package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestReleaseMissingRun(t *testing.T) {
	dir := t.TempDir()
	libErr := gobble.Release(dir)
	res := runCLI("release", "--workspace", dir)
	requireDomainError(t, res, libErr)
	if _, err := os.Stat(filepath.Join(dir, ".gobble")); !os.IsNotExist(err) {
		t.Fatalf("release created .gobble")
	}
}

func TestReleaseSuccess(t *testing.T) {
	dir := occupiedWorkspace(t)
	forceDeadOwner(t, dir)
	res := runCLI("release", "--workspace", dir)
	if res.code != 0 {
		t.Fatalf("exit = %d\nstderr: %s", res.code, res.stderr)
	}
	if len(res.stderr) != 0 {
		t.Fatalf("stderr = %q, want empty", res.stderr)
	}
	if string(res.stdout) != "{\"op\":\"release\"}\n" {
		t.Fatalf("stdout = %q, want {\"op\":\"release\"}\\n", res.stdout)
	}
	var raw map[string]any
	if err := json.Unmarshal(res.stdout, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["schema_version"]; ok {
		t.Fatalf("release JSON has schema_version: %#v", raw)
	}
}

func forceDeadOwner(t *testing.T, workspace string) {
	t.Helper()
	path := filepath.Join(workspace, ".gobble", "run.json")
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
	occ["pid"] = deadPID(t)
	out, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func deadPID(t *testing.T) int {
	t.Helper()
	for pid := 1 << 22; pid > 2; pid-- {
		if err := syscall.Kill(pid, 0); err != nil && err != syscall.EPERM {
			return pid
		}
	}
	t.Fatal("no dead pid")
	return 0
}
