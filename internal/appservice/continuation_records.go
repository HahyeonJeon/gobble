package appservice

import (
	"errors"
	"os"
	"path/filepath"
	"sort"

	"github.com/HahyeonJeon/gobble/internal/preparation"
)

type continuationOperation struct {
	RequestID     string                             `json:"requestId"`
	State         string                             `json:"state"`
	Issue         string                             `json:"issue"`
	Receipt       *preparation.ContinuationAdmission `json:"receipt,omitempty"`
	Observation   *admittedState                     `json:"observation,omitempty"`
	StopRequestID string                             `json:"stopRequestId"`
	StopLease     string                             `json:"stopLease"`
	StopState     string                             `json:"stopState"`
}
type continuationView struct {
	ProjectID      string                          `json:"projectId"`
	PipelineID     string                          `json:"pipelineId"`
	LaunchReviewID string                          `json:"launchReviewId"`
	RunRef         string                          `json:"runRef"`
	RequestID      string                          `json:"requestId"`
	CreatedAt      string                          `json:"createdAt"`
	State          string                          `json:"state"`
	Issue          string                          `json:"issue"`
	Review         *preparation.ContinuationReview `json:"review,omitempty"`
	Operation      *continuationOperation          `json:"operation,omitempty"`
}
type continuationRecord struct {
	SchemaVersion int                             `json:"schemaVersion"`
	Checksum      string                          `json:"checksum"`
	Value         continuationView                `json:"value"`
	Intent        *preparation.ContinuationIntent `json:"intent,omitempty"`
}
type continuationCheckInput struct {
	RequestID      string `json:"requestId"`
	LaunchReviewID string `json:"launchReviewId"`
	RunRef         string `json:"runRef"`
}
type continuationConfirmInput struct {
	RequestID    string `json:"requestId"`
	ReviewDigest string `json:"reviewDigest"`
}

func (s *Service) continuationDirectory(project, review string) string {
	return filepath.Join(s.store.dir, "continuations", project, review)
}
func (s *Service) continuationBundle(rec continuationRecord) string {
	return filepath.Join(s.continuationDirectory(rec.Value.ProjectID, rec.Value.RequestID), "bundle")
}
func (s *Service) saveContinuation(rec continuationRecord) error {
	rec.Checksum = ""
	rec.Checksum = requestDigest(rec)
	return writeInspectionJSON(filepath.Join(s.continuationDirectory(rec.Value.ProjectID, rec.Value.RequestID), "record.json"), rec)
}
func (s *Service) continuationLaunch(v continuationView) (launchRecord, error) {
	rec, err := s.readLaunch(v.ProjectID, v.LaunchReviewID)
	if err != nil {
		return rec, err
	}
	if rec.Value.Operation == nil || rec.Value.Operation.RunRef == "" || rec.Value.Operation.RunRef != v.RunRef || rec.Value.PipelineID != v.PipelineID {
		return rec, problem("request_conflict", "This continuation belongs to a different Run.")
	}
	p, err := s.store.project(v.ProjectID)
	if err != nil {
		return rec, err
	}
	if err = launchTargetIdentity(p, rec); err != nil {
		return rec, err
	}
	return rec, nil
}
func (s *Service) readContinuation(project, review string) (continuationRecord, error) {
	var rec continuationRecord
	if !validID(project, "prj") || !validID(review, "req") {
		return rec, problem("invalid_request", "An exact continuation review is required.")
	}
	if _, err := s.store.project(project); err != nil {
		return rec, err
	}
	if err := readProposalFile(filepath.Join(s.continuationDirectory(project, review), "record.json"), &rec); err != nil {
		return rec, problem("not_found", "This continuation review is unavailable.")
	}
	hash := rec.Checksum
	rec.Checksum = ""
	if rec.SchemaVersion != 1 || requestDigest(rec) != hash || rec.Value.ProjectID != project || rec.Value.RequestID != review {
		return rec, problem("internal", "The continuation record is inconsistent. Preserve the profile.")
	}
	rec.Checksum = hash
	original, err := s.continuationLaunch(rec.Value)
	if err != nil {
		return rec, err
	}
	if rec.Value.Review != nil {
		if err := validateContinuationReview(*rec.Value.Review, original); err != nil {
			return rec, err
		}
	}
	if op := rec.Value.Operation; op != nil {
		if rec.Intent == nil || rec.Value.Review == nil || !validID(op.RequestID, "req") || *rec.Intent != continuationIntentFor(rec.Value, op.RequestID) {
			return rec, problem("internal", "The saved continuation intent is inconsistent.")
		}
		if op.Receipt != nil {
			if err := validateContinuationReceipt(*op.Receipt, *rec.Intent); err != nil {
				return rec, err
			}
		}
	} else if rec.Intent != nil {
		return rec, problem("internal", "Continuation intent has no operation.")
	}
	return rec, nil
}
func continuationIntentFor(v continuationView, request string) preparation.ContinuationIntent {
	return preparation.ContinuationIntent{SchemaVersion: 1, RequestID: request, WorkspaceID: v.Review.WorkspaceID, OriginDigest: v.Review.OriginDigest, ReviewDigest: v.Review.Digest, PreviousHead: v.Review.PreviousHead}
}
func (s *Service) continuationRecords(project string) ([]continuationRecord, error) {
	if _, err := s.store.project(project); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.continuationDirectory(project, ""))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	records := []continuationRecord{}
	for _, entry := range entries {
		if entry.IsDir() {
			rec, e := s.readContinuation(project, entry.Name())
			if e != nil {
				return nil, e
			}
			records = append(records, rec)
		}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Value.CreatedAt < records[j].Value.CreatedAt })
	return records, nil
}
func (s *Service) continuationViewLocked(rec continuationRecord) continuationView {
	v := rec.Value
	if v.State == "checking" && s.continuationFlights[v.ProjectID+"/"+v.RequestID] == nil {
		v.State = "interrupted"
		v.Issue = "The check was interrupted. Request a new review; no analysis was started."
	}
	return v
}
func (s *Service) continuations(project string) ([]continuationView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.continuationRecords(project)
	if err != nil {
		return nil, err
	}
	out := []continuationView{}
	for _, rec := range records {
		out = append(out, s.continuationViewLocked(rec))
	}
	return out, nil
}
func (s *Service) continuation(project, review string) (continuationView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, err := s.readContinuation(project, review)
	return s.continuationViewLocked(rec), err
}
