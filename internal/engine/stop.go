package engine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	RunStopping         = "stopping"
	RunStopped          = "stopped"
	RunInterrupted      = "interrupted"
	controlPollInterval = 200 * time.Millisecond
)

// StopResult distinguishes accepting a request from proving task termination.
type StopResult struct {
	Status string `json:"status"`
	Lease  string `json:"lease,omitempty"`
}

type stopRequest struct {
	Lease     string `json:"lease"`
	Requested string `json:"requested"`
}

// Stop addresses a durable request to exactly one owner. It never signals a PID.
// A canceled wait leaves the request intact and returns status "requested".
func Stop(ctx context.Context, workspace string, supplied *InstallIdentity) (StopResult, []Defect) {
	if ctx == nil {
		ctx = context.Background()
	}
	if supplied != nil {
		if d := ValidateInstallIdentity(supplied); len(d) > 0 {
			return StopResult{}, d
		}
	}
	if d := CheckResumeStart(workspace, 0); len(d) > 0 {
		return StopResult{}, d
	}
	run, _, _, _, _, d := readCoherentControl(workspace)
	if len(d) > 0 {
		return StopResult{}, d
	}
	if d := workspaceIdentityDefects(run.Identity, installIdentityForWorkspace(run.Identity, supplied), identityRelease); len(d) > 0 {
		return StopResult{}, d
	}
	if !occupancyIsActive(run) {
		return StopResult{Status: "settled"}, nil
	}
	lease := run.Occupancy.Lease
	path, err := stopPath(workspace, lease)
	if err != nil {
		return StopResult{}, pathDefects(err)
	}
	data, err := json.Marshal(stopRequest{Lease: lease, Requested: time.Now().UTC().Format(time.RFC3339Nano)})
	if err == nil {
		err = writeAtomic(path, append(data, '\n'))
	}
	if err != nil {
		return StopResult{}, pathDefects(err)
	}
	result := StopResult{Status: "requested", Lease: lease}
	tick := time.NewTicker(controlPollInterval)
	defer tick.Stop()
	for {
		run, _, _, _, _, d = readCoherentControl(workspace)
		if len(d) > 0 {
			return result, d
		}
		if !occupancyIsActive(run) || run.Occupancy.Lease != lease {
			result.Status = "settled"
			return result, nil
		}
		if len(run.Occupancy.Unknown) > 0 && ownerLive(workspace) {
			result.Status = "recovery-required"
			return result, unknownBackendDefects(run.Occupancy.Unknown)
		}
		if run.Status != StatusRunning && run.Status != RunStopping || !ownerLive(workspace) {
			d := settleStop(workspace, supplied, lease)
			if len(d) == 0 {
				result.Status = "settled"
				return result, nil
			}
			if !hasDefectCode(d, DefectOccupiedWorkspace) {
				result.Status = "recovery-required"
				return result, d
			}
		}
		select {
		case <-ctx.Done():
			return result, nil
		case <-tick.C:
		}
	}
}

// A delayed Stop must not reconcile or release a newer owner's lease.
func settleStop(workspace string, supplied *InstallIdentity, lease string) []Defect {
	lock, previous, d := claimResume(filepath.Join(workspace, ControlDir), workspace)
	if len(d) > 0 {
		return d
	}
	if previous != nil {
		defer previous.mutator.Unlock()
	} else {
		defer lock.Close()
	}
	current, _, _, _, _, d := readCoherentControl(workspace)
	if len(d) > 0 {
		return d
	}
	if !occupancyIsActive(current) || current.Occupancy.Lease != lease {
		return nil
	}
	d = reconcileWorkspaceLocked(workspace, supplied, previous != nil)
	if len(d) == 0 && previous != nil {
		DropHeldLease(workspace)
	}
	return d
}

func stopPath(workspace, lease string) (string, error) {
	if !validCheckpointID(lease) {
		return "", errInvalidPath
	}
	// Keep requests out of immutable checkpoint generations. The owner is the
	// only writer of run state; a request for a prior lease is harmless.
	path, _, err := containedRel(workspace, ControlDir+"/stop-"+lease+".json", false)
	return path, err
}

func stopRequested(workspace, lease string) (bool, error) {
	path, err := stopPath(workspace, lease)
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var request stopRequest
	if err := json.Unmarshal(data, &request); err != nil {
		return false, err
	}
	return request.Lease == lease, nil
}
