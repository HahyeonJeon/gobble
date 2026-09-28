package appservice

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Birth is retained even after Current points to a later refinement. No Run authority.
type creationAdoption struct {
	RequestID   string `json:"requestId"`
	PipelineID  string `json:"pipelineId"`
	CandidateID string `json:"candidateId"`
	ArtifactID  string `json:"artifactId"`
	Generation  int64  `json:"generation"`
	Name        string `json:"name"`
}

// Current source has exactly one origin: an existing proposal OR a creation candidate.
type creationRevision struct {
	DraftID     string `json:"draftId"`
	CandidateID string `json:"candidateId"`
}
type adoptCreationInput struct {
	RequestID          string `json:"requestId"`
	CandidateID        string `json:"candidateId"`
	ArtifactID         string `json:"artifactId"`
	ExpectedGeneration int64  `json:"expectedGeneration"`
	Name               string `json:"name"`
}
type creationAdoptionResult struct {
	ProjectID string            `json:"projectId"`
	DraftID   string            `json:"draftId"`
	RequestID string            `json:"requestId"`
	State     string            `json:"state"`
	Adoption  *creationAdoption `json:"adoption,omitempty"`
}

func validCreationName(name string) bool {
	return utf8.ValidString(name) && len(name) > 0 && len(name) <= 200 && strings.TrimSpace(name) == name && !strings.ContainsAny(name, "\x00\r\n")
}
func validateCreationAdoption(c catalog, d creationDraft) error {
	bad := func() error { return errors.New("invalid creation adoption") }
	if d.State != "adopted" {
		if d.Adoption != nil {
			return bad()
		}
		return nil
	}
	a := d.Adoption
	if a == nil || d.Input == nil || !validID(a.RequestID, "req") || !validID(a.PipelineID, "pip") || !validID(a.CandidateID, "req") || !digestPattern.MatchString(a.ArtifactID) || a.Generation != d.Generation-1 || !validCreationName(a.Name) {
		return bad()
	}
	r := c.Requests[a.RequestID]
	if r.DraftID != d.DraftID || r.ProjectID != d.ProjectID || r.Digest != creationAdoptionIntent(d.ProjectID, d.DraftID, adoptCreationInput{a.RequestID, a.CandidateID, a.ArtifactID, a.Generation, a.Name}) {
		return bad()
	}
	for _, other := range c.Drafts {
		if other.DraftID != d.DraftID && other.Adoption != nil && other.Adoption.PipelineID == a.PipelineID {
			return bad()
		}
	}
	for _, p := range c.Pipelines {
		if p.PipelineID == a.PipelineID && p.ProjectID == d.ProjectID && p.Origin.Kind == "managed" {
			return nil
		}
	}
	return bad()
}
func validManagedSource(c catalog, r managedRevision) bool {
	if r.Creation == nil {
		return validID(r.ProposalID, "req")
	}
	d := c.Drafts[r.Creation.DraftID]
	a := d.Adoption
	return r.ProposalID == "" && d.State == "adopted" && a != nil && a.PipelineID == r.PipelineID && a.CandidateID == r.Creation.CandidateID && a.ArtifactID == r.Artifact.ArtifactID
}
func (s *Service) creationAdoptionOutcome(projectID, draftID, requestID string) (creationAdoptionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	return s.creationAdoptionOutcomeLocked(projectID, draftID, requestID)
}

// Both writer locks held. Never infers absence from poisoned in-memory state.
func (s *Service) creationAdoptionOutcomeLocked(projectID, draftID, requestID string) (creationAdoptionResult, error) {
	out := creationAdoptionResult{ProjectID: projectID, DraftID: draftID, RequestID: requestID, State: "not-recorded"}
	if !validID(requestID, "req") {
		return out, problem("invalid_request", "An adoption identity is required.")
	}
	d, err := s.store.draftLocked(projectID, draftID)
	if err != nil {
		return out, err
	}
	r, ok := s.store.data.Requests[requestID]
	if !ok {
		return out, nil
	}
	a := d.Adoption
	if r.ProjectID != projectID || r.DraftID != draftID || a == nil || r.Digest != creationAdoptionIntent(projectID, draftID, adoptCreationInput{requestID, a.CandidateID, a.ArtifactID, a.Generation, a.Name}) {
		return out, problem("request_conflict", "This identity belongs to another action.")
	}
	out.State = "adopted"
	out.Adoption = a
	return out, nil
}
func creationAdoptionIntent(projectID, draftID string, input adoptCreationInput) string {
	return requestDigest([]any{"adopt-creation", projectID, draftID, input.CandidateID, input.ArtifactID, input.ExpectedGeneration, input.Name})
}
func (s *Service) adoptCreation(ctx context.Context, projectID, draftID string, input adoptCreationInput) (creationAdoptionResult, error) {
	if !validID(input.RequestID, "req") || !validID(input.CandidateID, "req") || !digestPattern.MatchString(input.ArtifactID) || input.ExpectedGeneration < 1 || input.ExpectedGeneration >= maxDraftGeneration || !validCreationName(input.Name) {
		return creationAdoptionResult{}, problem("invalid_request", "Provide the exact checked proposal and a Pipeline name of up to 200 bytes.")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing {
		return creationAdoptionResult{}, problem("runtime_unavailable", "The service is closing.")
	}
	intent := creationAdoptionIntent(projectID, draftID, input)
	s.store.mu.Lock()
	d, err := s.store.draftLocked(projectID, draftID)
	if err != nil {
		s.store.mu.Unlock()
		return creationAdoptionResult{}, err
	}
	if r, ok := s.store.data.Requests[input.RequestID]; ok {
		if r.Digest != intent {
			s.store.mu.Unlock()
			return creationAdoptionResult{}, problem("request_conflict", "This adoption identity belongs to different content.")
		}
		out, e := s.creationAdoptionOutcomeLocked(projectID, draftID, input.RequestID)
		s.store.mu.Unlock()
		return out, e
	}
	// A second identical User intent resolves to the already registered Pipeline.
	if a := d.Adoption; a != nil {
		if a.CandidateID != input.CandidateID || a.ArtifactID != input.ArtifactID || a.Generation != input.ExpectedGeneration || a.Name != input.Name {
			s.store.mu.Unlock()
			return creationAdoptionResult{}, problem("stale_revision", "This draft was already adopted. Open its Pipeline.")
		}
		out, e := s.publishCreationAdoptionLocked(d, input, intent, nil)
		s.store.mu.Unlock()
		return out, e
	}
	s.store.mu.Unlock()
	if d.State != "draft" || d.Generation != input.ExpectedGeneration {
		return creationAdoptionResult{}, problem("stale_revision", "The draft changed. Review its current proposal.")
	}
	candidate, err := s.readCreationCandidateLocked(projectID, draftID, input.CandidateID)
	if err != nil {
		return creationAdoptionResult{}, err
	}
	m, err := s.checkedCreationSource(candidate)
	if err != nil {
		return creationAdoptionResult{}, err
	}
	if candidate.State != "ready" || candidate.Generation != input.ExpectedGeneration || candidate.Artifact.ArtifactID != input.ArtifactID || requestDigest(d) != requestDigest(m.Draft) {
		return creationAdoptionResult{}, problem("stale_revision", "Choose the exact checked proposal for this draft version.")
	}
	current, err := s.readCreationRuntime()
	if err != nil {
		return creationAdoptionResult{}, err
	}
	if current.ID != m.Runtime.ID {
		return creationAdoptionResult{}, problem("stale_revision", "The analysis engine selection changed. Check a new proposal.")
	}
	if err = s.runtime.verify(ctx, m.Runtime.Binding); err != nil {
		return creationAdoptionResult{}, err
	}
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	// Input and generation are observed immediately before the one durable publication.
	d, err = s.store.mutableDraftLocked(projectID, draftID, input.ExpectedGeneration)
	if err != nil {
		return creationAdoptionResult{}, err
	}
	p, err := s.store.draftProjectLocked(projectID)
	if err != nil {
		return creationAdoptionResult{}, err
	}
	observed, err := observeCreationInput(p, m.Draft.Input.ResourceID)
	if err != nil || observed != *m.Draft.Input {
		return creationAdoptionResult{}, problem("stale_revision", "The selected file changed. Select the data again and request a new check.")
	}
	if ctx.Err() != nil {
		return creationAdoptionResult{}, problem("runtime_unavailable", "Adoption was interrupted before publication.")
	}
	// Detect retained source drift after the runtime query, too.
	if err = verifyCreationSource(filepath.Join(s.creationDirectory(draftID, input.CandidateID), "project"), m); err != nil {
		return creationAdoptionResult{}, err
	}
	return s.publishCreationAdoptionLocked(d, input, intent, candidate.Artifact)
}
func (s *Service) publishCreationAdoptionLocked(d creationDraft, input adoptCreationInput, intent string, artifact *creationArtifact) (creationAdoptionResult, error) {
	if len(s.store.data.Requests) >= 10000 {
		return creationAdoptionResult{}, problem("unsupported", "The catalog request limit has been reached.")
	}
	next := s.store.copyLocked()
	if d.Adoption == nil {
		if artifact == nil || len(next.Pipelines) >= maxPipelines {
			return creationAdoptionResult{}, problem("unsupported", "A Pipeline cannot be registered.")
		}
		id := newID("pip")
		d.Adoption = &creationAdoption{input.RequestID, id, input.CandidateID, input.ArtifactID, input.ExpectedGeneration, input.Name}
		d.State = "adopted"
		d.Generation++
		next.Drafts[d.DraftID] = cloneCreationDraft(d)
		next.Pipelines = append(next.Pipelines, PipelineDefinition{d.ProjectID, id, input.Name, PipelineOrigin{Kind: "managed"}})
		next.Revisions[id] = managedRevision{PipelineID: id, Creation: &creationRevision{d.DraftID, input.CandidateID}, Artifact: pipelineArtifact{artifact.ArtifactID, artifact.SourceRevision, artifact.CheckedAt, artifact.Check.Review.Flow}}
	}
	next.Requests[input.RequestID] = receipt{Digest: intent, ProjectID: d.ProjectID, DraftID: d.DraftID}
	if err := s.store.commitLocked(next); err != nil {
		return creationAdoptionResult{}, err
	}
	s.cancelDraftFlightLocked(d)
	return s.creationAdoptionOutcomeLocked(d.ProjectID, d.DraftID, input.RequestID)
}
