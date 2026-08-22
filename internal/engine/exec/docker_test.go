package exec

import (
	"context"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
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
	if !hasArgPair(args, "-e", "FOO") || !hasArgPair(args, "-e", "HOME") {
		t.Fatalf("non-zero docker argv %v, want sorted -e KEY", args)
	}
	if hasArgPair(args, "-e", "HOME=/tmp") || hasArgPair(args, "-e", "FOO=bar") {
		t.Fatalf("docker argv %v contains env values", args)
	}
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "/tmp") || strings.Contains(joined, "bar") {
		t.Fatalf("docker argv %v contains env values", args)
	}
	env := dockerClientEnv(job.Env)
	if !contains(env, "HOME=/tmp") || !contains(env, "FOO=bar") {
		t.Fatalf("docker client env got %v, want values", env)
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

func TestDockerSubmitEnvIsCallScoped(t *testing.T) {
	orig := DockerCLI
	t.Cleanup(func() { DockerCLI = orig })
	type rec struct {
		args []string
		env  []string
	}
	var mu sync.Mutex
	var recs []rec
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	DockerCLI = func(ctx context.Context, args []string, env []string, stdout, stderr io.Writer) (int, error) {
		cpArgs := append([]string(nil), args...)
		cpEnv := append([]string(nil), env...)
		if len(args) > 0 && args[0] == "run" {
			entered <- struct{}{}
			<-release
		}
		mu.Lock()
		recs = append(recs, rec{args: cpArgs, env: cpEnv})
		mu.Unlock()
		if len(args) > 0 && args[0] == "run" {
			id := "cid-a"
			for _, e := range env {
				if strings.HasPrefix(e, "TOKEN=") {
					id = "cid-" + strings.TrimPrefix(e, "TOKEN=")
				}
			}
			_, _ = io.WriteString(stdout, id+"\n")
			return 0, nil
		}
		if strings.Contains(strings.Join(args, " "), "{{.Image}}") {
			_, _ = io.WriteString(stdout, "sha256:launched\n")
			return 0, nil
		}
		if len(args) >= 1 && args[0] == "image" {
			return 0, nil
		}
		return 0, nil
	}
	d := NewDocker()
	var wg sync.WaitGroup
	errc := make(chan error, 2)
	jobs := []Job{
		{Image: pinnedAlpine, Argv: []string{"true"}, Isolate: t.TempDir(), Env: map[string]string{"TOKEN": "alpha"}},
		{Image: pinnedAlpine, Argv: []string{"true"}, Isolate: t.TempDir(), Env: map[string]string{"TOKEN": "beta"}},
	}
	for i := range jobs {
		wg.Add(1)
		go func(job Job) {
			defer wg.Done()
			_, _, err := d.Submit(context.Background(), job)
			errc <- err
		}(jobs[i])
	}
	<-entered
	<-entered
	close(release)
	wg.Wait()
	close(errc)
	for err := range errc {
		if err != nil {
			t.Fatalf("Submit() error = %v", err)
		}
	}
	var runEnvs [][]string
	mu.Lock()
	for _, r := range recs {
		if len(r.args) > 0 && r.args[0] == "run" {
			runEnvs = append(runEnvs, r.env)
			joined := strings.Join(r.args, " ")
			if strings.Contains(joined, "alpha") || strings.Contains(joined, "beta") {
				t.Fatalf("run argv contains env value: %v", r.args)
			}
		}
	}
	mu.Unlock()
	if len(runEnvs) != 2 {
		t.Fatalf("run calls got %d, want 2", len(runEnvs))
	}
	seen := map[string]bool{}
	for _, env := range runEnvs {
		hasA, hasB := contains(env, "TOKEN=alpha"), contains(env, "TOKEN=beta")
		if hasA && hasB {
			t.Fatalf("mixed env on one Submit: %v", env)
		}
		if hasA {
			seen["alpha"] = true
		}
		if hasB {
			seen["beta"] = true
		}
	}
	if !seen["alpha"] || !seen["beta"] {
		t.Fatalf("env vectors got %#v, want alpha and beta unmixed", runEnvs)
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
