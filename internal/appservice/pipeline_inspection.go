package appservice

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const maxFlowBytes = 768 << 10
const inspectionTimeout = 120 * time.Second

type pipelineArtifact struct {
	ArtifactID     string          `json:"artifactId"`
	SourceRevision string          `json:"sourceRevision"`
	CheckedAt      string          `json:"checkedAt"`
	Flow           json.RawMessage `json:"flow"`
}
type pipelineInspection struct {
	Managed    bool              `json:"managed,omitempty"`
	ProjectID  string            `json:"projectId"`
	PipelineID string            `json:"pipelineId"`
	State      string            `json:"state"`
	JobID      string            `json:"jobId,omitempty"`
	Issue      string            `json:"issue,omitempty"`
	Artifact   *pipelineArtifact `json:"artifact,omitempty"`
}
type pipelineFlight struct {
	id     string
	cancel context.CancelFunc
}

// Inspection records have a separate owner and lifetime from the Project catalog
// and Workspace presentation. A single state pointer adopts only completed facts;
// every candidate, manifest and successful artifact is retained independently.
func (s *Service) inspectionDirectory(pipelineID string) string {
	return filepath.Join(s.store.dir, "pipeline-inspections", pipelineID)
}

func (s *Service) readInspectionLocked(projectID, pipelineID string) (pipelineInspection, error) {
	value := pipelineInspection{ProjectID: projectID, PipelineID: pipelineID, State: "idle"}
	s.store.mu.Lock()
	managed, ok := s.store.data.Revisions[pipelineID]
	s.store.mu.Unlock()
	if ok && managed.PipelineID == pipelineID {
		value.State = "ready"
		value.Managed = true
		value.Artifact = &managed.Artifact
		return value, nil
	}
	raw, err := readBoundedFile(filepath.Join(s.inspectionDirectory(pipelineID), "state.json"), maxFlowBytes+(64<<10))
	if errors.Is(err, os.ErrNotExist) {
		return value, nil
	}
	if err != nil || validateJSON(raw) != nil || json.Unmarshal(raw, &value) != nil || value.ProjectID != projectID || value.PipelineID != pipelineID {
		return value, problem("internal", "The saved flow is unavailable. Preserve the analysis profile and restore its inspection record.")
	}
	if value.State == "checking" && s.inspections[pipelineID] == nil {
		value.State, value.Issue = "cancelled", "The previous check was interrupted. Your earlier flow is still available."
	}
	return value, nil
}

func writeInspectionJSON(path string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return atomicWrite(filepath.Dir(path), filepath.Base(path), data)
}

func (s *Service) pipelineInspection(projectID, pipelineID string) (pipelineInspection, error) {
	if _, _, err := s.store.pipelineProject(projectID, pipelineID); err != nil {
		return pipelineInspection{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readInspectionLocked(projectID, pipelineID)
}

func (s *Service) checkPipeline(projectID, pipelineID, requestID string) (pipelineInspection, error) {
	if !validID(requestID, "req") {
		return pipelineInspection{}, problem("invalid_request", "A check identity is required.")
	}
	project, pipeline, err := s.store.pipelineProject(projectID, pipelineID)
	if err != nil {
		return pipelineInspection{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing {
		return pipelineInspection{}, problem("runtime_unavailable", "The analysis service is closing.")
	}
	value, err := s.readInspectionLocked(projectID, pipelineID)
	if err != nil {
		return value, err
	}
	s.store.mu.Lock()
	revision, hasRevision := s.store.data.Revisions[pipelineID]
	managed := hasRevision && revision.PipelineID == pipelineID
	s.store.mu.Unlock()
	if managed {
		return value, problem("unsupported", "This Pipeline uses an app-managed current version. Discuss changes to request a new checked proposal.")
	}
	if value.JobID == requestID {
		return value, nil
	}
	if s.inspections[pipelineID] != nil {
		return value, problem("request_conflict", "A check is already in progress for this pipeline.")
	}
	if s.analysisFlightCountLocked() >= 2 || s.proposals[pipelineID] != nil {
		return value, problem("runtime_unavailable", "Two analysis checks are already in progress. Try again shortly.")
	}
	dir := filepath.Join(s.inspectionDirectory(pipelineID), "candidates")
	entries, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return value, err
	}
	if len(entries) >= 32 {
		return value, problem("unsupported", "This pipeline has reached the retained inspection limit (32). Preserve its history before managing storage.")
	}
	// A completed request can never be replayed as a new evaluation.
	if _, err := os.Stat(filepath.Join(dir, requestID)); !errors.Is(err, os.ErrNotExist) {
		return value, problem("request_conflict", "This check identity was already used. Start a new check.")
	}
	if err := os.MkdirAll(filepath.Join(dir, requestID), 0o700); err != nil {
		return value, err
	}
	value.State, value.JobID, value.Issue = "checking", requestID, ""
	if err := writeInspectionJSON(filepath.Join(s.inspectionDirectory(pipelineID), "state.json"), value); err != nil {
		return value, err
	}
	ctx, cancel := context.WithTimeout(s.ctx, inspectionTimeout)
	if s.inspections == nil {
		s.inspections = map[string]*pipelineFlight{}
	}
	s.inspections[pipelineID] = &pipelineFlight{requestID, cancel}
	s.workers.Add(1)
	go s.inspectPipeline(ctx, project, pipeline, value)
	return value, nil
}

func (s *Service) cancelPipeline(projectID, pipelineID, jobID string) (pipelineInspection, error) {
	if _, _, err := s.store.pipelineProject(projectID, pipelineID); err != nil {
		return pipelineInspection{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.readInspectionLocked(projectID, pipelineID)
	if err != nil {
		return value, err
	}
	if value.JobID != jobID {
		return value, problem("stale_revision", "This check has already been replaced.")
	}
	if flight := s.inspections[pipelineID]; flight != nil && flight.id == jobID {
		value.State, value.Issue = "cancelled", "Check cancelled. Your earlier flow is still available."
		if err := writeInspectionJSON(filepath.Join(s.inspectionDirectory(pipelineID), "candidates", jobID, "result.json"), value); err != nil {
			return value, err
		}
		if err := writeInspectionJSON(filepath.Join(s.inspectionDirectory(pipelineID), "state.json"), value); err != nil {
			return value, err
		}
		flight.cancel()
	}
	return value, nil
}

func (s *Service) inspectPipeline(ctx context.Context, project projectRecord, pipeline PipelineDefinition, value pipelineInspection) {
	defer s.workers.Done()
	dir := filepath.Join(s.inspectionDirectory(pipeline.PipelineID), "candidates", value.JobID)
	artifact, err := s.evaluatePipeline(ctx, project, pipeline, dir)
	s.mu.Lock()
	defer s.mu.Unlock()
	flight := s.inspections[pipeline.PipelineID]
	if flight == nil || flight.id != value.JobID {
		return
	}
	flight.cancel()
	current, readErr := s.readInspectionLocked(project.ProjectID, pipeline.PipelineID)
	delete(s.inspections, pipeline.PipelineID)
	if readErr != nil || current.JobID != value.JobID || current.State == "cancelled" {
		return
	}
	if err != nil {
		value.State, value.Issue = "failed", inspectionIssue(err)
	} else {
		value.State, value.Artifact = "ready", artifact
	}
	// On publication failure, the previous state and artifact remain authoritative.
	if err := writeInspectionJSON(filepath.Join(dir, "result.json"), value); err != nil {
		s.store.mu.Lock()
		s.store.failed = true
		s.store.mu.Unlock()
		return
	}
	if err := writeInspectionJSON(filepath.Join(s.inspectionDirectory(pipeline.PipelineID), "state.json"), value); err != nil {
		s.store.mu.Lock()
		s.store.failed = true
		s.store.mu.Unlock()
	}
}

func inspectionIssue(err error) string {
	var failure *apiError
	if errors.As(err, &failure) {
		return failure.Message
	}
	return "The analysis could not be checked. Ask the Agent to check its setup and try again."
}

func (s *Service) evaluatePipeline(ctx context.Context, project projectRecord, pipeline PipelineDefinition, dir string) (*pipelineArtifact, error) {
	binding, err := s.runtime.selectBinding(ctx, project, "inspection")
	if err != nil {
		return nil, problem("runtime_unavailable", "The analysis runtime is not ready. Ask the Agent to check the Project setup, then try again.")
	}
	retained := filepath.Join(dir, "project")
	manifest, err := retainInspection(project, pipeline, binding, retained)
	if err != nil {
		return nil, err
	}
	if err := writeInspectionJSON(filepath.Join(dir, "manifest.json"), manifest); err != nil {
		return nil, err
	}
	raw, err := s.runtime.pipelineFlow(ctx, retained, manifest)
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, problem("runtime_unavailable", "The analysis check was interrupted or exceeded two minutes.")
	}
	if err := s.runtime.verify(ctx, binding); err != nil {
		return nil, err
	}
	current, err := retainInspection(project, pipeline, binding, "")
	if err != nil || requestDigest(current) != requestDigest(manifest) {
		return nil, problem("stale_revision", "Analysis inputs changed while checking. Your earlier flow is preserved; check the updated analysis again.")
	}
	artifact := &pipelineArtifact{requestDigest([]any{manifest, json.RawMessage(raw)}), requestDigest(manifest), time.Now().UTC().Format(time.RFC3339Nano), raw}
	if err := writeInspectionJSON(filepath.Join(dir, "artifact.json"), artifact); err != nil {
		return nil, err
	}
	return artifact, nil
}
