package appservice

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"
)

type launchOperation struct {
	RequestID     string `json:"requestId"`
	State         string `json:"state"`
	RunRef        string `json:"runRef"`
	RunStatus     string `json:"runStatus"`
	Snapshot      string `json:"snapshot"`
	OwnerLease    string `json:"ownerLease"`
	ObservedAt    string `json:"observedAt"`
	Issue         string `json:"issue"`
	StopRequestID string `json:"stopRequestId"`
	StopState     string `json:"stopState"`
	StopLease     string `json:"stopLease"`
}
type launchStartInput struct {
	RequestID string `json:"requestId"`
	ReviewID  string `json:"reviewId"`
}
type launchStopInput struct {
	RequestID     string `json:"requestId"`
	ExpectedLease string `json:"expectedLease"`
}

func (s *Service) startLaunch(project string, in launchStartInput) (launchReview, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !validID(in.RequestID, "req") {
		return launchReview{}, problem("invalid_request", "A launch request identity is required.")
	}
	rec, err := s.readLaunch(project, in.ReviewID)
	if err != nil {
		return rec.Value, err
	}
	if op := rec.Value.Operation; op != nil {
		if op.RequestID != in.RequestID {
			return rec.Value, problem("request_conflict", "This review already belongs to another Start request.")
		}
		return rec.Value, nil
	}
	if err = s.unconfirmedLaunchLocked(project, rec.Value.PipelineID, in.ReviewID); err != nil {
		return rec.Value, err
	}
	if s.closing || rec.Value.State != "ready" {
		return rec.Value, problem("stale_revision", "A ready launch review is required.")
	}
	if err = s.launchFreshLocked(rec); err != nil {
		return rec.Value, err
	}
	rec.Intent.RequestID = in.RequestID
	rec.Value.State = "accepted"
	rec.Value.Fresh = false
	rec.Value.Operation = &launchOperation{RequestID: in.RequestID, State: "accepted"}
	if err = writeLaunchIntent(s.launchDirectory(project, in.ReviewID), rec); err != nil {
		return rec.Value, err
	}
	if err = s.saveLaunch(rec); err != nil {
		return rec.Value, err
	}
	s.launchWorkerLocked(rec)
	return rec.Value, nil
}
func (s *Service) launchWorkerLocked(rec launchRecord) {
	if s.closing {
		return
	}
	if s.launchFlights == nil {
		s.launchFlights = map[string]*pipelineFlight{}
	}
	key := rec.Value.ProjectID + "/" + rec.Value.RequestID
	if s.launchFlights[key] != nil {
		return
	}
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	s.launchFlights[key] = &pipelineFlight{rec.Value.RequestID, cancel}
	s.workers.Add(1)
	go func() {
		defer s.workers.Done()
		defer func() { s.mu.Lock(); cancel(); delete(s.launchFlights, key); s.mu.Unlock() }()
		s.reconcileLaunchWorker(ctx, rec)
	}()
}

// Refresh reconciles the already accepted intent; it never manufactures a new Start.
func (s *Service) refreshLaunch(project, review string) (launchReview, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, err := s.readLaunch(project, review)
	if err != nil {
		return rec.Value, err
	}
	if rec.Value.Operation != nil && rec.Value.Operation.State != "rejected" {
		s.launchWorkerLocked(rec)
	}
	return rec.Value, nil
}
func (s *Service) stopLaunch(project, review string, in launchStopInput) (launchReview, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, err := s.readLaunch(project, review)
	if err != nil {
		return rec.Value, err
	}
	op := rec.Value.Operation
	if !validID(in.RequestID, "req") || op == nil || op.OwnerLease == "" || in.ExpectedLease != op.OwnerLease {
		return rec.Value, problem("stale_revision", "Refresh this Run before requesting Stop.")
	}
	if op.StopRequestID != "" {
		if op.StopRequestID != in.RequestID {
			return rec.Value, problem("request_conflict", "A Stop request is already recorded for this Run.")
		}
		return rec.Value, nil
	}
	op.StopRequestID = in.RequestID
	op.StopLease = in.ExpectedLease
	op.StopState = "requested"
	if err = s.saveLaunch(rec); err != nil {
		return rec.Value, err
	}
	s.launchWorkerLocked(rec)
	return rec.Value, nil
}
func (s *Service) persistLaunchProgress(rec launchRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Stop can be accepted while a status query is in flight. Preserve its intent.
	if saved, err := s.readLaunch(rec.Value.ProjectID, rec.Value.RequestID); err == nil && saved.Value.Operation != nil && rec.Value.Operation != nil {
		if rec.Value.Operation.StopRequestID == "" {
			rec.Value.Operation.StopRequestID = saved.Value.Operation.StopRequestID
			rec.Value.Operation.StopState = saved.Value.Operation.StopState
			rec.Value.Operation.StopLease = saved.Value.Operation.StopLease
		}
	}
	return s.saveLaunch(rec)
}
func (s *Service) reconcileLaunchWorker(ctx context.Context, rec launchRecord) {
	op := rec.Value.Operation
	p, err := s.store.project(rec.Value.ProjectID)
	if err != nil {
		return
	}
	if op.State == "accepted" {
		// Current and original source were checked when the User confirmed.
		// Later edits cannot substitute or invalidate this accepted data copy.
		err = verifyLaunchTarget(ctx, p, rec)
		if err == nil {
			err = s.runtime.verify(ctx, rec.Runtime)
		}
		if err != nil {
			op.State = "rejected"
			op.Issue = launchError(err)
			_ = s.persistLaunchProgress(rec)
			return
		}
		op.State = "dispatching"
		if s.persistLaunchProgress(rec) != nil {
			return
		}
		if err = s.createLaunchController(ctx, p, rec); err != nil {
			op.State = "reconciling"
			op.Issue = "Start acknowledgement is unavailable. Check the same Run status; do not start another."
			_ = s.persistLaunchProgress(rec)
			return
		}
	}
	// An admitted checkpoint remains authoritative even after Docker removes the
	// controller. Only an unconfirmed admission needs created-container dispatch.
	state, err := s.queryLaunchState(ctx, p, rec)
	if err != nil && op.OwnerLease == "" {
		if e := s.startOrObserveController(ctx, p, rec); e == nil {
			state, err = s.queryLaunchState(ctx, p, rec)
		}
	}
	if err != nil {
		op.State = "reconciling"
		op.Issue = "The execution observation is unavailable. Check the same Run; do not start another."
		_ = s.persistLaunchProgress(rec)
		return
	}
	op.State = "admitted"
	op.RunStatus = state.Status
	op.Snapshot = state.Snapshot
	op.OwnerLease = state.Admission.Lease
	op.ObservedAt = time.Now().UTC().Format(time.RFC3339Nano)
	op.Issue = ""
	if state.Status == "interrupted" || !state.OwnerLive && (state.Status == "running" || state.Status == "stopping") {
		op.State = "recovery-required"
		op.Issue = "The execution owner is unavailable. Preserve this Run; automatic Resume is not enabled."
	}
	if op.RunRef == "" {
		op.RunRef, err = s.registerLaunchedRun(ctx, p, rec)
		if err != nil {
			op.Issue = "The admitted Run could not yet be registered. Check status to reconnect."
		}
	}
	// Reload a Stop accepted concurrently with admission observation.
	s.mu.Lock()
	latest, e := s.readLaunch(p.ProjectID, rec.Value.RequestID)
	s.mu.Unlock()
	if e == nil && latest.Value.Operation != nil && latest.Value.Operation.StopRequestID != "" {
		op.StopRequestID = latest.Value.Operation.StopRequestID
		op.StopState = latest.Value.Operation.StopState
		op.StopLease = latest.Value.Operation.StopLease
	}
	if op.StopRequestID != "" && op.StopState != "settled" && op.StopState != "owner-changed" && op.StopState != "recovery-required" {
		op.StopState, err = s.sendLaunchStop(ctx, p, rec)
		if err != nil {
			op.StopState = "requested"
			op.Issue = "Stop was requested; settlement is not confirmed. Check status."
		}
	}
	_ = s.persistLaunchProgress(rec)
}
func (s *Service) registerLaunchedRun(ctx context.Context, p projectRecord, rec launchRecord) (string, error) {
	resource := resourceID(p.ProjectID, rec.Value.OutputPath)
	identity, err := s.runtime.inspect(ctx, p, rec.Runtime, "identity", "")
	if err != nil {
		return "", err
	}
	required, err := readIdentity(identity)
	if err != nil {
		return "", err
	}
	raw, err := s.runtime.inspect(ctx, p, rec.Runtime, "monitor", "")
	if err != nil {
		return "", err
	}
	header, err := readMonitor(raw)
	if err != nil {
		return "", err
	}
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	next := s.store.copyLocked()
	for _, r := range next.Runs {
		if r.ProjectID == p.ProjectID && r.WorkspaceResourceID == resource {
			if r.Binding != rec.Runtime || r.EngineRunID != header.Run.ID {
				return "", problem("request_conflict", "A different Run occupies the target.")
			}
			return r.RunRef, nil
		}
	}
	if len(next.Runs) >= 1000 {
		return "", problem("unsupported", "The Run registration limit was reached.")
	}
	for i := range next.Projects {
		if next.Projects[i].ProjectID == p.ProjectID {
			if len(next.Projects[i].Resources) >= 10000 {
				return "", problem("unsupported", "The Project resource limit was reached.")
			}
			next.Projects[i].Resources[resource] = rec.Value.OutputPath
		}
	}
	name := fmt.Sprintf("Analysis %d", len(next.Runs)+1)
	registration := RunRegistration{p.ProjectID, newID("run"), name, resource, header.Run.ID}
	next.Runs = append(next.Runs, runRecord{registration, rec.Runtime, json.RawMessage(required)})
	if err = s.store.commitLocked(next); err != nil {
		return "", err
	}
	return registration.RunRef, nil
}

func (s *Service) launchBundle(rec launchRecord) string {
	return filepath.Join(s.launchDirectory(rec.Value.ProjectID, rec.Value.RequestID), "bundle")
}
