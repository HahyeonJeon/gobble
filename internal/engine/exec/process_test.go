package exec

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

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

func TestProcessUnprovedPIDUnknown(t *testing.T) {
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
