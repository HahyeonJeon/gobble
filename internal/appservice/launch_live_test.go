package appservice

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Explicit opt-in: only the synthetic fixture below is executed.
func TestLivePreparedLaunch(t *testing.T) {
	image := os.Getenv("GOBBLE_LAUNCH_IMAGE")
	if image == "" {
		t.Skip("requires qualified local execution engine and tool images")
	}
	s, p, file, d := draftFixture(t)
	project, _ := s.store.project(p.ProjectID)
	var reads strings.Builder
	readTotal := 100
	stopMode := os.Getenv("GOBBLE_LAUNCH_STOP") == "1"
	if stopMode {
		readTotal = 200000
	}
	for i := 0; i < readTotal; i++ {
		fmt.Fprintf(&reads, "@synthetic-%d\n%s\n+\n%s\n", i, strings.Repeat("ACGT", 20), strings.Repeat("I", 80))
	}
	// draftFixture registers this exact input; observation happens after replacement.
	var data bytes.Buffer
	gz := gzip.NewWriter(&data)
	_, _ = gz.Write([]byte(reads.String()))
	_ = gz.Close()
	if err := os.WriteFile(filepath.Join(project.Root, project.Resources[file]), data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	d, err := s.store.updateDraft(p.ProjectID, d.DraftID, updateDraftInput{"req_input", 1, "Synthetic read quality", file, "single-end"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	endpoint, err := s.runtime.command(ctx, []string{"context", "inspect", "--format", "{{.Endpoints.docker.Host}}"}, os.Environ())
	if err != nil {
		t.Fatal(err)
	}
	daemon, err := s.runtime.commandAt(ctx, strings.TrimSpace(string(endpoint)), "info", "--format", "{{.ID}}")
	if err != nil {
		t.Fatal(err)
	}
	binding := RuntimeBinding{strings.TrimSpace(string(endpoint)), strings.TrimSpace(string(daemon)), image, "linux/amd64", "/gobble/project", "/gobble/project/work"}
	if _, err = s.bindCreationRuntime(ctx, binding); err != nil {
		t.Fatal(err)
	}
	scope, err := s.creationAuthoring(p.ProjectID, d.DraftID, d.Generation)
	if err != nil {
		t.Fatal(err)
	}
	submitCreationFixture(t, s, scope, "req_creation")
	candidate := awaitCreation(t, s, scope, "req_creation")
	if candidate.State != "ready" {
		t.Fatalf("creation %+v", candidate)
	}
	adopted, err := s.adoptCreation(context.Background(), p.ProjectID, d.DraftID, adoptCreationInput{"req_adopt", "req_creation", candidate.Artifact.ArtifactID, d.Generation, "Synthetic analysis"})
	if err != nil {
		t.Fatal(err)
	}
	pipeline := adopted.Adoption.PipelineID
	if _, err = s.preparePipeline(p.ProjectID, pipeline, prepareInput{RequestID: "req_prepare", ArtifactID: candidate.Artifact.ArtifactID, EngineID: image}); err != nil {
		t.Fatal(err)
	}
	prep := awaitPreparation(t, s, p.ProjectID, pipeline, "req_prepare")
	if prep.State != "ready" {
		t.Fatalf("preparation %+v", prep)
	}
	if _, err = s.checkLaunch(p.ProjectID, pipeline, launchReviewInput{"req_review", "req_prepare"}); err != nil {
		t.Fatal(err)
	}
	var review launchReview
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		review, err = s.launchView(p.ProjectID, "req_review")
		if err != nil {
			t.Fatal(err)
		}
		if review.State != "checking" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if review.State != "ready" || !review.Fresh {
		t.Fatalf("launch check %+v", review)
	}
	if len(s.store.data.Runs) != 0 {
		t.Fatal("check started a Run")
	}
	if _, err = s.startLaunch(p.ProjectID, launchStartInput{"req_start", "req_review"}); err != nil {
		t.Fatal(err)
	}
	rec, _ := s.readLaunch(p.ProjectID, "req_review")
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = s.runtime.commandAt(ctx, binding.Endpoint, "rm", "--force", launchControllerName(rec))
	})
	restarted := false
	deadline = time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		review, err = s.launchView(p.ProjectID, "req_review")
		if err != nil {
			t.Fatal(err)
		}
		if op := review.Operation; op != nil {
			t.Logf("launch=%s run=%s status=%s issue=%s", op.State, op.RunRef, op.RunStatus, op.Issue)
			if op.State == "reconciling" && op.OwnerLease == "" {
				raw, _ := s.runtime.commandAt(context.Background(), binding.Endpoint, "inspect", "--format", "{{.State.Status}}", launchControllerName(rec))
				if strings.TrimSpace(string(raw)) == "exited" {
					logs, _ := s.runtime.commandAt(context.Background(), binding.Endpoint, "logs", launchControllerName(rec))
					t.Fatalf("controller exited without admission: %s", logs)
				}
			}
			if op.State == "rejected" {
				t.Fatal(op.Issue)
			}
			if op.RunRef != "" && !restarted {
				profile := filepath.Dir(s.store.dir)
				if err = s.Close(); err != nil {
					t.Fatal(err)
				}
				s, err = Open(context.Background(), profile)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = s.Close() })
				restarted = true
				t.Log("native service restarted; original controller remains independent")
				if stopMode {
					if _, err = s.stopLaunch(p.ProjectID, "req_review", launchStopInput{"req_stop", op.OwnerLease}); err != nil {
						t.Fatal(err)
					}
				}
			}
			if stopMode && op.StopState == "settled" {
				break
			}
			if op.RunRef != "" && (op.RunStatus == "succeeded" || op.RunStatus == "failed") {
				break
			}
		}
		_, _ = s.refreshLaunch(p.ProjectID, "req_review")
		time.Sleep(2 * time.Second)
	}
	op := review.Operation
	if op == nil || op.RunRef == "" || !restarted || (!stopMode && op.RunStatus != "succeeded") || (stopMode && op.StopState != "settled") {
		raw, _ := s.runtime.commandAt(context.Background(), binding.Endpoint, "logs", launchControllerName(rec))
		t.Fatalf("Run did not complete: %+v logs=%s", op, raw)
	}
	same, err := s.startLaunch(p.ProjectID, launchStartInput{"req_start", "req_review"})
	if err != nil || same.Operation.RunRef != op.RunRef {
		t.Fatal("retry changed Run", err)
	}
	if _, err = s.startLaunch(p.ProjectID, launchStartInput{"req_other", "req_review"}); err == nil {
		t.Fatal("new Start accepted for used review")
	}
	if len(s.store.data.Runs) != 1 {
		t.Fatal("duplicate Run")
	}
	snapshot, err := s.runtime.inspect(context.Background(), project, rec.Runtime, "monitor", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("completed snapshot=%s", snapshot)
	// Removal of a completed controller cannot lose the admitted Run.
	_, _ = s.runtime.commandAt(context.Background(), binding.Endpoint, "rm", launchControllerName(rec))
	_, _ = s.refreshLaunch(p.ProjectID, "req_review")
	s.workers.Wait()
	final, err := s.launchView(p.ProjectID, "req_review")
	if err != nil || final.Operation.State != "admitted" || final.Operation.RunRef != op.RunRef {
		t.Fatal("reconnect lost admission", err)
	}
	if os.Getenv("GOBBLE_REPORT_EVIDENCE_DIR") != "" && !stopMode {
		verifyLiveRunReport(t, s, p.ProjectID, op.RunRef)
	}

}
