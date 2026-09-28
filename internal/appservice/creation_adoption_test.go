package appservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func firstAdoptionFixture(t *testing.T) (*Service, Project, creationScope, adoptCreationInput) {
	t.Helper()
	s, p, scope := creationFixture(t, func(context.Context, []string) ([]byte, error) { return mockCreationCheck(), nil })
	submitCreationFixture(t, s, scope, "req_candidate")
	v := awaitCreation(t, s, scope, "req_candidate")
	if v.State != "ready" {
		t.Fatalf("candidate: %+v", v)
	}
	return s, p, scope, adoptCreationInput{"req_birth", "req_candidate", v.Artifact.ArtifactID, scope.Draft.Generation, "Read quality analysis"}
}
func TestFirstAdoptionConcurrentRetryRestartAndRefinementSource(t *testing.T) {
	s, p, scope, input := firstAdoptionFixture(t)
	sibling := PipelineDefinition{p.ProjectID, "pip_sibling", "Existing study", PipelineOrigin{Kind: "managed"}}
	siblingArtifact := pipelineArtifact{ArtifactID: digest([]byte("sibling artifact")), SourceRevision: digest([]byte("sibling source")), CheckedAt: "2026-09-12T00:00:00Z", Flow: json.RawMessage(emptyFlow)}
	s.store.mu.Lock()
	nextCatalog := s.store.copyLocked()
	nextCatalog.Pipelines = append(nextCatalog.Pipelines, sibling)
	nextCatalog.Revisions[sibling.PipelineID] = managedRevision{PipelineID: sibling.PipelineID, ProposalID: "req_sibling", Artifact: siblingArtifact}
	saveErr := s.store.commitLocked(nextCatalog)
	s.store.mu.Unlock()
	if saveErr != nil {
		t.Fatal(saveErr)
	}
	other := testProject(t, s, t.TempDir())
	_, err := s.adoptCreation(context.Background(), other.ProjectID, scope.Draft.DraftID, input)
	wantCode(t, err, "not_found")
	var wg sync.WaitGroup
	results := make(chan creationAdoptionResult, 8)
	for i := range 8 {
		wg.Go(func() {
			next := input
			next.RequestID = fmt.Sprintf("req_birth_%d", i)
			v, e := s.adoptCreation(context.Background(), p.ProjectID, scope.Draft.DraftID, next)
			if e != nil {
				t.Error(e)
				return
			}
			results <- v
		})
	}
	wg.Wait()
	close(results)
	var pipelineID string
	for v := range results {
		if pipelineID != "" && pipelineID != v.Adoption.PipelineID {
			t.Fatal("duplicate Pipeline")
		}
		pipelineID = v.Adoption.PipelineID
	}
	if pipelineID == "" || len(s.store.data.Pipelines) != 2 || len(s.store.data.Runs) != 0 {
		t.Fatal("publication was not singular, or started Run")
	}
	if s.store.data.Revisions[sibling.PipelineID].Artifact.ArtifactID != siblingArtifact.ArtifactID {
		t.Fatal("sibling Current changed")
	}
	result, err := s.adoptCreation(context.Background(), p.ProjectID, scope.Draft.DraftID, input)
	if err != nil {
		t.Fatal(err)
	}
	input.Name = "different"
	_, err = s.adoptCreation(context.Background(), p.ProjectID, scope.Draft.DraftID, input)
	wantCode(t, err, "request_conflict")
	input.Name = "Read quality analysis"
	inspection, err := s.pipelineInspection(p.ProjectID, pipelineID)
	if err != nil || inspection.Artifact.ArtifactID != input.ArtifactID {
		t.Fatal("Current", err)
	}
	source, err := s.pipelineProposalSource(p.ProjectID, pipelineID, input.ArtifactID)
	if err != nil || len(source.Files) != 1 {
		t.Fatalf("refinement source: %+v %v", source, err)
	}
	_, err = s.store.updateDraft(p.ProjectID, scope.Draft.DraftID, updateDraftInput{"req_after", scope.Draft.Generation + 1, "changed", "", ""})
	wantCode(t, err, "stale_revision")
	s.store.mu.Lock()
	copy := s.store.copyLocked()
	copy.Drafts[scope.Draft.DraftID].Adoption.Name = "mutated"
	copy.Revisions[pipelineID].Creation.CandidateID = "req_changed"
	s.store.mu.Unlock()
	if s.store.data.Drafts[scope.Draft.DraftID].Adoption.Name != input.Name || s.store.data.Revisions[pipelineID].Creation.CandidateID != input.CandidateID {
		t.Fatal("aliased copy")
	}
	// Birth remains a historical record when Current later points to a refinement.
	s.store.mu.Lock()
	next := s.store.copyLocked()
	r := next.Revisions[pipelineID]
	r.Creation = nil
	r.ProposalID = "req_later"
	next.Revisions[pipelineID] = r
	err = s.store.commitLocked(next)
	s.store.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	profile := filepath.Dir(s.store.dir)
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(context.Background(), profile)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	// Saved results must resolve even with an offline engine and advanced Current.
	got, err := reopened.adoptCreation(context.Background(), p.ProjectID, scope.Draft.DraftID, input)
	if err != nil || *got.Adoption != *result.Adoption {
		t.Fatal("lost outcome on restart", err)
	}
	history, err := reopened.creationCandidate(p.ProjectID, scope.Draft.DraftID, input.CandidateID)
	if err != nil || history.Artifact.ArtifactID != input.ArtifactID {
		t.Fatal("lost birth history", err)
	}
}
func TestFirstAdoptionRefusesDriftAndUnreadyEvidence(t *testing.T) {
	for _, action := range []string{"generation", "data", "runtime", "source", "discard", "cancel", "artifact", "gap", "manifest", "checking", "failed", "unsupported", "cancelled"} {
		t.Run(action, func(t *testing.T) {
			s, p, scope, input := firstAdoptionFixture(t)
			ctx := context.Background()
			var err error
			switch action {
			case "generation":
				input.ExpectedGeneration--
			case "data":
				project, _ := s.store.project(p.ProjectID)
				err = os.WriteFile(filepath.Join(project.Root, "reads.fastq.gz"), []byte("changed"), 0600)
			case "runtime":
				err = os.Remove(filepath.Join(s.store.dir, "creation-runtime.json"))
			case "source":
				err = os.WriteFile(filepath.Join(s.creationDirectory(scope.Draft.DraftID, input.CandidateID), "project", "extra.go"), []byte("package extra"), 0600)
			case "discard":
				_, err = s.store.discardDraft(p.ProjectID, scope.Draft.DraftID, discardDraftInput{"req_discard", scope.Draft.Generation})
			case "cancel":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "artifact":
				input.ArtifactID = digest([]byte("wrong"))
			case "checking", "failed", "unsupported", "cancelled":
				v, _ := s.creationCandidate(p.ProjectID, scope.Draft.DraftID, input.CandidateID)
				v.State = action
				err = writeInspectionJSON(filepath.Join(s.creationDirectory(scope.Draft.DraftID, input.CandidateID), "candidate.json"), v)
			case "gap":
				v, _ := s.creationCandidate(p.ProjectID, scope.Draft.DraftID, input.CandidateID)
				v.Artifact.Check.Gaps = []string{"unsupported-setting"}
				err = writeInspectionJSON(filepath.Join(s.creationDirectory(scope.Draft.DraftID, input.CandidateID), "candidate.json"), v)
			case "manifest":
				err = os.WriteFile(filepath.Join(s.creationDirectory(scope.Draft.DraftID, input.CandidateID), "manifest.json"), []byte("{}"), 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.adoptCreation(ctx, p.ProjectID, scope.Draft.DraftID, input); err == nil {
				t.Fatalf("accepted %s", action)
			}
			if len(s.store.data.Pipelines) != 0 || len(s.store.data.Runs) != 0 || s.store.data.Drafts[scope.Draft.DraftID].Adoption != nil {
				t.Fatal("partial publication")
			}
		})
	}
}
func TestFirstAdoptionUncertainWriteReconcilesFromDisk(t *testing.T) {
	for _, published := range []bool{false, true} {
		t.Run(fmt.Sprint(published), func(t *testing.T) {
			s, p, scope, input := firstAdoptionFixture(t)
			s.store.writeFile = func(dir, name string, data []byte) error {
				if name != "catalog.json" {
					return atomicWrite(dir, name, data)
				}
				if published {
					if err := atomicWrite(dir, name, data); err != nil {
						return err
					}
				}
				return errors.New("injected persistence interruption")
			}
			_, err := s.adoptCreation(context.Background(), p.ProjectID, scope.Draft.DraftID, input)
			wantCode(t, err, "internal")
			if _, err = s.creationAdoptionOutcome(p.ProjectID, scope.Draft.DraftID, input.RequestID); err == nil {
				t.Fatal("poisoned memory claimed an outcome")
			}
			profile := filepath.Dir(s.store.dir)
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			reopened, err := Open(context.Background(), profile)
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			got, err := reopened.creationAdoptionOutcome(p.ProjectID, scope.Draft.DraftID, input.RequestID)
			if err != nil {
				t.Fatal(err)
			}
			want := "not-recorded"
			count := 0
			if published {
				want = "adopted"
				count = 1
			}
			if got.State != want || len(reopened.store.data.Pipelines) != count || len(reopened.store.data.Revisions) != count || len(reopened.store.data.Runs) != 0 {
				t.Fatalf("non-atomic recovery: %+v", got)
			}
		})
	}
}
