package exec

import (
	"errors"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCreateAttemptFileRefusesSymlink(t *testing.T) {
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "secret.log")
	if err := os.WriteFile(sentinel, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "stdout")
	if err := os.Symlink(sentinel, path); err != nil {
		t.Fatal(err)
	}
	f, err := createAttemptFile(path)
	if f != nil {
		f.Close()
	}
	if !errors.Is(err, ErrEscapedPath) {
		t.Fatalf("createAttemptFile() error = %v, want ErrEscapedPath", err)
	}
	got, err := os.ReadFile(sentinel)
	if err != nil || string(got) != "keep" {
		t.Fatalf("sentinel got %q, want keep", got)
	}
}

func TestCreateAttemptFileRefusesExistingRegular(t *testing.T) {
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "secret.log")
	if err := os.WriteFile(sentinel, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "stdout")
	if err := os.WriteFile(path, []byte("prior"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := createAttemptFile(path)
	if f != nil {
		f.Close()
	}
	if !errors.Is(err, ErrEscapedPath) {
		t.Fatalf("createAttemptFile() error = %v, want ErrEscapedPath", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "prior" {
		t.Fatalf("existing log got %q, want prior", got)
	}
	got, err = os.ReadFile(sentinel)
	if err != nil || string(got) != "keep" {
		t.Fatalf("sentinel got %q, want keep", got)
	}
}

func TestCreateAttemptFileRefusesHardlink(t *testing.T) {
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "secret.log")
	if err := os.WriteFile(sentinel, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "stdout")
	if err := os.Link(sentinel, path); err != nil {
		t.Fatal(err)
	}
	f, err := createAttemptFile(path)
	if f != nil {
		f.Close()
	}
	if !errors.Is(err, ErrEscapedPath) {
		t.Fatalf("createAttemptFile() error = %v, want ErrEscapedPath", err)
	}
	got, err := os.ReadFile(sentinel)
	if err != nil || string(got) != "keep" {
		t.Fatalf("sentinel got %q, want keep", got)
	}
}

func TestProcessSubmitRefusesEscapingLogSymlink(t *testing.T) {
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "secret.log")
	if err := os.WriteFile(sentinel, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	iso := filepath.Join(root, "work")
	if err := os.Mkdir(iso, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sentinel, filepath.Join(root, "stdout")); err != nil {
		t.Fatal(err)
	}
	_, _, err := NewProcess().Submit(t.Context(), Job{
		Identity: "copy",
		Isolate:  iso,
		Argv:     []string{"true"},
	})
	if !errors.Is(err, ErrEscapedPath) {
		t.Fatalf("Submit() error = %v, want ErrEscapedPath", err)
	}
	got, err := os.ReadFile(sentinel)
	if err != nil || string(got) != "keep" {
		t.Fatalf("sentinel got %q, want keep", got)
	}
}

func TestProcessCancelKillsGroup(t *testing.T) {
	dir := t.TempDir()
	p := NewProcess()
	h, _, err := p.Submit(t.Context(), Job{
		Identity: "sleep",
		Isolate:  dir,
		Argv:     []string{"sh", "-c", "sleep 30"},
	})
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if h.Backend != BackendProcess {
		t.Fatalf("backend got %q, want %s", h.Backend, BackendProcess)
	}
	pid, err := strconv.Atoi(h.RuntimeID)
	if err != nil || pid <= 0 {
		t.Fatalf("runtime_id got %q, want pid", h.RuntimeID)
	}
	if err := p.Cancel(t.Context(), h); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		r, err := p.Poll(t.Context(), h)
		if err != nil {
			t.Fatalf("Poll() error = %v", err)
		}
		if !r.Running {
			err := syscall.Kill(pid, 0)
			if err == nil {
				t.Fatalf("pid %d still live after cancel", pid)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("process still running after cancel")
}

func TestProcessCancelAfterWaitDoesNotSignal(t *testing.T) {
	orig := signalProcess
	t.Cleanup(func() { signalProcess = orig })
	const sentinelPID = 4242
	var signaled []int
	signalProcess = func(pid int, sig syscall.Signal) error {
		signaled = append(signaled, pid)
		return nil
	}
	done := make(chan struct{})
	close(done)
	p := NewProcess()
	p.live[strconv.Itoa(sentinelPID)] = &proc{
		cmd:  &osexec.Cmd{Process: &os.Process{Pid: sentinelPID}},
		done: done,
	}
	h := Handle{Identity: "complete", Backend: BackendProcess, RuntimeID: strconv.Itoa(sentinelPID)}
	if err := p.Cancel(t.Context(), h); err != nil {
		t.Fatalf("Cancel() after Wait error = %v", err)
	}
	if len(signaled) != 0 {
		t.Fatalf("Cancel() after Wait signaled sentinel PID: %v", signaled)
	}
}

func TestProcessCancelAfterWaitReturnDoesNotSignal(t *testing.T) {
	orig := signalProcess
	t.Cleanup(func() { signalProcess = orig })
	var signaled []int
	signalProcess = func(pid int, sig syscall.Signal) error {
		signaled = append(signaled, pid)
		return nil
	}
	root := t.TempDir()
	isolate := filepath.Join(root, "work")
	if err := os.Mkdir(isolate, 0o755); err != nil {
		t.Fatal(err)
	}
	p := NewProcess()
	h, _, err := p.Submit(t.Context(), Job{
		Identity: "wait-returned",
		Isolate:  isolate,
		Argv:     []string{"sh", "-c", "sleep 30"},
	})
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	pid, err := strconv.Atoi(h.RuntimeID)
	if err != nil || pid <= 0 {
		t.Fatalf("runtime_id got %q, want pid", h.RuntimeID)
	}
	pr := p.live[h.RuntimeID]
	t.Cleanup(func() { _ = pr.cmd.Process.Kill() })
	pr.mu.Lock()
	locked := true
	defer func() {
		if locked {
			pr.mu.Unlock()
		}
	}()
	if err := orig(-pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("stop test process group: %v", err)
	}
	select {
	case <-pr.waited:
	case <-time.After(2 * time.Second):
		t.Fatal("cmd.Wait() did not return")
	}
	select {
	case <-pr.done:
		t.Fatal("done closed before pending Wait cleanup completed")
	default:
	}
	if err := p.Cancel(t.Context(), h); err != nil {
		t.Fatalf("Cancel() after Wait return error = %v", err)
	}
	if len(signaled) != 0 {
		t.Fatalf("Cancel() after Wait return signaled sentinel PID: %v", signaled)
	}
	pr.mu.Unlock()
	locked = false
	select {
	case <-pr.done:
	case <-time.After(2 * time.Second):
		t.Fatal("Wait cleanup did not complete")
	}
}

func TestProcessUnprovedPIDUnknown(t *testing.T) {
	orig := signalProcess
	t.Cleanup(func() { signalProcess = orig })
	var signaled []int
	signalProcess = func(pid int, sig syscall.Signal) error {
		signaled = append(signaled, pid)
		return nil
	}
	p := NewProcess()
	h := Handle{Identity: "copy", Backend: BackendProcess, RuntimeID: "1"}
	if err := p.Cancel(t.Context(), h); err == nil {
		t.Fatal("Cancel unproved PID error = nil, want unproved")
	}
	if _, err := p.Poll(t.Context(), h); err == nil {
		t.Fatal("Poll unproved PID error = nil, want unproved")
	}
	if _, err := p.Reconcile(t.Context(), h); err == nil {
		t.Fatal("Reconcile unproved PID error = nil, want unproved")
	}
	if len(signaled) != 0 {
		t.Fatalf("unproved operations signaled PID: %v", signaled)
	}
}

func TestResolveArgv0IgnoresParentPATH(t *testing.T) {
	dir := t.TempDir()
	poison := filepath.Join(dir, "sh")
	if err := os.WriteFile(poison, []byte("#!/bin/sh\nexit 42\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":/usr/bin:/bin")
	got, err := ResolveArgv0("sh", nil)
	if err != nil {
		t.Fatalf("ResolveArgv0() error = %v", err)
	}
	if got == poison {
		t.Fatalf("ResolveArgv0 used parent PATH binary %q", got)
	}
	if !strings.HasPrefix(got, "/usr/bin/") && !strings.HasPrefix(got, "/bin/") {
		t.Fatalf("ResolveArgv0 got %q, want a /usr/bin or /bin path", got)
	}
	declared := filepath.Join(dir, "bin")
	if err := os.Mkdir(declared, 0o755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(declared, "sh")
	if err := os.WriteFile(want, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err = ResolveArgv0("sh", map[string]string{"PATH": declared})
	if err != nil {
		t.Fatalf("declared PATH ResolveArgv0() error = %v", err)
	}
	if got != want {
		t.Fatalf("declared PATH got %q, want %q", got, want)
	}
}

func TestProcessEnvDefaultPATH(t *testing.T) {
	got := processEnv(nil)
	if len(got) != 1 || got[0] != "PATH=/usr/bin:/bin" {
		t.Fatalf("default env got %v, want PATH=/usr/bin:/bin", got)
	}
	got = processEnv(map[string]string{"PATH": "/opt/bin", "FOO": "bar"})
	joined := ""
	for _, e := range got {
		joined += e + "\n"
	}
	if !contains(got, "PATH=/opt/bin") || contains(got, "PATH=/usr/bin:/bin") {
		t.Fatalf("author PATH env got %v", got)
	}
	if !contains(got, "FOO=bar") {
		t.Fatalf("declared env got %v, want FOO=bar", got)
	}
	os.Setenv("SECRET", "no")
	t.Cleanup(func() { os.Unsetenv("SECRET") })
	for _, e := range processEnv(map[string]string{"HOME": "/tmp"}) {
		if e == "SECRET=no" {
			t.Fatalf("inherited host env: %v", got)
		}
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
