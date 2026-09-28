package appservice

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLiveCreationAdoptionAndRefinement(t *testing.T) {
	image := os.Getenv("GOBBLE_CREATION_IMAGE")
	if image == "" {
		t.Skip("requires pinned local creation engine")
	}
	s, p, file, d := draftFixture(t)
	d, err := s.store.updateDraft(p.ProjectID, d.DraftID, updateDraftInput{"req_input", 1, "Trim reads and inspect quality", file, "single-end"})
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
	scope.Files[0].Content = strings.Replace(scope.Files[0].Content, "Quality: 0, Length: 0", "Quality: 25, Length: 40", 1)
	submitCreationFixture(t, s, scope, "req_creation")
	candidate := awaitCreation(t, s, scope, "req_creation")
	if candidate.State != "ready" {
		t.Fatalf("creation: %+v", candidate)
	}
	intent := adoptCreationInput{"req_first", "req_creation", candidate.Artifact.ArtifactID, d.Generation, "Read quality"}
	adopted, err := s.adoptCreation(context.Background(), p.ProjectID, d.DraftID, intent)
	if err != nil {
		t.Fatal(err)
	}
	id := adopted.Adoption.PipelineID
	if os.Getenv("GOBBLE_PREPARATION_LIVE") == "1" {
		checkLivePreparation(t, s, p.ProjectID, id, intent.ArtifactID, "req_prepare_birth")
	}
	source, err := s.pipelineProposalSource(p.ProjectID, id, intent.ArtifactID)
	if err != nil {
		t.Fatal(err)
	}
	source.Files[0].Content = strings.Replace(source.Files[0].Content, "Quality: 25", "Quality: 30", 1)
	if _, err = s.proposePipeline(p.ProjectID, id, proposalInput{nil, "req_refine", intent.ArtifactID, "Raise quality threshold", source.Files}); err != nil {
		t.Fatal(err)
	}
	var proposed pipelineProposal
	deadline := time.Now().Add(2*inspectionTimeout + 10*time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		proposed, err = s.readProposalLocked(p.ProjectID, id, "req_refine")
		s.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		if proposed.State != "checking" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if proposed.State != "ready" || proposed.Comparison == nil || len(proposed.Comparison.Gaps) != 0 || len(proposed.Comparison.Changes) != 1 {
		t.Fatalf("refinement: %+v", proposed)
	}
	if _, err = s.adoptPipeline(context.Background(), p.ProjectID, id, "req_refine", "req_next_current", proposed.Proposed.ArtifactID); err != nil {
		t.Fatal(err)
	}
	birth, err := s.creationAdoptionOutcome(p.ProjectID, d.DraftID, "req_first")
	if err != nil || birth.Adoption.ArtifactID != intent.ArtifactID {
		t.Fatal("birth changed", err)
	}
	current, err := s.pipelineInspection(p.ProjectID, id)
	if err != nil || current.Artifact.ArtifactID != proposed.Proposed.ArtifactID {
		t.Fatal("refinement not Current", err)
	}
	if len(s.store.data.Pipelines) != 1 || len(s.store.data.Runs) != 0 {
		t.Fatal("unexpected Pipeline or Run")
	}
	if os.Getenv("GOBBLE_PREPARATION_LIVE") == "1" {
		checkLivePreparation(t, s, p.ProjectID, id, current.Artifact.ArtifactID, "req_prepare_refined")
	}
	t.Logf("image=%s birth=%s refined=%s zero-runs", image, intent.ArtifactID, current.Artifact.ArtifactID)
}

func checkLivePreparation(t *testing.T, s *Service, project, pipeline, artifact, request string) {
	t.Helper()
	if _, err := s.preparePipeline(project, pipeline, prepareInput{RequestID: request, ArtifactID: artifact, EngineID: os.Getenv("GOBBLE_PREPARATION_TARGET_IMAGE")}); err != nil {
		t.Fatal(err)
	}
	ready := awaitPreparation(t, s, project, pipeline, request)
	if ready.State != "ready" || !ready.Fresh || ready.Prepared == nil || len(ready.Prepared.Steps) != 2 {
		t.Fatalf("live preparation: %+v", ready)
	}
	if _, err := s.preparePipeline(project, pipeline, prepareInput{RequestID: request, ArtifactID: artifact, EngineID: os.Getenv("GOBBLE_PREPARATION_TARGET_IMAGE")}); err != nil {
		t.Fatal("live replay", err)
	}
	if len(s.store.data.Runs) != 0 {
		t.Fatal("preparation launched a Run")
	}
	t.Logf("prepared=%s engine=%s current=%s no-runs", ready.Prepared.Digest, ready.Prepared.Binding.RuntimeID, artifact)
}
