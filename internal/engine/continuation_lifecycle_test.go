package engine

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HahyeonJeon/gobble/internal/engine/exec"
	"github.com/HahyeonJeon/gobble/internal/preparation"
)

func TestContinuationStaleReviewAndCanceledIntent(t *testing.T) {
	for _, scenario := range []string{"stale", "canceled", "occupied", "schema1", "payload", "workspace"} {
		t.Run(scenario, func(t *testing.T) {
			ws, raw, in := stoppedContinuation(t, "qc")
			r, e := ReviewPreparedContinuation(t.Context(), ws, raw, in, testInstallIdentity())
			if e != nil {
				t.Fatal(e)
			}
			ctx := t.Context()
			switch scenario {
			case "stale":
				mutateContinuationControl(t, ws, func(*jsonRun, *jsonPlan, *jsonTasksFile) {})
			case "canceled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "occupied":
				lock, d := claimOccupy(workspaceFile(ws, ControlDir))
				if len(d) > 0 {
					t.Fatal(d)
				}
				defer lock.Close()
			case "schema1":
				in.SchemaVersion = 1
			case "payload":
				raw = append(raw, ' ')
			case "workspace":
				in.WorkspaceID = "req_other"
			}
			before, _, _ := readRunIdentity(ws)
			var calls atomic.Int32
			useExec(t, continuationExecutor(t, in, "", func(exec.Job) { calls.Add(1) }))
			receipt, d := ContinuePrepared(ctx, ws, raw, in, continuationIntent(r), testInstallIdentity())
			after, _, _ := readRunIdentity(ws)
			if len(d) == 0 || receipt.Lease != "" || calls.Load() != 0 || !reflect.DeepEqual(before, after) {
				t.Fatalf("refused intent caused effects: %+v %v", receipt, d)
			}
		})
	}
}

func TestContinuationCompetingIntentAndStopEpoch(t *testing.T) {
	ws, raw, in := stoppedContinuation(t, "qc")
	origin, _, _ := readRunIdentity(ws)
	r, e := ReviewPreparedContinuation(t.Context(), ws, raw, in, testInstallIdentity())
	if e != nil {
		t.Fatal(e)
	}
	intent := continuationIntent(r)
	var calls atomic.Int32
	useExec(t, continuationExecutor(t, in, "qc", func(exec.Job) { calls.Add(1) }))
	type result struct {
		receipt preparation.ContinuationAdmission
		defects []Defect
	}
	done := make(chan result, 1)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	go func() {
		receipt, d := ContinuePrepared(ctx, ws, raw, in, intent, testInstallIdentity())
		done <- result{receipt, d}
	}()
	for {
		run, _, err := readRunIdentity(ws)
		if err == nil && run.Occupancy.Lease != origin.Occupancy.Lease && calls.Load() == 1 {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("continuation did not start", ctx.Err())
		case <-time.After(10 * time.Millisecond):
		}
	}
	current, _, _ := readRunIdentity(ws)
	if current.Occupancy.Lease == origin.Occupancy.Lease {
		t.Fatal("no new owner")
	}
	replay, d := ContinuePrepared(ctx, ws, raw, in, intent, testInstallIdentity())
	if len(d) > 0 || replay.Lease != current.Occupancy.Lease {
		t.Fatal("live exact replay", d)
	}
	competitor := intent
	competitor.RequestID = "req_competing"
	if _, d := ContinuePrepared(ctx, ws, raw, in, competitor, testInstallIdentity()); len(d) == 0 {
		t.Fatal("competing intent admitted")
	}
	if _, e := ReviewPreparedContinuation(ctx, ws, raw, in, testInstallIdentity()); e == nil {
		t.Fatal("active owner reviewed")
	}
	if result, d := StopOwner(ctx, ws, testInstallIdentity(), origin.Occupancy.Lease); len(d) != 0 || result.Status != "owner-changed" {
		t.Fatal("old Stop affected new owner", result, d)
	}
	if _, d := StopOwner(ctx, ws, testInstallIdentity(), current.Occupancy.Lease); len(d) > 0 {
		t.Fatal(d)
	}
	select {
	case got := <-done:
		if got.receipt != replay || !hasDefectCode(got.defects, DefectCanceled) {
			t.Fatal(got)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	DropHeldLease(ws)
	next, e := ReviewPreparedContinuation(ctx, ws, raw, in, testInstallIdentity())
	if e != nil {
		t.Fatal(e)
	}
	if next.PreviousHead == r.PreviousHead || next.Steps[1].PlannedAttempt != 3 {
		t.Fatal(next)
	}
	nextIntent := continuationIntent(next)
	nextIntent.RequestID = "req_continue_again"
	useExec(t, continuationExecutor(t, in, "", func(exec.Job) { calls.Add(1) }))
	if _, d := ContinuePrepared(ctx, ws, raw, in, nextIntent, testInstallIdentity()); len(d) > 0 {
		t.Fatal(d)
	}
	after, _, _ := readRunIdentity(ws)
	if calls.Load() != 2 || after.ID != origin.ID || len(after.ExecutionHistory.Continuations) != 2 {
		t.Fatal("duplicate scheduling or lost history", after)
	}
	saved, ok, e := ReadContinuationAdmission(ws, intent, testInstallIdentity())
	if e != nil || !ok || saved != replay {
		t.Fatal("old acknowledgement lost", e)
	}
	_, _, _, tasks, _, d := readCoherentControl(ws)
	if len(d) > 0 {
		t.Fatal(d)
	}
	attempts := map[int]string{}
	for _, task := range tasks.Tasks {
		if task.ID == "qc" {
			attempts[task.Attempt] = task.Status
		}
	}
	if !reflect.DeepEqual(attempts, map[int]string{1: StatusIncomplete, 2: StatusIncomplete, 3: StatusSucceeded}) {
		t.Fatal(attempts)
	}
}

func TestContinuationChecksStagedReviewedInputBeforeSubmit(t *testing.T) {
	ws, raw, in := stoppedContinuation(t, "qc")
	checked, e := inspectContinuation(t.Context(), ws, raw, in, testInstallIdentity())
	if e != nil {
		t.Fatal(e)
	}
	s := continuationScheduler(ws, checked, preparation.ContinuationAdmission{})
	task := checked.doc.Tasks[1]
	task.Attempt = 2
	// The mutation occurs after review and scheduler construction. Staging must
	// compare the bytes actually copied, before allowing any backend submission.
	path := declaredIOFiles(checked.doc.Tasks[0].Outputs)[0].path
	writeCheckFile(t, filepath.Join(ws, path), "changed after admission")
	var calls atomic.Int32
	ex := continuationExecutor(t, in, "", func(exec.Job) { calls.Add(1) })
	reports := make(chan report, 1)
	s.runJob(t.Context(), ws, task, ex, nil, make(chan startEvent, 1), reports)
	got := <-reports
	if calls.Load() != 0 || got.Message != "reviewed input changed before task execution" {
		t.Fatalf("staged drift was submitted: %+v calls=%d", got, calls.Load())
	}
}

func TestContinuationReceiptSurvivesUncertainPublication(t *testing.T) {
	for _, boundary := range []string{"generation", "pointer"} {
		t.Run(boundary, func(t *testing.T) {
			ws, raw, in := stoppedContinuation(t, "qc")
			checked, e := inspectContinuation(t.Context(), ws, raw, in, testInstallIdentity())
			if e != nil {
				t.Fatal(e)
			}
			intent := continuationIntent(checked.review)
			receipt := preparation.ContinuationAdmission{Intent: intent, IntentDigest: continuationIntentDigest(intent), Lease: newOccupancyID(), Snapshot: newOccupancyID()}
			// Use the checkpoint's existing per-call fault boundary: no global production
			// injection or fake successful scheduler is needed to exercise acknowledgement.
			run := checked.run
			history := *run.ExecutionHistory
			history.Continuations = append(history.Continuations, receipt)
			run.ExecutionHistory = &history
			run.Snapshot = receipt.Snapshot
			occupancy := *run.Occupancy
			occupancy.Lease = receipt.Lease
			occupancy.Active = true
			run.Occupancy = &occupancy
			run.Status = StatusRunning
			_, plan, _, tasks, _, _ := readCoherentControl(ws)
			plan.Snapshot = run.Snapshot
			tasks.Snapshot = run.Snapshot
			rb, _ := json.Marshal(run)
			pb, _ := json.Marshal(plan)
			tb, _ := json.Marshal(tasks)
			fault := errors.New("publication interrupted")
			e = commitCheckpointAt(ws, run.Snapshot, pb, tb, rb, func(step string) error {
				if step == boundary {
					return fault
				}
				return nil
			})
			if !errors.Is(e, fault) {
				t.Fatal(e)
			}
			got, exists, e := ReadContinuationAdmission(ws, intent, testInstallIdentity())
			if e != nil {
				t.Fatal(e)
			}
			if boundary == "generation" {
				if exists {
					t.Fatal("uncommitted intent acknowledged")
				}
				return
			}
			if !exists || got != receipt {
				t.Fatal("committed receipt lost", got)
			}
			var calls atomic.Int32
			useExec(t, continuationExecutor(t, in, "", func(exec.Job) { calls.Add(1) }))
			got, d := ContinuePrepared(t.Context(), ws, raw, in, intent, testInstallIdentity())
			if len(d) > 0 || got != receipt || calls.Load() != 0 {
				t.Fatal("uncertain acknowledgement rescheduled", d)
			}
		})
	}
}

func TestContinuationNeverOverwritesOutputCreatedAfterReview(t *testing.T) {
	ws, raw, in := stoppedContinuation(t, "qc")
	review, err := ReviewPreparedContinuation(t.Context(), ws, raw, in, testInstallIdentity())
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(ws, declaredIOFiles(creationDocument().Tasks[1].Outputs)[0].path)
	useExec(t, continuationExecutor(t, in, "", func(exec.Job) { writeCheckFile(t, destination, "external result after admission") }))
	receipt, d := ContinuePrepared(t.Context(), ws, raw, in, continuationIntent(review), testInstallIdentity())
	if receipt.Lease == "" || len(d) == 0 {
		t.Fatal("publication collision did not fail", receipt, d)
	}
	content, err := os.ReadFile(destination)
	if err != nil || string(content) != "external result after admission" {
		t.Fatal("unreviewed output overwritten", err)
	}
}
