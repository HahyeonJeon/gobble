package gobble

import (
	"context"
	"github.com/HahyeonJeon/gobble/internal/engine"
)

// StopResult reports whether termination was requested, settled, or requires
// backend recovery. A requested result never claims that tasks have stopped.
type StopResult struct {
	Status string `json:"status"`
	Lease  string `json:"lease,omitempty"`
}

// Stop requests graceful termination of the current run and waits for settlement.
// Canceling ctx stops waiting, leaving the durable request in place. Repeated
// calls are safe. Resume automatically reconciles before restarting work.
func Stop(ctx context.Context, workspace string, opts ...OccupyOption) (StopResult, error) {
	identity, err := parseOccupyOptions("stop", false, opts)
	if err != nil {
		return StopResult{}, err
	}
	result, defects := engine.Stop(ctx, workspace, toEngineIdentity(identity))
	return StopResult{Status: result.Status, Lease: result.Lease}, publicError("stop", defects)
}
