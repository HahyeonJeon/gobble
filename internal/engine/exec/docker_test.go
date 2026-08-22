package exec

import (
	"context"
	"errors"
	"io"
	"os"
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
		Image:   pinnedAlpine,
		Argv:    []string{"true"},
		CPU:     1.5,
		Memory:  "512m",
		Env:     map[string]string{"HOME": "/tmp", "FOO": "bar"},
		Isolate: "/iso",
	}
	args := dockerRunArgs(job)
	if !hasArgPair(args, "--cpus", "1.5") {
		t.Fatalf("non-zero docker argv %v, want --cpus 1.5", args)
	}
	if !hasArgPair(args, "--memory", "512m") {
		t.Fatalf("non-zero docker argv %v, want --memory 512m", args)
	}
	if !hasArgPair(args, "-e", "HOME=/tmp") || !hasArgPair(args, "-e", "FOO=bar") {
		t.Fatalf("non-zero docker argv %v, want -e HOME=/tmp and -e FOO=bar", args)
	}
	for i, arg := range args {
		if arg == "-e" && i+1 < len(args) && !strings.Contains(args[i+1], "=") {
			t.Fatalf("value-less -e in %v", args)
		}
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

func TestDockerSubmitPreservesContextDeadline(t *testing.T) {
	orig := DockerCLI
	t.Cleanup(func() { DockerCLI = orig })
	DockerCLI = func(ctx context.Context, args []string, stdout, stderr io.Writer) (int, error) {
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
	DockerCLI = func(ctx context.Context, args []string, stdout, stderr io.Writer) (int, error) {
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
	_, err := runDockerCLI(ctx, []string{"version"}, io.Discard, io.Discard)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runDockerCLI() error = %v, want context.Canceled", err)
	}
}

func TestEmptyImageNeverInvokesDockerAdapter(t *testing.T) {
	orig := DockerCLI
	t.Cleanup(func() { DockerCLI = orig })
	DockerCLI = func(ctx context.Context, args []string, stdout, stderr io.Writer) (int, error) {
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
