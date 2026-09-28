package appservice

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/HahyeonJeon/gobble/internal/preparation"
)

type launchReviewInput struct {
	RequestID     string `json:"requestId"`
	PreparationID string `json:"preparationId"`
}
type launchReview struct {
	FollowUp      *runFollowUp        `json:"followUp,omitempty"`
	ProjectID     string              `json:"projectId"`
	PipelineID    string              `json:"pipelineId"`
	RequestID     string              `json:"requestId"`
	PreparationID string              `json:"preparationId"`
	CreatedAt     string              `json:"createdAt"`
	State         string              `json:"state"`
	Issue         string              `json:"issue"`
	CopiedBytes   int64               `json:"copiedBytes"`
	TotalBytes    int64               `json:"totalBytes"`
	ReadCount     int64               `json:"readCount"`
	OutputPath    string              `json:"outputPath"`
	InputSHA256   string              `json:"inputSHA256"`
	Fresh         bool                `json:"fresh"`
	Preparation   pipelinePreparation `json:"preparation"`
	Operation     *launchOperation    `json:"operation,omitempty"`
}

// Private service record; never returned wholesale over HTTP.
type launchRecord struct {
	SchemaVersion  int                      `json:"schemaVersion"`
	Checksum       string                   `json:"checksum"`
	Value          launchReview             `json:"value"`
	Runtime        RuntimeBinding           `json:"runtime"`
	TargetIdentity string                   `json:"targetIdentity"`
	Intent         preparation.LaunchIntent `json:"intent"`
}

func (s *Service) launchDirectory(project, request string) string {
	return filepath.Join(s.store.dir, "launch-reviews", project, request)
}
func (s *Service) saveLaunch(rec launchRecord) error {
	rec.Checksum = ""
	rec.Checksum = requestDigest(rec)
	return writeInspectionJSON(filepath.Join(s.launchDirectory(rec.Value.ProjectID, rec.Value.RequestID), "record.json"), rec)
}
func (s *Service) readLaunch(project, request string) (launchRecord, error) {
	var rec launchRecord
	if !validID(project, "prj") || !validID(request, "req") {
		return rec, problem("invalid_request", "An exact launch review is required.")
	}
	if _, err := s.store.project(project); err != nil {
		return rec, err
	}
	if err := readProposalFile(filepath.Join(s.launchDirectory(project, request), "record.json"), &rec); err != nil {
		return rec, problem("not_found", "This launch review is unavailable.")
	}
	hash := rec.Checksum
	rec.Checksum = ""
	if rec.SchemaVersion != 1 || requestDigest(rec) != hash || rec.Value.ProjectID != project || rec.Value.RequestID != request || !validID(rec.Value.PipelineID, "pip") || !validBinding(rec.Runtime) {
		return rec, problem("internal", "The saved launch review is inconsistent. Preserve the profile.")
	}
	rec.Checksum = hash
	return rec, nil
}
func (s *Service) launchReviews(project string) ([]launchReview, error) {
	if _, err := s.store.project(project); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.launchDirectory(project, ""))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	out := []launchReview{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		v, err := s.launchView(project, e.Name())
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return out, nil
}
func (s *Service) launchView(project, request string) (launchReview, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, err := s.readLaunch(project, request)
	if err != nil {
		return rec.Value, err
	}
	v := rec.Value
	v.Fresh = false
	if v.State == "checking" {
		if f := s.launchChecks[project]; f == nil || f.id != request {
			v.State = "cancelled"
			v.Issue = "The check was interrupted. Check this review again."
		}
	}
	if v.State == "ready" {
		v.Fresh = s.launchFreshLocked(rec) == nil
		if !v.Fresh {
			v.Issue = "Current, source data, or the result location changed. Check the run again."
		}
	}
	return v, nil
}
func (s *Service) launchFreshLocked(rec launchRecord) error {
	v, err := s.readPreparationLocked(rec.Value.ProjectID, rec.Value.PipelineID, rec.Value.PreparationID)
	if err != nil || v.State != "ready" || !v.Fresh || v.Prepared.Digest != rec.Value.Preparation.Prepared.Digest {
		return problem("stale_revision", "Review the current analysis and data again.")
	}
	p, err := s.store.project(rec.Value.ProjectID)
	if err != nil {
		return err
	}
	return launchTargetIdentity(p, rec)
}
func launchTargetIdentity(p projectRecord, rec launchRecord) error {
	root, err := openProject(p)
	if err != nil {
		return err
	}
	defer root.Close()
	info, err := root.Stat(rec.Value.OutputPath)
	if err != nil || rootIdentity(info) != rec.TargetIdentity {
		return problem("stale_revision", "The result location changed.")
	}
	f, err := openResource(root, rec.Value.OutputPath+"/.gobble-launch-target")
	if err != nil {
		return err
	}
	defer f.Close()
	b := make([]byte, 128)
	n, _ := f.Read(b)
	if string(b[:n]) != rec.Value.RequestID {
		return problem("stale_revision", "The reserved result location changed.")
	}
	return nil
}
func (s *Service) checkLaunch(project, pipeline string, in launchReviewInput) (launchReview, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !validID(in.RequestID, "req") || !validID(in.PreparationID, "req") {
		return launchReview{}, problem("invalid_request", "Select the exact preparation to check.")
	}
	if old, err := s.readLaunch(project, in.RequestID); err == nil {
		if old.Value.PreparationID != in.PreparationID || old.Value.PipelineID != pipeline {
			return launchReview{}, problem("request_conflict", "This request belongs to another launch review.")
		}
		return old.Value, nil
	}
	if _, err := os.Stat(s.launchDirectory(project, in.RequestID)); err == nil {
		return launchReview{}, problem("internal", "An incomplete launch record exists. Preserve the profile.")
	}
	if err := s.unconfirmedLaunchLocked(project, pipeline, ""); err != nil {
		return launchReview{}, err
	}
	if s.closing || s.launchChecks[project] != nil || len(s.launchChecks) >= 2 {
		return launchReview{}, problem("runtime_unavailable", "A data check is in progress. Try again shortly.")
	}
	old, _ := os.ReadDir(s.launchDirectory(project, ""))
	if len(old) >= 32 {
		return launchReview{}, problem("unsupported", "The retained launch-review limit (32) was reached.")
	}
	prep, err := s.readPreparationLocked(project, pipeline, in.PreparationID)
	if err != nil {
		return launchReview{}, err
	}
	if prep.State != "ready" || !prep.Fresh {
		return launchReview{}, problem("stale_revision", "Prepare the current analysis before checking a Run.")
	}
	if prep.Prepared.Input.Size <= 0 || prep.Prepared.Input.Size > maxLaunchInput {
		return launchReview{}, problem("unsupported", "Choose a nonempty read file no larger than 64 GiB.")
	}
	var saved preparationRecord
	if err := readProposalFile(filepath.Join(s.preparationDirectory(pipeline, in.PreparationID), "record.json"), &saved); err != nil {
		return launchReview{}, err
	}
	p, err := s.store.project(project)
	if err != nil {
		return launchReview{}, err
	}
	followUp, err := s.currentFollowUpLocked(project, pipeline, prep.ArtifactID)
	if err != nil {
		return launchReview{}, err
	}
	v := launchReview{FollowUp: followUp, ProjectID: project, PipelineID: pipeline, RequestID: in.RequestID, PreparationID: in.PreparationID, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), State: "checking", TotalBytes: prep.Prepared.Input.Size, OutputPath: "runs/analysis-" + strings.TrimPrefix(in.RequestID, "req_"), Preparation: prep}
	rec := launchRecord{SchemaVersion: 1, Value: v, Runtime: saved.Runtime}
	rec.Runtime.WorkspacePath = workspaceContainerPath(v.OutputPath)
	rec.Intent = preparation.LaunchIntent{SchemaVersion: 1, PreparedDigest: prep.Prepared.Digest, Binding: prep.Prepared.Binding, WorkspaceID: in.RequestID, EngineImage: saved.Runtime.ImageID, DaemonID: saved.Runtime.DaemonID, Tools: map[string]string{}}
	if err := os.MkdirAll(s.launchDirectory(project, in.RequestID), 0700); err != nil {
		return v, err
	}
	if err := s.saveLaunch(rec); err != nil {
		return v, err
	}
	ctx, cancel := context.WithCancel(s.ctx)
	if s.launchChecks == nil {
		s.launchChecks = map[string]*pipelineFlight{}
	}
	s.launchChecks[project] = &pipelineFlight{in.RequestID, cancel}
	s.workers.Add(1)
	go s.checkLaunchWorker(ctx, rec, p)
	return v, nil
}
func (s *Service) cancelLaunch(project, request string) (launchReview, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, err := s.readLaunch(project, request)
	if err != nil {
		return rec.Value, err
	}
	if rec.Value.State == "checking" {
		if f := s.launchChecks[project]; f != nil && f.id == request {
			f.cancel()
		}
	}
	if rec.Value.State == "ready" {
		rec.Value.State = "cancelled"
		rec.Value.Issue = "Run check cancelled. No analysis started."
		err = s.saveLaunch(rec)
	}
	return rec.Value, err
}
func (s *Service) checkLaunchWorker(ctx context.Context, rec launchRecord, p projectRecord) {
	defer s.workers.Done()
	err := s.stageLaunch(ctx, &rec, p)
	s.mu.Lock()
	defer s.mu.Unlock()
	defer func() {
		if f := s.launchChecks[p.ProjectID]; f != nil && f.id == rec.Value.RequestID {
			f.cancel()
			delete(s.launchChecks, p.ProjectID)
		}
	}()
	if ctx.Err() != nil {
		rec.Value.State = "cancelled"
		rec.Value.Issue = "Data check cancelled or interrupted. No analysis started."
	} else if err != nil {
		rec.Value.State = "failed"
		rec.Value.Issue = publicPreparationIssue(err)
	} else if err = s.launchFreshLocked(rec); err != nil {
		rec.Value.State = "failed"
		rec.Value.Issue = "The analysis or data changed while checking. Check again."
	} else {
		rec.Value.State = "ready"
		rec.Value.Fresh = true
	}
	if err := s.saveLaunch(rec); err != nil {
		s.store.mu.Lock()
		s.store.failed = true
		s.store.mu.Unlock()
	}
}

// Ambiguous dispatch must be reconciled before the same Pipeline can receive
// another launch intent, including through a newly prepared review.
func (s *Service) unconfirmedLaunchLocked(project, pipeline, except string) error {
	if _, err := s.store.project(project); err != nil {
		return err
	}
	entries, err := os.ReadDir(s.launchDirectory(project, ""))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() || e.Name() == except {
			continue
		}
		rec, err := s.readLaunch(project, e.Name())
		if err != nil {
			return err
		}
		op := rec.Value.Operation
		if rec.Value.PipelineID == pipeline && op != nil && op.State != "admitted" && op.State != "rejected" {
			return problem("request_conflict", "Reconnect to the earlier Start before starting another Run of this Pipeline.")
		}
	}
	return nil
}
