package appservice

import (
	"context"
	"github.com/HahyeonJeon/gobble/internal/preparation"
	"os"
	"path/filepath"
	"testing"
)

func readyLaunchFixture(t *testing.T) (*Service, Project, launchRecord) {
	t.Helper()
	s, p, pipeline, artifact := preparationFixture(t, false)
	_, e := s.preparePipeline(p.ProjectID, pipeline, prepareInput{RequestID: "req_prepare", ArtifactID: artifact})
	if e != nil {
		t.Fatal(e)
	}
	v := awaitPreparation(t, s, p.ProjectID, pipeline, "req_prepare")
	if v.State != "ready" {
		t.Fatal(v)
	}
	var prior preparationRecord
	_ = readProposalFile(filepath.Join(s.preparationDirectory(pipeline, v.RequestID), "record.json"), &prior)
	project, _ := s.store.project(p.ProjectID)
	target := "runs/analysis-test"
	if e = os.MkdirAll(filepath.Join(project.Root, target, "inputs"), 0700); e != nil {
		t.Fatal(e)
	}
	writeTestFile(t, filepath.Join(project.Root, target, ".gobble-launch-target"), []byte("req_test"))
	info, _ := os.Stat(filepath.Join(project.Root, target))
	rec := launchRecord{SchemaVersion: 1, Runtime: prior.Runtime, TargetIdentity: rootIdentity(info), Value: launchReview{ProjectID: p.ProjectID, PipelineID: pipeline, RequestID: "req_test", PreparationID: v.RequestID, CreatedAt: v.CreatedAt, State: "ready", Fresh: true, Preparation: v, OutputPath: target, InputSHA256: digest([]byte("copy")), TotalBytes: 4, CopiedBytes: 4, ReadCount: 1}, Intent: preparation.LaunchIntent{SchemaVersion: 1, PreparedDigest: v.Prepared.Digest, Binding: v.Prepared.Binding, WorkspaceID: "req_test", EngineImage: prior.Runtime.ImageID, DaemonID: prior.Runtime.DaemonID, InputSHA256: digest([]byte("copy")), InputSize: 4, Tools: map[string]string{}}}
	rec.Runtime.WorkspacePath = workspaceContainerPath(target)
	if e = os.MkdirAll(filepath.Join(s.launchDirectory(p.ProjectID, "req_test"), "bundle"), 0700); e != nil {
		t.Fatal(e)
	}
	if e = s.saveLaunch(rec); e != nil {
		t.Fatal(e)
	}
	// Hold the worker slot to inspect the durable accepted boundary without dispatch.
	s.launchFlights = map[string]*pipelineFlight{p.ProjectID + "/req_test": {id: "req_test", cancel: func() {}}}
	return s, p, rec
}
func TestLaunchStartReplayAndStopExpectedOwner(t *testing.T) {
	s, p, rec := readyLaunchFixture(t)
	accepted, e := s.startLaunch(p.ProjectID, launchStartInput{"req_start", "req_test"})
	if e != nil || accepted.Operation.State != "accepted" {
		t.Fatal(accepted, e)
	}
	saved, e := s.readLaunch(p.ProjectID, "req_test")
	if e != nil || saved.Intent.RequestID != "req_start" {
		t.Fatal("intent not durable", e)
	}
	same, e := s.startLaunch(p.ProjectID, launchStartInput{"req_start", "req_test"})
	if e != nil || same.Operation.RequestID != "req_start" {
		t.Fatal(e)
	}
	_, e = s.startLaunch(p.ProjectID, launchStartInput{"req_other", "req_test"})
	wantCode(t, e, "request_conflict")
	saved.Value.Operation.OwnerLease = "observed-owner"
	_ = s.saveLaunch(saved)
	_, e = s.stopLaunch(p.ProjectID, "req_test", launchStopInput{"req_stop", "earlier-owner"})
	wantCode(t, e, "stale_revision")
	stopped, e := s.stopLaunch(p.ProjectID, "req_test", launchStopInput{"req_stop", "observed-owner"})
	if e != nil || stopped.Operation.StopState != "requested" || stopped.Operation.StopLease != "observed-owner" {
		t.Fatal(stopped, e)
	}
	// A concurrent observation cannot erase a committed Stop request.
	saved.Value.Operation.StopRequestID = ""
	saved.Value.Operation.StopState = ""
	saved.Value.Operation.StopLease = ""
	if e = s.persistLaunchProgress(saved); e != nil {
		t.Fatal(e)
	}
	latest, _ := s.readLaunch(p.ProjectID, rec.Value.RequestID)
	if latest.Value.Operation.StopRequestID != "req_stop" {
		t.Fatal("Stop intent lost")
	}
}
func TestLaunchFreshnessAndCorruptRecordRejectBeforeStart(t *testing.T) {
	s, p, rec := readyLaunchFixture(t)
	pr, _ := s.store.project(p.ProjectID)
	writeTestFile(t, filepath.Join(pr.Root, rec.Value.OutputPath, ".gobble-launch-target"), []byte("other"))
	_, e := s.startLaunch(p.ProjectID, launchStartInput{"req_start", "req_test"})
	wantCode(t, e, "stale_revision")
	saved, _ := s.readLaunch(p.ProjectID, "req_test")
	if saved.Value.Operation != nil {
		t.Fatal("stale start admitted")
	}
	saved.Value.OutputPath = "runs/analysis-replaced"
	if e = writeInspectionJSON(filepath.Join(s.launchDirectory(p.ProjectID, "req_test"), "record.json"), saved); e != nil {
		t.Fatal(e)
	}
	_, e = s.readLaunch(p.ProjectID, "req_test")
	wantCode(t, e, "internal")
}
func TestLaunchTargetRejectsChangedContentAndExtraFiles(t *testing.T) {
	s, p, rec := readyLaunchFixture(t)
	pr, _ := s.store.project(p.ProjectID)
	input := filepath.Join(pr.Root, rec.Value.OutputPath, rec.Intent.Binding.InputPath)
	writeTestFile(t, input, []byte("copy"))
	if e := verifyLaunchTarget(context.Background(), pr, rec); e != nil {
		t.Fatal(e)
	}
	writeTestFile(t, input, []byte("edit"))
	if e := verifyLaunchTarget(context.Background(), pr, rec); e == nil {
		t.Fatal("changed input accepted")
	}
	writeTestFile(t, input, []byte("copy"))
	writeTestFile(t, filepath.Join(pr.Root, rec.Value.OutputPath, "foreign"), []byte("keep"))
	if e := verifyLaunchTarget(context.Background(), pr, rec); e == nil {
		t.Fatal("foreign result file accepted")
	}
}
func TestLaunchCheckCancellationRetainsNoRun(t *testing.T) {
	s, p, rec := readyLaunchFixture(t)
	v, e := s.cancelLaunch(p.ProjectID, rec.Value.RequestID)
	if e != nil || v.State != "cancelled" {
		t.Fatal(v, e)
	}
	_, e = s.startLaunch(p.ProjectID, launchStartInput{"req_start", rec.Value.RequestID})
	wantCode(t, e, "stale_revision")
	if len(s.store.data.Runs) != 0 {
		t.Fatal("cancel started Run")
	}
}

func TestLaunchAmbiguityCannotBeBypassedWithAnotherReview(t *testing.T) {
	s, p, rec := readyLaunchFixture(t)
	if _, err := s.startLaunch(p.ProjectID, launchStartInput{"req_start", rec.Value.RequestID}); err != nil {
		t.Fatal(err)
	}
	_, err := s.checkLaunch(p.ProjectID, rec.Value.PipelineID, launchReviewInput{"req_new", rec.Value.PreparationID})
	wantCode(t, err, "request_conflict")
	if _, err := os.Stat(s.launchDirectory(p.ProjectID, "req_new")); !os.IsNotExist(err) {
		t.Fatal("another review was allocated during ambiguous admission")
	}
}

func TestLaunchServiceCloseIsIdempotent(t *testing.T) {
	s := testService(t)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal("repeated shutdown", err)
	}
}
