package appservice

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
)

// One Pipeline current pointer and its operation receipt publish in a single
// catalog commit. All referenced bytes and checked review facts precede it.
type managedRevision struct {
	Creation   *creationRevision `json:"creation,omitempty"`
	PipelineID string            `json:"pipelineId"`
	ProposalID string            `json:"proposalId,omitempty"`
	Artifact   pipelineArtifact  `json:"artifact"`
}
type adoptionResult struct {
	ProjectID  string `json:"projectId"`
	PipelineID string `json:"pipelineId"`
	RequestID  string `json:"requestId"`
	ProposalID string `json:"proposalId"`
	State      string `json:"state"`
	ArtifactID string `json:"artifactId"`
}

func validateManagedRevisions(c catalog, projects map[string]projectRecord, pipelines map[string]string) error {
	if c.Revisions == nil || len(c.Revisions) > maxPipelines {
		return errors.New("invalid managed revisions")
	}
	for pipelineID, r := range c.Revisions {
		project := pipelines[pipelineID]
		if projects[project].ProjectID == "" || r.PipelineID != pipelineID || !validManagedSource(c, r) || !digestPattern.MatchString(r.Artifact.ArtifactID) || !digestPattern.MatchString(r.Artifact.SourceRevision) || !json.Valid(r.Artifact.Flow) {
			return errors.New("invalid current source revision")
		}
	}
	return nil
}
func (s *Service) adoptionOutcome(projectID, pipelineID, requestID string) (adoptionResult, error) {
	if !validID(requestID, "req") {
		return adoptionResult{}, problem("invalid_request", "An adoption identity is required.")
	}
	if _, _, err := s.store.pipelineProject(projectID, pipelineID); err != nil {
		return adoptionResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.adoptionOutcomeLocked(projectID, pipelineID, requestID)
}
func (s *Service) adoptionOutcomeLocked(projectID, pipelineID, requestID string) (adoptionResult, error) {
	out := adoptionResult{ProjectID: projectID, PipelineID: pipelineID, RequestID: requestID, State: "not-recorded"}
	s.store.mu.Lock()
	// A failed atomic write may have renamed the durable catalog before its
	// directory sync failed. The old in-memory map cannot prove absence.
	if s.store.failed {
		s.store.mu.Unlock()
		return out, problem("runtime_unavailable", "The saved adoption outcome is uncertain. Restart the App before checking it again.")
	}
	receipt, ok := s.store.data.Requests[requestID]
	s.store.mu.Unlock()
	if !ok {
		return out, nil
	}
	if receipt.ProjectID != projectID || receipt.PipelineID != pipelineID || receipt.ProposalID == "" {
		return out, problem("request_conflict", "This operation identity belongs to a different action.")
	}
	value, err := s.readProposalLocked(projectID, pipelineID, receipt.ProposalID)
	if err != nil || value.Proposed == nil {
		return out, problem("internal", "The adopted comparison is unavailable. Preserve the profile.")
	}
	out.ProposalID = receipt.ProposalID
	out.State = "adopted"
	out.ArtifactID = value.Proposed.ArtifactID
	return out, nil
}
func (s *Service) adoptPipeline(ctx context.Context, projectID, pipelineID, proposalID, requestID, expectedArtifactID string) (adoptionResult, error) {
	if !validID(requestID, "req") || !validID(proposalID, "req") {
		return adoptionResult{}, problem("invalid_request", "Exact proposal and operation identities are required.")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	hash := requestDigest([]string{"adopt-pipeline", projectID, pipelineID, proposalID, expectedArtifactID})
	s.store.mu.Lock()
	prior, exists := s.store.data.Requests[requestID]
	s.store.mu.Unlock()
	if exists {
		if prior.Digest != hash {
			return adoptionResult{}, problem("request_conflict", "This adoption identity belongs to different content.")
		}
		return s.adoptionOutcomeLocked(projectID, pipelineID, requestID)
	}
	value, err := s.readProposalLocked(projectID, pipelineID, proposalID)
	if err != nil {
		return adoptionResult{}, err
	}
	if value.State != "ready" || value.Proposed == nil || value.Proposed.ArtifactID != expectedArtifactID || value.Comparison == nil || len(value.Comparison.Gaps) > 0 || len(value.Comparison.Changes) == 0 {
		return adoptionResult{}, problem("unsupported", "Only the exact, fully explained checked proposal can be adopted.")
	}
	base, err := s.proposalBaseLocked(projectID, pipelineID, value.Base.ArtifactID)
	if err != nil {
		return adoptionResult{}, err
	}
	if err = s.runtime.verify(ctx, base.Manifest.Runtime); err != nil {
		return adoptionResult{}, err
	}
	// Read every adopted file again before publishing the pointer, detecting both
	// incomplete storage and any external modification to retained profile bytes.
	dir := s.proposalDirectory(pipelineID, proposalID)
	var manifest inspectionSource
	if readProposalFile(filepath.Join(dir, "manifest.json"), &manifest) != nil || requestDigest(manifest) != value.Proposed.SourceRevision {
		return adoptionResult{}, problem("internal", "Proposed source provenance is unavailable.")
	}
	for _, f := range manifest.Files {
		raw, err := readBoundedFile(filepath.Join(dir, "project", filepath.FromSlash(f.Path)), maxFileBytes)
		if err != nil || digest(raw) != f.SHA256 {
			return adoptionResult{}, problem("internal", "Proposed source failed its integrity check.")
		}
	}
	if ctx.Err() != nil {
		return adoptionResult{}, problem("runtime_unavailable", "Adoption was interrupted before publication.")
	}
	if _, err = s.proposalBaseLocked(projectID, pipelineID, value.Base.ArtifactID); err != nil {
		return adoptionResult{}, err
	}
	s.store.mu.Lock()
	next := s.store.copyLocked()
	// Service.mu prevents competing proposal/adoption/check publication. Catalog
	// still rechecks participating source ownership under its own writer lock.
	if !canRefinePipeline(next, projectID, pipelineID) {
		s.store.mu.Unlock()
		return adoptionResult{}, problem("stale_revision", "Project Pipeline ownership changed during review.")
	}

	next.Revisions[pipelineID] = managedRevision{PipelineID: pipelineID, ProposalID: proposalID, Artifact: *value.Proposed}
	next.Requests[requestID] = receipt{Digest: hash, ProjectID: projectID, PipelineID: pipelineID, ProposalID: proposalID}
	err = s.store.commitLocked(next)
	s.store.mu.Unlock()
	if err != nil {
		return adoptionResult{}, err
	}
	return s.adoptionOutcomeLocked(projectID, pipelineID, requestID)
}
