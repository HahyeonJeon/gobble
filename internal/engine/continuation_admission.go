package engine

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/HahyeonJeon/gobble/internal/engine/exec"
	"github.com/HahyeonJeon/gobble/internal/preparation"
)

func validateContinuationIntent(in preparation.ContinuationIntent) error {
	if in.SchemaVersion != 1 || !launchRequestPattern.MatchString(in.RequestID) || !launchRequestPattern.MatchString(in.WorkspaceID) || !preparedDigestPattern.MatchString(in.OriginDigest) || !preparedDigestPattern.MatchString(in.ReviewDigest) || !preparedDigestPattern.MatchString(in.PreviousHead) {
		return errors.New("invalid continuation intent")
	}
	return nil
}

// ReadContinuationAdmission resolves one exact request, including after later
// checkpoints or an uncertain acknowledgement. It never schedules or reconciles.
func ReadContinuationAdmission(workspace string, in preparation.ContinuationIntent, identity *InstallIdentity) (preparation.ContinuationAdmission, bool, error) {
	var zero preparation.ContinuationAdmission
	if err := validateContinuationIntent(in); err != nil {
		return zero, false, err
	}
	run, exists, err := readRunIdentity(workspace)
	if err != nil {
		return zero, false, err
	}
	if !exists || !validAdmission(run) || run.ExecutionHistory == nil || run.Admission.IntentDigest != in.OriginDigest || run.Admission.WorkspaceID != in.WorkspaceID {
		return zero, false, errors.New("continuation Run binding differs")
	}
	if d := ValidateInstallIdentity(identity); len(d) != 0 {
		return zero, false, errors.New(d[0].Message)
	}
	if d := workspaceIdentityDefects(run.Identity, identity, identityResume); len(d) != 0 {
		return zero, false, errors.New(d[0].Message)
	}
	if in.RequestID == run.Admission.RequestID {
		return zero, false, errors.New("continuation cannot reuse the Start request")
	}
	for _, receipt := range run.ExecutionHistory.Continuations {
		if receipt.Intent.RequestID != in.RequestID {
			continue
		}
		if receipt.Intent != in {
			return zero, false, errors.New("continuation request already names different effects")
		}
		return receipt, true, nil
	}
	return zero, false, nil
}

// ContinuePrepared admits the exact reviewed plan before scheduling. A receipt
// can accompany execution defects: admission is not successful completion.
// Replaying an admitted intent only returns the receipt, even after owner death.
func ContinuePrepared(ctx context.Context, workspace string, raw []byte, launch preparation.LaunchIntent, in preparation.ContinuationIntent, identity *InstallIdentity) (preparation.ContinuationAdmission, []Defect) {
	var zero preparation.ContinuationAdmission
	bad := func(err error) []Defect { return []Defect{{Code: DefectInvalidRequest, Message: err.Error()}} }
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return zero, bad(err)
	}
	if err := validateLaunch(launch); err != nil {
		return zero, bad(err)
	}
	if launch.SchemaVersion != 2 || LaunchDigest(launch) != in.OriginDigest {
		return zero, bad(errors.New("original launch differs"))
	}
	if _, err := DecodePrepared(raw, launch.PreparedDigest, launch.Binding); err != nil {
		return zero, bad(err)
	}
	if receipt, exists, err := ReadContinuationAdmission(workspace, in, identity); err != nil || exists {
		if err != nil {
			return zero, bad(err)
		}
		return receipt, nil
	}
	lock, d := claimOccupy(workspaceFile(workspace, ControlDir))
	if len(d) != 0 {
		if receipt, exists, err := ReadContinuationAdmission(workspace, in, identity); err == nil && exists {
			return receipt, nil
		}
		return zero, d
	}
	transferred := false
	defer func() {
		if !transferred {
			_ = lock.Close()
		}
	}()
	if receipt, exists, err := ReadContinuationAdmission(workspace, in, identity); err != nil || exists {
		if err != nil {
			return zero, bad(err)
		}
		return receipt, nil
	}
	checked, err := inspectContinuation(ctx, workspace, raw, launch, identity)
	if err != nil {
		return zero, bad(err)
	}
	if checked.review.Digest != in.ReviewDigest || checked.review.PreviousHead != in.PreviousHead {
		return zero, []Defect{{Code: DefectConflict, Message: "Continuation review changed; check again."}}
	}
	if d := checkCapacity(checked.doc, readHostCapacity()); len(d) != 0 {
		return zero, d
	}
	host, err := currentHost()
	if err != nil {
		return zero, pathDefects(err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	receipt := preparation.ContinuationAdmission{Intent: in, IntentDigest: continuationIntentDigest(in), Lease: newOccupancyID(), Snapshot: newOccupancyID()}
	s := continuationScheduler(workspace, checked, receipt)
	s.run.Status, s.run.Ended = StatusRunning, ""
	s.run.Occupancy = &jsonOccupancy{Active: true, Host: host, PID: os.Getpid(), Lease: receipt.Lease, Started: now}
	if err := ctx.Err(); err != nil {
		return zero, bad(err)
	}
	if err := s.writeControl(); err != nil {
		// A directory sync error may follow visible pointer publication. Report
		// any committed receipt, but never schedule after an uncertain write.
		committed, _, _ := ReadContinuationAdmission(workspace, in, identity)
		return committed, pathDefects(err)
	}
	s.lease = retainLease(workspace, lock, s.exec)
	transferred = true
	return receipt, s.loop(exec.WithInstalledImages(ctx, launch.Tools), launch.Binding.Cap)
}

func continuationScheduler(workspace string, checked checkedContinuation, receipt preparation.ContinuationAdmission) *sched {
	run := checked.run
	history := *run.ExecutionHistory
	history.Continuations = append(append([]preparation.ContinuationAdmission(nil), history.Continuations...), receipt)
	run.ExecutionHistory = &history
	s := &sched{workspace: workspace, doc: checked.doc, run: run, snapshot: receipt.Snapshot, tasks: make(map[string]*jsonTaskState), history: priorAttempts(checked.tasks), resume: make(map[string]reuseDecision), launched: make(map[string]bool), budget: newBudget(readHostCapacity()), exec: schedulerExecutor(), continuationInputs: make(map[string]string)}
	for _, f := range checked.review.Files {
		s.continuationInputs[f.Path] = f.SHA256
	}
	latest := map[string]jsonTaskState{}
	for _, st := range latestAttempts(checked.tasks) {
		latest[st.ID] = st
	}
	for i, task := range checked.doc.Tasks {
		step := checked.review.Steps[i]
		old := latest[task.ID]
		ident := reservedIdentity(task)
		if step.Action == "reuse" {
			s.tasks[ident] = &old
			s.resume[ident] = reuseDecision{Identity: ident, Decision: reuseReused, Change: changeUnchanged, Reason: reasonReusedIdentityMatched}
			applyResumeDecision(&old, s.resume[ident], true)
			continue
		}
		if step.Action == "restart" {
			s.history = append(s.history, old)
		}
		fresh := initialTask(task)
		fresh.Attempt = step.PlannedAttempt
		s.tasks[ident] = &fresh
		reason := reasonPreviousUnsuccessful
		if step.Action == "restart" {
			reason = reasonPreviousIncomplete
		}
		s.resume[ident] = reuseDecision{Identity: ident, Decision: reuseRerun, Change: changeUnchanged, Reason: reason}
	}
	return s
}
