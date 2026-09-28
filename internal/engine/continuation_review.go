package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/HahyeonJeon/gobble/internal/engine/exec"
	"github.com/HahyeonJeon/gobble/internal/preparation"
)

var verifyContinuationSubmission = exec.VerifySubmissionAbsent

// ReviewPreparedContinuation verifies the saved Run without occupying, repairing
// or executing it. Callers must retain Digest for the explicit User intent.
func ReviewPreparedContinuation(ctx context.Context, workspace string, raw []byte, launch preparation.LaunchIntent, identity *InstallIdentity) (preparation.ContinuationReview, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if ownerLive(workspace) {
		return preparation.ContinuationReview{}, errors.New("execution owner is still active")
	}
	checked, err := inspectContinuation(ctx, workspace, raw, launch, identity)
	if err != nil {
		return preparation.ContinuationReview{}, err
	}
	if ownerLive(workspace) {
		return preparation.ContinuationReview{}, errors.New("execution owner changed; check again")
	}
	return checked.review, nil
}

type checkedContinuation struct {
	review preparation.ContinuationReview
	doc    Document
	run    jsonRun
	tasks  []jsonTaskState
}

func inspectContinuation(ctx context.Context, workspace string, raw []byte, launch preparation.LaunchIntent, identity *InstallIdentity) (checkedContinuation, error) {
	var out checkedContinuation
	if err := ctx.Err(); err != nil {
		return out, err
	}
	if err := validateLaunch(launch); err != nil {
		return out, err
	}
	if launch.SchemaVersion != 2 {
		return out, errors.New("earlier engine admission does not support continuation")
	}
	doc, err := DecodePrepared(raw, launch.PreparedDigest, launch.Binding)
	if err != nil {
		return out, err
	}
	run, plan, _, tasks, _, defects := readCoherentControl(workspace)
	if len(defects) != 0 {
		return out, errors.New(defects[0].Message)
	}
	epoch, valid := admissionEpoch(run)
	if !valid || run.ExecutionHistory == nil || run.Admission.IntentDigest != LaunchDigest(launch) {
		return out, errors.New("Run does not match the original launch")
	}
	if d := ValidateInstallIdentity(identity); len(d) != 0 {
		return out, errors.New(d[0].Message)
	}
	if d := workspaceIdentityDefects(run.Identity, identity, identityResume); len(d) != 0 {
		return out, errors.New(d[0].Message)
	}
	if run.Status != RunStopped || occupancyIsActive(run) || len(run.Occupancy.Unknown) != 0 {
		return out, errors.New("continuation requires a settled stopped Run")
	}
	if len(run.ExecutionHistory.Continuations) >= preparation.MaxContinuations {
		return out, errors.New("continuation history limit reached")
	}
	marker, err := os.ReadFile(filepath.Join(workspace, ".gobble-launch-target"))
	if err != nil || string(marker) != launch.WorkspaceID {
		return out, errors.New("launch workspace reservation changed")
	}
	for i := range doc.Tasks {
		applyReservedDefaults(&doc.Tasks[i])
	}
	encoded, err := marshalControlPlan(doc, run.Snapshot)
	if err != nil {
		return out, err
	}
	var expected jsonPlan
	if err := json.Unmarshal(encoded, &expected); err != nil {
		return out, err
	}
	if !reflect.DeepEqual(expected, plan) {
		return out, errors.New("saved plan differs from the sealed design")
	}
	input, present, err := containedRel(workspace, launch.Binding.InputPath, false)
	if err != nil || !present {
		return out, errors.New("saved input is unavailable")
	}
	info, err := os.Lstat(input)
	if err != nil || !info.Mode().IsRegular() || info.Size() != launch.InputSize {
		return out, errors.New("saved input changed")
	}
	sum, err := sha256File(input)
	if err != nil || "sha256:"+sum != launch.InputSHA256 {
		return out, errors.New("saved input content changed")
	}
	for _, task := range doc.Tasks {
		id := launch.Tools[task.Image]
		if id == "" || lookupImageID(id) != id || lookupImageID(task.Image) != id {
			return out, errors.New("accepted tool image unavailable or changed")
		}
	}
	seenAttempts := map[string]map[int]bool{}
	for _, st := range tasks.Tasks {
		key := reservedIdentity(taskPlanFromState(st))
		if seenAttempts[key] == nil {
			seenAttempts[key] = map[int]bool{}
		}
		if st.Attempt < 1 || st.Attempt > preparation.MaxContinuations+1 || seenAttempts[key][st.Attempt] {
			return out, errors.New("ambiguous task attempts")
		}
		seenAttempts[key][st.Attempt] = true
	}
	latest := latestAttempts(tasks.Tasks)
	if len(latest) != len(doc.Tasks) {
		return out, errors.New("unexpected task history")
	}
	byID := map[string]jsonTaskState{}
	for _, st := range latest {
		byID[reservedIdentity(taskPlanFromState(st))] = st
	}
	if len(byID) != len(doc.Tasks) {
		return out, errors.New("ambiguous task history")
	}
	review := preparation.ContinuationReview{SchemaVersion: 1, WorkspaceID: launch.WorkspaceID, OriginDigest: run.Admission.IntentDigest, PreviousHead: epoch.Head, Snapshot: run.Snapshot, Steps: []preparation.ContinuationStep{}, Files: []preparation.ContinuationFile{{Path: launch.Binding.InputPath, SHA256: sum}}}
	unfinished := false
	for _, task := range doc.Tasks {
		st, exists := byID[reservedIdentity(task)]
		if !exists || st.Attempt < 1 || st.Attempt > preparation.MaxContinuations+1 || st.Instance != "" || st.ShardIndex != 0 || st.ShardCount != 1 || !sameStrings(st.Command, task.Command) || st.Script != task.Script || !sameParams(decodeParams(st.Params), task.Params) || envIdentityChanged(st, task) || st.Image != task.Image {
			return out, errors.New("task identity differs from the saved design")
		}
		if st.Submission != nil && st.Submission.DaemonID != "" && st.Submission.DaemonID != launch.DaemonID {
			return out, errors.New("task daemon differs from the accepted runtime")
		}
		if err := verifyContinuationSubmission(ctx, exec.Handle{RuntimeID: st.RuntimeID, Submission: st.Submission}); err != nil {
			return out, err
		}
		step := preparation.ContinuationStep{TaskID: task.ID, PriorAttempt: st.Attempt, PlannedAttempt: st.Attempt}
		switch st.Status {
		case StatusSucceeded:
			if unfinished || st.ImageDigest != launch.Tools[task.Image] {
				return out, errors.New("completed task cannot be reused")
			}
			if reason, _ := compareInputIdentity(workspace, st, task, true); reason != "" {
				return out, errors.New("completed task input changed")
			}
			files, err := continuationOutputs(workspace, task, st)
			if err != nil {
				return out, err
			}
			review.Files = append(review.Files, files...)
			step.Action = "reuse"
		case StatusIncomplete, StatusNotStarted:
			unfinished = true
			if st.Status == StatusNotStarted && (st.Started != "" || st.RuntimeID != "" || st.Submission != nil || len(st.Fingerprints) != 0) {
				return out, errors.New("unstarted task has execution evidence")
			}
			step.Action = "start"
			if st.Status == StatusIncomplete {
				if st.Reason != "canceled" {
					return out, errors.New("task was not stopped cleanly")
				}
				step.Action, step.PlannedAttempt = "restart", st.Attempt+1
			}
			if len(st.Checksums) != 0 || len(st.Lineage) != 0 {
				return out, errors.New("stopped task has ambiguous published output")
			}
			for _, file := range declaredIOFiles(task.Outputs) {
				_, present, err := containedRel(workspace, file.path, false)
				if err != nil || present {
					return out, errors.New("uncompleted task output already exists")
				}
			}
			next := task
			next.Attempt = step.PlannedAttempt
			_, present, err := containedRel(workspace, isolateRel(next), false)
			if err != nil || present {
				return out, errors.New("next attempt directory already exists")
			}
		default:
			return out, fmt.Errorf("task state %q does not support continuation", st.Status)
		}
		review.Steps = append(review.Steps, step)
	}
	if !unfinished {
		return out, errors.New("no unfinished work to continue")
	}
	state, err := json.Marshal(struct {
		Run   jsonRun
		Plan  jsonPlan
		Tasks jsonTasksFile
	}{run, plan, tasks})
	if err != nil {
		return out, err
	}
	review.StateDigest = preparedDigest(state)
	encoded, err = json.Marshal(review)
	if err != nil {
		return out, err
	}
	review.Digest = preparedDigest(encoded)
	current, _, err := readRunIdentity(workspace)
	if err != nil || current.Snapshot != run.Snapshot {
		return out, errors.New("Run changed during review; check again")
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	return checkedContinuation{review: review, doc: doc, run: run, tasks: tasks.Tasks}, nil
}

func continuationOutputs(workspace string, task TaskPlan, st jsonTaskState) ([]preparation.ContinuationFile, error) {
	files := declaredIOFiles(task.Outputs)
	if len(st.Checksums) != len(files) {
		return nil, errors.New("completed output evidence is incomplete")
	}
	byPath := map[string]string{}
	for _, record := range st.Checksums {
		if record.SHA256 == "" || byPath[record.Path] != "" {
			return nil, errors.New("completed output evidence is ambiguous")
		}
		byPath[record.Path] = record.SHA256
	}
	out := make([]preparation.ContinuationFile, 0, len(files))
	for _, file := range files {
		path, present, err := containedRel(workspace, file.path, false)
		if err != nil || !present || !regularFile(path) {
			return nil, errors.New("completed output is unavailable")
		}
		sum, err := sha256File(path)
		if err != nil || byPath[file.path] != sum {
			return nil, errors.New("completed output content changed")
		}
		out = append(out, preparation.ContinuationFile{Path: file.path, SHA256: sum})
	}
	return out, nil
}
