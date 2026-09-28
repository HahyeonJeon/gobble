package appservice

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestRunDiscoveryDoesNotBlockOnSpecialFiles(t *testing.T) {
	s := testService(t)
	root := t.TempDir()
	p := testProject(t, s, root)
	path := filepath.Join(root, "runs")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := s.store.listRuns(p.ProjectID)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		// Release a regressed blocking reader before reporting the failure.
		if writer, err := os.OpenFile(path, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			writer.Close()
		}
		t.Fatal("Run discovery blocked on a FIFO")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), path); err != nil {
		t.Fatal(err)
	}
	_, err := s.store.listRuns(p.ProjectID)
	wantCode(t, err, "outside_project")
}

type runtimeFixture struct {
	mu                   sync.Mutex
	image, daemon, runID string
	match                bool
	monitorSchema        int
	attempt              int
	commands             [][]string
}

func newRuntimeFixture() *runtimeFixture {
	return &runtimeFixture{image: "sha256:" + strings.Repeat("a", 64), daemon: "fixture-daemon", runID: "engine-run", match: true, monitorSchema: 2, attempt: 2}
}
func (f *runtimeFixture) command(ctx context.Context, args, env []string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commands = append(f.commands, append([]string{}, args...))
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch args[0] {
	case "context":
		return []byte("unix:///fixture/docker.sock"), nil
	case "info":
		return []byte(f.daemon + " linux"), nil
	case "image":
		return []byte(f.image + " linux/amd64"), nil
	case "rm":
		return nil, nil
	case "run":
		if !contains(env, "DOCKER_HOST=unix:///fixture/docker.sock") {
			return nil, errors.New("query did not use registered endpoint")
		}
		for i, arg := range args {
			if arg == "inspect" && i+1 < len(args) {
				if args[i+1] == "identity" {
					return json.Marshal(map[string]any{"schema_version": 2, "view": "identity", "match": f.match, "required": map[string]string{"gobble_module": "fixture", "identity_mode": "local-pin"}})
				}
				return json.Marshal(map[string]any{"schema_version": f.monitorSchema, "snapshot": "snapshot-4", "run": map[string]string{"id": f.runID, "status": "running"}, "tasks": []any{map[string]any{"identity": "align", "attempt": f.attempt}}, "logs": []any{map[string]any{"identity": "align", "stdout_tail": "sample S03", "stdout_size": 8000, "stderr_tail": "", "stderr_size": 0}}})
			}
		}
	}
	return nil, errors.New("unexpected command")
}
func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func attachFixtureRun(t *testing.T) (*Service, *runtimeFixture, Project, RunRegistration) {
	t.Helper()
	t.Setenv("DOCKER_HOST", "")
	t.Setenv("DOCKER_CONTEXT", "")
	s := testService(t)
	fixture := newRuntimeFixture()
	s.runtime.command = fixture.command
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "runs", "existing", ".gobble"), 0o700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{"format": 1, "image": fixture.image, "daemon": fixture.daemon})
	writeTestFile(t, filepath.Join(root, ".gobble-runtime.json"), raw)
	p := testProject(t, s, root)
	listed, err := s.store.listRuns(p.ProjectID)
	if err != nil || len(listed.Candidates) != 1 {
		t.Fatalf("discovery: %+v %v", listed, err)
	}
	run, err := s.attachRun(t.Context(), p.ProjectID, listed.Candidates[0].WorkspaceResourceID, "req_attach")
	if err != nil {
		t.Fatal(err)
	}
	return s, fixture, p, run
}

func TestRunAttachmentAndReadOnlyPinnedQuery(t *testing.T) {
	s, fixture, p, run := attachFixtureRun(t)
	same, err := s.attachRun(t.Context(), p.ProjectID, run.WorkspaceResourceID, "req_attach")
	if err != nil || same != run {
		t.Fatalf("attachment retry: %+v %v", same, err)
	}
	snapshot, err := s.coalescedQuery(t.Context(), p.ProjectID, run.RunRef, "align", 2)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.EngineRevision != "snapshot-4" || snapshot.Availability != "available" || snapshot.ObservedAt <= 0 {
		t.Fatalf("snapshot: %+v", snapshot)
	}
	if snapshot.RuntimeBinding.ImageID != fixture.image || snapshot.RuntimeBinding.WorkspacePath != "/gobble/project/runs/existing" {
		t.Fatalf("binding: %+v", snapshot.RuntimeBinding)
	}
	for _, args := range fixture.commands {
		if args[0] == "run" {
			for _, required := range []string{"--read-only", "--network=none", "--pull=never", "--cap-drop=ALL", "GOBBLE_CONTAINER_BOOTSTRAP=0", "/usr/local/bin/gobble"} {
				if !contains(args, required) {
					t.Fatalf("missing %s in query", required)
				}
			}
			mountReadOnly := false
			for _, arg := range args {
				if strings.Contains(arg, "dst=/gobble/project,readonly") {
					mountReadOnly = true
				}
				if strings.Contains(arg, "/var/run/docker.sock") {
					t.Fatal("query mounted the Docker socket")
				}
			}
			if !mountReadOnly {
				t.Fatal("query omitted readonly project mount")
			}
		}
	}
}

func TestRunReadCancellationPreservesSharedQuery(t *testing.T) {
	s, fixture, p, run := attachFixtureRun(t)
	started, release := make(chan struct{}), make(chan struct{})
	var monitors atomic.Int32
	s.runtime.command = func(ctx context.Context, args, env []string) ([]byte, error) {
		if args[0] == "run" && contains(args, "monitor") {
			if monitors.Add(1) == 1 {
				close(started)
			}
			select {
			case <-release:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		return fixture.command(ctx, args, env)
	}
	done := make(chan error, 1)
	go func() {
		_, err := s.coalescedQuery(t.Context(), p.ProjectID, run.RunRef, "", 0)
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("shared query did not start")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Millisecond)
	_, err := s.coalescedQuery(ctx, p.ProjectID, run.RunRef, "", 0)
	cancel()
	wantCode(t, err, "runtime_unavailable")
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("one cancelled reader interrupted the other reader: %v", err)
	}
	if monitors.Load() != 1 {
		t.Fatalf("identical reads launched %d monitor queries", monitors.Load())
	}
	if _, err := s.coalescedQuery(t.Context(), p.ProjectID, run.RunRef, "", 0); err != nil {
		t.Fatal(err)
	}
	if monitors.Load() != 2 {
		t.Fatal("a later observation reused a completed query")
	}
}

func TestRunRejectsStaleAttemptsCrossProjectAndIdentity(t *testing.T) {
	s, fixture, p, run := attachFixtureRun(t)
	_, err := s.coalescedQuery(t.Context(), p.ProjectID, run.RunRef, "align", 1)
	wantCode(t, err, "stale_revision")
	other := testProject(t, s, t.TempDir())
	_, err = s.coalescedQuery(t.Context(), other.ProjectID, run.RunRef, "", 0)
	wantCode(t, err, "not_found")
	fixture.match = false
	_, err = s.coalescedQuery(t.Context(), p.ProjectID, run.RunRef, "", 0)
	wantCode(t, err, "incompatible_runtime")
	fixture.match = true
	fixture.runID = "replacement"
	_, err = s.coalescedQuery(t.Context(), p.ProjectID, run.RunRef, "", 0)
	wantCode(t, err, "stale_revision")
}

func TestRunRejectsDaemonImageAndSchemaMismatch(t *testing.T) {
	s, fixture, p, run := attachFixtureRun(t)
	original := fixture.daemon
	fixture.daemon = "other"
	_, err := s.coalescedQuery(t.Context(), p.ProjectID, run.RunRef, "", 0)
	wantCode(t, err, "incompatible_runtime")
	fixture.daemon = original
	image := fixture.image
	fixture.image = "sha256:" + strings.Repeat("b", 64)
	_, err = s.coalescedQuery(t.Context(), p.ProjectID, run.RunRef, "", 0)
	wantCode(t, err, "incompatible_runtime")
	fixture.image = image
	fixture.monitorSchema = 3
	_, err = s.coalescedQuery(t.Context(), p.ProjectID, run.RunRef, "", 0)
	wantCode(t, err, "incompatible_runtime")
}

func TestMissingRuntimeRemainsUnavailable(t *testing.T) {
	s := testService(t)
	root := t.TempDir()
	p := testProject(t, s, root)
	_, err := s.attachRun(t.Context(), p.ProjectID, p.RootResourceID, "req_no_runtime")
	wantCode(t, err, "incompatible_runtime")
	if len(s.store.data.Runs) != 0 {
		t.Fatal("unsupported runtime was attached")
	}
}

func TestRuntimeCancellationCleansOnlyItsQuery(t *testing.T) {
	s, fixture, p, run := attachFixtureRun(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := s.coalescedQuery(ctx, p.ProjectID, run.RunRef, "", 0)
	wantCode(t, err, "runtime_unavailable")
	for _, args := range fixture.commands {
		if args[0] == "rm" && (len(args) != 3 || !strings.HasPrefix(args[2], "gobble-query-req_")) {
			t.Fatalf("cleanup targeted a non-query container: %v", args)
		}
	}
}

func TestActiveRuntimeCancellationRemovesOwnedQuery(t *testing.T) {
	s, fixture, p, run := attachFixtureRun(t)
	project, err := s.store.project(p.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	binding := s.store.data.Runs[0].Binding
	started := make(chan struct{})
	adapter := runtimeAdapter{command: func(ctx context.Context, args, env []string) ([]byte, error) {
		if args[0] == "run" {
			close(started)
			<-ctx.Done()
			return nil, ctx.Err()
		}
		return fixture.command(ctx, args, env)
	}}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { _, err := adapter.inspect(ctx, project, binding, "monitor", ""); done <- err }()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled query: %v", err)
	}
	last := fixture.commands[len(fixture.commands)-1]
	if last[0] != "rm" || last[1] != "--force" || !strings.HasPrefix(last[2], "gobble-query-") || strings.Contains(last[2], run.EngineRunID) {
		t.Fatalf("unsafe cleanup: %v", last)
	}
}
