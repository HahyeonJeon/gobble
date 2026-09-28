package appservice

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func legacyV3(c catalog) catalogV3 {
	old := catalogV3{SchemaVersion: 3, Projects: c.Projects, Runs: c.Runs, Pipelines: []catalogPipelineV3{}, Requests: map[string]catalogReceiptV3{}, Revisions: map[string]catalogRevisionV4{}}
	for _, p := range c.Pipelines {
		old.Pipelines = append(old.Pipelines, catalogPipelineV3{p.ProjectID, p.PipelineID, p.Name, p.Origin.PackageResourceID, p.Origin.SourceResourceID})
	}
	for id, r := range c.Requests {
		old.Requests[id] = catalogReceiptV3{r.Digest, r.ProjectID, r.RunRef, r.PipelineID, r.ProposalID}
	}
	for _, r := range c.Revisions {
		for _, p := range c.Pipelines {
			if p.PipelineID == r.PipelineID {
				old.Revisions[p.ProjectID] = catalogRevisionV4{r.PipelineID, r.ProposalID, r.Artifact}
			}
		}
	}
	return old
}
func TestCatalogV3MigrationKeepsCurrentReceiptsAndOriginalArchive(t *testing.T) {
	s, project, p, base := proposalFixture(t)
	proposal := installCheckedProposal(t, s, p, base, "req_review", []string{})
	adopted, err := s.adoptPipeline(context.Background(), project.ProjectID, p.PipelineID, proposal.ProposalID, "req_adopt", proposal.Proposed.ArtifactID)
	if err != nil {
		t.Fatal(err)
	}
	old := legacyV3(s.store.data)
	raw, err := json.MarshalIndent(old, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	migrated, original, err := decodeCatalog(raw)
	if err != nil || !bytes.Equal(original, raw) || migrated.Revisions[p.PipelineID].Artifact.ArtifactID != adopted.ArtifactID || len(migrated.Drafts) != 0 {
		t.Fatalf("migration: %+v %v", migrated, err)
	}
	reader := &Service{store: &store{dir: s.store.dir, data: migrated}}
	result, err := reader.adoptionOutcome(project.ProjectID, p.PipelineID, "req_adopt")
	if err != nil || result != adopted {
		t.Fatalf("historical outcome changed: %+v %v", result, err)
	}
	inspected, err := reader.pipelineInspection(project.ProjectID, p.PipelineID)
	if err != nil || inspected.Artifact.ArtifactID != adopted.ArtifactID {
		t.Fatalf("current after migration: %+v %v", inspected, err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.json")
	writeTestFile(t, path, raw)
	store, err := openStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	if !bytes.Equal(before, raw) {
		t.Fatal("read rewrote old catalog")
	}
	created, err := store.createDraft(project.ProjectID, createDraftInput{"req_newdraft", "new"})
	if err != nil {
		t.Fatal(err)
	}
	archived, err := os.ReadFile(filepath.Join(dir, "catalog.v3.backup.json"))
	if err != nil || !bytes.Equal(archived, raw) {
		t.Fatal("v3 archive was not preserved", err)
	}
	if err := store.close(); err != nil {
		t.Fatal(err)
	}
	restarted, err := openStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.close()
	got, err := restarted.creationDraft(project.ProjectID, created.DraftID)
	if err != nil || got.Generation != 1 || restarted.data.Revisions[p.PipelineID].Artifact.ArtifactID != adopted.ArtifactID {
		t.Fatal("restart lost Current or draft", err)
	}
}
func TestCatalogV3RejectsWrongOwnerNewFieldsAndArchiveConflict(t *testing.T) {
	s, project, p, base := proposalFixture(t)
	value := installCheckedProposal(t, s, p, base, "req_old", []string{})
	_, err := s.adoptPipeline(context.Background(), project.ProjectID, p.PipelineID, value.ProposalID, "req_oldadopt", value.Proposed.ArtifactID)
	if err != nil {
		t.Fatal(err)
	}
	old := legacyV3(s.store.data)
	revision := old.Revisions[project.ProjectID]
	delete(old.Revisions, project.ProjectID)
	old.Revisions["prj_other"] = revision
	raw, _ := json.Marshal(old)
	if _, _, err := decodeCatalog(raw); err == nil {
		t.Fatal("legacy current reassigned across Project")
	}
	old = legacyV3(s.store.data)
	raw, _ = json.Marshal(old)
	for _, body := range [][]byte{
		bytes.Replace(raw, []byte(`"schemaVersion":3`), []byte(`"schemaVersion":3,"drafts":{}`), 1),
		bytes.Replace(raw, []byte(`"packageResourceId"`), []byte(`"origin":{"kind":"managed"},"packageResourceId"`), 1),
		bytes.Replace(raw, []byte(`"digest"`), []byte(`"draftId":"drf_smuggled","digest"`), 1),
	} {
		if _, _, err := decodeCatalog(body); err == nil {
			t.Fatal("new fields accepted inside frozen v3")
		}
	}
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "catalog.json"), raw)
	writeTestFile(t, filepath.Join(dir, "catalog.v3.backup.json"), []byte("preserve other archive"))
	st, err := openStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.close()
	_, err = st.createDraft(project.ProjectID, createDraftInput{"req_archive", ""})
	wantCode(t, err, "internal")
	after, _ := os.ReadFile(filepath.Join(dir, "catalog.json"))
	if !bytes.Equal(after, raw) || len(st.data.Drafts) != 0 {
		t.Fatal("archive failure published migration")
	}
}
func TestPipelineCurrentOwnershipIsIndependentAndImportedGuardRemains(t *testing.T) {
	s, project, p, base := proposalFixture(t)
	proposal := installCheckedProposal(t, s, p, base, "req_a", []string{})
	first, err := s.adoptPipeline(context.Background(), project.ProjectID, p.PipelineID, proposal.ProposalID, "req_a_adopt", proposal.Proposed.ArtifactID)
	if err != nil {
		t.Fatal(err)
	}
	// A managed-origin fixture exercises catalog ownership only, not creation qualification.
	managed := PipelineDefinition{project.ProjectID, "pip_managed", "Other analysis", PipelineOrigin{Kind: "managed"}}
	artifact := pipelineArtifact{ArtifactID: digest([]byte("other artifact")), SourceRevision: digest([]byte("other source")), CheckedAt: base.Artifact.CheckedAt, Flow: json.RawMessage(emptyFlow)}
	s.store.mu.Lock()
	next := s.store.copyLocked()
	next.Pipelines = append(next.Pipelines, managed)
	next.Revisions[managed.PipelineID] = managedRevision{PipelineID: managed.PipelineID, ProposalID: "req_managed", Artifact: artifact}
	err = s.store.commitLocked(next)
	s.store.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.pipelineInspection(project.ProjectID, managed.PipelineID)
	if err != nil || got.Artifact.ArtifactID != artifact.ArtifactID {
		t.Fatalf("managed Current: %+v %v", got, err)
	}
	kept, err := s.pipelineInspection(project.ProjectID, p.PipelineID)
	if err != nil || kept.Artifact.ArtifactID != first.ArtifactID {
		t.Fatal("other Current overwritten", err)
	}
	if _, err := s.pipelineProposalSource(project.ProjectID, p.PipelineID, first.ArtifactID); err != nil {
		t.Fatal("retained independent source blocked by sibling", err)
	}
	_, err = s.pipelineInspection("prj_other", managed.PipelineID)
	wantCode(t, err, "not_found")
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	copied := s.store.copyLocked()
	copied.Revisions[p.PipelineID].Artifact.Flow[0] = 'x'
	if !json.Valid(s.store.data.Revisions[p.PipelineID].Artifact.Flow) {
		t.Fatal("copy aliases retained artifact bytes")
	}
	guarded := s.store.copyLocked()
	delete(guarded.Revisions, p.PipelineID)
	if !canRefinePipeline(guarded, project.ProjectID, p.PipelineID) {
		t.Fatal("isolated managed sibling counted as shared imported source")
	}
	sibling := p
	sibling.PipelineID = "pip_imported"
	guarded.Pipelines = append(guarded.Pipelines, sibling)
	if canRefinePipeline(guarded, project.ProjectID, p.PipelineID) {
		t.Fatal("shared imported source restriction lost")
	}
	invalid := s.store.copyLocked()
	delete(invalid.Revisions, managed.PipelineID)
	if validateCatalog(invalid) == nil {
		t.Fatal("managed Pipeline without Current accepted")
	}
}
func TestMissingPrimaryDoesNotIgnoreV3Archive(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "catalog.v3.backup.json"), []byte("retained"))
	if store, err := openStore(dir); err == nil {
		store.close()
		t.Fatal("missing primary reset despite v3 archive")
	}
}

func TestPipelineOriginRejectsMixedAndUnknownJSON(t *testing.T) {
	for _, raw := range []string{`null`, `{"kind":"managed","sourceResourceId":""}`, `{"kind":"managed","path":"/private"}`, `{"kind":"imported","packageResourceId":"res_pkg"}`} {
		var origin PipelineOrigin
		if json.Unmarshal([]byte(raw), &origin) == nil {
			t.Fatalf("invalid origin accepted: %s", raw)
		}
	}
	for _, raw := range []string{`{"kind":"managed"}`, `{"kind":"imported","packageResourceId":"res_pkg","sourceResourceId":"res_source"}`} {
		var origin PipelineOrigin
		if err := json.Unmarshal([]byte(raw), &origin); err != nil {
			t.Fatalf("valid origin rejected: %s %v", raw, err)
		}
	}
}
