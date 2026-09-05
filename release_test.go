package gobble_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/internal/engine"
	"github.com/HahyeonJeon/gobble/internal/testutil"
)

func TestReleaseMissingRun(t *testing.T) {
	err := gobble.Release(t.TempDir())
	requireReleaseError(t, "missing run", err, gobble.DefectNotFound, "")
}

func TestReleaseLiveAndDeadOwner(t *testing.T) {
	dir := readyRunWorkspace(t)
	g := mustCompose(processCopyPipeline)(t)
	if err := gobble.Run(t.Context(), g, dir, 0, testOccupyOption(t)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("occupying-process Release() error = %v, want nil", err)
	}
	if _, statErr := os.Stat(testutil.ControlPath(t, filepath.Join(dir, engine.ControlDir, engine.RunIdentityFile))); statErr != nil {
		t.Fatalf("Release deleted run.json: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "out", "sample.txt")); statErr != nil {
		t.Fatalf("Release deleted dest: %v", statErr)
	}

	err := gobble.Release(dir)
	requireReleaseError(t, "already released", err, gobble.DefectAlreadyReleased, "")

	err = gobble.Run(t.Context(), g, dir, 0, testOccupyOption(t))
	requireRunError(t, "Run after Release", err, gobble.DefectOutputExists, "copy.out")
}

func TestReleaseOp(t *testing.T) {
	err := gobble.Release(t.TempDir())
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		t.Fatalf("error = %v, want *Error", err)
	}
	if ge.Op != "release" {
		t.Fatalf("Error.Op got %q, want release", ge.Op)
	}
}

func requireReleaseError(t *testing.T, name string, err error, code gobble.DefectCode, unit string) *gobble.Error {
	t.Helper()
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		t.Fatalf("case %s: error = %v, want *Error", name, err)
	}
	if ge.Op != "release" {
		t.Fatalf("case %s: Error.Op got %q, want release", name, ge.Op)
	}
	found := false
	for _, d := range ge.Defects {
		if d.Code == code && (unit == "" || d.Unit == unit) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("case %s: defects %#v, want code %s unit %q", name, ge.Defects, code, unit)
	}
	return ge
}

func forcePublicDeadOwner(t *testing.T, workspace string) {
	t.Helper()
	gobble.DropHeldLease(workspace)
	path := testutil.ControlPath(t, filepath.Join(workspace, engine.ControlDir, engine.RunIdentityFile))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var run map[string]any
	if err := json.Unmarshal(data, &run); err != nil {
		t.Fatal(err)
	}
	occ, _ := run["occupancy"].(map[string]any)
	if occ == nil {
		occ = map[string]any{"active": true}
		run["occupancy"] = occ
	}
	host, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}
	occ["active"] = true
	occ["host"] = host
	occ["pid"] = publicDeadPID(t)
	out, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func publicDeadPID(t *testing.T) int {
	t.Helper()
	for pid := 1 << 22; pid > 2; pid-- {
		if err := syscall.Kill(pid, 0); err != nil && err != syscall.EPERM {
			return pid
		}
	}
	t.Fatal("no dead pid")
	return 0
}
