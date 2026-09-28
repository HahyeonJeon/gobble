package gobble

import (
	"context"
	"errors"
	"github.com/HahyeonJeon/gobble/internal/containerenv"
	"github.com/HahyeonJeon/gobble/internal/engine"
	"github.com/HahyeonJeon/gobble/internal/preparation"
)

// LaunchIntent binds User-authorized effects to a private prepared pipeline.
type LaunchIntent = preparation.LaunchIntent

// PreparedRunState contains an exact engine admission and observed Run state.
type PreparedRunState = engine.PreparedRunState

// RunPreparedPipeline executes only the previously sealed graph. It never loads
// Project source. The trusted controller must have verified its actual container
// through containerenv.Prepare before calling this entry.
func RunPreparedPipeline(ctx context.Context, workspace string, raw []byte, in LaunchIntent) error {
	runtime := containerenv.Current()
	if runtime.Image != in.EngineImage || runtime.Daemon != in.DaemonID {
		return errors.New("launch runtime differs from the accepted environment")
	}
	id, err := engine.PreparedIdentity(in)
	if err != nil {
		return err
	}
	return publicError("prepared-run", engine.RunPrepared(ctx, workspace, raw, in, id))
}

// ReadPreparedRun reads the committed admission through the exact installed engine.
func ReadPreparedRun(workspace string) (PreparedRunState, error) {
	return engine.InspectAdmission(workspace)
}

// StopPreparedRun requests settlement of the observed owner only. It cannot Stop
// a replacement owner, and cancelling the wait does not erase a durable request.
func StopPreparedRun(ctx context.Context, workspace, expectedLease string) (StopResult, error) {
	r, d := engine.StopOwner(ctx, workspace, nil, expectedLease)
	return StopResult{r.Status, r.Lease}, publicError("prepared-stop", d)
}

// ContinuationReview describes the exact reusable and unfinished steps in a
// stopped Run. It carries no editable or executable source.
type ContinuationReview = preparation.ContinuationReview

type ContinuationIntent = preparation.ContinuationIntent
type ContinuationAdmission = preparation.ContinuationAdmission

// ReviewPreparedContinuation only observes the original saved design and Run.
// The trusted controller presents these facts before collecting User intent.
func ReviewPreparedContinuation(ctx context.Context, workspace string, raw []byte, original LaunchIntent) (ContinuationReview, error) {
	id, err := preparedContinuationIdentity(original)
	if err != nil {
		return ContinuationReview{}, err
	}
	return engine.ReviewPreparedContinuation(ctx, workspace, raw, original, id)
}

// ContinuePreparedPipeline executes an explicitly authorized, unchanged review.
// A nonzero receipt acknowledges admission even when execution returns an error.
// Retrying the exact intent returns that receipt without scheduling more work.
func ContinuePreparedPipeline(ctx context.Context, workspace string, raw []byte, original LaunchIntent, intent ContinuationIntent) (ContinuationAdmission, error) {
	id, err := preparedContinuationIdentity(original)
	if err != nil {
		return ContinuationAdmission{}, err
	}
	receipt, defects := engine.ContinuePrepared(ctx, workspace, raw, original, intent, id)
	return receipt, publicError("prepared-continue", defects)
}

// ReadPreparedContinuation resolves an exact continuation acknowledgement. It
// never executes or repairs the Run, including after an interrupted controller.
func ReadPreparedContinuation(workspace string, original LaunchIntent, intent ContinuationIntent) (ContinuationAdmission, bool, error) {
	id, err := preparedContinuationIdentity(original)
	if err != nil {
		return ContinuationAdmission{}, false, err
	}
	if engine.LaunchDigest(original) != intent.OriginDigest {
		return ContinuationAdmission{}, false, errors.New("original launch differs")
	}
	return engine.ReadContinuationAdmission(workspace, intent, id)
}

func preparedContinuationIdentity(original LaunchIntent) (*engine.InstallIdentity, error) {
	runtime := containerenv.Current()
	if runtime.Image != original.EngineImage || runtime.Daemon != original.DaemonID {
		return nil, errors.New("continuation runtime differs from the accepted environment")
	}
	return engine.PreparedIdentity(original)
}
