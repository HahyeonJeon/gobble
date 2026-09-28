package engine

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"

	"github.com/HahyeonJeon/gobble/internal/engine/exec"
	"github.com/HahyeonJeon/gobble/internal/preparation"
)

var launchRequestPattern = regexp.MustCompile(`^req_[A-Za-z0-9_-]{1,100}$`)

// LaunchDigest names all effects in a launch intent, independently of P3 metadata.
func LaunchDigest(in preparation.LaunchIntent) string {
	raw, _ := json.Marshal(in)
	return preparedDigest(raw)
}

func validateLaunch(in preparation.LaunchIntent) error {
	if (in.SchemaVersion != 1 && in.SchemaVersion != 2) || !launchRequestPattern.MatchString(in.RequestID) || !launchRequestPattern.MatchString(in.WorkspaceID) || !preparedDigestPattern.MatchString(in.PreparedDigest) || !validPreparationBinding(in.Binding) || !preparedDigestPattern.MatchString(in.InputSHA256) || in.InputSize <= 0 || !preparedDigestPattern.MatchString(in.EngineImage) || in.DaemonID == "" || len(in.DaemonID) > 256 || len(in.Tools) != 2 {
		return errors.New("invalid launch intent")
	}
	for ref, id := range in.Tools {
		if invalidImage(ref) != "" || !preparedDigestPattern.MatchString(id) {
			return errors.New("invalid installed tool binding")
		}
	}
	return nil
}

// PreparedIdentity identifies the trusted reader executable plus sealed source,
// without loading a Project module or running the Go compiler.
func PreparedIdentity(in preparation.LaunchIntent) (*InstallIdentity, error) {
	sum, err := executableDigest()
	if err != nil {
		return nil, err
	}
	version := "prepared-v1"
	if in.SchemaVersion == 2 {
		version = "prepared-v2"
	}
	return &InstallIdentity{GobbleModule: installGobbleModule, GobbleVersion: version, GobbleVCSRevision: in.EngineImage, GobbleExecutableSHA256: sum, PipelineModule: "gobble.local/prepared", PipelineImport: "gobble.local/prepared", PipelineVersion: version, PipelineVCSRevision: in.Binding.SourceRevision, GOOS: "linux", GOARCH: "amd64", InstallKind: "module", IdentityMode: "local-pin"}, nil
}

// RunPrepared validates sealed semantics and staged bytes before occupying. The
// first committed checkpoint contains admission BEFORE the scheduler can submit.
// A repeated admitted intent never calls the scheduler again, even after Stop.
func RunPrepared(ctx context.Context, workspace string, raw []byte, in preparation.LaunchIntent, identity *InstallIdentity) []Defect {
	bad := func(err error) []Defect { return []Defect{{Code: DefectInvalidRequest, Message: err.Error()}} }
	if err := validateLaunch(in); err != nil {
		return bad(err)
	}
	doc, err := DecodePrepared(raw, in.PreparedDigest, in.Binding)
	if err != nil {
		return bad(err)
	}
	for _, task := range doc.Tasks {
		if in.Tools[task.Image] == "" {
			return bad(errors.New("missing exact installed tool"))
		}
	}
	if d := ValidateInstallIdentity(identity); len(d) > 0 {
		return d
	}
	if a, exists, e := ReadAdmission(workspace); e != nil {
		return pathDefects(e)
	} else if exists {
		if a.IntentDigest == LaunchDigest(in) {
			return nil
		}
		return []Defect{{Code: DefectConflict, Message: "Workspace already admitted another launch."}}
	}
	// The service also checks its allocation; Gobble independently binds the marker.
	marker, err := os.ReadFile(filepath.Join(workspace, ".gobble-launch-target"))
	if err != nil || string(marker) != in.WorkspaceID {
		return bad(errors.New("launch workspace reservation changed"))
	}
	input, present, err := containedRel(workspace, in.Binding.InputPath, false)
	if err != nil || !present {
		return bad(errors.New("staged input unavailable"))
	}
	info, err := os.Lstat(input)
	if err != nil || !info.Mode().IsRegular() || info.Size() != in.InputSize {
		return bad(errors.New("staged input changed"))
	}
	sum, err := sha256File(input)
	if err != nil || "sha256:"+sum != in.InputSHA256 {
		return bad(errors.New("staged input content changed"))
	}
	req := Request{Workspace: workspace, Identity: identity, Document: doc, Cap: in.Binding.Cap}
	if d := Check(req); len(d) > 0 {
		return d
	}
	a := preparation.Admission{SchemaVersion: in.SchemaVersion, IntentDigest: LaunchDigest(in), RequestID: in.RequestID, PreparedDigest: in.PreparedDigest, WorkspaceID: in.WorkspaceID}
	s, d := occupyAdmission(req, &a)
	if len(d) > 0 {
		return d
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return s.loop(exec.WithInstalledImages(ctx, in.Tools), in.Binding.Cap)
}

// ReadAdmission is an exact committed read, not controller name inference.
func ReadAdmission(workspace string) (preparation.Admission, bool, error) {
	var zero preparation.Admission
	run, exists, err := readRunIdentity(workspace)
	if err != nil || !exists {
		return zero, false, err
	}
	if run.Admission == nil {
		return zero, false, errors.New("workspace has an unrelated Run")
	}
	a := *run.Admission
	if !validAdmission(run) {
		return zero, false, errors.New("invalid admission record")
	}
	return a, true, nil
}

func existingAdmissionPresent(workspace string) bool {
	_, exists, err := readRunIdentity(workspace)
	return exists || err != nil
}

// PreparedRunState reports engine truth with a distinct observation snapshot.
type PreparedRunState struct {
	Epoch         *preparation.ExecutionEpoch `json:"epoch,omitempty"`
	SchemaVersion int                         `json:"schemaVersion"`
	Admission     preparation.Admission       `json:"admission"`
	Status        string                      `json:"status"`
	Snapshot      string                      `json:"snapshot"`
	OwnerActive   bool                        `json:"ownerActive"`
	OwnerLive     bool                        `json:"ownerLive"`
}

func InspectAdmission(workspace string) (PreparedRunState, error) {
	var out PreparedRunState
	run, _, _, _, _, d := readCoherentControl(workspace)
	if len(d) > 0 {
		return out, errors.New(d[0].Message)
	}
	if run.Admission == nil || !validAdmission(run) {
		return out, errors.New("no valid admission in this workspace")
	}
	if d := workspaceIdentityDefects(run.Identity, processInstallIdentity(run.Identity), identityInspect); len(d) > 0 {
		return out, errors.New(d[0].Message)
	}
	out = PreparedRunState{SchemaVersion: 1, Admission: *run.Admission, Status: run.Status, Snapshot: run.Snapshot, OwnerActive: occupancyIsActive(run), OwnerLive: ownerLive(workspace)}
	if run.ExecutionHistory != nil {
		epoch, _ := admissionEpoch(run)
		out.SchemaVersion, out.Epoch = 2, &epoch
	}
	return out, nil
}
func validAdmission(run jsonRun) bool {
	_, valid := admissionEpoch(run)
	return valid
}
