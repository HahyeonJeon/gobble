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

// Standard setup for adapter unit tests; submission-order and recovery tests
// provide their own complete fake daemon instead.
func dockerSetupFixture(args []string, out io.Writer) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "context":
		_, _ = io.WriteString(out, "unix:///var/run/docker.sock\n")
	case "info":
		_, _ = io.WriteString(out, "test-daemon\n")
	case "start":
	default:
		return false
	}
	return true
}

func TestDockerCreateArgs(t *testing.T) {
	job := Job{
		Submission: &Submission{Token: strings.Repeat("a", 32)},
		Record:     func(context.Context, Handle, Report) error { return nil },
		Image:      pinnedAlpine,
		Argv:       []string{"cp", "in/sample.txt", "out/docker/sample.txt"},
		Isolate:    "/iso",
	}
	args := dockerCreateArgs(job)
	joined := strings.Join(args, " ")
	for _, banned := range []string{"--cpus", "--memory"} {
		if strings.Contains(joined, banned) {
			t.Fatalf("docker argv %v contains %s", args, banned)
		}
	}
	want := []string{
		"create",
		"--platform", "linux/amd64",
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

func TestDockerCreateArgsNonZeroResources(t *testing.T) {
	job := Job{
		Submission:  &Submission{Token: strings.Repeat("a", 32)},
		Record:      func(context.Context, Handle, Report) error { return nil },
		Image:       pinnedAlpine,
		Argv:        []string{"true"},
		CPU:         1.5,
		MemoryBytes: 512 << 20,
		Env:         map[string]string{"HOME": "/tmp", "FOO": "bar"},
		Isolate:     "/iso",
	}
	args := dockerCreateArgs(job)
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

func TestDockerCreateArgsMemory15g(t *testing.T) {
	job := Job{
		Submission:  &Submission{Token: strings.Repeat("a", 32)},
		Record:      func(context.Context, Handle, Report) error { return nil },
		Image:       pinnedAlpine,
		Argv:        []string{"true"},
		MemoryBytes: 1610612736,
		Isolate:     "/iso",
	}
	args := dockerCreateArgs(job)
	if !hasArgPair(args, "--memory", "1610612736") {
		t.Fatalf("1.5g docker argv %v, want --memory 1610612736", args)
	}
}

func TestDockerCreateArgsZeroResourcesOmitFlags(t *testing.T) {
	job := Job{
		Submission: &Submission{Token: strings.Repeat("a", 32)},
		Record:     func(context.Context, Handle, Report) error { return nil },
		Image:      pinnedAlpine,
		Argv:       []string{"true"},
		Isolate:    "/iso",
	}
	args := dockerCreateArgs(job)
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "--cpus") || strings.Contains(joined, "--memory") {
		t.Fatalf("zero docker argv %v contains resource flags", args)
	}
	job.MemoryBytes = 0
	job.Memory = "0m"
	args = dockerCreateArgs(job)
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
		if dockerSetupFixture(args, stdout) {
			return 0, nil
		}
		recs = append(recs, rec{
			args: append([]string(nil), args...),
			env:  append([]string(nil), env...),
		})
		joined := strings.Join(args, " ")
		if len(args) > 0 && args[0] == "image" {
			return 0, nil
		}
		if len(args) > 0 && args[0] == "create" {
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
		Submission: &Submission{Token: strings.Repeat("a", 32)},
		Record:     func(context.Context, Handle, Report) error { return nil },
		Image:      pinnedAlpine,
		Argv:       []string{"true"},
		Isolate:    t.TempDir(),
		Env:        taskEnv,
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

	seen := map[string]bool{}
	for _, r := range recs {
		wantEnv := dockerClientEnv()
		if r.args[0] == "image" || r.args[0] == "create" || r.args[len(r.args)-1] == "cid-submit" {
			wantEnv = append(wantEnv, "DOCKER_HOST=unix:///var/run/docker.sock")
		}
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
		case len(r.args) > 0 && r.args[0] == "create":
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
		name       string
		operation  string
		logsFail   bool
		removeFail bool
	}{
		{name: "poll logs", operation: "Poll", logsFail: true},
		{name: "poll remove", operation: "Poll", removeFail: true},
		{name: "poll both", operation: "Poll", logsFail: true, removeFail: true},
		{name: "reconcile logs", operation: "Reconcile", logsFail: true},
		{name: "reconcile remove", operation: "Reconcile", removeFail: true},
		{name: "reconcile both", operation: "Reconcile", logsFail: true, removeFail: true},
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
				if dockerSetupFixture(args, stdout) {
					return 0, nil
				}
				joined := strings.Join(args, " ")
				switch {
				case strings.Contains(joined, "State.Running"):
					_, _ = io.WriteString(stdout, "false 17\n")
					return 0, nil
				case strings.Contains(joined, "Mounts"):
					_, _ = io.WriteString(stdout, work+"\n")
					return 0, nil
				case len(args) > 0 && args[0] == "logs":
					calls["logs"]++
					_, _ = io.WriteString(stdout, "partial stdout\n")
					_, _ = io.WriteString(stderr, "partial stderr\n")
					if tt.logsFail {
						return 23, nil
					}
					return 0, nil
				case len(args) > 0 && args[0] == "rm":
					calls["rm"]++
					if tt.removeFail {
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
			if err != nil {
				t.Fatalf("%s() error = %v, want proved-stop cleanup success", tt.operation, err)
			}
			if r.Running || r.Exit != 17 || r.Message != "exit 17" {
				t.Fatalf("%s() report = %#v, want stopped exit 17", tt.operation, r)
			}
			wantReason := ""
			if tt.logsFail {
				wantReason = "log-copy-failed"
			}
			if r.Reason != wantReason {
				t.Fatalf("%s() reason = %q, want %q", tt.operation, r.Reason, wantReason)
			}
			wantRuntimeID := ""
			if tt.removeFail {
				wantRuntimeID = h.RuntimeID
			}
			if r.RuntimeID != wantRuntimeID {
				t.Fatalf("%s() runtime_id = %q, want %q", tt.operation, r.RuntimeID, wantRuntimeID)
			}
			cached, ok := d.done[h.RuntimeID]
			if !ok || cached != r {
				t.Fatalf("%s() cache = %#v, want %#v", tt.operation, cached, r)
			}
			if calls["logs"] != 1 || calls["rm"] != 1 {
				t.Fatalf("%s() cleanup calls = %v, want logs=1 rm=1", tt.operation, calls)
			}
		})
	}
}

func TestDockerCachedLeftoverRetriesRemoveAndClearsRuntimeID(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "work")
	if err := os.Mkdir(work, 0o755); err != nil {
		t.Fatal(err)
	}
	orig := DockerCLI
	t.Cleanup(func() { DockerCLI = orig })
	calls := map[string]int{}
	DockerCLI = func(ctx context.Context, args []string, env []string, stdout, stderr io.Writer) (int, error) {
		if dockerSetupFixture(args, stdout) {
			return 0, nil
		}
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "State.Running"):
			calls["inspect"]++
			_, _ = io.WriteString(stdout, "false 19\n")
			return 0, nil
		case strings.Contains(joined, "Mounts"):
			_, _ = io.WriteString(stdout, work+"\n")
			return 0, nil
		case len(args) > 0 && args[0] == "logs":
			calls["logs"]++
			return 0, nil
		case len(args) > 0 && args[0] == "rm":
			calls["rm"]++
			if calls["rm"] == 1 {
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
	first, err := d.Poll(context.Background(), h)
	if err != nil || first.Exit != 19 || first.RuntimeID != h.RuntimeID {
		t.Fatalf("first Poll() report=%#v error=%v, want cached leftover exit 19", first, err)
	}
	second, err := d.Reconcile(context.Background(), h)
	if err != nil || second.Exit != 19 || second.RuntimeID != "" {
		t.Fatalf("second Reconcile() report=%#v error=%v, want cleared leftover exit 19", second, err)
	}
	third, err := d.Poll(context.Background(), h)
	if err != nil || third != second {
		t.Fatalf("third Poll() report=%#v error=%v, want cached %#v", third, err, second)
	}
	if calls["inspect"] != 1 || calls["logs"] != 1 || calls["rm"] != 2 {
		t.Fatalf("docker calls = %v, want inspect=1 logs=1 rm=2", calls)
	}
}

func TestDockerStoppedUnknownExitIsUnproved(t *testing.T) {
	for _, output := range []string{"false\n", "false nope\n", "false 0 extra\n"} {
		t.Run(strings.TrimSpace(output), func(t *testing.T) {
			orig := DockerCLI
			t.Cleanup(func() { DockerCLI = orig })
			calls := 0
			DockerCLI = func(ctx context.Context, args []string, env []string, stdout, stderr io.Writer) (int, error) {
				if dockerSetupFixture(args, stdout) {
					return 0, nil
				}
				calls++
				if !strings.Contains(strings.Join(args, " "), "State.Running") {
					t.Fatalf("unexpected docker argv: %v", args)
				}
				_, _ = io.WriteString(stdout, output)
				return 0, nil
			}
			d := NewDocker()
			h := Handle{Identity: "unknown", Backend: BackendDocker, RuntimeID: "cid-unknown"}
			r, err := d.Poll(context.Background(), h)
			if err == nil {
				t.Fatalf("Poll() report=%#v error=nil, want unproved inspect error", r)
			}
			if _, ok := d.done[h.RuntimeID]; ok {
				t.Fatal("Poll() cached unproved stop")
			}
			if calls != 1 {
				t.Fatalf("docker calls=%d, want inspect only", calls)
			}
		})
	}
}

func TestDockerSubmitPersistsLaunchedContainerImage(t *testing.T) {
	orig := DockerCLI
	t.Cleanup(func() { DockerCLI = orig })
	DockerCLI = func(ctx context.Context, args []string, env []string, stdout, stderr io.Writer) (int, error) {
		if dockerSetupFixture(args, stdout) {
			return 0, nil
		}
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "{{.Image}}") && strings.Contains(joined, "cid-launched") {
			_, _ = io.WriteString(stdout, "sha256:launched-container\n")
			return 0, nil
		}
		if strings.Contains(joined, "{{.Id}}") {
			_, _ = io.WriteString(stdout, "sha256:authored-tag\n")
			return 0, nil
		}
		if len(args) > 0 && args[0] == "create" {
			_, _ = io.WriteString(stdout, "cid-launched\n")
			return 0, nil
		}
		if len(args) >= 1 && args[0] == "image" {
			return 0, nil
		}
		return 0, nil
	}
	_, rep, err := NewDocker().Submit(context.Background(), Job{
		Submission: &Submission{Token: strings.Repeat("a", 32)},
		Record:     func(context.Context, Handle, Report) error { return nil },
		Image:      pinnedAlpine,
		Argv:       []string{"true"},
		Isolate:    t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if rep.ImageDigest != "sha256:launched-container" {
		t.Fatalf("ImageDigest got %q, want launched container id", rep.ImageDigest)
	}
}

func TestDockerPollLogCreateFailureMarksIncompleteLogs(t *testing.T) {
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
		if dockerSetupFixture(args, stdout) {
			return 0, nil
		}
		joined := strings.Join(args, " ")
		if len(args) > 0 && args[0] == "create" {
			_, _ = io.WriteString(stdout, "cid-log\n")
			return 0, nil
		}
		if args[0] == "container" {
			_, _ = io.WriteString(stdout, "cid-log\n")
			return 0, nil
		}
		if strings.Contains(joined, "Config.Labels") {
			_, _ = io.WriteString(stdout, strings.Repeat("a", 32))
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
		Submission: &Submission{Token: strings.Repeat("a", 32)},
		Record:     func(context.Context, Handle, Report) error { return nil },
		Image:      pinnedAlpine,
		Argv:       []string{"true"},
		Isolate:    work,
	})
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	report, err := d.Poll(context.Background(), h)
	if err != nil {
		t.Fatalf("Poll() error = %v, want proved-stop cleanup success", err)
	}
	if report.Reason != "log-copy-failed" || report.RuntimeID != "" || report.Running {
		t.Fatalf("Poll() report = %#v, want stopped incomplete logs with removed container", report)
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
		if dockerSetupFixture(args, stdout) {
			return 0, nil
		}
		<-ctx.Done()
		return -1, ctx.Err()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	_, _, err := NewDocker().Submit(ctx, Job{Submission: &Submission{Token: strings.Repeat("a", 32)}, Record: func(context.Context, Handle, Report) error { return nil }, Image: pinnedAlpine, Argv: []string{"true"}, Isolate: t.TempDir()})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Submit() error = %v, want context.DeadlineExceeded", err)
	}
}

func TestDockerSubmitKilledCLIReturnsContextErr(t *testing.T) {
	orig := DockerCLI
	t.Cleanup(func() { DockerCLI = orig })
	DockerCLI = func(ctx context.Context, args []string, env []string, stdout, stderr io.Writer) (int, error) {
		if dockerSetupFixture(args, stdout) {
			return 0, nil
		}
		<-ctx.Done()
		return -1, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	_, _, err := NewDocker().Submit(ctx, Job{Submission: &Submission{Token: strings.Repeat("a", 32)}, Record: func(context.Context, Handle, Report) error { return nil }, Image: pinnedAlpine, Argv: []string{"true"}, Isolate: t.TempDir()})
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
		if dockerSetupFixture(args, stdout) {
			return 0, nil
		}
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
