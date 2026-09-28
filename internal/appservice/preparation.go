package appservice

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/HahyeonJeon/gobble/internal/preparation"
)

type prepareInput struct {
	EngineID   string `json:"engineId,omitempty"`
	RequestID  string `json:"requestId"`
	ArtifactID string `json:"artifactId"`
}
type preparationArtifact struct {
	Flow      json.RawMessage     `json:"flow"`
	Digest    string              `json:"digest"`
	Binding   preparation.Binding `json:"binding"`
	Input     creationInput       `json:"input"`
	Steps     []preparation.Step  `json:"steps"`
	CheckedAt string              `json:"checkedAt"`
}
type pipelinePreparation struct {
	EngineID   string               `json:"engineId"`
	CreatedAt  string               `json:"createdAt"`
	ProjectID  string               `json:"projectId"`
	PipelineID string               `json:"pipelineId"`
	RequestID  string               `json:"requestId"`
	ArtifactID string               `json:"artifactId"`
	State      string               `json:"state"`
	Issue      string               `json:"issue"`
	Prepared   *preparationArtifact `json:"prepared,omitempty"`
	// Derived on read; never an execution authorization.
	Fresh bool `json:"fresh"`
}
type preparationRecord struct {
	Checksum      string              `json:"checksum"`
	SchemaVersion int                 `json:"schemaVersion"`
	Value         pipelinePreparation `json:"value"`
	Runtime       RuntimeBinding      `json:"runtime"`
}

func (s *Service) preparationDirectory(pipelineID, requestID string) string {
	return filepath.Join(s.store.dir, "pipeline-preparations", pipelineID, requestID)
}
func (s *Service) readPreparationLocked(projectID, pipelineID, requestID string) (pipelinePreparation, error) {
	var rec preparationRecord
	if !validID(requestID, "req") {
		return rec.Value, problem("invalid_request", "An exact preparation identity is required.")
	}
	if _, _, err := s.store.pipelineProject(projectID, pipelineID); err != nil {
		return rec.Value, err
	}
	if err := readProposalFile(filepath.Join(s.preparationDirectory(pipelineID, requestID), "record.json"), &rec); err != nil {
		return rec.Value, problem("not_found", "This run review is unavailable.")
	}
	v := rec.Value
	if rec.SchemaVersion != 1 || rec.Checksum != requestDigest([]any{rec.Value, rec.Runtime}) || !validBinding(rec.Runtime) || rec.Runtime.ImageID != v.EngineID || v.ProjectID != projectID || v.PipelineID != pipelineID || v.RequestID != requestID {
		return v, problem("internal", "The saved preparation is inconsistent. Preserve the profile.")
	}
	if f := s.preparations[pipelineID]; v.State == "preparing" && (f == nil || f.id != requestID) {
		v.State = "cancelled"
		v.Issue = "Preparation was interrupted. Prepare again when you are ready."
	}
	v.Fresh = false
	if v.State == "ready" {
		if v.Prepared == nil {
			return v, problem("internal", "The saved run review is incomplete.")
		}
		raw, err := readBoundedFile(filepath.Join(s.preparationDirectory(pipelineID, requestID), "payload.json"), 3<<20)
		if err != nil || digest(raw) != v.Prepared.Digest {
			return v, problem("internal", "The private prepared plan failed its integrity check. Preserve the profile.")
		}
		_, _, _, err = s.preparationSourceLocked(projectID, pipelineID, v.ArtifactID)
		if err == nil {
			p, _, e := s.store.pipelineProject(projectID, pipelineID)
			observed, e2 := observeCreationInput(p, v.Prepared.Input.ResourceID)
			v.Fresh = e == nil && e2 == nil && observed == v.Prepared.Input && v.Prepared.Binding.RuntimeID == requestDigest(rec.Runtime)
		}
		if !v.Fresh {
			v.Issue = "Current, data or the retained source changed. This is an earlier run review; prepare again."
		}
	}
	return v, nil
}
func (s *Service) preparation(projectID, pipelineID, requestID string) (pipelinePreparation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readPreparationLocked(projectID, pipelineID, requestID)
}
func (s *Service) preparationsList(projectID, pipelineID string) ([]pipelinePreparation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, _, err := s.store.pipelineProject(projectID, pipelineID); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.preparationDirectory(pipelineID, ""))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	out := []pipelinePreparation{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(s.preparationDirectory(pipelineID, e.Name()), "record.json")); errors.Is(err, os.ErrNotExist) {
			continue
		}
		v, err := s.readPreparationLocked(projectID, pipelineID, e.Name())
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}
func (s *Service) preparePipeline(projectID, pipelineID string, in prepareInput) (pipelinePreparation, error) {
	v := pipelinePreparation{ProjectID: projectID, PipelineID: pipelineID, RequestID: in.RequestID, ArtifactID: in.ArtifactID, State: "preparing", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	if !validID(in.RequestID, "req") || !digestPattern.MatchString(in.ArtifactID) || (in.EngineID != "" && !digestPattern.MatchString(in.EngineID)) {
		return v, problem("invalid_request", "Select the exact Current flow to prepare.")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing {
		return v, problem("runtime_unavailable", "The service is closing.")
	}
	if _, _, err := s.store.pipelineProject(projectID, pipelineID); err != nil {
		return v, err
	}
	dir := s.preparationDirectory(pipelineID, in.RequestID)
	if _, err := os.Stat(dir); err == nil {
		old, e := s.readPreparationLocked(projectID, pipelineID, in.RequestID)
		if e != nil {
			return v, e
		}
		if old.ArtifactID != in.ArtifactID || (in.EngineID != "" && old.EngineID != in.EngineID) {
			return v, problem("request_conflict", "This preparation identity belongs to another Current version.")
		}
		return old, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return v, err
	}
	if s.preparations[pipelineID] != nil || s.analysisFlightCountLocked() >= 2 {
		return v, problem("runtime_unavailable", "An analysis check is in progress. Try again shortly.")
	}
	entries, err := os.ReadDir(s.preparationDirectory(pipelineID, ""))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return v, err
	}
	if len(entries) >= 32 {
		return v, problem("unsupported", "The retained run review limit (32) has been reached.")
	}
	base, input, binding, err := s.preparationSourceLocked(projectID, pipelineID, in.ArtifactID)
	if err != nil {
		return v, err
	}
	if in.EngineID != "" {
		ctx, cancel := context.WithTimeout(s.ctx, 20*time.Second)
		choices, e := s.analysisEngineBindings(ctx, "io.gobble.preparation.scope=single-end-trim-fastqc-v1")
		cancel()
		if e != nil {
			return v, e
		}
		found := false
		for _, candidate := range choices {
			if candidate.ImageID == in.EngineID {
				base.Manifest.Runtime = candidate
				found = true
				break
			}
		}
		if !found {
			return v, problem("incompatible_runtime", "Choose an installed engine that supports run preparation.")
		}
	}
	v.EngineID = base.Manifest.Runtime.ImageID
	binding.RuntimeID = requestDigest(base.Manifest.Runtime)
	if err = retainPreparationSource(base, binding, dir); err != nil {
		return v, err
	}
	if err = writeInspectionJSON(filepath.Join(dir, "record.json"), preparationRecord{requestDigest([]any{v, base.Manifest.Runtime}), 1, v, base.Manifest.Runtime}); err != nil {
		return v, err
	}
	ctx, cancel := context.WithTimeout(s.ctx, inspectionTimeout)
	if s.preparations == nil {
		s.preparations = map[string]*pipelineFlight{}
	}
	s.preparations[pipelineID] = &pipelineFlight{in.RequestID, cancel}
	s.workers.Add(1)
	go s.evaluatePreparation(ctx, v, base, input, binding)
	return v, nil
}
func (s *Service) cancelPreparation(projectID, pipelineID, requestID string) (pipelinePreparation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, err := s.readPreparationLocked(projectID, pipelineID, requestID)
	if err != nil {
		return v, err
	}
	if f := s.preparations[pipelineID]; f != nil && f.id == requestID {
		f.cancel()
	}
	return v, nil
}
func (s *Service) evaluatePreparation(ctx context.Context, v pipelinePreparation, base proposalBase, input creationInput, binding preparation.Binding) {
	defer s.workers.Done()
	dir := s.preparationDirectory(v.PipelineID, v.RequestID)
	result, err := s.runtime.preparedPipeline(ctx, filepath.Join(dir, "project"), base.Manifest, binding)
	s.mu.Lock()
	defer s.mu.Unlock()
	defer func() {
		if f := s.preparations[v.PipelineID]; f != nil && f.id == v.RequestID {
			f.cancel()
			delete(s.preparations, v.PipelineID)
		}
	}()
	if err == nil {
		var current proposalBase
		var observed creationInput
		var b preparation.Binding
		current, observed, b, err = s.preparationSourceLocked(v.ProjectID, v.PipelineID, v.ArtifactID)
		b.RuntimeID = requestDigest(base.Manifest.Runtime)
		if err == nil && (observed != input || b != binding || !sameReviewJSON(result.Review.Review.Flow, current.Artifact.Flow)) {
			err = problem("stale_revision", "Current or data changed during preparation. Review the flow and prepare again.")
		}
	}
	if ctx.Err() != nil {
		v.State = "cancelled"
		v.Issue = "Preparation was cancelled or interrupted. No analysis started."
	} else if err != nil {
		v.State = "failed"
		v.Issue = publicPreparationIssue(err)
	} else {
		if err = atomicWrite(dir, "payload.json", result.Payload); err == nil {
			v.State = "ready"
			v.Prepared = &preparationArtifact{result.Review.Review.Flow, result.Digest, binding, input, result.Steps, time.Now().UTC().Format(time.RFC3339Nano)}
		} else {
			v.State = "failed"
			v.Issue = "The private plan could not be saved. Preserve the profile and prepare again."
		}
	}
	// No ready pointer is published before the private payload is durable. A failed
	// record write remains preparing on disk and becomes interrupted on restart.
	if e := writeInspectionJSON(filepath.Join(dir, "record.json"), preparationRecord{requestDigest([]any{v, base.Manifest.Runtime}), 1, v, base.Manifest.Runtime}); e != nil {
		s.store.mu.Lock()
		s.store.failed = true
		s.store.mu.Unlock()
	}
}
func publicPreparationIssue(err error) string {
	var p *apiError
	if errors.As(err, &p) {
		return p.Message
	}
	return "The analysis could not be prepared. Confirm the pinned engine supports run preparation, then ask the Agent to check the Current design."
}
