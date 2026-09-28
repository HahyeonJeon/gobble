package engine

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/HahyeonJeon/gobble/internal/engine/exec"
	"github.com/HahyeonJeon/gobble/internal/preparation"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func launchFixture(t *testing.T) (string, []byte, preparation.LaunchIntent) {
	t.Helper()
	ws := t.TempDir()
	t.Cleanup(func() { DropHeldLease(ws) })
	doc, b := creationDocument(), preparationBinding()
	raw, id, err := Prepare(doc, b)
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("@read\nACGT\n+\nIIII\n")
	writeCheckFile(t, filepath.Join(ws, b.InputPath), string(data))
	writeCheckFile(t, filepath.Join(ws, ".gobble-launch-target"), "req_target")
	image := "sha256:" + strings.Repeat("b", 64)
	tools := map[string]string{}
	for _, task := range doc.Tasks {
		tools[task.Image] = image
	}
	return ws, raw, preparation.LaunchIntent{SchemaVersion: 1, RequestID: "req_start", PreparedDigest: id, Binding: b, InputSHA256: preparedDigest(data), InputSize: int64(len(data)), WorkspaceID: "req_target", EngineImage: image, DaemonID: "daemon", Tools: tools}
}
func TestAdmissionPublishedBeforeTaskAndNeverRescheduled(t *testing.T) {
	ws, raw, in := launchFixture(t)
	calls := 0
	useExec(t, &fnExec{submit: func(ctx context.Context, job exec.Job) (exec.Handle, exec.Report, error) {
		calls++
		a, exists, err := ReadAdmission(ws)
		if err != nil || !exists || a.IntentDigest != LaunchDigest(in) {
			t.Errorf("task before exact admission: %+v %v", a, err)
		}
		return exec.Handle{}, exec.Report{}, errors.New("fixture failure after admission")
	}})
	_ = RunPrepared(t.Context(), ws, raw, in, testInstallIdentity())
	if calls != 1 {
		t.Fatalf("submissions=%d", calls)
	}
	a, exists, err := ReadAdmission(ws)
	if err != nil || !exists {
		t.Fatal(a, err)
	}
	DropHeldLease(ws)
	if d := RunPrepared(t.Context(), ws, raw, in, testInstallIdentity()); len(d) > 0 || calls != 1 {
		t.Fatalf("replay scheduled: %v calls=%d", d, calls)
	}
	other := in
	other.RequestID = "req_second"
	if d := RunPrepared(t.Context(), ws, raw, other, testInstallIdentity()); len(d) == 0 {
		t.Fatal("different launch admitted")
	}
	p, _ := os.ReadFile(filepath.Join(ws, ControlDir, checkpointPointerFile))
	var ptr checkpointPointer
	_ = json.Unmarshal(p, &ptr)
	if ptr.Format != 2 {
		t.Fatal("old readers could interpret admitted checkpoint")
	}
	if d := Resume(t.Context(), Request{Workspace: ws, Identity: testInstallIdentity(), Document: creationDocument()}); !hasDefectCode(d, DefectUnsupportedSchema) {
		t.Fatal("unreviewed resume accepted", d)
	}
}
func TestAdmissionRejectsChangedInputPayloadAndTargetBeforeOccupancy(t *testing.T) {
	for _, which := range []string{"data", "payload", "target", "tool", "schema"} {
		t.Run(which, func(t *testing.T) {
			ws, raw, in := launchFixture(t)
			switch which {
			case "data":
				writeCheckFile(t, filepath.Join(ws, in.Binding.InputPath), "changed")
			case "payload":
				raw = append(raw, ' ')
			case "target":
				writeCheckFile(t, filepath.Join(ws, ".gobble-launch-target"), "req_other")
			case "tool":
				in.Tools = map[string]string{}
			case "schema":
				in.SchemaVersion = 99
			}
			if d := RunPrepared(t.Context(), ws, raw, in, testInstallIdentity()); len(d) == 0 {
				t.Fatal("invalid launch accepted")
			}
			if _, exists, _ := readRunIdentity(ws); exists {
				t.Fatal("invalid launch occupied workspace")
			}
		})
	}
}
func TestStopOwnerRejectsUnobservedLease(t *testing.T) {
	req, _ := seedCheckpoint(t)
	before, _, _ := readRunIdentity(req.Workspace)
	wrong := strings.Repeat("a", 64)
	if wrong == before.Occupancy.Lease {
		wrong = strings.Repeat("b", 64)
	}
	result, d := StopOwner(t.Context(), req.Workspace, req.Identity, wrong)
	if len(d) > 0 || result.Status != "owner-changed" {
		t.Fatal(result, d)
	}
	path, _ := stopPath(req.Workspace, wrong)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("wrote stale stop")
	}
	path, _ = stopPath(req.Workspace, before.Occupancy.Lease)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("stopped different owner")
	}
}

func TestAdmissionPointerCannotDowngrade(t *testing.T) {
	ws, raw, in := launchFixture(t)
	useExec(t, &fnExec{submit: func(context.Context, exec.Job) (exec.Handle, exec.Report, error) {
		return exec.Handle{}, exec.Report{}, errors.New("fixture")
	}})
	_ = RunPrepared(t.Context(), ws, raw, in, testInstallIdentity())
	path := filepath.Join(ws, ControlDir, checkpointPointerFile)
	b, _ := os.ReadFile(path)
	var pointer checkpointPointer
	_ = json.Unmarshal(b, &pointer)
	pointer.Format = 1
	b, _ = json.Marshal(pointer)
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadAdmission(ws); err == nil {
		t.Fatal("admission was read through a legacy pointer")
	}
}
