package appservice

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"
	"unicode/utf8"
)

type creationCandidateInput struct {
	RequestID          string         `json:"requestId"`
	ExpectedGeneration int64          `json:"expectedGeneration"`
	ScopeID            string         `json:"scopeId"`
	Summary            string         `json:"summary"`
	Files              []proposalFile `json:"files"`
}
type creationCandidate struct {
	ProjectID   string            `json:"projectId"`
	DraftID     string            `json:"draftId"`
	CandidateID string            `json:"candidateId"`
	Generation  int64             `json:"generation"`
	Digest      string            `json:"digest"`
	State       string            `json:"state"`
	Issue       string            `json:"issue"`
	Summary     string            `json:"summary"`
	CreatedAt   string            `json:"createdAt"`
	Artifact    *creationArtifact `json:"artifact,omitempty"`
}
type creationArtifact struct {
	ArtifactID     string        `json:"artifactId"`
	SourceRevision string        `json:"sourceRevision"`
	Input          creationInput `json:"input"`
	RuntimeID      string        `json:"runtimeId"`
	CheckedAt      string        `json:"checkedAt"`
	Check          creationCheck `json:"check"`
}
type creationCheck struct {
	SchemaVersion int           `json:"schemaVersion"`
	Scope         string        `json:"scope"`
	InputPath     string        `json:"inputPath"`
	Review        checkedReview `json:"review"`
	Gaps          []string      `json:"gaps"`
}

func (s *Service) readCreationCandidateLocked(projectID, draftID, candidateID string) (creationCandidate, error) {
	var v creationCandidate
	if _, err := s.store.creationDraft(projectID, draftID); err != nil {
		return v, err
	}
	if !validID(candidateID, "req") {
		return v, problem("invalid_request", "A candidate identity is required.")
	}
	err := readProposalFile(filepath.Join(s.creationDirectory(draftID, candidateID), "candidate.json"), &v)
	if err != nil {
		return v, err
	}
	if v.ProjectID != projectID || v.DraftID != draftID || v.CandidateID != candidateID {
		return creationCandidate{}, problem("internal", "Creation candidate identity mismatch.")
	}
	if v.State == "checking" && (s.creations[draftID] == nil || s.creations[draftID].id != candidateID) {
		v.State = "cancelled"
		v.Issue = "The previous check was interrupted. Request a new check; this history is retained."
	}
	return v, nil
}
func (s *Service) creationCandidate(projectID, draftID, id string) (creationCandidate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, err := s.readCreationCandidateLocked(projectID, draftID, id)
	if errors.Is(err, os.ErrNotExist) {
		err = problem("not_found", "Creation candidate not found.")
	}
	return v, err
}
func (s *Service) submitCreation(projectID, draftID string, input creationCandidateInput) (creationCandidate, error) {
	if !validID(input.RequestID, "req") || !digestPattern.MatchString(input.ScopeID) || input.ExpectedGeneration < 1 || len(input.Summary) > 2000 || !utf8.ValidString(input.Summary) {
		return creationCandidate{}, problem("invalid_request", "Provide the exact creation scope, generation and bounded explanation.")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing {
		return creationCandidate{}, problem("runtime_unavailable", "The service is closing.")
	}
	intent := requestDigest(input)
	old, err := s.readCreationCandidateLocked(projectID, draftID, input.RequestID)
	if err == nil {
		if old.Digest != intent {
			return old, problem("request_conflict", "This candidate identity belongs to different source.")
		}
		return old, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return creationCandidate{}, err
	}
	scope, runtime, err := s.creationScopeLocked(projectID, draftID, input.ExpectedGeneration)
	if err != nil {
		return creationCandidate{}, err
	}
	if scope.ScopeID != input.ScopeID {
		return creationCandidate{}, problem("stale_revision", "The creation scope changed. Request its latest saved source.")
	}
	if s.creations[draftID] != nil || s.analysisFlightCountLocked() >= 2 {
		return creationCandidate{}, problem("runtime_unavailable", "An analysis check is already in progress.")
	}
	entries, err := os.ReadDir(s.creationDirectory(draftID, ""))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return creationCandidate{}, err
	}
	if len(entries) >= 8 {
		return creationCandidate{}, problem("unsupported", "This draft reached its retained candidate limit (8).")
	}
	dir := s.creationDirectory(draftID, input.RequestID)
	if err = os.MkdirAll(filepath.Dir(dir), 0700); err != nil {
		return creationCandidate{}, err
	}
	if err = os.Mkdir(dir, 0700); err != nil {
		return creationCandidate{}, problem("request_conflict", "This candidate identity was already used. Preserve its saved outcome.")
	}
	v := creationCandidate{ProjectID: projectID, DraftID: draftID, CandidateID: input.RequestID, Generation: input.ExpectedGeneration, Digest: intent, State: "checking", Summary: input.Summary, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	manifest, err := retainCreation(scope, runtime, input.Files, dir)
	if err != nil {
		v.State = "failed"
		v.Issue = inspectionIssue(err)
	}
	if saveErr := writeInspectionJSON(filepath.Join(dir, "candidate.json"), v); saveErr != nil {
		return creationCandidate{}, saveErr
	}
	if err != nil {
		return v, err
	}
	if s.creations == nil {
		s.creations = map[string]*pipelineFlight{}
	}
	ctx, cancel := context.WithTimeout(s.ctx, inspectionTimeout)
	s.creations[draftID] = &pipelineFlight{input.RequestID, cancel}
	s.workers.Add(1)
	go s.checkCreation(ctx, manifest, v)
	return v, nil
}
func (s *Service) cancelCreation(projectID, draftID, id string) (creationCandidate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, err := s.readCreationCandidateLocked(projectID, draftID, id)
	if err != nil {
		return v, err
	}
	if f := s.creations[draftID]; f != nil && f.id == id {
		f.cancel()
	}
	return v, nil
}
func (s *Service) analysisFlightCountLocked() int {
	return len(s.inspections) + len(s.proposals) + len(s.creations) + len(s.preparations)
}

func (s *Service) listCreationCandidates(projectID, draftID string) ([]creationCandidate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.store.creationDraft(projectID, draftID); err != nil {
		return nil, err
	}
	out := []creationCandidate{}
	entries, err := os.ReadDir(s.creationDirectory(draftID, ""))
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	if len(entries) > 8 {
		return nil, problem("internal", "Creation history exceeds its retained limit.")
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			return nil, problem("internal", "Unexpected creation history entry.")
		}
		value, err := s.readCreationCandidateLocked(projectID, draftID, entry.Name())
		if err != nil {
			return nil, problem("internal", "A saved candidate is incomplete. Preserve the profile before recovery.")
		}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return out, nil
}
