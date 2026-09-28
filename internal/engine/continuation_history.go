package engine

import (
	"encoding/json"
	"errors"

	"github.com/HahyeonJeon/gobble/internal/preparation"
)

func continuationIntentDigest(in preparation.ContinuationIntent) string {
	raw, _ := json.Marshal(in)
	return preparedDigest(raw)
}

func continuationAdmissionDigest(in preparation.ContinuationAdmission) string {
	raw, _ := json.Marshal(in)
	return preparedDigest(raw)
}

// admissionEpoch validates the complete retained chain before exposing ownership.
// The initial Start receipt stays unchanged even when the current lease advances.
func admissionEpoch(run jsonRun) (preparation.ExecutionEpoch, bool) {
	var zero preparation.ExecutionEpoch
	a := run.Admission
	if a == nil || (a.SchemaVersion != 1 && a.SchemaVersion != 2) || !preparedDigestPattern.MatchString(a.IntentDigest) || !preparedDigestPattern.MatchString(a.PreparedDigest) || !launchRequestPattern.MatchString(a.RequestID) || !launchRequestPattern.MatchString(a.WorkspaceID) || !validCheckpointID(a.Lease) || run.Occupancy == nil {
		return zero, false
	}
	epoch := preparation.ExecutionEpoch{Head: a.IntentDigest, Lease: a.Lease}
	h := run.ExecutionHistory
	if a.SchemaVersion == 1 {
		return epoch, h == nil && run.Occupancy.Lease == a.Lease
	}
	if h == nil || h.SchemaVersion != 1 || h.Continuations == nil || len(h.Continuations) > preparation.MaxContinuations {
		return zero, false
	}
	requests := map[string]bool{a.RequestID: true}
	leases := map[string]bool{a.Lease: true}
	snapshots := make(map[string]bool, len(h.Continuations))
	for _, receipt := range h.Continuations {
		in := receipt.Intent
		if in.SchemaVersion != 1 || !launchRequestPattern.MatchString(in.RequestID) || requests[in.RequestID] || in.WorkspaceID != a.WorkspaceID || in.OriginDigest != a.IntentDigest || in.PreviousHead != epoch.Head || !preparedDigestPattern.MatchString(in.ReviewDigest) || receipt.IntentDigest != continuationIntentDigest(in) || !validCheckpointID(receipt.Lease) || leases[receipt.Lease] || !validCheckpointID(receipt.Snapshot) || snapshots[receipt.Snapshot] {
			return zero, false
		}
		requests[in.RequestID], leases[receipt.Lease], snapshots[receipt.Snapshot] = true, true, true
		epoch = preparation.ExecutionEpoch{Head: continuationAdmissionDigest(receipt), Lease: receipt.Lease}
	}
	return epoch, run.Occupancy.Lease == epoch.Lease
}

func admissionCheckpointFormat(run jsonRun) (int, error) {
	if run.Admission == nil {
		if run.ExecutionHistory != nil {
			return 0, errCheckpointSchema
		}
		return checkpointFormat, nil
	}
	if !validAdmission(run) {
		return 0, errCheckpointSchema
	}
	if run.ExecutionHistory != nil {
		return 3, nil
	}
	return 2, nil
}

// validateAdmissionTransition protects immutable history through ordinary status
// commits as well as a future continuation admission. It does not schedule work.
func validateAdmissionTransition(old, next jsonRun) error {
	if old.Admission == nil && next.Admission == nil {
		return nil
	}
	if old.Admission == nil || next.Admission == nil || *old.Admission != *next.Admission || old.ID != next.ID || old.Started != next.Started {
		return errors.New("initial Run admission cannot be replaced")
	}
	if old.Identity == nil || next.Identity == nil || *old.Identity != *next.Identity {
		return errors.New("admitted engine identity cannot be replaced")
	}
	if old.ExecutionHistory == nil {
		if next.ExecutionHistory != nil {
			return errors.New("existing admission cannot be converted")
		}
		return nil
	}
	if next.ExecutionHistory == nil {
		return errors.New("execution history cannot be removed")
	}
	before, after := old.ExecutionHistory.Continuations, next.ExecutionHistory.Continuations
	if len(after) < len(before) || len(after) > len(before)+1 {
		return errors.New("execution history must append one admission at a time")
	}
	for i := range before {
		if before[i] != after[i] {
			return errors.New("retained continuation cannot be replaced")
		}
	}
	if len(after) > len(before) {
		if old.Status != RunStopped || occupancyIsActive(old) {
			return errors.New("continuation requires a settled stopped Run")
		}
		if after[len(before)].Snapshot != next.Snapshot {
			return errors.New("continuation must name its admission snapshot")
		}
	}
	return nil
}
