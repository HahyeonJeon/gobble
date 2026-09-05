package engine

import (
	"context"
	"encoding/json"
	"os"
	osexec "os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestStopAcrossProcessesAndAutomaticResume(t *testing.T) {
	workspace := t.TempDir()
	t.Cleanup(func() { DropHeldLease(workspace) })
	writeCheckFile(t, filepath.Join(workspace, "in", "sample.txt"), "reads")
	req := Request{Identity: testInstallIdentity(), Workspace: workspace, Document: sampleDoc("", "", "in/sample.txt", "out/sample.txt")}
	req.Document.Tasks[0].Command = []string{"sh", "-c", "sleep 60; cp in/sample.txt out/sample.txt"}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	done := make(chan []Defect, 1)
	go func() { done <- Run(ctx, req) }()
	waitRuntimeID(t, workspace, "copy")
	if d := Resume(ctx, req); !hasDefectCode(d, DefectOccupiedWorkspace) {
		t.Fatalf("live Resume = %v", d)
	}
	before, _, _ := readRunIdentity(workspace)
	cmd := osexec.CommandContext(ctx, os.Args[0], "-test.run=^TestStopChild$")
	cmd.Env = append(os.Environ(), "GOBBLE_TEST_STOP_WORKSPACE="+workspace)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Stop child: %v\n%s", err, out)
	}
	if d := <-done; !hasDefectCode(d, DefectCanceled) {
		t.Fatalf("Run = %v", d)
	}
	stopped, _, _ := readRunIdentity(workspace)
	if stopped.Status != RunStopped || occupancyIsActive(stopped) {
		t.Fatalf("Stop not settled: %+v", stopped)
	}
	if result, d := Stop(ctx, workspace, req.Identity); len(d) > 0 || result.Status != "settled" {
		t.Fatalf("repeated Stop = %+v %v", result, d)
	}
	// A request addressed to the old lease must not cancel a later attempt.
	req.Document.Tasks[0].Command = []string{"cp", "in/sample.txt", "out/sample.txt"}
	if d := Resume(ctx, req); len(d) > 0 {
		t.Fatalf("Resume = %v", d)
	}
	after, _, _ := readRunIdentity(workspace)
	if after.Occupancy.Lease == before.Occupancy.Lease || after.Status != StatusSucceeded {
		t.Fatalf("new owner = %+v", after)
	}
	if task := taskStates(t, workspace)["copy"]; task.Attempt != 2 || task.Status != StatusSucceeded {
		t.Fatalf("resumed task = %+v", task)
	}
}

func TestStopChild(t *testing.T) {
	workspace := os.Getenv("GOBBLE_TEST_STOP_WORKSPACE")
	if workspace == "" {
		return
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	result, d := Stop(ctx, workspace, testInstallIdentity())
	if len(d) > 0 || result.Status != "settled" {
		t.Fatalf("Stop = %+v %v", result, d)
	}
}

func TestStopWaitCancellationKeepsOwnerAddressedRequest(t *testing.T) {
	req, _ := seedCheckpoint(t)
	// Simulate a live owner that does not consume its control channel.
	run, plan, _, tasks, _, d := readCoherentControl(req.Workspace)
	if len(d) > 0 {
		t.Fatal(d)
	}
	lock, d := claimOccupy(filepath.Join(req.Workspace, ControlDir))
	if len(d) > 0 {
		t.Fatal(d)
	}
	defer lock.Close()
	run.Status = StatusRunning
	s := releaseSched(req.Workspace, run, tasks.Tasks)
	s.snapshot = newOccupancyID()
	if err := s.writeReleasedCheckpoint(plan); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	result, d := Stop(ctx, req.Workspace, req.Identity)
	if len(d) > 0 || result.Status != "requested" {
		t.Fatalf("Stop = %+v %v", result, d)
	}
	path, err := stopPath(req.Workspace, run.Occupancy.Lease)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var request stopRequest
	if err := json.Unmarshal(data, &request); err != nil || request.Lease != run.Occupancy.Lease {
		t.Fatalf("request = %s %v", data, err)
	}
}

func TestDelayedStopDoesNotReleaseNewerLease(t *testing.T) {
	req, _ := seedCheckpoint(t)
	if d := Resume(t.Context(), req); len(d) > 0 {
		t.Fatal(d)
	}
	held := heldLeaseFor(req.Workspace)
	before, _, _ := readRunIdentity(req.Workspace)
	if held == nil {
		t.Fatal("missing new lease")
	}
	if d := settleStop(req.Workspace, req.Identity, newOccupancyID()); len(d) > 0 {
		t.Fatal(d)
	}
	after, _, _ := readRunIdentity(req.Workspace)
	if heldLeaseFor(req.Workspace) != held || before.Snapshot != after.Snapshot {
		t.Fatal("stale Stop changed the new owner")
	}
}

func TestResumeRefusalDoesNotRetainAClosedOwnerLock(t *testing.T) {
	req, _ := seedCheckpoint(t)
	if d := Resume(t.Context(), req); len(d) > 0 {
		t.Fatal(d)
	}
	writeCheckFile(t, filepath.Join(req.Workspace, "out", "unrelated.txt"), "keep")
	req.Document.Tasks[0].Outputs[0].Path = "out/unrelated.txt"
	if d := Resume(t.Context(), req); !hasDefectCode(d, DefectOutputExists) {
		t.Fatalf("Resume = %v", d)
	}
	if heldLeaseFor(req.Workspace) != nil || flockHeld(req.Workspace) {
		t.Fatal("refused Resume retained the reconciled owner's lock")
	}
}
