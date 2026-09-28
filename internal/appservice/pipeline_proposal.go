package appservice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"
	"unicode/utf8"

	"github.com/HahyeonJeon/gobble/internal/pipelinereview"
)

// Eight records plus response framing fit the 16 MiB client response budget.
const maxProposalRecordBytes = 1792 << 10

type pipelineProposal struct {
	FollowUp   *runFollowUp               `json:"followUp,omitempty"`
	CreatedAt  string                     `json:"createdAt"`
	ProjectID  string                     `json:"projectId"`
	PipelineID string                     `json:"pipelineId"`
	ProposalID string                     `json:"proposalId"`
	Digest     string                     `json:"digest"`
	State      string                     `json:"state"`
	Summary    string                     `json:"summary"`
	Issue      string                     `json:"issue"`
	Base       pipelineArtifact           `json:"base"`
	Proposed   *pipelineArtifact          `json:"proposed,omitempty"`
	Comparison *pipelinereview.Comparison `json:"comparison,omitempty"`
	Managed    bool                       `json:"managed"`
}
type proposalInput struct {
	FollowUp       *runFollowUpInput `json:"followUp,omitempty"`
	RequestID      string            `json:"requestId"`
	BaseArtifactID string            `json:"baseArtifactId"`
	Summary        string            `json:"summary"`
	Files          []proposalFile    `json:"files"`
}
type proposalList struct {
	ProjectID         string             `json:"projectId"`
	PipelineID        string             `json:"pipelineId"`
	CurrentArtifactID string             `json:"currentArtifactId"`
	Managed           bool               `json:"managed"`
	Proposals         []pipelineProposal `json:"proposals"`
}

func (s *Service) readProposalLocked(projectID, pipelineID, id string) (pipelineProposal, error) {
	var value pipelineProposal
	if !validID(id, "req") {
		return value, problem("invalid_request", "A proposal identity is required.")
	}
	if err := readProposalFile(filepath.Join(s.proposalDirectory(pipelineID, id), "review.json"), &value); err != nil {
		return value, problem("not_found", "This proposal is unavailable.")
	}
	if value.ProjectID != projectID || value.PipelineID != pipelineID || value.ProposalID != id {
		return pipelineProposal{}, problem("internal", "Proposal identity mismatch.")
	}
	if value.State == "checking" && s.proposals[pipelineID] == nil {
		value.State = "failed"
		value.Issue = "The proposal check was interrupted. Keep this history and request a new proposal."
	}
	return value, nil
}
func (s *Service) pipelineProposals(projectID, pipelineID string) (proposalList, error) {
	if _, _, err := s.store.pipelineProject(projectID, pipelineID); err != nil {
		return proposalList{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	value := proposalList{ProjectID: projectID, PipelineID: pipelineID, Proposals: []pipelineProposal{}}
	current, err := s.readInspectionLocked(projectID, pipelineID)
	if err != nil {
		return value, err
	}
	if current.Artifact != nil {
		value.CurrentArtifactID = current.Artifact.ArtifactID
	}
	s.store.mu.Lock()
	_, value.Managed = s.store.data.Revisions[pipelineID]
	s.store.mu.Unlock()
	entries, err := os.ReadDir(s.proposalDirectory(pipelineID, ""))
	if errors.Is(err, os.ErrNotExist) {
		return value, nil
	}
	if err != nil {
		return value, err
	}
	if len(entries) > 8 {
		return value, problem("internal", "Proposal storage exceeds its retained limit.")
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		item, err := s.readProposalLocked(projectID, pipelineID, entry.Name())
		if err != nil {
			return value, err
		}
		value.Proposals = append(value.Proposals, item)
	}
	sort.SliceStable(value.Proposals, func(i, j int) bool { return value.Proposals[i].CreatedAt < value.Proposals[j].CreatedAt })
	return value, nil
}
func (s *Service) proposePipeline(projectID, pipelineID string, input proposalInput) (pipelineProposal, error) {
	if !validID(input.RequestID, "req") || !digestPattern.MatchString(input.BaseArtifactID) || len(input.Summary) > 2000 || !utf8.ValidString(input.Summary) {
		return pipelineProposal{}, problem("invalid_request", "A valid proposal identity and bounded explanation are required.")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing {
		return pipelineProposal{}, problem("runtime_unavailable", "The analysis service is closing.")
	}
	hash := requestDigest(input)
	if old, err := s.readProposalLocked(projectID, pipelineID, input.RequestID); err == nil {
		if old.Digest != hash {
			return old, problem("request_conflict", "This proposal identity belongs to different content.")
		}
		return old, nil
	}
	base, err := s.proposalBaseLocked(projectID, pipelineID, input.BaseArtifactID)
	if err != nil {
		return pipelineProposal{}, err
	}
	followUp, err := s.resolveRunFollowUpLocked(projectID, pipelineID, input.FollowUp)
	if err != nil {
		return pipelineProposal{}, err
	}
	if s.proposals[pipelineID] != nil || s.analysisFlightCountLocked() >= 2 {
		return pipelineProposal{}, problem("runtime_unavailable", "An analysis check is already in progress. Wait for it to finish.")
	}
	entries, err := os.ReadDir(s.proposalDirectory(pipelineID, ""))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return pipelineProposal{}, err
	}
	if len(entries) >= 8 {
		return pipelineProposal{}, problem("unsupported", "This Pipeline has reached its retained proposal limit (8).")
	}
	dir := s.proposalDirectory(pipelineID, input.RequestID)
	if err = os.MkdirAll(filepath.Dir(dir), 0700); err != nil {
		return pipelineProposal{}, err
	}
	if err = os.Mkdir(dir, 0700); err != nil {
		return pipelineProposal{}, problem("request_conflict", "This proposal identity was already used. Its outcome must be preserved.")
	}
	value := pipelineProposal{FollowUp: followUp, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), ProjectID: projectID, PipelineID: pipelineID, ProposalID: input.RequestID, Digest: hash, State: "checking", Summary: input.Summary, Base: base.Artifact, Managed: base.Managed}
	manifest, err := retainProposal(base, input.Files, dir)
	if err != nil {
		value.State = "failed"
		value.Issue = inspectionIssue(err)
		if e := writeInspectionJSON(filepath.Join(dir, "review.json"), value); e != nil {
			return value, e
		}
		return value, err
	}
	if err = writeInspectionJSON(filepath.Join(dir, "review.json"), value); err != nil {
		return value, err
	}
	ctx, cancel := context.WithTimeout(s.ctx, 2*inspectionTimeout)
	if s.proposals == nil {
		s.proposals = map[string]*pipelineFlight{}
	}
	s.proposals[pipelineID] = &pipelineFlight{input.RequestID, cancel}
	s.workers.Add(1)
	go s.checkProposal(ctx, base, manifest, value)
	return value, nil
}

type checkedReview struct {
	SchemaVersion int                       `json:"schemaVersion"`
	Flow          json.RawMessage           `json:"flow"`
	Definition    pipelinereview.Definition `json:"definition"`
}

func (s *Service) evaluateReview(ctx context.Context, dir string, manifest inspectionSource) (checkedReview, error) {
	var value checkedReview
	raw, err := s.runtime.pipelineEvaluation(ctx, dir, manifest, "review")
	if err != nil {
		return value, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if len(raw) > 2<<20 || validateJSON(raw) != nil || decoder.Decode(&value) != nil || value.SchemaVersion != 1 || value.Definition.SchemaVersion != 1 || len(value.Flow) > maxFlowBytes || len(value.Definition.Steps) > 500 || len(value.Definition.Edges) > 2000 {
		return value, problem("unsupported", "The pinned analysis runtime does not provide a supported complete review.")
	}
	if err := pipelinereview.Validate(value.Definition); err != nil {
		return value, problem("unsupported", "The checked review definition is invalid.")
	}
	return value, nil
}
func (s *Service) checkProposal(ctx context.Context, base proposalBase, manifest inspectionSource, value pipelineProposal) {
	defer s.workers.Done()
	dir := s.proposalDirectory(value.PipelineID, value.ProposalID)
	before, err := s.evaluateReview(ctx, base.Directory, base.Manifest)
	if err == nil && !sameReviewJSON(before.Flow, base.Artifact.Flow) {
		err = problem("stale_revision", "The checked source no longer reproduces the current flow. Review its deterministic inputs.")
	}
	var after checkedReview
	if err == nil {
		after, err = s.evaluateReview(ctx, filepath.Join(dir, "project"), manifest)
	}
	if err == nil {
		err = s.runtime.verify(ctx, manifest.Runtime)
	}
	if err == nil && ctx.Err() != nil {
		err = ctx.Err()
	}
	if err == nil {
		result := pipelinereview.Compare(before.Definition, after.Definition)
		value.Proposed = &pipelineArtifact{requestDigest([]any{manifest, json.RawMessage(after.Flow)}), requestDigest(manifest), time.Now().UTC().Format(time.RFC3339Nano), after.Flow}
		value.Comparison = &result
		value.State = "ready"
	} else {
		value.State = "failed"
		value.Issue = inspectionIssue(err)
	}
	if raw, marshalErr := json.Marshal(value); marshalErr != nil || len(raw) > maxProposalRecordBytes {
		value.State = "failed"
		value.Issue = "This comparison exceeds the retained review limit. Request a smaller refinement."
		value.Proposed = nil
		value.Comparison = nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	defer func() {
		if flight := s.proposals[value.PipelineID]; flight != nil {
			flight.cancel()
		}
		delete(s.proposals, value.PipelineID)
	}()
	if err = writeInspectionJSON(filepath.Join(dir, "review.json"), value); err != nil {
		s.store.mu.Lock()
		s.store.failed = true
		s.store.mu.Unlock()
	}
}

func sameReviewJSON(a, b []byte) bool {
	var left, right any
	if json.Unmarshal(a, &left) != nil || json.Unmarshal(b, &right) != nil {
		return false
	}
	return requestDigest(left) == requestDigest(right)
}
