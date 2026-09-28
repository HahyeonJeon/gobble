package engine

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HahyeonJeon/gobble/internal/engine/exec"
	"github.com/HahyeonJeon/gobble/internal/preparation"
)

// These tests exercise the real scheduler, Stop, persistence and publication,
// with deterministic tool execution and installed-image/daemon observations.
func continuationTools(t *testing.T, in preparation.LaunchIntent) {
	t.Helper()
	oldImage, oldAbsent := lookupImageID, verifyContinuationSubmission
	lookupImageID = func(image string) string {
		if v := in.Tools[image]; v != "" {
			return v
		}
		return image
	}
	verifyContinuationSubmission = func(context.Context, exec.Handle) error { return nil }
	t.Cleanup(func() { lookupImageID, verifyContinuationSubmission = oldImage, oldAbsent })
}
func continuationExecutor(t *testing.T, in preparation.LaunchIntent, block string, submitted func(exec.Job)) *fnExec {
	t.Helper()
	var canceled atomic.Bool
	return &fnExec{cancel: func(context.Context, exec.Handle) error { canceled.Store(true); return nil }, submit: func(ctx context.Context, j exec.Job) (exec.Handle, exec.Report, error) {
		if submitted != nil {
			submitted(j)
		}
		if j.Identity != block {
			for _, task := range creationDocument().Tasks {
				if task.ID == j.Identity {
					for _, f := range declaredIOFiles(task.Outputs) {
						writeCheckFile(t, filepath.Join(j.Isolate, f.path), "result:"+f.path)
					}
				}
			}
		}
		return exec.Handle{Identity: j.Identity, RuntimeID: j.Identity}, exec.Report{Identity: j.Identity, RuntimeID: j.Identity, ImageDigest: in.Tools[j.Image], Running: true}, nil
	}, poll: func(ctx context.Context, h exec.Handle) (exec.Report, error) {
		if err := ctx.Err(); err != nil {
			return exec.Report{}, err
		}
		return exec.Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: h.Identity == block && !canceled.Load()}, nil
	}}
}
func stoppedContinuation(t *testing.T, block string) (string, []byte, preparation.LaunchIntent) {
	t.Helper()
	ws, raw, in := launchFixture(t)
	in.SchemaVersion = 2
	continuationTools(t, in)
	useExec(t, continuationExecutor(t, in, block, nil))
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)
	done := make(chan []Defect, 1)
	go func() { done <- RunPrepared(ctx, ws, raw, in, testInstallIdentity()) }()
	waitRuntimeID(t, ws, block)
	before, _, _ := readRunIdentity(ws)
	if r, d := StopOwner(ctx, ws, testInstallIdentity(), before.Occupancy.Lease); len(d) != 0 || r.Status != "settled" {
		t.Fatalf("stop: %+v %v", r, d)
	}
	select {
	case d := <-done:
		if !hasDefectCode(d, DefectCanceled) {
			t.Fatalf("run stop: %v", d)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	DropHeldLease(ws)
	return ws, raw, in
}
func continuationIntent(r preparation.ContinuationReview) preparation.ContinuationIntent {
	return preparation.ContinuationIntent{SchemaVersion: 1, RequestID: "req_continue", WorkspaceID: r.WorkspaceID, OriginDigest: r.OriginDigest, ReviewDigest: r.Digest, PreviousHead: r.PreviousHead}
}
func workspaceContents(t *testing.T, ws string) map[string]string {
	t.Helper()
	out := map[string]string{}
	if err := filepath.WalkDir(ws, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			b, e := os.ReadFile(path)
			if e != nil {
				return e
			}
			out[path] = string(b)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return out
}
func TestContinuationReviewAndSameRunExecution(t *testing.T) {
	ws, raw, in := stoppedContinuation(t, "qc")
	before, _, _ := readRunIdentity(ws)
	files := workspaceContents(t, ws)
	review, err := ReviewPreparedContinuation(t.Context(), ws, raw, in, testInstallIdentity())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(files, workspaceContents(t, ws)) {
		t.Fatal("review wrote workspace")
	}
	want := []preparation.ContinuationStep{{TaskID: "trim", PriorAttempt: 1, PlannedAttempt: 1, Action: "reuse"}, {TaskID: "qc", PriorAttempt: 1, PlannedAttempt: 2, Action: "restart"}}
	if !reflect.DeepEqual(review.Steps, want) {
		t.Fatalf("steps: %+v", review.Steps)
	}
	intent := continuationIntent(review)
	var calls atomic.Int32
	useExec(t, continuationExecutor(t, in, "", func(j exec.Job) {
		calls.Add(1)
		if j.Identity != "qc" {
			t.Error("reused trim was submitted")
		}
		if _, ok, e := ReadContinuationAdmission(ws, intent, testInstallIdentity()); e != nil || !ok {
			t.Errorf("submit before receipt: %v", e)
		}
	}))
	receipt, d := ContinuePrepared(t.Context(), ws, raw, in, intent, testInstallIdentity())
	if len(d) != 0 {
		t.Fatal(d)
	}
	DropHeldLease(ws)
	after, _, _ := readRunIdentity(ws)
	if calls.Load() != 1 || after.Status != StatusSucceeded || after.ID != before.ID || after.Started != before.Started || *after.Admission != *before.Admission || receipt.Lease == before.Occupancy.Lease {
		t.Fatalf("same Run: calls=%d before=%+v after=%+v", calls.Load(), before, after)
	}
	tasks := taskStates(t, ws)
	if tasks["trim"].Attempt != 1 || tasks["qc"].Attempt != 2 || tasks["trim"].Decision != reuseReused {
		t.Fatalf("attempts: %+v", tasks)
	}
	for _, f := range review.Files {
		sum, e := sha256File(filepath.Join(ws, f.Path))
		if e != nil || sum != f.SHA256 {
			t.Fatal("reused content changed", e)
		}
	}
	for n := 0; n < 2; n++ {
		got, d := ContinuePrepared(t.Context(), ws, raw, in, intent, testInstallIdentity())
		if len(d) != 0 || got != receipt || calls.Load() != 1 {
			t.Fatalf("replay: %+v %v", got, d)
		}
	}
	other := intent
	other.ReviewDigest = "sha256:" + strings.Repeat("f", 64)
	if _, d := ContinuePrepared(t.Context(), ws, raw, in, other, testInstallIdentity()); len(d) == 0 {
		t.Fatal("request reused with different effects")
	}
	if d := RunPrepared(t.Context(), ws, raw, in, testInstallIdentity()); len(d) != 0 || calls.Load() != 1 {
		t.Fatal("Start replay rescheduled", d)
	}
}
func mutateContinuationControl(t *testing.T, ws string, change func(*jsonRun, *jsonPlan, *jsonTasksFile)) {
	t.Helper()
	run, plan, _, tasks, _, d := readCoherentControl(ws)
	if len(d) > 0 {
		t.Fatal(d)
	}
	change(&run, &plan, &tasks)
	run.Snapshot = newOccupancyID()
	plan.Snapshot = run.Snapshot
	tasks.Snapshot = run.Snapshot
	r, _ := json.Marshal(run)
	p, _ := json.Marshal(plan)
	s, _ := json.Marshal(tasks)
	if err := commitCheckpoint(ws, run.Snapshot, p, s, r); err != nil {
		t.Fatal(err)
	}
}
func TestContinuationRefusesChangedEvidence(t *testing.T) {
	cases := map[string]func(*testing.T, string, preparation.LaunchIntent){
		"duplicate attempt": func(t *testing.T, ws string, in preparation.LaunchIntent) {
			mutateContinuationControl(t, ws, func(r *jsonRun, p *jsonPlan, s *jsonTasksFile) { s.Tasks = append(s.Tasks, s.Tasks[0]) })
		},
		"plan changed": func(t *testing.T, ws string, in preparation.LaunchIntent) {
			mutateContinuationControl(t, ws, func(r *jsonRun, p *jsonPlan, s *jsonTasksFile) { p.Tasks[0].Command = []string{"changed"} })
		},
		"history limit": func(t *testing.T, ws string, in preparation.LaunchIntent) {
			for range preparation.MaxContinuations {
				run, _, _ := readRunIdentity(ws)
				appendHistoryFixture(t, &run)
				if err := saveHistoryFixture(ws, run); err != nil {
					t.Fatal(err)
				}
			}
		},
		"saved input": func(t *testing.T, ws string, in preparation.LaunchIntent) {
			writeCheckFile(t, filepath.Join(ws, in.Binding.InputPath), "@read\nTGCA\n+\nIIII\n")
		},
		"completed output": func(t *testing.T, ws string, in preparation.LaunchIntent) {
			p := filepath.Join(ws, declaredIOFiles(creationDocument().Tasks[0].Outputs)[0].path)
			info, _ := os.Stat(p)
			b, _ := os.ReadFile(p)
			b[0] = 'X'
			if err := os.WriteFile(p, b, 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.Chtimes(p, info.ModTime(), info.ModTime()); err != nil {
				t.Fatal(err)
			}
		},
		"missing output": func(t *testing.T, ws string, in preparation.LaunchIntent) {
			if err := os.Remove(filepath.Join(ws, declaredIOFiles(creationDocument().Tasks[0].Outputs)[0].path)); err != nil {
				t.Fatal(err)
			}
		},
		"partial output": func(t *testing.T, ws string, in preparation.LaunchIntent) {
			writeCheckFile(t, filepath.Join(ws, declaredIOFiles(creationDocument().Tasks[1].Outputs)[0].path), "partial")
		},
		"tool changed": func(t *testing.T, ws string, in preparation.LaunchIntent) {
			lookupImageID = func(string) string { return "" }
		},
		"backend uncertain": func(t *testing.T, ws string, in preparation.LaunchIntent) {
			verifyContinuationSubmission = func(context.Context, exec.Handle) error { return errors.New("daemon unavailable") }
		},
		"failed": func(t *testing.T, ws string, in preparation.LaunchIntent) {
			mutateContinuationControl(t, ws, func(r *jsonRun, p *jsonPlan, s *jsonTasksFile) {
				for i := range s.Tasks {
					if s.Tasks[i].ID == "qc" {
						s.Tasks[i].Status = StatusFailed
					}
				}
			})
		},
		"not canceled": func(t *testing.T, ws string, in preparation.LaunchIntent) {
			mutateContinuationControl(t, ws, func(r *jsonRun, p *jsonPlan, s *jsonTasksFile) {
				for i := range s.Tasks {
					if s.Tasks[i].ID == "qc" {
						s.Tasks[i].Reason = "unknown"
					}
				}
			})
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			ws, raw, in := stoppedContinuation(t, "qc")
			review, e := ReviewPreparedContinuation(t.Context(), ws, raw, in, testInstallIdentity())
			if e != nil {
				t.Fatal(e)
			}
			change(t, ws, in)
			files := workspaceContents(t, ws)
			if _, e := ReviewPreparedContinuation(t.Context(), ws, raw, in, testInstallIdentity()); e == nil {
				t.Fatal("unsafe review accepted")
			}
			if !reflect.DeepEqual(files, workspaceContents(t, ws)) {
				t.Fatal("failed review wrote workspace")
			}
			var calls atomic.Int32
			useExec(t, continuationExecutor(t, in, "", func(exec.Job) { calls.Add(1) }))
			receipt, d := ContinuePrepared(t.Context(), ws, raw, in, continuationIntent(review), testInstallIdentity())
			if len(d) == 0 || receipt.Lease != "" || calls.Load() != 0 {
				t.Fatalf("unsafe continuation: %+v %v", receipt, d)
			}
		})
	}
}
func TestContinuationRestartAndNeverStarted(t *testing.T) {
	ws, raw, in := stoppedContinuation(t, "trim")
	r, e := ReviewPreparedContinuation(t.Context(), ws, raw, in, testInstallIdentity())
	if e != nil {
		t.Fatal(e)
	}
	if r.Steps[0].Action != "restart" || r.Steps[0].PlannedAttempt != 2 || r.Steps[1].Action != "start" || r.Steps[1].PlannedAttempt != 1 {
		t.Fatal(r.Steps)
	}
	useExec(t, continuationExecutor(t, in, "", nil))
	if _, d := ContinuePrepared(t.Context(), ws, raw, in, continuationIntent(r), testInstallIdentity()); len(d) > 0 {
		t.Fatal(d)
	}
	tasks := taskStates(t, ws)
	if tasks["trim"].Attempt != 2 || tasks["qc"].Attempt != 1 {
		t.Fatal(tasks)
	}
}
