package appservice

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HahyeonJeon/gobble/internal/pipelinereview"
)

func proposalFixture(t *testing.T) (*Service, projectRecord, PipelineDefinition, pipelineInspection) {
	t.Helper()
	s, project, pipeline := inspectionFixture(t)
	s.runtime.command = inspectionCommand(func(_ context.Context, args []string) ([]byte, error) {
		verb := ""
		mount := ""
		for i, a := range args {
			if a == "flow" || a == "review" {
				verb = a
			}
			if a == "--mount" {
				mount = args[i+1]
			}
		}
		if verb == "flow" {
			return emptyFlow, nil
		}
		if verb != "review" {
			t.Errorf("unexpected evaluator invocation: %v", args)
		}
		root := ""
		for _, part := range strings.Split(mount, ",") {
			if strings.HasPrefix(part, "src=") {
				root = strings.TrimPrefix(part, "src=")
			}
			if strings.HasPrefix(part, "source=") {
				root = strings.TrimPrefix(part, "source=")
			}
		}
		data, err := os.ReadFile(filepath.Join(root, "pipeline.go"))
		if err != nil {
			return nil, err
		}
		def := pipelinereview.Definition{SchemaVersion: 1, Context: pipelinereview.Digest("context"), Steps: []pipelinereview.Step{}, Edges: []pipelinereview.Edge{}}
		flow := emptyFlow
		if strings.Contains(string(data), "candidate") {
			// Portable service tests deliberately do not execute Go; engine/CLI tests
			// separately qualify the actual fingerprints and module recipes.
			def.Steps = []pipelinereview.Step{{ID: "new", Recipe: "unknown", Fingerprint: pipelinereview.Digest("new"), Residual: pipelinereview.Digest("new")}}
		}
		return json.Marshal(checkedReview{1, flow, def})
	})
	_, err := s.checkPipeline(project.ProjectID, pipeline.PipelineID, "req_current")
	if err != nil {
		t.Fatal(err)
	}
	current := awaitInspection(t, s, pipeline)
	if current.Artifact == nil {
		t.Fatal(current)
	}
	return s, project, pipeline, current
}
func awaitProposal(t *testing.T, s *Service, p PipelineDefinition, id string) pipelineProposal {
	t.Helper()
	end := time.Now().Add(5 * time.Second)
	for time.Now().Before(end) {
		s.mu.Lock()
		v, err := s.readProposalLocked(p.ProjectID, p.PipelineID, id)
		s.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		if v.State != "checking" {
			return v
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("proposal did not finish")
	return pipelineProposal{}
}
func TestProposalScopeHistoryAndFailedReview(t *testing.T) {
	s, project, p, current := proposalFixture(t)
	source, err := s.pipelineProposalSource(project.ProjectID, p.PipelineID, current.Artifact.ArtifactID)
	if err != nil || len(source.Files) != 1 || source.Files[0].Path != "pipeline.go" {
		t.Fatal(source, err)
	}
	for _, path := range []string{"../pipeline.go", "private.txt", "go.mod", ".gobble-runtime.json", "new.go"} {
		input := proposalInput{RequestID: newID("req"), BaseArtifactID: current.Artifact.ArtifactID, Summary: "Outside scope", Files: []proposalFile{{path, "changed"}}}
		_, err := s.proposePipeline(project.ProjectID, p.PipelineID, input)
		wantCode(t, err, "forbidden")
	}
	input := proposalInput{RequestID: "req_proposed", BaseArtifactID: current.Artifact.ArtifactID, Summary: "Candidate", Files: []proposalFile{{"pipeline.go", entrySource + "\n// candidate"}}}
	value, err := s.proposePipeline(project.ProjectID, p.PipelineID, input)
	if err != nil || value.State != "checking" {
		t.Fatal(value, err)
	}
	value = awaitProposal(t, s, p, input.RequestID)
	if value.State != "failed" {
		t.Fatal("unsupported recipe accepted", value)
	}
	same, err := s.proposePipeline(project.ProjectID, p.PipelineID, input)
	if err != nil || same.State != "failed" {
		t.Fatal("replayed failed proposal", same, err)
	}
	changed := input
	changed.Summary = "Different"
	_, err = s.proposePipeline(project.ProjectID, p.PipelineID, changed)
	wantCode(t, err, "request_conflict")
	original, _ := os.ReadFile(filepath.Join(project.Root, "pipeline.go"))
	if string(original) != entrySource {
		t.Fatal("mutated import")
	}
	observed, _ := s.pipelineInspection(project.ProjectID, p.PipelineID)
	if observed.Artifact.ArtifactID != current.Artifact.ArtifactID || len(s.store.data.Runs) > 0 {
		t.Fatal("changed current or started Run")
	}
	os.WriteFile(filepath.Join(project.Root, "pipeline.go"), []byte(entrySource+"\n// external"), 0600)
	_, err = s.pipelineProposalSource(project.ProjectID, p.PipelineID, current.Artifact.ArtifactID)
	wantCode(t, err, "stale_revision")
}
func installCheckedProposal(t *testing.T, s *Service, p PipelineDefinition, base pipelineInspection, id string, gaps []string) pipelineProposal {
	t.Helper()
	s.mu.Lock()
	snapshot, err := s.proposalBaseLocked(p.ProjectID, p.PipelineID, base.Artifact.ArtifactID)
	s.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	dir := s.proposalDirectory(p.PipelineID, id)
	manifest, err := retainProposal(snapshot, []proposalFile{{"pipeline.go", entrySource + "\n// retained proposal " + id}}, dir)
	if err != nil {
		t.Fatal(err)
	}
	artifact := pipelineArtifact{requestDigest([]any{manifest, json.RawMessage(emptyFlow)}), requestDigest(manifest), time.Now().UTC().Format(time.RFC3339Nano), emptyFlow}
	value := pipelineProposal{ProjectID: p.ProjectID, PipelineID: p.PipelineID, ProposalID: id, Digest: requestDigest(id), State: "ready", Base: *base.Artifact, Proposed: &artifact, Comparison: &pipelinereview.Comparison{SchemaVersion: 1, Changes: []pipelinereview.Change{{ID: "fixture-change", Kind: "setting"}}, Gaps: gaps}}
	if err := writeInspectionJSON(filepath.Join(dir, "review.json"), value); err != nil {
		t.Fatal(err)
	}
	return value
}
func TestAdoptionAtomicReceiptConflictAndRestart(t *testing.T) {
	s, project, p, base := proposalFixture(t)
	value := installCheckedProposal(t, s, p, base, "req_review", []string{})
	_, err := s.adoptPipeline(context.Background(), project.ProjectID, p.PipelineID, value.ProposalID, "req_adoption", value.Proposed.ArtifactID)
	if err != nil {
		t.Fatal(err)
	}
	// Exact repetition resolves its original receipt, including after a newer current.
	result, err := s.adoptionOutcome(project.ProjectID, p.PipelineID, "req_adoption")
	if err != nil || result.State != "adopted" || result.ArtifactID != value.Proposed.ArtifactID {
		t.Fatal(result, err)
	}
	_, err = s.adoptPipeline(context.Background(), project.ProjectID, p.PipelineID, value.ProposalID, "req_adoption", base.Artifact.ArtifactID)
	wantCode(t, err, "request_conflict")
	raw, err := os.ReadFile(filepath.Join(s.store.dir, "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	restored, _, err := decodeCatalog(raw)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Revisions[p.PipelineID].Artifact.ArtifactID != result.ArtifactID || restored.Requests["req_adoption"].ProposalID != value.ProposalID {
		t.Fatal("current and receipt were not saved together")
	}
	// Construct a restarted reader from saved catalog, without competing profile lock.
	reader := &Service{store: &store{dir: s.store.dir, data: restored}}
	read, err := reader.pipelineInspection(project.ProjectID, p.PipelineID)
	if err != nil || read.Artifact.ArtifactID != result.ArtifactID {
		t.Fatal(read, err)
	}
	source, err := reader.pipelineProposalSource(project.ProjectID, p.PipelineID, result.ArtifactID)
	if err != nil || !strings.Contains(source.Files[0].Content, "retained proposal") {
		t.Fatal(source, err)
	}
	original, _ := os.ReadFile(filepath.Join(project.Root, "pipeline.go"))
	if string(original) != entrySource {
		t.Fatal("adoption overwrote imported folder")
	}
	if len(restored.Runs) != 0 {
		t.Fatal("adoption registered a Run")
	}
}
func TestAdoptionRejectsGapsStaleBaseAndCorruptBytes(t *testing.T) {
	for _, mode := range []string{"gap", "external", "bytes"} {
		t.Run(mode, func(t *testing.T) {
			s, project, p, base := proposalFixture(t)
			gaps := []string{}
			if mode == "gap" {
				gaps = append(gaps, "unknown behavior")
			}
			value := installCheckedProposal(t, s, p, base, "req_review", gaps)
			if mode == "external" {
				os.WriteFile(filepath.Join(project.Root, "pipeline.go"), []byte("changed"), 0600)
			}
			if mode == "bytes" {
				path := filepath.Join(s.proposalDirectory(p.PipelineID, value.ProposalID), "project/pipeline.go")
				os.Chmod(path, 0600)
				os.WriteFile(path, []byte("corrupt"), 0600)
			}
			if _, err := s.adoptPipeline(context.Background(), project.ProjectID, p.PipelineID, value.ProposalID, "req_adoption", value.Proposed.ArtifactID); err == nil {
				t.Fatal("invalid adoption accepted")
			}
			if len(s.store.data.Revisions) != 0 || s.store.data.Requests["req_adoption"].ProposalID != "" {
				t.Fatal("failed adoption changed catalog")
			}
		})
	}
}

func TestCompetingAdoptionsPublishOnlyOneCurrent(t *testing.T) {
	s, project, p, base := proposalFixture(t)
	first := installCheckedProposal(t, s, p, base, "req_first", []string{})
	second := installCheckedProposal(t, s, p, base, "req_second", []string{})
	outcomes := make(chan error, 2)
	for _, v := range []pipelineProposal{first, second} {
		go func(v pipelineProposal) {
			_, err := s.adoptPipeline(context.Background(), project.ProjectID, p.PipelineID, v.ProposalID, "req_adopt_"+v.ProposalID, v.Proposed.ArtifactID)
			outcomes <- err
		}(v)
	}
	successes := 0
	for range 2 {
		if err := <-outcomes; err == nil {
			successes++
		} else {
			wantCode(t, err, "stale_revision")
		}
	}
	if successes != 1 {
		t.Fatalf("published %d competing proposals", successes)
	}
}

func TestAdoptionFailedCatalogWritePublishesNeitherPointerNorReceipt(t *testing.T) {
	s, project, p, base := proposalFixture(t)
	value := installCheckedProposal(t, s, p, base, "req_review", []string{})
	// Block the backup write without removing the saved current catalog.
	backup := filepath.Join(s.store.dir, "catalog.backup.json")
	if err := os.Remove(backup); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := os.Mkdir(backup, 0700); err != nil {
		t.Fatal(err)
	}
	_, err := s.adoptPipeline(context.Background(), project.ProjectID, p.PipelineID, value.ProposalID, "req_adoption", value.Proposed.ArtifactID)
	wantCode(t, err, "internal")
	raw, err := os.ReadFile(filepath.Join(s.store.dir, "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	persisted, _, err := decodeCatalog(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []catalog{s.store.data, persisted} {
		if len(c.Revisions) != 0 || c.Requests["req_adoption"].ProposalID != "" {
			t.Fatal("failed adoption published authority")
		}
	}
}

func TestAdoptionOutcomeRefusesUncertainInMemoryCatalog(t *testing.T) {
	s, project, p, base := proposalFixture(t)
	value := installCheckedProposal(t, s, p, base, "req_review", []string{})
	s.store.mu.Lock()
	old := s.store.copyLocked()
	s.store.mu.Unlock()
	if _, err := s.adoptPipeline(context.Background(), project.ProjectID, p.PipelineID, value.ProposalID, "req_adopt", value.Proposed.ArtifactID); err != nil {
		t.Fatal(err)
	}
	// Simulate rename succeeded but directory-sync acknowledgement was lost:
	// durable catalog has the receipt; the stopped writer still has its old map.
	s.store.mu.Lock()
	s.store.data = old
	s.store.failed = true
	s.store.mu.Unlock()
	if _, err := s.adoptionOutcome(project.ProjectID, p.PipelineID, "req_adopt"); err == nil {
		t.Fatal("uncertain writer reported an absent receipt")
	} else {
		wantCode(t, err, "runtime_unavailable")
	}
	raw, err := os.ReadFile(filepath.Join(s.store.dir, "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	restored, _, err := decodeCatalog(raw)
	if err != nil {
		t.Fatal(err)
	}
	reader := &Service{store: &store{dir: s.store.dir, data: restored}}
	result, err := reader.adoptionOutcome(project.ProjectID, p.PipelineID, "req_adopt")
	if err != nil || result.State != "adopted" || result.ArtifactID != value.Proposed.ArtifactID {
		t.Fatal(result, err)
	}
}

func TestInterruptedProposalHistoryCannotBeAdoptedOrOverwritten(t *testing.T) {
	s, project, p, base := proposalFixture(t)
	value := installCheckedProposal(t, s, p, base, "req_interrupted", []string{})
	value.State = "checking"
	value.Proposed = nil
	value.Comparison = nil
	file := filepath.Join(s.proposalDirectory(p.PipelineID, value.ProposalID), "review.json")
	if err := writeInspectionJSON(file, value); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	read, err := s.readProposalLocked(project.ProjectID, p.PipelineID, value.ProposalID)
	s.mu.Unlock()
	if err != nil || read.State != "failed" || !strings.Contains(read.Issue, "interrupted") {
		t.Fatal(read, err)
	}
	_, err = s.adoptPipeline(context.Background(), project.ProjectID, p.PipelineID, value.ProposalID, "req_adopt", base.Artifact.ArtifactID)
	wantCode(t, err, "unsupported")
	_, err = s.proposePipeline(project.ProjectID, p.PipelineID, proposalInput{RequestID: value.ProposalID, BaseArtifactID: base.Artifact.ArtifactID, Files: []proposalFile{{"pipeline.go", "changed"}}})
	wantCode(t, err, "request_conflict")
	current, _ := s.pipelineInspection(project.ProjectID, p.PipelineID)
	if current.Artifact.ArtifactID != base.Artifact.ArtifactID {
		t.Fatal("interrupted check changed current")
	}
}
