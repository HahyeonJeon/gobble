package appservice

import (
	"context"
	"encoding/json"
	"path/filepath"
	"time"

	"github.com/HahyeonJeon/gobble/internal/pipelinereview"
)

func (s *Service) evaluateCreation(ctx context.Context, dir string, m creationManifest) (creationCheck, error) {
	var v creationCheck
	if err := verifyCreationSource(dir, m); err != nil {
		return v, err
	}
	if err := s.runtime.verify(ctx, m.Runtime.Binding); err != nil {
		return v, err
	}
	raw, err := s.runtime.pipelineEvaluation(ctx, dir, inspectionSource{Package: m.Runtime.Scaffold.Package, Runtime: m.Runtime.Binding}, "creation-review")
	if err != nil {
		return v, err
	}
	if len(raw) > maxProposalRecordBytes || decodeCreationJSON(raw, &v) != nil || validateCreationCheck(v) != nil || v.InputPath != m.InputPath {
		return v, problem("unsupported", "The engine did not return a complete creation check.")
	}
	if err = verifyCreationSource(dir, m); err != nil {
		return v, err
	}
	if err = s.runtime.verify(ctx, m.Runtime.Binding); err != nil {
		return v, err
	}
	return v, nil
}
func (s *Service) checkCreation(ctx context.Context, m creationManifest, v creationCandidate) {
	defer s.workers.Done()
	dir := s.creationDirectory(v.DraftID, v.CandidateID)
	check, err := s.evaluateCreation(ctx, filepath.Join(dir, "project"), m)
	s.mu.Lock()
	defer s.mu.Unlock()
	defer func() {
		if f := s.creations[v.DraftID]; f != nil {
			f.cancel()
		}
		delete(s.creations, v.DraftID)
	}()
	// Serialize the final generation fence and publication with every draft writer.
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	if err == nil {
		d, e := s.store.mutableDraftLocked(v.ProjectID, v.DraftID, v.Generation)
		if e != nil {
			err = e
		} else if requestDigest(d) != requestDigest(m.Draft) {
			err = problem("stale_revision", "The draft changed.")
		}
		if err == nil {
			p, e := s.store.draftProjectLocked(v.ProjectID)
			if e != nil {
				err = e
			} else {
				observed, e := observeCreationInput(p, m.Draft.Input.ResourceID)
				if e != nil || observed != *m.Draft.Input {
					err = problem("stale_revision", "The selected file changed. Review the selected file again.")
				}
			}
		}
		if err == nil {
			current, e := s.readCreationRuntime()
			if e != nil {
				err = e
			} else if current.ID != m.Runtime.ID {
				err = problem("stale_revision", "The selected analysis engine changed.")
			}
		}
	}
	if ctx.Err() != nil {
		v.State = "cancelled"
		v.Issue = "The check was cancelled or interrupted."
	} else if err != nil {
		v.State = "failed"
		v.Issue = inspectionIssue(err)
	} else {
		revision := requestDigest(m)
		v.Artifact = &creationArtifact{requestDigest([]any{revision, check}), revision, *m.Draft.Input, m.Runtime.ID, time.Now().UTC().Format(time.RFC3339Nano), check}
		v.State = "ready"
		if len(check.Gaps) > 0 {
			v.State = "unsupported"
			v.Issue = "This proposed design includes unsupported behavior. Ask the Agent to revise it."
		}
	}
	if raw, e := json.Marshal(v); e != nil || len(raw) > maxProposalRecordBytes {
		v.State = "failed"
		v.Issue = "The creation result exceeds the retained limit."
		v.Artifact = nil
	}
	if err = writeInspectionJSON(filepath.Join(dir, "candidate.json"), v); err != nil {
		s.store.failed = true
	}
}

// Validate envelope and graph correspondence, not scientific semantics. Gobble
// alone qualifies commands; the service refuses malformed or mismatched evidence.
func validateCreationCheck(v creationCheck) error {
	bad := func() error { return problem("unsupported", "Invalid complete creation evidence.") }
	if v.SchemaVersion != 1 || v.Scope != pipelinereview.CreationScope || !pipelinereview.ValidCreationPath(v.InputPath) || v.Gaps == nil || len(v.Gaps) > 100 || v.Review.SchemaVersion != 1 || len(v.Review.Flow) > maxFlowBytes || pipelinereview.Validate(v.Review.Definition) != nil {
		return bad()
	}
	var flow struct {
		SchemaVersion int               `json:"schemaVersion"`
		Name          string            `json:"name"`
		Inputs        []json.RawMessage `json:"inputs"`
		Steps         []struct {
			ID string `json:"id"`
		} `json:"steps"`
		Connections []struct {
			FromTask string   `json:"fromTask"`
			FromPort string   `json:"fromPort"`
			ToTask   string   `json:"toTask"`
			ToPort   string   `json:"toPort"`
			Wait     []string `json:"wait"`
		} `json:"connections"`
	}
	if json.Unmarshal(v.Review.Flow, &flow) != nil || flow.SchemaVersion != 2 || flow.Name == "" || len(flow.Name) > 256 || flow.Inputs == nil || len(flow.Inputs) > 500 || flow.Steps == nil || len(flow.Steps) != len(v.Review.Definition.Steps) || flow.Connections == nil || len(flow.Connections) != len(v.Review.Definition.Edges) {
		return bad()
	}
	for i, step := range flow.Steps {
		if step.ID != v.Review.Definition.Steps[i].ID {
			return bad()
		}
	}
	for i, e := range flow.Connections {
		expected := pipelinereview.Edge{FromTask: e.FromTask, FromPort: e.FromPort, ToTask: e.ToTask, ToPort: e.ToPort, Wait: e.Wait}
		if requestDigest(expected) != requestDigest(v.Review.Definition.Edges[i]) {
			return bad()
		}
	}
	if len(v.Gaps) == 0 && (len(flow.Inputs) != 1 || len(flow.Steps) != 2 || len(flow.Connections) != 2) {
		return bad()
	}
	return nil
}
