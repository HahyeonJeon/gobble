package engine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HahyeonJeon/gobble/internal/engine/exec"
)

func TestRunContextCancelPersistsIncomplete(t *testing.T) {
	submitted := make(chan struct{})
	var once syncClose
	var cancelCalled atomic.Bool
	useExec(t, &fnExec{
		submit: func(ctx context.Context, job exec.Job) (exec.Handle, exec.Report, error) {
			once.do(submitted)
			h := exec.Handle{Identity: job.Identity, Backend: exec.BackendProcess, RuntimeID: "9"}
			return h, exec.Report{Identity: job.Identity, RuntimeID: "9", Running: true}, nil
		},
		poll: func(ctx context.Context, h exec.Handle) (exec.Report, error) {
			if cancelCalled.Load() {
				return exec.Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: false, Exit: -1, Message: "killed"}, nil
			}
			return exec.Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: true}, nil
		},
		cancel: func(ctx context.Context, h exec.Handle) error {
			cancelCalled.Store(true)
			return nil
		},
	})
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan []Defect, 1)
	go func() {
		done <- Run(ctx, Request{
			Workspace: dir,
			Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
		})
	}()
	select {
	case <-submitted:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Submit")
	}
	waitRuntimeID(t, dir, "copy")
	cancel()
	select {
	case defects := <-done:
		if !hasDefect(defects, DefectCanceled, "") {
			t.Fatalf("canceled Run() defects %v, want canceled", defects)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for canceled Run")
	}
	if !cancelCalled.Load() {
		t.Fatal("Executor.Cancel not called")
	}
	st := taskStates(t, dir)["copy"]
	if st.Status != StatusIncomplete {
		t.Fatalf("canceled task status got %q, want incomplete", st.Status)
	}
	raw := mustJSONFile(t, filepath.Join(dir, ControlDir, RunIdentityFile))
	var run jsonRun
	if err := json.Unmarshal(raw, &run); err != nil {
		t.Fatalf("run.json: %v", err)
	}
	if run.Occupancy == nil || !run.Occupancy.Active {
		t.Fatalf("occupancy after cancel got %#v, want active", run.Occupancy)
	}
}

func TestResumeReconcileLiveCancelsIncomplete(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc("", "", "in/sample.txt", "out/sample.txt")
	if defects := Run(t.Context(), Request{Workspace: dir, Document: doc}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	forceDeadOwner(t, dir)
	if defects := Release(dir); len(defects) != 0 {
		t.Fatalf("Release() defects %v", defects)
	}
	patchAttempt(t, dir, func(st *jsonTaskState) {
		st.Status = StatusIncomplete
		st.RuntimeID = "99"
		st.Reason = reasonPreviousIncomplete
	})
	var cancelCalled atomic.Bool
	var canceledID string
	useExec(t, &fnExec{
		submit: func(ctx context.Context, job exec.Job) (exec.Handle, exec.Report, error) {
			writeCheckFile(t, filepath.Join(isolateWorkspace(job.Isolate), "out", "sample.txt"), "reads")
			h := exec.Handle{Identity: job.Identity, Backend: exec.BackendProcess, RuntimeID: "2"}
			return h, exec.Report{Identity: job.Identity, RuntimeID: "2", Running: true}, nil
		},
		poll: func(ctx context.Context, h exec.Handle) (exec.Report, error) {
			return exec.Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: false, Published: true}, nil
		},
		cancel: func(ctx context.Context, h exec.Handle) error {
			cancelCalled.Store(true)
			canceledID = h.RuntimeID
			return nil
		},
		reconcile: func(ctx context.Context, h exec.Handle) (exec.Report, error) {
			if h.RuntimeID == "99" && !cancelCalled.Load() {
				return exec.Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: true}, nil
			}
			return exec.Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: false}, nil
		},
	})
	defects := Resume(t.Context(), Request{Workspace: dir, Document: doc})
	if len(defects) != 0 {
		t.Fatalf("Resume() defects %v, want none", defects)
	}
	if !cancelCalled.Load() || canceledID != "99" {
		t.Fatalf("Cancel got called=%v id=%q, want runtime_id 99", cancelCalled.Load(), canceledID)
	}
	after := taskStates(t, dir)["copy"]
	if after.Attempt != 2 {
		t.Fatalf("resume attempt got %d, want 2 (no adopt)", after.Attempt)
	}
	if after.Decision != reuseRerun || after.ReuseReason != reasonPreviousIncomplete {
		t.Fatalf("resume decision got %q %q, want rerun previous-incomplete", after.Decision, after.ReuseReason)
	}
}

type syncClose struct {
	once atomic.Bool
}

func (s *syncClose) do(ch chan struct{}) {
	if s.once.CompareAndSwap(false, true) {
		close(ch)
	}
}

func waitRuntimeID(t *testing.T, workspace, ident string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		path := filepath.Join(workspace, ControlDir, TasksFile)
		data, err := os.ReadFile(path)
		if err == nil {
			var file jsonTasksFile
			if json.Unmarshal(data, &file) == nil {
				for _, st := range file.Tasks {
					if reservedIdentity(taskPlanFromState(st)) == ident && st.RuntimeID != "" {
						return
					}
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for runtime_id on %s", ident)
}

func patchAttempt(t *testing.T, workspace string, fn func(*jsonTaskState)) {
	t.Helper()
	path := filepath.Join(workspace, ControlDir, TasksFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read tasks.json: %v", err)
	}
	var file jsonTasksFile
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("tasks.json: %v", err)
	}
	if len(file.Tasks) == 0 {
		t.Fatal("tasks.json empty")
	}
	fn(&file.Tasks[len(file.Tasks)-1])
	out, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		t.Fatalf("marshal tasks.json: %v", err)
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		t.Fatalf("write tasks.json: %v", err)
	}
}
