package engine

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"github.com/HahyeonJeon/gobble/internal/preparation"
)

// OutputRequest addresses one declared output of an exact latest task attempt.
type OutputRequest struct {
	RunID    string `json:"runId"`
	Instance string `json:"instance"`
	Attempt  int    `json:"attempt"`
	Port     string `json:"port"`
}

// OutputEvidence attributes recorded bytes at Snapshot; it does not read the file
// or promise that the attempt remains current after this observation.
type OutputEvidence struct {
	SchemaVersion int    `json:"schemaVersion"`
	RunID         string `json:"runId"`
	Snapshot      string `json:"snapshot"`
	OriginDigest  string `json:"originDigest"`
	Instance      string `json:"instance"`
	Attempt       int    `json:"attempt"`
	Port          string `json:"port"`
	Recipe        string `json:"recipe,omitempty"`
	Path          string `json:"path"`
	SHA256        string `json:"sha256"`
	Size          int64  `json:"size"`
}

// ReadOutputEvidence reads one coherent checkpoint tied to the admitted sealed
// declaration. It never loads Project code, mutates state or checks tool liveness.
func ReadOutputEvidence(workspace string, raw []byte, launch preparation.LaunchIntent, request OutputRequest, identity *InstallIdentity) (OutputEvidence, error) {
	fail := func(message string) (OutputEvidence, error) { return OutputEvidence{}, errors.New(message) }
	if request.RunID == "" || len(request.RunID) > 256 || request.Instance == "" || len(request.Instance) > 256 || request.Attempt < 1 || request.Port == "" || len(request.Port) > 256 || strings.ContainsAny(request.RunID+request.Instance+request.Port, "\x00\r\n") {
		return fail("invalid output request")
	}
	if validateLaunch(launch) != nil {
		return fail("invalid output launch")
	}
	doc, err := DecodePrepared(raw, launch.PreparedDigest, launch.Binding)
	if err != nil {
		return fail("output declaration is unsupported")
	}
	run, plan, _, tasks, _, defects := readCoherentControl(workspace)
	if len(defects) != 0 {
		return fail("output state is unavailable")
	}
	if run.ID != request.RunID || !validAdmission(run) || run.Admission.IntentDigest != LaunchDigest(launch) {
		return fail("output Run origin changed")
	}
	if len(ValidateInstallIdentity(identity)) != 0 || len(workspaceIdentityDefects(run.Identity, identity, identityResume)) != 0 {
		return fail("output engine identity changed")
	}
	var declared *TaskPlan
	for i := range doc.Tasks {
		if reservedIdentity(doc.Tasks[i]) == request.Instance {
			if declared != nil {
				return fail("output declaration is ambiguous")
			}
			copy := doc.Tasks[i]
			declared = &copy
		}
	}
	if declared == nil {
		return fail("output task is unavailable")
	}
	recipe := ReviewStep(*declared).Recipe
	for i := range doc.Tasks {
		applyReservedDefaults(&doc.Tasks[i])
	}
	encoded, err := marshalControlPlan(doc, run.Snapshot)
	if err != nil {
		return fail("output declaration is unsupported")
	}
	var expected jsonPlan
	if json.Unmarshal(encoded, &expected) != nil || !reflect.DeepEqual(expected, plan) {
		return fail("output saved plan changed")
	}
	seen := map[string]map[int]bool{}
	var latest *jsonTaskState
	for i := range tasks.Tasks {
		st := tasks.Tasks[i]
		key := reservedIdentity(taskPlanFromState(st))
		if seen[key] == nil {
			seen[key] = map[int]bool{}
		}
		if st.Attempt < 1 || seen[key][st.Attempt] {
			return fail("output attempts are ambiguous")
		}
		seen[key][st.Attempt] = true
		if key == request.Instance && (latest == nil || latest.Attempt < st.Attempt) {
			copy := st
			latest = &copy
		}
	}
	if latest == nil {
		return fail("output task is unavailable")
	}
	if latest.Attempt != request.Attempt {
		return fail("output attempt is no longer current")
	}
	if latest.Status != StatusSucceeded {
		return fail("output attempt has not succeeded")
	}
	t := *declared
	if latest.Instance != "" || latest.ShardIndex != 0 || latest.ShardCount != 1 || !sameStrings(latest.Command, t.Command) || latest.Script != t.Script || !sameParams(decodeParams(latest.Params), t.Params) || envIdentityChanged(*latest, t) || latest.Image != t.Image || latest.ImageDigest != launch.Tools[t.Image] || launch.Tools[t.Image] == "" {
		return fail("output task identity changed")
	}
	path := ""
	for _, o := range t.Outputs {
		if o.Name == request.Port {
			if path != "" || !simpleReviewIO(o) {
				return fail("output declaration is ambiguous")
			}
			path = o.Path
		}
	}
	if path == "" {
		return fail("output port is unavailable")
	}
	var record *jsonFileHash
	checks := map[string]bool{}
	for _, c := range latest.Checksums {
		if checks[c.Path] || !preparedDigestPattern.MatchString("sha256:"+c.SHA256) || c.Size < 0 {
			return fail("output checksums are ambiguous")
		}
		checks[c.Path] = true
		if c.Path == path {
			copy := c
			record = &copy
		}
	}
	if record == nil {
		return fail("output checksum is unavailable")
	}
	return OutputEvidence{1, run.ID, run.Snapshot, run.Admission.IntentDigest, request.Instance, request.Attempt, request.Port, recipe, path, "sha256:" + record.SHA256, record.Size}, nil
}
