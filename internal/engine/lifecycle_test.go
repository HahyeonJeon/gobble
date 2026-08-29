package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/HahyeonJeon/gobble/internal/engine/exec"
)

func useBound(t *testing.T, d time.Duration) {
	t.Helper()
	orig := currentSettlementBound()
	setSettlementBound(d)
	t.Cleanup(func() { setSettlementBound(orig) })
}

func waitCtx(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestReleaseEmptyWorkspaceInvalidPath(t *testing.T) {
	defects := Release("", testInstallIdentity())
	if !hasDefect(defects, DefectInvalidPath, "") {
		t.Fatalf("Release(\"\", testInstallIdentity()) defects %v, want invalid-path", defects)
	}
	missing := filepath.Join(t.TempDir(), "absent")
	defects = Release(missing, testInstallIdentity())
	if !hasDefect(defects, DefectInvalidPath, "") {
		t.Fatalf("Release(missing, testInstallIdentity()) defects %v, want invalid-path", defects)
	}
}

func TestCanceledPollAlwaysRunningStopsAtSettlementBound(t *testing.T) {
	useBound(t, 40*time.Millisecond)
	submitted := make(chan struct{})
	var once syncClose
	useExec(t, &fnExec{
		submit: func(ctx context.Context, job exec.Job) (exec.Handle, exec.Report, error) {
			once.do(submitted)
			h := exec.Handle{Identity: job.Identity, Backend: exec.BackendProcess, RuntimeID: "1"}
			return h, exec.Report{Identity: job.Identity, RuntimeID: "1", Running: true}, nil
		},
		poll: func(ctx context.Context, h exec.Handle) (exec.Report, error) {
			return exec.Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: true}, nil
		},
		cancel: func(ctx context.Context, h exec.Handle) error {
			return nil
		},
	})
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan []Defect, 1)
	go func() {
		done <- Run(ctx, Request{
			Identity:  testInstallIdentity(),
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
	started := time.Now()
	cancel()
	var defects []Defect
	select {
	case defects = <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after settlement bound")
	}
	if elapsed := time.Since(started); elapsed > 200*time.Millisecond {
		t.Fatalf("canceled Run settled in %v, want near %v", elapsed, currentSettlementBound())
	}
	if !hasDefect(defects, DefectCanceled, "") {
		t.Fatalf("Run() defects %v, want canceled", defects)
	}
	if !hasDefect(defects, DefectUnknownBackend, "copy") {
		t.Fatalf("Run() defects %v, want unknown-backend copy", defects)
	}
	run, exists, err := readRunIdentity(dir)
	if err != nil || !exists {
		t.Fatalf("run.json exists=%v err=%v", exists, err)
	}
	if !occupancyIsActive(run) {
		t.Fatal("occupancy closed after unknown poll")
	}
	st := taskStates(t, dir)["copy"]
	if st.Status != StatusUnknown {
		t.Fatalf("task status got %q, want unknown", st.Status)
	}
	raw, inspectDefects := Inspect(dir, viewRun, "", testInstallIdentity())
	if len(inspectDefects) != 0 {
		t.Fatalf("Inspect(run, testInstallIdentity()) defects %v", inspectDefects)
	}
	if !bytesContains(raw, []byte(`"unknown": true`)) && !bytesContains(raw, []byte(`"unknown":true`)) {
		t.Fatalf("Inspect(run, testInstallIdentity()) missing run-scope unknown: %s", raw)
	}
}

func TestLaterProcessReleaseUnprovedProcessIncomplete(t *testing.T) {
	useBound(t, 40*time.Millisecond)
	submitted := make(chan struct{})
	var once syncClose
	useExec(t, &fnExec{
		submit: func(ctx context.Context, job exec.Job) (exec.Handle, exec.Report, error) {
			once.do(submitted)
			h := exec.Handle{Identity: job.Identity, Backend: exec.BackendProcess, RuntimeID: "9"}
			return h, exec.Report{Identity: job.Identity, RuntimeID: "9", Running: true}, nil
		},
		poll: func(ctx context.Context, h exec.Handle) (exec.Report, error) {
			return exec.Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: true}, nil
		},
		cancel: func(ctx context.Context, h exec.Handle) error {
			return nil
		},
	})
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan []Defect, 1)
	go func() {
		done <- Run(ctx, Request{
			Identity:  testInstallIdentity(),
			Workspace: dir,
			Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
		})
	}()
	<-submitted
	waitRuntimeID(t, dir, "copy")
	cancel()
	if defects := <-done; !hasDefect(defects, DefectUnknownBackend, "copy") {
		t.Fatalf("Run() defects %v, want unknown-backend", defects)
	}
	runExecutor = nil
	DropHeldLease(dir)
	defects := Release(dir, testInstallIdentity())
	if len(defects) != 0 {
		t.Fatalf("later-process Release() defects %v, want none", defects)
	}
	run, exists, err := readRunIdentity(dir)
	if err != nil || !exists {
		t.Fatalf("run.json exists=%v err=%v", exists, err)
	}
	if occupancyIsActive(run) {
		t.Fatal("later-process Release kept occupancy for unproved process")
	}
	released := taskStates(t, dir)["copy"]
	if released.Status != StatusIncomplete {
		t.Fatalf("later-process process status got %q, want incomplete", released.Status)
	}
	if released.RuntimeID != "9" {
		t.Fatalf("later-process process runtime_id got %q, want stored 9", released.RuntimeID)
	}

	doc := sampleDoc("", "", "in/sample.txt", "out/sample.txt")
	defects = Resume(t.Context(), Request{
		Identity:  testInstallIdentity(),
		Workspace: dir,
		Document:  doc,
	})
	if len(defects) != 0 {
		t.Fatalf("Resume() defects %v, want rerun without unproved reconciliation", defects)
	}
	after := taskStates(t, dir)["copy"]
	if after.Status != StatusSucceeded || after.Attempt != released.Attempt+1 {
		t.Fatalf("resumed process state got status=%q attempt=%d, want succeeded attempt %d", after.Status, after.Attempt, released.Attempt+1)
	}
	run, exists, err = readRunIdentity(dir)
	if err != nil || !exists {
		t.Fatalf("resumed run.json exists=%v err=%v", exists, err)
	}
	if run.Occupancy == nil || len(run.Occupancy.Unknown) != 0 {
		t.Fatalf("resumed occupancy unknown got %#v, want none", run.Occupancy)
	}
}

func TestBlockingCancelRecordsUnknown(t *testing.T) {
	useBound(t, 40*time.Millisecond)
	submitted := make(chan struct{})
	var once syncClose
	useExec(t, &fnExec{
		submit: func(ctx context.Context, job exec.Job) (exec.Handle, exec.Report, error) {
			once.do(submitted)
			h := exec.Handle{Identity: job.Identity, Backend: exec.BackendProcess, RuntimeID: "1"}
			return h, exec.Report{Identity: job.Identity, RuntimeID: "1", Running: true}, nil
		},
		poll: func(ctx context.Context, h exec.Handle) (exec.Report, error) {
			return exec.Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: true}, nil
		},
		cancel: func(ctx context.Context, h exec.Handle) error {
			return waitCtx(ctx)
		},
	})
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan []Defect, 1)
	go func() {
		done <- Run(ctx, Request{
			Identity:  testInstallIdentity(),
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
			t.Fatalf("blocking Cancel Run() defects %v, want canceled", defects)
		}
		if !hasDefect(defects, DefectUnknownBackend, "copy") {
			t.Fatalf("blocking Cancel Run() defects %v, want unknown-backend", defects)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for bounded Cancel")
	}
	run, exists, err := readRunIdentity(dir)
	if err != nil || !exists || !occupancyIsActive(run) {
		t.Fatalf("occupancy after blocking Cancel exists=%v err=%v", exists, err)
	}
}

func TestUncancelledLongTaskExceedsSettlementBound(t *testing.T) {
	useBound(t, 40*time.Millisecond)
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc("", "", "in/sample.txt", "out/sample.txt")
	doc.Tasks[0].Command = []string{"sh", "-c", "sleep 0.15; cp in/sample.txt out/sample.txt"}
	start := time.Now()
	defects := Run(t.Context(), Request{
		Identity:  testInstallIdentity(),
		Workspace: dir,
		Document:  doc,
	})
	if len(defects) != 0 {
		t.Fatalf("cooperative Run() defects %v, want none", defects)
	}
	if time.Since(start) < currentSettlementBound() {
		t.Fatalf("cooperative job finished in %v, want longer than settlement bound %v", time.Since(start), currentSettlementBound())
	}
	if taskStates(t, dir)["copy"].Status != StatusSucceeded {
		t.Fatalf("cooperative job status got %q, want succeeded", taskStates(t, dir)["copy"].Status)
	}
}

func TestCallerDeadlineDuringSubmitPersistsIncomplete(t *testing.T) {
	useBound(t, 40*time.Millisecond)
	useExec(t, &fnExec{
		submit: func(ctx context.Context, job exec.Job) (exec.Handle, exec.Report, error) {
			return exec.Handle{}, exec.Report{}, waitCtx(ctx)
		},
	})
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Millisecond)
	defer cancel()
	defects := Run(ctx, Request{
		Identity:  testInstallIdentity(),
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	})
	if !hasDefect(defects, DefectCanceled, "") {
		t.Fatalf("deadline Submit Run() defects %v, want canceled", defects)
	}
	st := taskStates(t, dir)["copy"]
	if st.Status != StatusIncomplete || st.RuntimeID != "" {
		t.Fatalf("timed-out Submit state got status=%q runtime_id=%q", st.Status, st.RuntimeID)
	}
	if defects := Release(dir, testInstallIdentity()); len(defects) != 0 {
		t.Fatalf("occupying Release after deadline defects %v, want none", defects)
	}
	if run, exists, err := readRunIdentity(dir); err != nil || !exists || occupancyIsActive(run) {
		t.Fatal("Release did not close occupancy after deadline before handle")
	}
}

func TestWrappedCallerDeadlineDuringSubmitPersistsIncomplete(t *testing.T) {
	useBound(t, 40*time.Millisecond)
	useExec(t, &fnExec{
		submit: func(ctx context.Context, job exec.Job) (exec.Handle, exec.Report, error) {
			return exec.Handle{}, exec.Report{}, fmt.Errorf("docker: %w", waitCtx(ctx))
		},
	})
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Millisecond)
	defer cancel()
	defects := Run(ctx, Request{
		Identity:  testInstallIdentity(),
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	})
	if !hasDefect(defects, DefectCanceled, "") {
		t.Fatalf("wrapped deadline Submit Run() defects %v, want canceled", defects)
	}
	st := taskStates(t, dir)["copy"]
	if st.Status != StatusIncomplete || st.RuntimeID != "" {
		t.Fatalf("wrapped deadline Submit state got status=%q runtime_id=%q", st.Status, st.RuntimeID)
	}
	if run, exists, err := readRunIdentity(dir); err != nil || !exists || !occupancyIsActive(run) {
		t.Fatal("wrapped deadline Submit did not retain occupancy")
	}
}

func TestEvaluatorSuccessfulSubmitWithoutRuntimeID(t *testing.T) {
	useBound(t, 40*time.Millisecond)
	useExec(t, &fnExec{
		submit: func(ctx context.Context, job exec.Job) (exec.Handle, exec.Report, error) {
			return exec.Handle{Identity: job.Identity, Backend: exec.BackendProcess}, exec.Report{Identity: job.Identity, Running: true}, nil
		},
		poll: func(ctx context.Context, h exec.Handle) (exec.Report, error) {
			return exec.Report{}, waitCtx(ctx)
		},
	})
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	defects := Run(t.Context(), Request{
		Identity:  testInstallIdentity(),
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	})
	st := taskStates(t, dir)["copy"]
	if st.Status == StatusRunning && st.RuntimeID == "" {
		t.Fatalf("persisted status=%q with empty runtime_id", st.Status)
	}
	if st.Status != StatusUnknown || st.RuntimeID != "" {
		t.Fatalf("empty-id Submit state got status=%q runtime_id=%q, want unknown", st.Status, st.RuntimeID)
	}
	if !hasDefect(defects, DefectUnknownBackend, "copy") {
		t.Fatalf("empty-id Submit Run() defects %v, want unknown-backend", defects)
	}
	if run, exists, err := readRunIdentity(dir); err != nil || !exists || !occupancyIsActive(run) {
		t.Fatal("occupancy closed after empty-id Submit")
	}
	runExecutor = nil
	DropHeldLease(dir)
	if defects := Release(dir, testInstallIdentity()); len(defects) != 0 {
		t.Fatalf("later-process Release() defects %v, want none", defects)
	}
	if run, exists, err := readRunIdentity(dir); err != nil || !exists || occupancyIsActive(run) {
		t.Fatal("later-process Release kept occupancy for empty-id process")
	}
	if st := taskStates(t, dir)["copy"]; st.Status != StatusIncomplete {
		t.Fatalf("empty-id process status got %q, want incomplete", st.Status)
	}
}

func TestLaterProcessCrashBeforeHandle(t *testing.T) {
	dir := t.TempDir()
	host, err := currentHost()
	if err != nil {
		t.Fatal(err)
	}
	writeOccupancy(t, dir, jsonOccupancy{Active: true, Host: host, PID: deadPID(t), Lease: "lease", Started: "2026-01-01T00:00:00Z"})
	writeCheckFile(t, filepath.Join(dir, ControlDir, TasksFile), `{
  "schema_version": 2,
  "tasks": [
    {
      "id": "copy",
      "instance": "",
      "shard_index": 0,
      "shard_count": 1,
      "attempt": 1,
      "status": "unknown",
      "executor": "process",
      "image": "",
      "command": ["true"],
      "resources": {"cpu": 0, "memory": ""},
      "params": [],
      "reason": "unknown-backend"
    }
  ]
}
`)
	defects := Release(dir, testInstallIdentity())
	if len(defects) != 0 {
		t.Fatalf("later-process crash-before-handle Release() defects %v, want none", defects)
	}
	run, exists, err := readRunIdentity(dir)
	if err != nil || !exists || occupancyIsActive(run) {
		t.Fatal("later-process crash-before-handle kept occupancy")
	}
	if taskStates(t, dir)["copy"].Status != StatusIncomplete {
		t.Fatalf("copy status got %q, want incomplete", taskStates(t, dir)["copy"].Status)
	}
}

func TestRunContextCancelUnknown(t *testing.T) {
	useBound(t, 40*time.Millisecond)
	submitted := make(chan struct{})
	var once syncClose
	useExec(t, &fnExec{
		submit: func(ctx context.Context, job exec.Job) (exec.Handle, exec.Report, error) {
			once.do(submitted)
			h := exec.Handle{Identity: job.Identity, Backend: exec.BackendProcess, RuntimeID: "1"}
			return h, exec.Report{Identity: job.Identity, RuntimeID: "1", Running: true}, nil
		},
		poll: func(ctx context.Context, h exec.Handle) (exec.Report, error) {
			return exec.Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: true}, nil
		},
		cancel: func(ctx context.Context, h exec.Handle) error {
			return waitCtx(ctx)
		},
	})
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan []Defect, 1)
	go func() {
		done <- Run(ctx, Request{
			Identity:  testInstallIdentity(),
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
		if !hasDefect(defects, DefectCanceled, "") || !hasDefect(defects, DefectUnknownBackend, "copy") {
			t.Fatalf("Run cancel/unknown defects %v", defects)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for cancel/unknown Run")
	}
}

func TestResumeContextCancelUnknown(t *testing.T) {
	useBound(t, 40*time.Millisecond)
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc("", "", "in/sample.txt", "out/sample.txt")
	if defects := Run(t.Context(), Request{
		Identity:  testInstallIdentity(),
		Workspace: dir,
		Document:  doc,
	}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	if defects := Release(dir, testInstallIdentity()); len(defects) != 0 {
		t.Fatalf("Release() defects %v", defects)
	}
	if err := os.Remove(filepath.Join(dir, "out", "sample.txt")); err != nil {
		t.Fatal(err)
	}
	submitted := make(chan struct{})
	var once syncClose
	useExec(t, &fnExec{
		submit: func(ctx context.Context, job exec.Job) (exec.Handle, exec.Report, error) {
			once.do(submitted)
			h := exec.Handle{Identity: job.Identity, Backend: exec.BackendProcess, RuntimeID: "2"}
			return h, exec.Report{Identity: job.Identity, RuntimeID: "2", Running: true}, nil
		},
		poll: func(ctx context.Context, h exec.Handle) (exec.Report, error) {
			return exec.Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: true}, nil
		},
		cancel: func(ctx context.Context, h exec.Handle) error {
			return waitCtx(ctx)
		},
		reconcile: func(ctx context.Context, h exec.Handle) (exec.Report, error) {
			return exec.Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: false}, nil
		},
	})
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan []Defect, 1)
	go func() {
		done <- Resume(ctx, Request{
			Identity:  testInstallIdentity(),
			Workspace: dir,
			Document:  doc,
		})
	}()
	select {
	case <-submitted:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Resume Submit")
	}
	waitRuntimeID(t, dir, "copy")
	cancel()
	select {
	case defects := <-done:
		if !hasDefect(defects, DefectCanceled, "") || !hasDefect(defects, DefectUnknownBackend, "copy") {
			t.Fatalf("Resume cancel/unknown defects %v", defects)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for cancel/unknown Resume")
	}
}

func TestSecondOccupyOccupiedWorkspace(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	req := Request{
		Identity:  testInstallIdentity(),
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt")}
	if defects := Run(t.Context(), req); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	defects := Run(t.Context(), req)
	if !hasDefect(defects, DefectOccupiedWorkspace, "") {
		t.Fatalf("second Run() defects %v, want occupied-workspace", defects)
	}
}

func TestInspectSnapshotMismatchRefused(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	if defects := Run(t.Context(), Request{
		Identity:  testInstallIdentity(),
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	path := filepath.Join(dir, ControlDir, TasksFile)
	data := mustJSONFile(t, path)
	var file jsonTasksFile
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatal(err)
	}
	file.Snapshot = "other-snapshot"
	out, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeCheckFile(t, path, string(append(out, '\n')))
	_, defects := Inspect(dir, viewRun, "", testInstallIdentity())
	if !hasDefect(defects, DefectInvalidPath, "") {
		t.Fatalf("Inspect mixed snapshot defects %v, want invalid-path", defects)
	}
}

func TestConcurrentWorkspaceRunsIsolateExecutors(t *testing.T) {
	var wg sync.WaitGroup
	errs := make(chan []Defect, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dir := t.TempDir()
			writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
			errs <- Run(t.Context(), Request{
				Identity:  testInstallIdentity(),
				Workspace: dir,
				Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
			})
		}()
	}
	wg.Wait()
	close(errs)
	for d := range errs {
		if len(d) != 0 {
			t.Fatalf("concurrent Run() defects %v", d)
		}
	}
}

func bytesContains(b, sub []byte) bool {
	return bytes.Contains(b, sub)
}
