package appservice

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/HahyeonJeon/gobble/internal/preparation"
)

var continuationLeasePattern = regexp.MustCompile(`^[a-f0-9]{32,128}$`)

type continuationSupport struct {
	ProjectID      string `json:"projectId"`
	LaunchReviewID string `json:"launchReviewId"`
	Supported      bool   `json:"supported"`
	Reason         string `json:"reason"`
}

// Only an explicitly labelled engine is queried for the new protocol. Missing
// support leaves old Start/inspection available; it never converts an old Run.
func (s *Service) engineContinuationSupport(ctx context.Context, b RuntimeBinding) (bool, error) {
	if err := s.runtime.verify(ctx, b); err != nil {
		return false, err
	}
	label, err := s.runtime.commandAt(ctx, b.Endpoint, "image", "inspect", "--format", `{{index .Config.Labels "io.gobble.continuation.scope"}}`, b.ImageID)
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(string(label)) != "single-end-trim-fastqc-v1" {
		return false, nil
	}
	name := "gobble-capabilities-" + newID("req")
	defer s.removeRuntimeQuery(b, name)
	raw, err := s.runtime.commandAt(ctx, b.Endpoint, "run", "--rm", "--pull=never", "--name", name, "--platform", "linux/amd64", "--network=none", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--pids-limit=32", "--memory=128m", "--cpus=1", "--env", "GOBBLE_CONTAINER_BOOTSTRAP=0", "--entrypoint", "/usr/local/bin/gobble", b.ImageID, "prepared-capabilities")
	if err != nil {
		return false, err
	}
	var caps struct {
		SchemaVersion       int `json:"schemaVersion"`
		LaunchSchemaVersion int `json:"launchSchemaVersion"`
		ContinuationVersion int `json:"continuationVersion"`
	}
	if decodeCreationJSON(raw, &caps) != nil || caps.SchemaVersion != 1 || caps.LaunchSchemaVersion != 2 || caps.ContinuationVersion != 1 {
		return false, problem("incompatible_runtime", "The engine continuation protocol is unsupported.")
	}
	return true, nil
}
func (s *Service) continuationSupport(ctx context.Context, project, launch string) (continuationSupport, error) {
	rec, err := s.readLaunch(project, launch)
	if err != nil {
		return continuationSupport{}, err
	}
	if rec.Intent.SchemaVersion != 2 {
		return continuationSupport{project, launch, false, "This analysis uses an earlier engine."}, nil
	}
	ok, err := s.engineContinuationSupport(ctx, rec.Runtime)
	if err != nil {
		return continuationSupport{}, err
	}
	reason := ""
	if !ok {
		reason = "The original engine does not support continuation."
	}
	return continuationSupport{project, launch, ok, reason}, nil
}

func (s *Service) continuationQuery(ctx context.Context, p projectRecord, original launchRecord, rec continuationRecord, verb string) ([]byte, error) {
	if verb != "prepared-continuation-review" && verb != "prepared-continuation-receipt" && verb != "prepared-stop" {
		return nil, problem("invalid_request", "Unsupported continuation query.")
	}
	if err := s.runtime.verify(ctx, original.Runtime); err != nil {
		return nil, err
	}
	target := filepath.Join(p.Root, filepath.FromSlash(original.Value.OutputPath))
	mount := readOnlyMount(target, original.Runtime.WorkspacePath)
	if verb == "prepared-stop" {
		mount = writableMount(target, original.Runtime.WorkspacePath)
	}
	name := "gobble-continuation-query-" + newID("req")
	defer s.removeRuntimeQuery(original.Runtime, name)
	args := []string{"run", "--rm", "--pull=never", "--name", name, "--platform", "linux/amd64", "--network=none", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--user", "0:0", "--pids-limit=32", "--memory=256m", "--cpus=1", "--workdir", original.Runtime.WorkspacePath, "--env", "GOBBLE_CONTAINER_BOOTSTRAP=0", "--env", "GOBBLE_CONTROLLER=" + name, "--env", "GOBBLE_DAEMON_ID=" + original.Runtime.DaemonID, "--env", "GOBBLE_RUNTIME_IMAGE_ID=" + original.Runtime.ImageID, "--env", "DOCKER_HOST=unix:///var/run/docker.sock", "--mount", mount, "--mount", readOnlyMount(s.continuationBundle(rec), "/gobble/launch"), "--mount", writableMount(launchSocket(original.Runtime.Endpoint), "/var/run/docker.sock"), "--entrypoint", "/usr/local/bin/gobble", original.Runtime.ImageID, verb, "--workspace", original.Runtime.WorkspacePath}
	raw, err := s.runtime.commandAt(ctx, original.Runtime.Endpoint, args...)
	if verb == "prepared-continuation-review" {
		err = continuationReviewError(err)
	}
	return raw, err
}
func (s *Service) continuationController(rec continuationRecord) executionController {
	id := requestDigest(*rec.Intent)
	return executionController{"gobble-continuation-" + strings.TrimPrefix(id, "sha256:")[:32], s.continuationBundle(rec), "prepared-continue", id}
}
func validateContinuationReview(v preparation.ContinuationReview, original launchRecord) error {
	bad := func() error {
		return problem("incompatible_runtime", "The continuation review does not match this exact saved Run.")
	}
	hash := v.Digest
	v.Digest = ""
	if original.Intent.SchemaVersion != 2 || v.SchemaVersion != 1 || !digestPattern.MatchString(hash) || requestDigest(v) != hash || v.WorkspaceID != original.Intent.WorkspaceID || v.OriginDigest != launchIntentDigest(original.Intent) || !digestPattern.MatchString(v.PreviousHead) || !digestPattern.MatchString(v.StateDigest) || !continuationLeasePattern.MatchString(v.Snapshot) || len(v.Steps) != 2 || len(v.Files) < 1 || len(v.Files) > 5 {
		return bad()
	}
	seen := map[string]bool{}
	unfinished := false
	for i, step := range v.Steps {
		if len(original.Value.Preparation.Prepared.Steps) != 2 || step.TaskID != original.Value.Preparation.Prepared.Steps[i].ID || seen[step.TaskID] || step.PriorAttempt < 1 || step.PlannedAttempt > 33 {
			return bad()
		}
		seen[step.TaskID] = true
		switch step.Action {
		case "reuse":
			if unfinished || step.PlannedAttempt != step.PriorAttempt {
				return bad()
			}
		case "restart":
			unfinished = true
			if step.PlannedAttempt != step.PriorAttempt+1 {
				return bad()
			}
		case "start":
			unfinished = true
			if step.PlannedAttempt != step.PriorAttempt {
				return bad()
			}
		default:
			return bad()
		}
	}
	if !unfinished {
		return bad()
	}
	files := map[string]bool{}
	for _, f := range v.Files {
		if !validRelative(f.Path) || len(f.Path) > 4096 || !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(f.SHA256) || files[f.Path] {
			return bad()
		}
		files[f.Path] = true
	}
	if v.Files[0].Path != original.Intent.Binding.InputPath || "sha256:"+v.Files[0].SHA256 != original.Intent.InputSHA256 {
		return bad()
	}
	return nil
}
func validateContinuationReceipt(r preparation.ContinuationAdmission, in preparation.ContinuationIntent) error {
	if r.Intent != in || r.IntentDigest != requestDigest(in) || !continuationLeasePattern.MatchString(r.Lease) || !continuationLeasePattern.MatchString(r.Snapshot) {
		return problem("incompatible_runtime", "The continuation receipt belongs to another intent.")
	}
	return nil
}
func (s *Service) queryContinuationReceipt(ctx context.Context, p projectRecord, original launchRecord, rec continuationRecord) (*preparation.ContinuationAdmission, error) {
	raw, err := s.continuationQuery(ctx, p, original, rec, "prepared-continuation-receipt")
	if err != nil {
		return nil, err
	}
	var result struct {
		SchemaVersion int                                `json:"schemaVersion"`
		Found         bool                               `json:"found"`
		Receipt       *preparation.ContinuationAdmission `json:"receipt,omitempty"`
	}
	if decodeCreationJSON(raw, &result) != nil || result.SchemaVersion != 1 || result.Found != (result.Receipt != nil) {
		return nil, problem("incompatible_runtime", "The continuation acknowledgement is invalid.")
	}
	if result.Receipt != nil {
		if err := validateContinuationReceipt(*result.Receipt, *rec.Intent); err != nil {
			return nil, err
		}
	}
	return result.Receipt, nil
}
func (s *Service) sendContinuationStop(ctx context.Context, p projectRecord, original launchRecord, rec continuationRecord) (string, error) {
	op := rec.Value.Operation
	data, _ := json.Marshal(map[string]string{"expectedLease": op.StopLease})
	if err := atomicWrite(s.continuationBundle(rec), "stop.json", data); err != nil {
		return "requested", err
	}
	raw, err := s.continuationQuery(ctx, p, original, rec, "prepared-stop")
	if err != nil {
		return "requested", err
	}
	var result struct {
		Status string `json:"status"`
		Lease  string `json:"lease,omitempty"`
	}
	if decodeCreationJSON(raw, &result) != nil || (result.Lease != "" && result.Lease != op.StopLease) {
		return "requested", problem("incompatible_runtime", "Stop acknowledgement belongs to another execution.")
	}
	switch result.Status {
	case "requested", "settled", "owner-changed", "recovery-required":
		return result.Status, nil
	}
	return "requested", problem("incompatible_runtime", "Unknown Stop acknowledgement.")
}

// Only known, path-free CLI review failures become public guidance. Arbitrary
// process stderr, including unknown engine messages, retains its generic error.
func continuationReviewError(err error) error {
	var failure *runtimeCommandError
	if !errors.As(err, &failure) {
		return err
	}
	var diagnostic struct {
		Op      string `json:"op"`
		Defects []struct {
			Code    string   `json:"code"`
			Unit    string   `json:"unit"`
			Message string   `json:"message"`
			Paths   []string `json:"paths"`
		} `json:"defects"`
	}
	if decodeCreationJSON(failure.diagnostic, &diagnostic) != nil || diagnostic.Op != "cli" || len(diagnostic.Defects) != 1 {
		return err
	}
	defect := diagnostic.Defects[0]
	if defect.Code != "invalid-request" || defect.Unit != "" || len(defect.Paths) != 0 {
		return err
	}
	var message string
	switch defect.Message {
	case "completed output is unavailable":
		message = "A completed result is missing. Restore the original result and recheck, or review a new analysis."
	case "completed output content changed":
		message = "A completed result has changed. Restore the original result and recheck, or review a new analysis."
	case "saved input is unavailable":
		message = "The checked input copy is missing. Restore the original copy and recheck, or review a new analysis."
	case "saved input changed", "saved input content changed", "completed task input changed":
		message = "The checked input copy has changed. Restore the original copy and recheck, or review a new analysis."
	case "accepted tool image unavailable or changed":
		message = "An analysis tool is missing or has changed. Restore the original tool version before rechecking."
	case "execution owner is still active", "continuation requires a settled stopped Run":
		message = "Stop has not settled. Refresh the Run before checking continuation again."
	case "execution owner changed; check again", "Run changed during review; check again":
		message = "The execution state changed during this check. Refresh the Run and check again."
	case "uncompleted task output already exists":
		message = "An unfinished step already has a result. Keep it for review; continuation will not overwrite it."
	default:
		return err
	}
	return problem("invalid_request", message)
}
