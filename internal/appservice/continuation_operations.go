package appservice

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/HahyeonJeon/gobble/internal/preparation"
)

func (s *Service) checkContinuation(project string, in continuationCheckInput) (continuationView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !validID(in.RequestID, "req") || !validID(in.LaunchReviewID, "req") || !validID(in.RunRef, "run") {
		return continuationView{}, problem("invalid_request", "Select an exact Run to review.")
	}
	if old, err := s.readContinuation(project, in.RequestID); err == nil {
		if old.Value.LaunchReviewID != in.LaunchReviewID || old.Value.RunRef != in.RunRef {
			return old.Value, problem("request_conflict", "The request belongs to another Run.")
		}
		return s.continuationViewLocked(old), nil
	}
	if _, err := os.Stat(s.continuationDirectory(project, in.RequestID)); err == nil {
		return continuationView{}, problem("internal", "An incomplete continuation record exists. Preserve the profile.")
	}
	records, err := s.continuationRecords(project)
	if err != nil {
		return continuationView{}, err
	}
	if len(records) >= 64 {
		return continuationView{}, problem("unsupported", "The retained continuation-review limit (64) was reached.")
	}
	if s.closing || len(s.continuationFlights) >= 2 {
		return continuationView{}, problem("runtime_unavailable", "Another continuation operation is in progress.")
	}
	original, err := s.readLaunch(project, in.LaunchReviewID)
	if err != nil {
		return continuationView{}, err
	}
	v := continuationView{ProjectID: project, PipelineID: original.Value.PipelineID, LaunchReviewID: in.LaunchReviewID, RunRef: in.RunRef, RequestID: in.RequestID, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), State: "checking"}
	if _, err = s.continuationLaunch(v); err != nil {
		return v, err
	}
	for _, old := range records {
		if old.Value.RunRef == in.RunRef && old.Value.Operation != nil && old.Value.Operation.State != "admitted" && old.Value.Operation.State != "rejected" {
			return v, problem("request_conflict", "Resolve the existing continuation acknowledgement first.")
		}
	}
	rec := continuationRecord{SchemaVersion: 1, Value: v}
	if err = s.saveContinuation(rec); err != nil {
		return v, err
	}
	s.continuationWorkerLocked(rec, func(ctx context.Context, owned continuationRecord) { s.checkContinuationWorker(ctx, owned, original) })
	return v, nil
}
func (s *Service) continuationWorkerLocked(rec continuationRecord, work func(context.Context, continuationRecord)) {
	if s.closing {
		return
	}
	if s.continuationFlights == nil {
		s.continuationFlights = map[string]*pipelineFlight{}
	}
	key := rec.Value.ProjectID + "/" + rec.Value.RequestID
	if s.continuationFlights[key] != nil {
		return
	}
	// The HTTP response and worker must not share mutable operation pointers.
	data, _ := json.Marshal(rec)
	var owned continuationRecord
	_ = json.Unmarshal(data, &owned)
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	s.continuationFlights[key] = &pipelineFlight{rec.Value.RequestID, cancel}
	s.workers.Add(1)
	go func() {
		defer s.workers.Done()
		defer func() { s.mu.Lock(); cancel(); delete(s.continuationFlights, key); s.mu.Unlock() }()
		work(ctx, owned)
	}()
}
func (s *Service) checkContinuationWorker(ctx context.Context, rec continuationRecord, original launchRecord) {
	p, err := s.store.project(rec.Value.ProjectID)
	if err == nil {
		if original.Intent.SchemaVersion != 2 {
			err = problem("unsupported", "This analysis uses an earlier engine.")
		} else {
			var ok bool
			ok, err = s.engineContinuationSupport(ctx, original.Runtime)
			if err == nil && !ok {
				err = problem("unsupported", "The original engine does not support continuation.")
			}
		}
	}
	if err == nil {
		err = s.writeContinuationBundle(rec, original)
	}
	if err == nil {
		var raw []byte
		raw, err = s.continuationQuery(ctx, p, original, rec, "prepared-continuation-review")
		if err == nil {
			var review preparation.ContinuationReview
			err = decodeCreationJSON(raw, &review)
			if err == nil {
				err = validateContinuationReview(review, original)
			}
			if err == nil {
				rec.Value.Review = &review
			}
		}
	}
	rec.Value.State = "ready"
	if err != nil {
		rec.Value.State = "blocked"
		rec.Value.Review = nil
		rec.Value.Issue = publicPreparationIssue(err)
	}
	if ctx.Err() != nil {
		rec.Value.State = "interrupted"
		rec.Value.Review = nil
		rec.Value.Issue = "The check was interrupted. Request a new review."
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.saveContinuation(rec)
}
func (s *Service) writeContinuationBundle(rec continuationRecord, original launchRecord) error {
	raw, err := os.ReadFile(filepath.Join(s.launchBundle(original), "payload.json"))
	if err != nil {
		return err
	}
	if len(raw) > 3<<20 || digest(raw) != original.Intent.PreparedDigest {
		return problem("internal", "The saved execution payload changed.")
	}
	dir := s.continuationBundle(rec)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if err = atomicWrite(dir, "payload.json", raw); err != nil {
		return err
	}
	in, _ := json.Marshal(original.Intent)
	return atomicWrite(dir, "intent.json", in)
}
func (s *Service) confirmContinuation(project, review string, in continuationConfirmInput) (continuationView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, err := s.readContinuation(project, review)
	if err != nil {
		return rec.Value, err
	}
	if !validID(in.RequestID, "req") || rec.Value.Review == nil || in.ReviewDigest != rec.Value.Review.Digest {
		return rec.Value, problem("stale_revision", "Confirm the exact saved continuation review.")
	}
	if op := rec.Value.Operation; op != nil {
		if op.RequestID != in.RequestID {
			return rec.Value, problem("request_conflict", "This review already has a continuation intent.")
		}
		return rec.Value, nil
	}
	if s.closing || rec.Value.State != "ready" || s.continuationFlights[project+"/"+review] != nil {
		return rec.Value, problem("stale_revision", "A completed continuation review is required.")
	}
	original, err := s.continuationLaunch(rec.Value)
	if err != nil {
		return rec.Value, err
	}
	if original.Intent.RequestID == in.RequestID {
		return rec.Value, problem("request_conflict", "Continuation cannot reuse the Start request.")
	}
	records, err := s.continuationRecords(project)
	if err != nil {
		return rec.Value, err
	}
	for _, other := range records {
		if op := other.Value.Operation; op != nil && (op.RequestID == in.RequestID || (other.Value.RunRef == rec.Value.RunRef && op.State != "admitted" && op.State != "rejected")) {
			return rec.Value, problem("request_conflict", "An earlier continuation must be resolved first.")
		}
	}
	intent := continuationIntentFor(rec.Value, in.RequestID)
	rec.Intent = &intent
	rec.Value.State = "accepted"
	rec.Value.Operation = &continuationOperation{RequestID: in.RequestID, State: "accepted"}
	// The engine bundle alone is not authority. Save the service operation before
	// spawning its single dispatch worker; a failed save cannot execute anything.
	data, _ := json.Marshal(intent)
	if err = atomicWrite(s.continuationBundle(rec), "continuation.json", data); err != nil {
		return rec.Value, err
	}
	if err = s.saveContinuation(rec); err != nil {
		return rec.Value, err
	}
	s.continuationWorkerLocked(rec, func(ctx context.Context, owned continuationRecord) { s.dispatchContinuation(ctx, owned, original) })
	return rec.Value, nil
}
func (s *Service) dispatchContinuation(ctx context.Context, rec continuationRecord, original launchRecord) {
	p, err := s.store.project(rec.Value.ProjectID)
	if err != nil {
		return
	}
	if err = launchTargetIdentity(p, original); err == nil {
		var ok bool
		ok, err = s.engineContinuationSupport(ctx, original.Runtime)
		if err == nil && !ok {
			err = problem("unsupported", "Continuation support is unavailable.")
		}
	}
	if err != nil {
		rec.Value.Operation.State = "rejected"
		rec.Value.Operation.Issue = publicPreparationIssue(err)
		_ = s.persistContinuationProgress(rec)
		return
	}
	rec.Value.Operation.State = "dispatching"
	if s.persistContinuationProgress(rec) != nil {
		return
	}
	spec := s.continuationController(rec)
	err = s.createExecutionController(ctx, p, original, spec)
	if err == nil {
		err = s.startExecutionController(ctx, p, original, spec)
	}
	// Even a failed create/start response may hide an admitted execution. Only
	// the exact receipt can confirm it. No later read/refresh repeats dispatch.
	s.observeContinuation(ctx, rec, original)
}
func (s *Service) persistContinuationProgress(rec continuationRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	saved, err := s.readContinuation(rec.Value.ProjectID, rec.Value.RequestID)
	if err != nil {
		return err
	}
	if rec.Value.Operation != nil && saved.Value.Operation != nil && rec.Value.Operation.StopRequestID == "" {
		rec.Value.Operation.StopRequestID = saved.Value.Operation.StopRequestID
		rec.Value.Operation.StopLease = saved.Value.Operation.StopLease
		rec.Value.Operation.StopState = saved.Value.Operation.StopState
	}
	return s.saveContinuation(rec)
}
func (s *Service) refreshContinuation(project, review string) (continuationView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, err := s.readContinuation(project, review)
	if err != nil {
		return rec.Value, err
	}
	if rec.Value.Operation == nil {
		return s.continuationViewLocked(rec), nil
	}
	original, err := s.continuationLaunch(rec.Value)
	if err != nil {
		return rec.Value, err
	}
	s.continuationWorkerLocked(rec, func(ctx context.Context, owned continuationRecord) { s.observeContinuation(ctx, owned, original) })
	return rec.Value, nil
}
func (s *Service) observeContinuation(ctx context.Context, rec continuationRecord, original launchRecord) {
	p, err := s.store.project(rec.Value.ProjectID)
	if err != nil {
		return
	}
	receipt, err := s.queryContinuationReceipt(ctx, p, original, rec)
	op := rec.Value.Operation
	if err == nil && receipt == nil && op.Receipt == nil {
		// Only a verified terminal controller followed by an exact absent
		// receipt proves rejection. Missing observation alone is uncertainty.
		status, inspectErr := s.executionControllerState(ctx, p, original, s.continuationController(rec))
		if inspectErr == nil && status == "exited" {
			receipt, err = s.queryContinuationReceipt(ctx, p, original, rec)
			if err == nil && receipt == nil {
				op.State = "rejected"
				op.Issue = "The engine did not admit this continuation. Check the saved Run again before confirming new work."
				_ = s.persistContinuationProgress(rec)
				return
			}
		}
	}
	if err != nil || receipt == nil {
		// Preserve a receipt already observed; temporary unavailability cannot erase
		// admitted authority or allow another confirmation to replace it.
		if op.Receipt == nil {
			op.State = "unknown"
		}
		op.Issue = "The continuation acknowledgement is unavailable. The analysis may already be running. Refresh this same request; it will not submit new work."
		_ = s.persistContinuationProgress(rec)
		return
	}
	op.State = "admitted"
	op.Receipt = receipt
	op.Issue = ""
	state, err := s.queryLaunchState(ctx, p, original)
	if err == nil {
		op.Observation = &state
	} else {
		op.Issue = "Continuation was admitted; current execution status is unavailable."
	}
	s.mu.Lock()
	saved, e := s.readContinuation(rec.Value.ProjectID, rec.Value.RequestID)
	s.mu.Unlock()
	if e == nil && saved.Value.Operation != nil && saved.Value.Operation.StopRequestID != "" {
		op.StopRequestID = saved.Value.Operation.StopRequestID
		op.StopLease = saved.Value.Operation.StopLease
		op.StopState = saved.Value.Operation.StopState
	}
	if op.StopState == "requested" {
		op.StopState, err = s.sendContinuationStop(ctx, p, original, rec)
		if err != nil {
			op.StopState = "requested"
			op.Issue = "Stop was requested; settlement is not confirmed."
		}
	}
	_ = s.persistContinuationProgress(rec)
}
func (s *Service) stopContinuation(project, review string, in launchStopInput) (continuationView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, err := s.readContinuation(project, review)
	if err != nil {
		return rec.Value, err
	}
	op := rec.Value.Operation
	if !validID(in.RequestID, "req") || op == nil || op.Receipt == nil || op.Observation == nil || op.Observation.Epoch == nil || in.ExpectedLease != op.Receipt.Lease || in.ExpectedLease != op.Observation.Epoch.Lease {
		return rec.Value, problem("stale_revision", "Refresh the current continuation before requesting Stop.")
	}
	if op.StopRequestID != "" {
		if op.StopRequestID != in.RequestID || op.StopLease != in.ExpectedLease {
			return rec.Value, problem("request_conflict", "This execution already has a Stop request.")
		}
		return rec.Value, nil
	}
	original, err := s.continuationLaunch(rec.Value)
	if err != nil {
		return rec.Value, err
	}
	op.StopRequestID = in.RequestID
	op.StopLease = in.ExpectedLease
	op.StopState = "requested"
	if err = s.saveContinuation(rec); err != nil {
		return rec.Value, err
	}
	s.continuationWorkerLocked(rec, func(ctx context.Context, owned continuationRecord) { s.observeContinuation(ctx, owned, original) })
	return rec.Value, nil
}
