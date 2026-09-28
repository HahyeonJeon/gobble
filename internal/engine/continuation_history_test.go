package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/internal/engine/exec"
	"github.com/HahyeonJeon/gobble/internal/preparation"
)

func seedPreparedHistory(t *testing.T) (string, []byte, preparation.LaunchIntent) {
	t.Helper()
	ws, raw, in := launchFixture(t)
	in.SchemaVersion = 2
	useExec(t, &fnExec{submit: func(context.Context, exec.Job) (exec.Handle, exec.Report, error) {
		return exec.Handle{}, exec.Report{}, errors.New("fixture stops after initial admission")
	}})
	_ = RunPrepared(t.Context(), ws, raw, in, testInstallIdentity())
	DropHeldLease(ws)
	if _, exists, err := ReadAdmission(ws); err != nil || !exists {
		t.Fatalf("seed admission exists=%v err=%v", exists, err)
	}
	run, _, err := readRunIdentity(ws)
	if err != nil {
		t.Fatal(err)
	}
	// Storage fixture only: no claim that a failed task is resumable. The real
	// continuation reviewer will enforce task/output eligibility in the next slice.
	run.Status, run.Occupancy.Active = RunStopped, false
	run.Snapshot = newOccupancyID()
	if err := saveHistoryFixture(ws, run); err != nil {
		t.Fatal(err)
	}
	return ws, raw, in
}

func appendHistoryFixture(t *testing.T, run *jsonRun) {
	t.Helper()
	epoch, ok := admissionEpoch(*run)
	if !ok {
		t.Fatal("cannot append to invalid fixture")
	}
	intent := preparation.ContinuationIntent{SchemaVersion: 1,
		RequestID:   fmt.Sprintf("req_continue_%d", len(run.ExecutionHistory.Continuations)+1),
		WorkspaceID: run.Admission.WorkspaceID, OriginDigest: run.Admission.IntentDigest,
		ReviewDigest: "sha256:" + strings.Repeat("c", 64), PreviousHead: epoch.Head}
	run.Snapshot = newOccupancyID()
	r := preparation.ContinuationAdmission{Intent: intent, IntentDigest: continuationIntentDigest(intent), Lease: newOccupancyID(), Snapshot: run.Snapshot}
	run.ExecutionHistory.Continuations = append(run.ExecutionHistory.Continuations, r)
	run.Occupancy.Lease = r.Lease
}

func saveHistoryFixture(workspace string, run jsonRun) error {
	_, plan, _, tasks, _, d := readCoherentControl(workspace)
	if len(d) > 0 {
		return fmt.Errorf("read controls: %v", d)
	}
	plan.Snapshot, tasks.Snapshot = run.Snapshot, run.Snapshot
	r, err := json.Marshal(run)
	if err != nil {
		return err
	}
	p, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	s, err := json.Marshal(tasks)
	if err != nil {
		return err
	}
	return commitCheckpoint(workspace, run.Snapshot, p, s, r)
}

func TestPreparedHistoryKeepsStartAndReceiptsAfterPruning(t *testing.T) {
	ws, raw, in := seedPreparedHistory(t)
	origin, _, _ := ReadAdmission(ws)
	for range 3 {
		run, _, err := readRunIdentity(ws)
		if err != nil {
			t.Fatal(err)
		}
		appendHistoryFixture(t, &run)
		if err := saveHistoryFixture(ws, run); err != nil {
			t.Fatal(err)
		}
	}
	if err := copyCheckpoint(ws, newOccupancyID(), nil); err != nil {
		t.Fatal(err)
	}
	run, _, err := readRunIdentity(ws)
	if err != nil || len(run.ExecutionHistory.Continuations) != 3 {
		t.Fatalf("retained history: %+v %v", run, err)
	}
	got, _, err := ReadAdmission(ws)
	if err != nil || got != origin {
		t.Fatalf("origin changed: got=%+v want=%+v err=%v", got, origin, err)
	}
	epoch, ok := admissionEpoch(run)
	if !ok || epoch.Lease == origin.Lease || epoch.Lease != run.Occupancy.Lease {
		t.Fatalf("incorrect epoch: %+v valid=%v", epoch, ok)
	}
	entries, err := os.ReadDir(filepath.Join(ws, ControlDir, checkpointDirectory))
	if err != nil || len(entries) != 2 {
		t.Fatalf("checkpoint retention: count=%d err=%v", len(entries), err)
	}
	if d := RunPrepared(t.Context(), ws, raw, in, testInstallIdentity()); len(d) > 0 {
		t.Fatalf("original Start replay: %v", d)
	}
	if d := Resume(t.Context(), Request{Workspace: ws, Identity: testInstallIdentity(), Document: creationDocument()}); !hasDefectCode(d, DefectUnsupportedSchema) {
		t.Fatalf("ordinary Resume bypassed admission: %v", d)
	}
}

func TestPreparedHistoryRejectsInvalidChain(t *testing.T) {
	ws, _, _ := seedPreparedHistory(t)
	base, _, _ := readRunIdentity(ws)
	appendHistoryFixture(t, &base)
	b, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*jsonRun){
		"history-version":  func(r *jsonRun) { r.ExecutionHistory.SchemaVersion = 99 },
		"missing-history":  func(r *jsonRun) { r.ExecutionHistory = nil },
		"null-chain":       func(r *jsonRun) { r.ExecutionHistory.Continuations = nil },
		"schema-downgrade": func(r *jsonRun) { r.Admission.SchemaVersion = 1 },
		"owner":            func(r *jsonRun) { r.Occupancy.Lease = r.Admission.Lease },
		"origin": func(r *jsonRun) {
			r.ExecutionHistory.Continuations[0].Intent.OriginDigest = "sha256:" + strings.Repeat("d", 64)
		},
		"workspace": func(r *jsonRun) { r.ExecutionHistory.Continuations[0].Intent.WorkspaceID = "req_elsewhere" },
		"head": func(r *jsonRun) {
			r.ExecutionHistory.Continuations[0].Intent.PreviousHead = "sha256:" + strings.Repeat("d", 64)
		},
		"review": func(r *jsonRun) { r.ExecutionHistory.Continuations[0].Intent.ReviewDigest = "invalid" },
		"digest": func(r *jsonRun) {
			r.ExecutionHistory.Continuations[0].IntentDigest = "sha256:" + strings.Repeat("d", 64)
		},
		"lease-reused": func(r *jsonRun) {
			r.ExecutionHistory.Continuations[0].Lease = r.Admission.Lease
			r.Occupancy.Lease = r.Admission.Lease
		},
		"request-reused": func(r *jsonRun) {
			c := &r.ExecutionHistory.Continuations[0]
			c.Intent.RequestID = r.Admission.RequestID
			c.IntentDigest = continuationIntentDigest(c.Intent)
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			var r jsonRun
			if err := json.Unmarshal(b, &r); err != nil {
				t.Fatal(err)
			}
			change(&r)
			if _, ok := admissionEpoch(r); ok {
				t.Fatal("invalid chain accepted")
			}
		})
	}
	for len(base.ExecutionHistory.Continuations) < preparation.MaxContinuations {
		appendHistoryFixture(t, &base)
	}
	if _, ok := admissionEpoch(base); !ok {
		t.Fatal("valid bounded history rejected")
	}
	appendHistoryFixture(t, &base)
	if _, ok := admissionEpoch(base); ok {
		t.Fatal("history limit accepted")
	}
}

func TestPreparedHistoryCannotDowngradePointer(t *testing.T) {
	for _, format := range []int{1, 2} {
		t.Run(fmt.Sprint(format), func(t *testing.T) {
			ws, _, _ := seedPreparedHistory(t)
			path := filepath.Join(ws, ControlDir, checkpointPointerFile)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var ptr checkpointPointer
			if err := json.Unmarshal(raw, &ptr); err != nil {
				t.Fatal(err)
			}
			if ptr.Format != 3 {
				t.Fatalf("format=%d want=3", ptr.Format)
			}
			ptr.Format = format
			raw, err = json.Marshal(ptr)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			if _, _, err := ReadAdmission(ws); err == nil {
				t.Fatal("downgraded pointer accepted")
			}
		})
	}
}

func TestPreparedHistoryRejectsRewriteBeforePublication(t *testing.T) {
	ws, _, _ := seedPreparedHistory(t)
	run, _, _ := readRunIdentity(ws)
	appendHistoryFixture(t, &run)
	if err := saveHistoryFixture(ws, run); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(ws, ControlDir, checkpointPointerFile))
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"drop", "replace", "origin", "snapshot"} {
		t.Run(kind, func(t *testing.T) {
			r, _, err := readRunIdentity(ws)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "drop":
				r.ExecutionHistory.Continuations = []preparation.ContinuationAdmission{}
				r.Occupancy.Lease = r.Admission.Lease
			case "replace":
				c := &r.ExecutionHistory.Continuations[0]
				c.Intent.ReviewDigest = "sha256:" + strings.Repeat("e", 64)
				c.IntentDigest = continuationIntentDigest(c.Intent)
			case "origin":
				r.Started = "changed"
			case "snapshot":
				appendHistoryFixture(t, &r)
			}
			r.Snapshot = newOccupancyID()
			if err := saveHistoryFixture(ws, r); err == nil {
				t.Fatal("history rewrite published")
			}
			after, err := os.ReadFile(filepath.Join(ws, ControlDir, checkpointPointerFile))
			if err != nil || string(after) != string(before) {
				t.Fatalf("pointer changed: %v", err)
			}
		})
	}
}

func TestPreparedHistoryRefusesOldFormatConversion(t *testing.T) {
	ws, raw, in := launchFixture(t)
	useExec(t, &fnExec{submit: func(context.Context, exec.Job) (exec.Handle, exec.Report, error) {
		return exec.Handle{}, exec.Report{}, errors.New("fixture")
	}})
	_ = RunPrepared(t.Context(), ws, raw, in, testInstallIdentity())
	DropHeldLease(ws)
	run, _, err := readRunIdentity(ws)
	if err != nil {
		t.Fatal(err)
	}
	run.Admission.SchemaVersion = 2
	run.ExecutionHistory = &preparation.ExecutionHistory{SchemaVersion: 1, Continuations: []preparation.ContinuationAdmission{}}
	run.Snapshot = newOccupancyID()
	if err := saveHistoryFixture(ws, run); err == nil {
		t.Fatal("old format converted")
	}
	got, exists, err := ReadAdmission(ws)
	if err != nil || !exists || got.SchemaVersion != 1 {
		t.Fatalf("old admission unreadable: %+v %v", got, err)
	}
}

func TestPreparedHistoryRequiresSettledStopToAppend(t *testing.T) {
	for _, state := range []string{StatusRunning, RunStopping, RunInterrupted, StatusFailed, "active-stopped"} {
		t.Run(state, func(t *testing.T) {
			ws, _, _ := seedPreparedHistory(t)
			run, _, _ := readRunIdentity(ws)
			run.Status = state
			if state == "active-stopped" {
				run.Status = RunStopped
				run.Occupancy.Active = true
			}
			run.Snapshot = newOccupancyID()
			if err := saveHistoryFixture(ws, run); err != nil {
				t.Fatal(err)
			}
			appendHistoryFixture(t, &run)
			if err := saveHistoryFixture(ws, run); err == nil {
				t.Fatal("non-settled Run accepted continuation")
			}
		})
	}
}

func TestPreparedHistoryRejectsFlatLayout(t *testing.T) {
	ws, _, _ := seedPreparedHistory(t)
	run, _, _ := readRunIdentity(ws)
	b, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, ControlDir, RunIdentityFile), b, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(ws, ControlDir, checkpointPointerFile)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadAdmission(ws); err == nil {
		t.Fatal("history accepted outside committed generation")
	}
	if _, _, _, _, _, d := readCoherentControl(ws); len(d) == 0 {
		t.Fatal("flat history accepted by inspection")
	}
}

func TestPreparedHistoryObservationSeparatesOriginAndCurrentEpoch(t *testing.T) {
	oldBuild, oldDigest := installReadBuildInfo, installExecutableDigest
	t.Cleanup(func() { installReadBuildInfo, installExecutableDigest = oldBuild, oldDigest })
	installReadBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Path: installGobbleModule}}, true
	}
	installExecutableDigest = func() (string, error) { return strings.Repeat("a", 64), nil }
	for _, version := range []int{1, 2} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			ws, raw, in := launchFixture(t)
			in.SchemaVersion = version
			useExec(t, &fnExec{submit: func(context.Context, exec.Job) (exec.Handle, exec.Report, error) {
				return exec.Handle{}, exec.Report{}, errors.New("fixture")
			}})
			_ = RunPrepared(t.Context(), ws, raw, in, testInstallIdentity())
			DropHeldLease(ws)
			state, err := InspectAdmission(ws)
			if err != nil || state.SchemaVersion != version {
				t.Fatalf("state version=%d want=%d err=%v", state.SchemaVersion, version, err)
			}
			if version == 1 {
				b, err := json.Marshal(state)
				if err != nil {
					t.Fatal(err)
				}
				if state.Epoch != nil || strings.Contains(string(b), "\"epoch\"") {
					t.Fatalf("legacy wire contains epoch: %s", b)
				}
				return
			}
			if state.Epoch == nil || state.Epoch.Lease != state.Admission.Lease || state.Epoch.Head != LaunchDigest(in) {
				t.Fatalf("initial epoch=%+v", state.Epoch)
			}
			origin := state.Admission
			run, _, _ := readRunIdentity(ws)
			run.Status = RunStopped
			run.Occupancy.Active = false
			run.Snapshot = newOccupancyID()
			if err := saveHistoryFixture(ws, run); err != nil {
				t.Fatal(err)
			}
			appendHistoryFixture(t, &run)
			if err := saveHistoryFixture(ws, run); err != nil {
				t.Fatal(err)
			}
			state, err = InspectAdmission(ws)
			if err != nil || state.Admission != origin || state.Epoch == nil || state.Epoch.Lease == origin.Lease || state.Epoch.Lease != run.Occupancy.Lease {
				t.Fatalf("origin/epoch mixed: %+v err=%v", state, err)
			}
		})
	}
}
