package exec

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const pinnedAlpine = "alpine:3.21"

func TestDockerRunArgs(t *testing.T) {
	job := Job{
		Image:   pinnedAlpine,
		Argv:    []string{"cp", "in/sample.txt", "out/docker/sample.txt"},
		Isolate: "/iso",
	}
	args := dockerRunArgs(job)
	joined := strings.Join(args, " ")
	for _, banned := range []string{"--cpus", "--memory"} {
		if strings.Contains(joined, banned) {
			t.Fatalf("docker argv %v contains %s", args, banned)
		}
	}
	want := []string{
		"run", "-d",
		"--user", strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid()),
		"--network=none",
		"--entrypoint", "cp",
		"-v", "/iso:" + containerWorkDir,
		"-w", containerWorkDir,
		pinnedAlpine,
		"in/sample.txt", "out/docker/sample.txt",
	}
	if len(args) != len(want) {
		t.Fatalf("docker argv got %#v, want %#v", args, want)
	}
	for i, arg := range want {
		if args[i] != arg {
			t.Fatalf("docker argv got %#v, want %#v", args, want)
		}
	}
}

func TestDockerRunArgsNonZeroResources(t *testing.T) {
	job := Job{
		Image:       pinnedAlpine,
		Argv:        []string{"true"},
		CPU:         1.5,
		MemoryBytes: 512 << 20,
		Env:         map[string]string{"HOME": "/tmp", "FOO": "bar"},
		Isolate:     "/iso",
	}
	args := dockerRunArgs(job)
	if !hasArgPair(args, "--cpus", "1.5") {
		t.Fatalf("non-zero docker argv %v, want --cpus 1.5", args)
	}
	if !hasArgPair(args, "--memory", "536870912") {
		t.Fatalf("non-zero docker argv %v, want --memory 536870912", args)
	}
	if !hasArgPair(args, "-e", "FOO=bar") || !hasArgPair(args, "-e", "HOME=/tmp") {
		t.Fatalf("non-zero docker argv %v, want sorted -e KEY=value", args)
	}
}

func TestDockerRunArgsMemory15g(t *testing.T) {
	job := Job{
		Image:       pinnedAlpine,
		Argv:        []string{"true"},
		MemoryBytes: 1610612736,
		Isolate:     "/iso",
	}
	args := dockerRunArgs(job)
	if !hasArgPair(args, "--memory", "1610612736") {
		t.Fatalf("1.5g docker argv %v, want --memory 1610612736", args)
	}
}

func TestDockerRunArgsZeroResourcesOmitFlags(t *testing.T) {
	job := Job{
		Image:   pinnedAlpine,
		Argv:    []string{"true"},
		Isolate: "/iso",
	}
	args := dockerRunArgs(job)
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "--cpus") || strings.Contains(joined, "--memory") {
		t.Fatalf("zero docker argv %v contains resource flags", args)
	}
	job.MemoryBytes = 0
	job.Memory = "0m"
	args = dockerRunArgs(job)
	if strings.Contains(strings.Join(args, " "), "--memory") {
		t.Fatalf("zero-memory docker argv %v contains --memory", args)
	}
}

func hasArgPair(args []string, flag, value string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}

func TestDockerOperationsUseEngineClientEnv(t *testing.T) {
	orig := DockerCLI
	t.Cleanup(func() { DockerCLI = orig })
	type rec struct {
		args []string
		env  []string
	}
	var recs []rec
	DockerCLI = func(ctx context.Context, args []string, env []string, stdout, stderr io.Writer) (int, error) {
		recs = append(recs, rec{
			args: append([]string(nil), args...),
			env:  append([]string(nil), env...),
		})
		joined := strings.Join(args, " ")
		if len(args) > 0 && args[0] == "image" {
			return 0, nil
		}
		if len(args) > 0 && args[0] == "run" {
			_, _ = io.WriteString(stdout, "cid-submit\n")
			return 0, nil
		}
		if strings.Contains(joined, "{{.Image}}") {
			_, _ = io.WriteString(stdout, "sha256:launched\n")
			return 0, nil
		}
		if strings.Contains(joined, "State.Running") {
			_, _ = io.WriteString(stdout, "true 0\n")
			return 0, nil
		}
		if len(args) > 0 && args[0] == "kill" {
			return 0, nil
		}
		t.Fatalf("unexpected docker argv: %v", args)
		return -1, nil
	}
	d := NewDocker()
	taskEnv := map[string]string{
		"DOCKER_CONFIG":  "/task/docker",
		"DOCKER_CONTEXT": "task-context",
		"DOCKER_HOST":    "tcp://task.invalid:2375",
		"HOME":           "/task/home",
		"PATH":           "/task/bin",
		"TOKEN":          "task-token",
	}
	if _, _, err := d.Submit(context.Background(), Job{
		Image:   pinnedAlpine,
		Argv:    []string{"true"},
		Isolate: t.TempDir(),
		Env:     taskEnv,
	}); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if _, err := d.Poll(context.Background(), Handle{Identity: "poll", Backend: BackendDocker, RuntimeID: "cid-poll"}); err != nil {
		t.Fatalf("Poll() error = %v", err)
	}
	if err := d.Cancel(context.Background(), Handle{Identity: "cancel", Backend: BackendDocker, RuntimeID: "cid-cancel"}); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if _, err := d.Reconcile(context.Background(), Handle{Identity: "reconcile", Backend: BackendDocker, RuntimeID: "cid-reconcile"}); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	wantEnv := []string{"PATH=/usr/bin:/bin"}
	seen := map[string]bool{}
	for _, r := range recs {
		if strings.Join(r.env, "\x00") != strings.Join(wantEnv, "\x00") {
			t.Fatalf("docker client env for %v got %v, want %v", r.args, r.env, wantEnv)
		}
		for key, value := range taskEnv {
			if contains(r.env, key+"="+value) {
				t.Fatalf("docker client env for %v contains task %s", r.args, key)
			}
		}
		joined := strings.Join(r.args, " ")
		switch {
		case len(r.args) > 0 && r.args[0] == "run":
			seen["Submit"] = true
			if !contains(r.args, "--network=none") {
				t.Fatalf("Submit docker argv %v lacks --network=none", r.args)
			}
			for key, value := range taskEnv {
				if !hasArgPair(r.args, "-e", key+"="+value) {
					t.Fatalf("Submit docker argv %v lacks -e %s=value", r.args, key)
				}
			}
		case strings.Contains(joined, "State.Running") && r.args[len(r.args)-1] == "cid-poll":
			seen["Poll"] = true
		case len(r.args) > 0 && r.args[0] == "kill":
			seen["Cancel"] = true
		case strings.Contains(joined, "State.Running") && r.args[len(r.args)-1] == "cid-reconcile":
			seen["Reconcile"] = true
		}
	}
	for _, operation := range []string{"Submit", "Poll", "Cancel", "Reconcile"} {
		if !seen[operation] {
			t.Fatalf("docker calls did not record %s: %#v", operation, recs)
		}
	}
}

func TestDockerTerminalCleanupFailuresAreVisible(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		failure   string
	}{
		{name: "poll logs", operation: "Poll", failure: "logs"},
		{name: "poll remove", operation: "Poll", failure: "rm"},
		{name: "reconcile logs", operation: "Reconcile", failure: "logs"},
		{name: "reconcile remove", operation: "Reconcile", failure: "rm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			work := filepath.Join(root, "work")
			if err := os.Mkdir(work, 0o755); err != nil {
				t.Fatal(err)
			}
			orig := DockerCLI
			t.Cleanup(func() { DockerCLI = orig })
			calls := map[string]int{}
			DockerCLI = func(ctx context.Context, args []string, env []string, stdout, stderr io.Writer) (int, error) {
				joined := strings.Join(args, " ")
				switch {
				case strings.Contains(joined, "State.Running"):
					_, _ = io.WriteString(stdout, "false 0\n")
					return 0, nil
				case strings.Contains(joined, "Mounts"):
					_, _ = io.WriteString(stdout, work+"\n")
					return 0, nil
				case len(args) > 0 && args[0] == "logs":
					calls["logs"]++
					if tt.failure == "logs" {
						return 23, nil
					}
					return 0, nil
				case len(args) > 0 && args[0] == "rm":
					calls["rm"]++
					if tt.failure == "rm" {
						_, _ = io.WriteString(stderr, "remove denied\n")
						return 24, nil
					}
					return 0, nil
				default:
					t.Fatalf("unexpected docker argv: %v", args)
					return -1, nil
				}
			}

			d := NewDocker()
			h := Handle{Identity: "cleanup", Backend: BackendDocker, RuntimeID: "cid-cleanup"}
			var r Report
			var err error
			if tt.operation == "Poll" {
				r, err = d.Poll(context.Background(), h)
			} else {
				r, err = d.Reconcile(context.Background(), h)
			}
			if err == nil || !strings.Contains(err.Error(), "docker "+tt.failure) {
				t.Fatalf("%s() error = %v, want docker %s failure", tt.operation, err, tt.failure)
			}
			if r.Running || r.Exit != 0 {
				t.Fatalf("%s() report = %#v, want stopped exit 0", tt.operation, r)
			}
			if _, ok := d.done[h.RuntimeID]; ok {
				t.Fatalf("%s() cached cleanup failure as ordinary success", tt.operation)
			}
			if calls["logs"] != 1 || calls["rm"] != 1 {
				t.Fatalf("%s() cleanup calls = %v, want logs=1 rm=1", tt.operation, calls)
			}
		})
	}
}

func TestDockerSubmitPersistsLaunchedContainerImage(t *testing.T) {
	orig := DockerCLI
	t.Cleanup(func() { DockerCLI = orig })
	DockerCLI = func(ctx context.Context, args []string, env []string, stdout, stderr io.Writer) (int, error) {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "{{.Image}}") && strings.Contains(joined, "cid-launched") {
			_, _ = io.WriteString(stdout, "sha256:launched-container\n")
			return 0, nil
		}
		if strings.Contains(joined, "{{.Id}}") {
			_, _ = io.WriteString(stdout, "sha256:authored-tag\n")
			return 0, nil
		}
		if len(args) > 0 && args[0] == "run" {
			_, _ = io.WriteString(stdout, "cid-launched\n")
			return 0, nil
		}
		if len(args) >= 1 && args[0] == "image" {
			return 0, nil
		}
		return 0, nil
	}
	_, rep, err := NewDocker().Submit(context.Background(), Job{
		Image:   pinnedAlpine,
		Argv:    []string{"true"},
		Isolate: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if rep.ImageDigest != "sha256:launched-container" {
		t.Fatalf("ImageDigest got %q, want launched container id", rep.ImageDigest)
	}
}

func TestDockerPollLogCreateFailureIsEscapedPath(t *testing.T) {
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "secret.log")
	if err := os.WriteFile(sentinel, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	work := filepath.Join(root, "work")
	if err := os.Mkdir(work, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(sentinel, filepath.Join(root, "stdout")); err != nil {
		t.Fatal(err)
	}
	orig := DockerCLI
	t.Cleanup(func() { DockerCLI = orig })
	DockerCLI = func(ctx context.Context, args []string, env []string, stdout, stderr io.Writer) (int, error) {
		joined := strings.Join(args, " ")
		if len(args) > 0 && args[0] == "run" {
			_, _ = io.WriteString(stdout, "cid-log\n")
			return 0, nil
		}
		if strings.Contains(joined, "State.Running") {
			_, _ = io.WriteString(stdout, "false 0\n")
			return 0, nil
		}
		if strings.Contains(joined, containerWorkDir) && strings.Contains(joined, "Mounts") {
			_, _ = io.WriteString(stdout, work+"\n")
			return 0, nil
		}
		if strings.Contains(joined, "{{.Image}}") {
			_, _ = io.WriteString(stdout, "sha256:img\n")
			return 0, nil
		}
		if len(args) > 0 && (args[0] == "image" || args[0] == "logs" || args[0] == "rm") {
			return 0, nil
		}
		return 0, nil
	}
	d := NewDocker()
	h, _, err := d.Submit(context.Background(), Job{
		Image:   pinnedAlpine,
		Argv:    []string{"true"},
		Isolate: work,
	})
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	_, err = d.Poll(context.Background(), h)
	if !errors.Is(err, ErrEscapedPath) {
		t.Fatalf("Poll() error = %v, want ErrEscapedPath", err)
	}
	got, err := os.ReadFile(sentinel)
	if err != nil || string(got) != "keep" {
		t.Fatalf("sentinel got %q, want keep", got)
	}
}

func TestDockerSubmitPreservesContextDeadline(t *testing.T) {
	orig := DockerCLI
	t.Cleanup(func() { DockerCLI = orig })
	DockerCLI = func(ctx context.Context, args []string, env []string, stdout, stderr io.Writer) (int, error) {
		<-ctx.Done()
		return -1, ctx.Err()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	_, _, err := NewDocker().Submit(ctx, Job{Image: pinnedAlpine, Argv: []string{"true"}, Isolate: t.TempDir()})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Submit() error = %v, want context.DeadlineExceeded", err)
	}
}

func TestDockerSubmitKilledCLIReturnsContextErr(t *testing.T) {
	orig := DockerCLI
	t.Cleanup(func() { DockerCLI = orig })
	DockerCLI = func(ctx context.Context, args []string, env []string, stdout, stderr io.Writer) (int, error) {
		<-ctx.Done()
		return -1, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	_, _, err := NewDocker().Submit(ctx, Job{Image: pinnedAlpine, Argv: []string{"true"}, Isolate: t.TempDir()})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Submit() killed CLI error = %v, want context.DeadlineExceeded", err)
	}
}

func TestRunDockerCLICanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := runDockerCLI(ctx, []string{"version"}, nil, io.Discard, io.Discard)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runDockerCLI() error = %v, want context.Canceled", err)
	}
}

func TestEmptyImageNeverInvokesDockerAdapter(t *testing.T) {
	orig := DockerCLI
	t.Cleanup(func() { DockerCLI = orig })
	DockerCLI = func(ctx context.Context, args []string, env []string, stdout, stderr io.Writer) (int, error) {
		t.Fatalf("docker invoked for empty Image: %v", args)
		return -1, nil
	}
	ex := Local()
	h, _, err := ex.Submit(t.Context(), Job{Identity: "copy", Isolate: t.TempDir(), Argv: []string{"true"}})
	if err != nil {
		t.Fatalf("process Submit() error = %v", err)
	}
	if h.Backend != BackendProcess {
		t.Fatalf("backend got %q, want %s", h.Backend, BackendProcess)
	}
	_ = ex.Cancel(t.Context(), h)
	for i := 0; i < 50; i++ {
		r, err := ex.Poll(t.Context(), h)
		if err != nil {
			t.Fatalf("Poll() error = %v", err)
		}
		if !r.Running {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("process still running")
}
