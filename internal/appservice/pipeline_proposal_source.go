package appservice

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/HahyeonJeon/gobble/internal/pipelinereview"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Authoring never targets the research folder. Only existing declared Go files
// in this Pipeline package enter the per-turn source scope; setup, dependencies,
// runtime bindings, samples and sibling packages remain immutable.
type proposalFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}
type proposalSource struct {
	Guidance       string         `json:"guidance"`
	ProjectID      string         `json:"projectId"`
	PipelineID     string         `json:"pipelineId"`
	BaseArtifactID string         `json:"baseArtifactId"`
	Files          []proposalFile `json:"files"`
}
type proposalBase struct {
	Artifact  pipelineArtifact
	Manifest  inspectionSource
	Directory string
	Managed   bool
}

func (s *Service) proposalDirectory(pipelineID, proposalID string) string {
	return filepath.Join(s.store.dir, "pipeline-proposals", pipelineID, proposalID)
}

// Called with Service.mu held. Catalog remains the sole owner of current.
func (s *Service) proposalBaseLocked(projectID, pipelineID, artifactID string) (proposalBase, error) {
	project, pipeline, err := s.store.pipelineProject(projectID, pipelineID)
	if err != nil {
		return proposalBase{}, err
	}
	s.store.mu.Lock()
	allowed := canRefinePipeline(s.store.data, projectID, pipelineID)
	managed, hasManaged := s.store.data.Revisions[pipelineID]
	s.store.mu.Unlock()
	if !allowed {
		return proposalBase{}, problem("unsupported", "Imported source proposals require one imported Pipeline in this Project. Shared source changes need a wider review.")
	}
	value, err := s.readInspectionLocked(projectID, pipelineID)
	if err != nil {
		return proposalBase{}, err
	}
	if value.State != "ready" || value.Artifact == nil || value.Artifact.ArtifactID != artifactID {
		return proposalBase{}, problem("stale_revision", "The current flow changed or is not ready. Open the current flow and request changes again.")
	}
	if hasManaged && managed.Creation != nil {
		return s.creationProposalBaseLocked(projectID, pipelineID, managed)
	}
	dir := filepath.Join(s.inspectionDirectory(pipelineID), "candidates", value.JobID)
	if hasManaged {
		if managed.PipelineID != pipelineID {
			return proposalBase{}, problem("stale_revision", "The managed Pipeline changed.")
		}
		dir = s.proposalDirectory(pipelineID, managed.ProposalID)
	}
	var manifest inspectionSource
	raw, err := readBoundedFile(filepath.Join(dir, "manifest.json"), 256<<10)
	if err != nil || validateJSON(raw) != nil || json.Unmarshal(raw, &manifest) != nil || requestDigest(manifest) != value.Artifact.SourceRevision {
		return proposalBase{}, problem("internal", "The retained source record is unavailable. Preserve the analysis profile.")
	}
	if !hasManaged {
		current, err := retainInspection(project, pipeline, manifest.Runtime, "")
		if err != nil || requestDigest(current) != requestDigest(manifest) {
			return proposalBase{}, problem("stale_revision", "Imported source changed. Check the flow before requesting a proposal.")
		}
	}
	return proposalBase{*value.Artifact, manifest, filepath.Join(dir, "project"), hasManaged}, nil
}

func scopedSource(base proposalBase) ([]proposalFile, error) {
	out := []proposalFile{}
	total := 0
	for _, f := range base.Manifest.Files {
		if filepath.ToSlash(filepath.Dir(f.Path)) != base.Manifest.Package || !strings.HasSuffix(f.Path, ".go") || strings.HasSuffix(f.Path, "_test.go") {
			continue
		}
		raw, err := readBoundedFile(filepath.Join(base.Directory, filepath.FromSlash(f.Path)), 32<<10)
		if err != nil || digest(raw) != f.SHA256 || !utf8.Valid(raw) {
			return nil, problem("unsupported", "This Pipeline source cannot be shared within the proposal limit.")
		}
		total += len(raw)
		if len(out) >= 16 || total > 48<<10 {
			return nil, problem("unsupported", "This Pipeline exceeds the bounded authoring scope.")
		}
		out = append(out, proposalFile{f.Path, string(raw)})
	}
	if len(out) == 0 {
		return nil, problem("unsupported", "No editable Pipeline source is available.")
	}
	return out, nil
}
func (s *Service) pipelineProposalSource(projectID, pipelineID, artifactID string) (proposalSource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	base, err := s.proposalBaseLocked(projectID, pipelineID, artifactID)
	if err != nil {
		return proposalSource{}, err
	}
	files, err := scopedSource(base)
	return proposalSource{Guidance: pipelinereview.AuthoringGuide, ProjectID: projectID, PipelineID: pipelineID, BaseArtifactID: artifactID, Files: files}, err
}

// Durably publish every evaluator-visible byte before a comparison or current
// pointer can reference it. Candidate files have no paths chosen by a renderer.
func retainProposal(base proposalBase, files []proposalFile, dir string) (inspectionSource, error) {
	scoped, err := scopedSource(base)
	if err != nil {
		return inspectionSource{}, err
	}
	allowed := map[string]bool{}
	for _, f := range scoped {
		allowed[f.Path] = true
	}
	replacements := map[string][]byte{}
	total := 0
	if len(files) == 0 || len(files) > 16 {
		return inspectionSource{}, problem("invalid_request", "Provide 1 to 16 complete Pipeline source files.")
	}
	for _, f := range files {
		total += len(f.Content)
		if !allowed[f.Path] || replacements[f.Path] != nil || len(f.Content) == 0 || len(f.Content) > 32<<10 || total > 48<<10 || !utf8.ValidString(f.Content) || strings.ContainsRune(f.Content, 0) {
			return inspectionSource{}, problem("forbidden", "The proposal exceeds its existing Pipeline source scope.")
		}
		replacements[f.Path] = []byte(f.Content)
	}
	manifest := base.Manifest
	manifest.Files = append([]inspectionFile{}, base.Manifest.Files...)
	changed := false
	for i, f := range manifest.Files {
		data, err := readBoundedFile(filepath.Join(base.Directory, filepath.FromSlash(f.Path)), maxFileBytes)
		if err != nil || digest(data) != f.SHA256 {
			return inspectionSource{}, problem("internal", "Retained analysis input failed its integrity check.")
		}
		if candidate, ok := replacements[f.Path]; ok {
			changed = changed || !bytes.Equal(data, candidate)
			data = candidate
		}
		target := filepath.Join(dir, "project", filepath.FromSlash(f.Path))
		if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return inspectionSource{}, err
		}
		if err = atomicWrite(filepath.Dir(target), filepath.Base(target), data); err != nil {
			return inspectionSource{}, err
		}
		// Docker's unprivileged evaluator must be able to read the immutable source.
		if err = os.Chmod(target, 0444); err != nil {
			return inspectionSource{}, err
		}
		manifest.Files[i] = inspectionFile{f.Path, digest(data), len(data)}
	}
	if !changed {
		return inspectionSource{}, problem("invalid_request", "The proposal contains no source change.")
	}
	return manifest, writeInspectionJSON(filepath.Join(dir, "manifest.json"), manifest)
}

func readProposalFile(path string, value any) error {
	raw, err := readBoundedFile(path, 3<<20)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if validateJSON(raw) != nil || decoder.Decode(value) != nil {
		return errors.New("invalid proposal record")
	}
	return nil
}

// Managed copies are independently retained. Imported source still needs a
// conservative Project-wide source-sharing review; isolated managed origins do not participate.
func canRefinePipeline(c catalog, projectID, pipelineID string) bool {
	for _, p := range c.Pipelines {
		if p.ProjectID != projectID || p.PipelineID != pipelineID {
			continue
		}
		if _, managed := c.Revisions[pipelineID]; managed {
			return true
		}
		if p.Origin.Kind != "imported" {
			return false
		}
		count := 0
		for _, sibling := range c.Pipelines {
			if sibling.ProjectID == projectID && sibling.Origin.Kind == "imported" {
				count++
			}
		}
		return count == 1
	}
	return false
}
