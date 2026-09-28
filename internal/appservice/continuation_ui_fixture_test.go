package appservice

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Explicit opt-in exporter for the Electron test. Only writes to its new, test-owned directory.
// It uses native serializers/checksums; no production seeding or bypass endpoint exists.
func TestExportContinuationUIFixture(t *testing.T) {
	base := os.Getenv("GOBBLE_CONTINUATION_UI_FIXTURE")
	if base == "" {
		t.Skip("Electron fixture export only")
	}
	if !filepath.IsAbs(base) {
		t.Fatal("absolute fixture path required")
	}
	s, p, original, f := continuationFixture(t)
	project, err := s.store.project(p.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	var flow json.RawMessage
	flow, err = os.ReadFile(filepath.Join(base, "flow.json"))
	if err != nil {
		t.Fatal(err)
	}
	original.Value.Preparation.Prepared.Flow = flow
	root := filepath.Join(base, "Continuation study")
	profile := filepath.Join(base, "profile")
	if err = os.CopyFS(root, os.DirFS(project.Root)); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	s.store.mu.Lock()
	next := s.store.copyLocked()
	current := next.Revisions[original.Value.PipelineID]
	current.Artifact.Flow = flow
	original.Value.Preparation.ArtifactID = digest([]byte("earlier checked design"))
	next.Revisions[original.Value.PipelineID] = current
	for i := range next.Projects {
		if next.Projects[i].ProjectID == p.ProjectID {
			next.Projects[i].Root = root
			next.Projects[i].RootIdentity = rootIdentity(info)
			next.Projects[i].Name = "Continuation study"
			next.Projects[i].Resources[resourceID(p.ProjectID, original.Value.OutputPath)] = original.Value.OutputPath
		}
	}
	identity := json.RawMessage(`{"identity_mode":"local-pin","gobble_module":"fixture"}`)
	next.Runs = append(next.Runs, runRecord{RunRegistration{p.ProjectID, "run_saved", "Analysis 1", resourceID(p.ProjectID, original.Value.OutputPath), "fixture-run"}, original.Runtime, identity})
	err = s.store.commitLocked(next)
	s.store.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	info, err = os.Stat(filepath.Join(root, original.Value.OutputPath))
	if err != nil {
		t.Fatal(err)
	}
	original.TargetIdentity = rootIdentity(info)
	if err = s.saveLaunch(original); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	if err = os.CopyFS(profile, os.DirFS(filepath.Dir(s.store.dir))); err != nil {
		t.Fatal(err)
	}
	// Snapshot names are the same immutable engine checkpoint used by the review.
	data := map[string]any{"profile": profile, "root": root, "projectId": p.ProjectID, "original": original, "review": f.review, "identity": identity}
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(base, "fixture.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
}
