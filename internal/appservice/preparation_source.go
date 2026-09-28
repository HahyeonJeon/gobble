package appservice

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/HahyeonJeon/gobble/internal/pipelinereview"
	"github.com/HahyeonJeon/gobble/internal/preparation"
)

// A retained birth binds selected research data across later source refinements.
// Imported pipelines have no equivalent data binding and cannot prepare in v1.
func (s *Service) preparationSourceLocked(projectID, pipelineID, artifactID string) (proposalBase, creationInput, preparation.Binding, error) {
	var input creationInput
	var binding preparation.Binding
	base, err := s.proposalBaseLocked(projectID, pipelineID, artifactID)
	if err != nil {
		return base, input, binding, err
	}
	s.store.mu.Lock()
	var born creationDraft
	for _, d := range s.store.data.Drafts {
		if d.ProjectID == projectID && d.Adoption != nil && d.Adoption.PipelineID == pipelineID {
			born = cloneCreationDraft(d)
			break
		}
	}
	s.store.mu.Unlock()
	if born.Input == nil {
		return base, input, binding, problem("unsupported", "Run preparation currently supports Pipelines created here with one selected single-end read file, including their refinements.")
	}
	p, _, err := s.store.pipelineProject(projectID, pipelineID)
	if err != nil {
		return base, input, binding, err
	}
	input, err = observeCreationInput(p, born.Input.ResourceID)
	if err != nil {
		return base, input, binding, err
	}
	logical, err := pipelinereview.CreationPath(input.RelativePath)
	if err != nil {
		return base, input, binding, err
	}
	if err = verifyCreationSource(base.Directory, creationManifest{Files: base.Manifest.Files}); err != nil {
		return base, input, binding, err
	}
	binding = preparation.Binding{SourceRevision: base.Artifact.SourceRevision, RuntimeID: requestDigest(base.Manifest.Runtime), InputIdentity: requestDigest(input), InputPath: logical, Cap: 1}
	return base, input, binding, nil
}
func retainPreparationSource(base proposalBase, b preparation.Binding, dir string) error {
	for _, f := range base.Manifest.Files {
		raw, err := readBoundedFile(filepath.Join(base.Directory, filepath.FromSlash(f.Path)), maxFileBytes)
		if err != nil || digest(raw) != f.SHA256 {
			return problem("stale_revision", "The retained Current source changed.")
		}
		target := filepath.Join(dir, "project", filepath.FromSlash(f.Path))
		if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err = atomicWrite(filepath.Dir(target), filepath.Base(target), raw); err != nil {
			return err
		}
		if err = os.Chmod(target, 0444); err != nil {
			return err
		}
	}
	path := filepath.Join(dir, "project", "gobble-preparation.json")
	if err := writeInspectionJSON(path, b); err != nil {
		return err
	}
	return os.Chmod(path, 0444)
}

type preparedResponse struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Digest        string              `json:"digest"`
	Payload       json.RawMessage     `json:"payload"`
	Binding       preparation.Binding `json:"binding"`
	Review        creationCheck       `json:"review"`
	Steps         []preparation.Step  `json:"steps"`
}

func (a runtimeAdapter) preparedPipeline(ctx context.Context, dir string, source inspectionSource, b preparation.Binding) (preparedResponse, error) {
	var out preparedResponse
	if err := a.verify(ctx, source.Runtime); err != nil {
		return out, err
	}
	if err := verifyPreparationSource(dir, source, b); err != nil {
		return out, err
	}
	raw, err := a.pipelineEvaluation(ctx, dir, source, "prepare")
	if err != nil {
		return out, err
	}
	if len(raw) > 3<<20 || decodeCreationJSON(raw, &out) != nil || out.SchemaVersion != 1 || out.Binding != b || validateCreationCheck(out.Review) != nil || len(out.Review.Gaps) != 0 || out.Review.InputPath != b.InputPath || len(out.Steps) != 2 {
		return out, problem("unsupported", "This pinned engine cannot prepare the complete Current design.")
	}
	// CLI formatting can indent the enclosing response. Retain canonical compact
	// engine payload bytes, with duplicate-key rejection before digest comparison.
	var compact bytes.Buffer
	if validateJSON(out.Payload) != nil || json.Compact(&compact, out.Payload) != nil || digest(compact.Bytes()) != out.Digest {
		return out, problem("internal", "The private prepared plan failed its integrity check.")
	}
	out.Payload = append([]byte{}, compact.Bytes()...)
	var payload struct {
		SchemaVersion int                 `json:"schemaVersion"`
		Binding       preparation.Binding `json:"binding"`
		Document      json.RawMessage     `json:"document"`
	}
	if decodeCreationJSON(out.Payload, &payload) != nil || payload.SchemaVersion != 1 || payload.Binding != b || len(payload.Document) == 0 {
		return out, problem("internal", "The private plan belongs to a different preparation.")
	}
	for i, step := range out.Steps {
		if step.ID != out.Review.Review.Definition.Steps[i].ID {
			return out, problem("internal", "The prepared steps differ from the review.")
		}
	}
	verified, e := a.readPrepared(ctx, filepath.Dir(dir), source.Runtime, out.Payload, out.Digest, b)
	if e != nil {
		return out, e
	}
	if !sameReviewJSON(verified.Flow, out.Review.Review.Flow) || requestDigest(verified.Steps) != requestDigest(out.Steps) {
		return out, problem("unsupported", "The displayed review does not match the sealed execution plan. Ask the Agent to check the design.")
	}
	out.Review.Review.Flow = verified.Flow
	out.Steps = verified.Steps
	if err = verifyPreparationSource(dir, source, b); err != nil {
		return out, err
	}
	if err = a.verify(ctx, source.Runtime); err != nil {
		return out, err
	}
	return out, nil
}

func verifyPreparationSource(dir string, source inspectionSource, b preparation.Binding) error {
	raw, err := json.Marshal(b)
	if err != nil {
		return err
	}
	files := append([]inspectionFile{}, source.Files...)
	files = append(files, inspectionFile{Path: "gobble-preparation.json", SHA256: digest(raw), Size: len(raw)})
	return verifyCreationSource(dir, creationManifest{Files: files})
}
