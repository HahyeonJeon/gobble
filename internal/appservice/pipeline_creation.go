package appservice

import (
	"errors"
	"sort"
	"strings"
	"unicode/utf8"
)

const maxCreationDrafts = 1000
const maxDraftGeneration = 9007199254740991

// creationDraft owns intent, never source validity, Current or a Run. Discard is
// a retained tombstone so delayed writers cannot recreate the same draft.
type creationDraft struct {
	Adoption   *creationAdoption `json:"adoption,omitempty"`
	ProjectID  string            `json:"projectId"`
	DraftID    string            `json:"draftId"`
	Generation int64             `json:"generation"`
	State      string            `json:"state"`
	Brief      string            `json:"brief"`
	Input      *creationInput    `json:"input,omitempty"`
}
type creationDrafts struct {
	ProjectID string          `json:"projectId"`
	Drafts    []creationDraft `json:"drafts"`
}
type createDraftInput struct {
	RequestID string `json:"requestId"`
	Brief     string `json:"brief"`
}
type updateDraftInput struct {
	RequestID          string `json:"requestId"`
	ExpectedGeneration int64  `json:"expectedGeneration"`
	Brief              string `json:"brief"`
	ResourceID         string `json:"resourceId"`
	ReadLayout         string `json:"readLayout"`
}
type discardDraftInput struct {
	RequestID          string `json:"requestId"`
	ExpectedGeneration int64  `json:"expectedGeneration"`
}

func validBrief(brief string) bool {
	return utf8.ValidString(brief) && len(brief) <= 8000 && !strings.ContainsRune(brief, 0)
}
func cloneCreationDraft(d creationDraft) creationDraft {
	if d.Adoption != nil {
		a := *d.Adoption
		d.Adoption = &a
	}
	if d.Input != nil {
		input := *d.Input
		d.Input = &input
	}
	return d
}
func validateCreationDrafts(c catalog, projects map[string]projectRecord) error {
	if c.Drafts == nil || len(c.Drafts) > maxCreationDrafts {
		return errors.New("invalid creation drafts")
	}
	for id, d := range c.Drafts {
		if !validID(id, "drf") || d.DraftID != id || projects[d.ProjectID].ProjectID == "" || d.Generation < 1 || d.Generation > maxDraftGeneration || (d.State != "draft" && d.State != "discarded" && d.State != "adopted") || !validBrief(d.Brief) {
			return errors.New("invalid creation draft")
		}
		if err := validateCreationAdoption(c, d); err != nil {
			return err
		}
		if d.Input != nil && !validCreationInput(*d.Input, projects[d.ProjectID]) {
			return errors.New("invalid draft input")
		}
	}
	return nil
}

// Receipt lookup and mutation use the same catalog lock. Retries return the
// latest saved draft (including a tombstone), and never replay a historical edit.
func (s *store) draftReceiptLocked(projectID, draftID, requestID, intent string) (creationDraft, bool, error) {
	if s.failed {
		return creationDraft{}, false, problem("internal", "Draft storage is unavailable. Restart before checking the saved outcome.")
	}
	old, ok := s.data.Requests[requestID]
	if !ok {
		return creationDraft{}, false, nil
	}
	if old.Digest != intent || old.ProjectID != projectID || old.DraftID == "" || (draftID != "" && old.DraftID != draftID) {
		return creationDraft{}, true, problem("request_conflict", "This request identity belongs to a different action.")
	}
	return cloneCreationDraft(s.data.Drafts[old.DraftID]), true, nil
}
func (s *store) createDraft(projectID string, input createDraftInput) (creationDraft, error) {
	if !validID(projectID, "prj") || !validID(input.RequestID, "req") || !validBrief(input.Brief) {
		return creationDraft{}, problem("invalid_request", "Provide a valid Project, request identity and brief of at most 8000 bytes.")
	}
	intent := requestDigest([]string{"create-draft", projectID, input.Brief})
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, found, err := s.draftReceiptLocked(projectID, "", input.RequestID, intent); found || err != nil {
		return d, err
	}
	if _, err := s.draftProjectLocked(projectID); err != nil {
		return creationDraft{}, err
	}
	if len(s.data.Drafts) >= maxCreationDrafts {
		return creationDraft{}, problem("unsupported", "The retained draft limit has been reached.")
	}
	d := creationDraft{ProjectID: projectID, DraftID: newID("drf"), Generation: 1, State: "draft", Brief: input.Brief}
	return s.saveDraftLocked(d, input.RequestID, intent)
}
func (s *store) draftProjectLocked(projectID string) (projectRecord, error) {
	for _, p := range s.data.Projects {
		if p.ProjectID == projectID {
			return p, nil
		}
	}
	return projectRecord{}, problem("not_found", "Project not found.")
}
func (s *store) creationDraft(projectID, draftID string) (creationDraft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.draftLocked(projectID, draftID)
}
func (s *store) draftLocked(projectID, draftID string) (creationDraft, error) {
	if s.failed {
		return creationDraft{}, problem("internal", "Draft storage is unavailable. Restart before continuing.")
	}
	d, ok := s.data.Drafts[draftID]
	if !ok || d.ProjectID != projectID {
		return creationDraft{}, problem("not_found", "Draft not found in this Project.")
	}
	return cloneCreationDraft(d), nil
}
func (s *store) listCreationDrafts(projectID string) (creationDrafts, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failed {
		return creationDrafts{}, problem("internal", "Draft storage is unavailable. Restart before continuing.")
	}
	if _, err := s.draftProjectLocked(projectID); err != nil {
		return creationDrafts{}, err
	}
	out := creationDrafts{projectID, []creationDraft{}}
	for _, d := range s.data.Drafts {
		if d.ProjectID == projectID && d.State == "draft" {
			out.Drafts = append(out.Drafts, cloneCreationDraft(d))
		}
	}
	sort.Slice(out.Drafts, func(i, j int) bool { return out.Drafts[i].DraftID < out.Drafts[j].DraftID })
	return out, nil
}
func (s *store) updateDraft(projectID, draftID string, input updateDraftInput) (creationDraft, error) {
	if !validID(input.RequestID, "req") || !validBrief(input.Brief) || input.ExpectedGeneration < 1 || input.ExpectedGeneration >= maxDraftGeneration ||
		(input.ResourceID == "" && input.ReadLayout != "") || (input.ResourceID != "" && (!validID(input.ResourceID, "res") || input.ReadLayout != "single-end")) {
		return creationDraft{}, problem("invalid_request", "Choose one single-end input and a valid draft generation and brief.")
	}
	intent := requestDigest(struct {
		Action, Project, Draft string
		Input                  updateDraftInput
	}{"update-draft", projectID, draftID, input})
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, found, err := s.draftReceiptLocked(projectID, draftID, input.RequestID, intent); found || err != nil {
		return d, err
	}
	d, err := s.mutableDraftLocked(projectID, draftID, input.ExpectedGeneration)
	if err != nil {
		return d, err
	}
	p, err := s.draftProjectLocked(projectID)
	if err != nil {
		return creationDraft{}, err
	}
	d.Input = nil
	if input.ResourceID != "" {
		observed, err := observeCreationInput(p, input.ResourceID)
		if err != nil {
			return creationDraft{}, err
		}
		d.Input = &observed
	}
	d.Brief = input.Brief
	d.Generation++
	return s.saveDraftLocked(d, input.RequestID, intent)
}
func (s *store) discardDraft(projectID, draftID string, input discardDraftInput) (creationDraft, error) {
	if !validID(input.RequestID, "req") || input.ExpectedGeneration < 1 || input.ExpectedGeneration >= maxDraftGeneration {
		return creationDraft{}, problem("invalid_request", "Provide the exact draft generation and request identity.")
	}
	intent := requestDigest(struct {
		Action, Project, Draft string
		Input                  discardDraftInput
	}{"discard-draft", projectID, draftID, input})
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, found, err := s.draftReceiptLocked(projectID, draftID, input.RequestID, intent); found || err != nil {
		return d, err
	}
	d, err := s.mutableDraftLocked(projectID, draftID, input.ExpectedGeneration)
	if err != nil {
		return d, err
	}
	d.State = "discarded"
	d.Generation++
	return s.saveDraftLocked(d, input.RequestID, intent)
}
func (s *store) mutableDraftLocked(projectID, draftID string, generation int64) (creationDraft, error) {
	d, err := s.draftLocked(projectID, draftID)
	if err != nil {
		return d, err
	}
	if d.State != "draft" || d.Generation != generation {
		return creationDraft{}, problem("stale_revision", "The draft changed or was discarded. Reopen its saved state before continuing.")
	}
	return d, nil
}
func (s *store) saveDraftLocked(d creationDraft, requestID, intent string) (creationDraft, error) {
	if len(s.data.Requests) >= 10000 {
		return creationDraft{}, problem("unsupported", "The catalog request limit has been reached.")
	}
	next := s.copyLocked()
	next.Drafts[d.DraftID] = cloneCreationDraft(d)
	next.Requests[requestID] = receipt{Digest: intent, ProjectID: d.ProjectID, DraftID: d.DraftID}
	if err := s.commitLocked(next); err != nil {
		return creationDraft{}, err
	}
	return cloneCreationDraft(d), nil
}
