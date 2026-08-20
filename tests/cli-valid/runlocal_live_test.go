//go:build live

package cli_valid_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

const runLocalPkg = "./tests/cli-valid/runlocal"

func requireDocker(t *testing.T) {
	t.Helper()
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Fatalf("docker info: %v", err)
	}
}

func TestRunLocalCLIRecover(t *testing.T) {
	requireDocker(t)
	bin := buildGobble(t)
	dir := t.TempDir()
	stageRunLocalInput(t, dir)

	compose := runGobble(t, bin, "compose", runLocalPkg)
	requireSuccess(t, compose)
	if string(compose.stdout) != "{\"op\":\"compose\",\"pipeline\":\"run-local\"}\n" {
		t.Fatalf("compose stdout = %q, want {\"op\":\"compose\",\"pipeline\":\"run-local\"}\\n", compose.stdout)
	}

	validate := runGobble(t, bin, "validate", runLocalPkg)
	requireSuccess(t, validate)
	if string(validate.stdout) != "{\"op\":\"validate\"}\n" {
		t.Fatalf("validate stdout = %q, want {\"op\":\"validate\"}\\n", validate.stdout)
	}

	plan := runGobble(t, bin, "plan", runLocalPkg)
	requireSuccess(t, plan)
	wantPlan := readGolden(t, "testdata/run-local/plan.json")
	if !bytes.Equal(plan.stdout, wantPlan) {
		t.Fatalf("plan stdout != testdata/run-local/plan.json\ngot:\n%s\nwant:\n%s", plan.stdout, wantPlan)
	}

	run := runGobble(t, bin, "run", runLocalPkg, "--workspace", dir, "--cap", "2")
	requireSuccess(t, run)
	if string(run.stdout) != "{\"op\":\"run\"}\n" {
		t.Fatalf("run stdout = %q, want {\"op\":\"run\"}\\n", run.stdout)
	}

	requireFixtureText(t, filepath.Join(dir, "out", "docker", "sample.txt"))
	requireFixtureText(t, filepath.Join(dir, "out", "process", "sample.txt"))
	pwd, err := os.ReadFile(filepath.Join(dir, "out", "docker", "pwd.txt"))
	if err != nil {
		t.Fatalf("published container cwd: %v", err)
	}
	if strings.TrimSpace(string(pwd)) != "/work" {
		t.Fatalf("container cwd got %q, want /work", pwd)
	}

	requireRemainingEmpty(t, bin, dir)

	occupied := runGobble(t, bin, "run", runLocalPkg, "--workspace", dir, "--cap", "2")
	requireOccupiedWorkspace(t, occupied)

	inspectRun := runGobble(t, bin, "inspect", "run", "--workspace", dir)
	requireSuccess(t, inspectRun)
	if !occupancyActive(inspectRun.stdout) {
		t.Fatalf("inspect run occupancy inactive: %s", inspectRun.stdout)
	}

	releaseWorkspace(t, bin, dir)

	resume := runGobble(t, bin, "resume", runLocalPkg, "--workspace", dir, "--cap", "2")
	requireSuccess(t, resume)
	if string(resume.stdout) != "{\"op\":\"resume\"}\n" {
		t.Fatalf("resume stdout = %q, want {\"op\":\"resume\"}\\n", resume.stdout)
	}

	requireRemainingEmpty(t, bin, dir)
	requireReuseIdentityMatched(t, bin, dir)
}

func stageRunLocalInput(t *testing.T, workspace string) {
	t.Helper()
	data := readGolden(t, "testdata/run-local/in/sample.txt")
	dst := filepath.Join(workspace, "in", "sample.txt")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(dst), err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", dst, err)
	}
}

func requireSuccess(t *testing.T, res gobbleResult) {
	t.Helper()
	if res.code != 0 {
		t.Fatalf("exit = %d, want 0\nstderr: %s", res.code, res.stderr)
	}
	if len(res.stderr) != 0 {
		t.Fatalf("stderr = %q, want empty", res.stderr)
	}
}

func requireFixtureText(t *testing.T, path string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	if string(got) != "fixture\n" && string(got) != "fixture" {
		t.Fatalf("%s got %q, want fixture", path, got)
	}
}

func requireRemainingEmpty(t *testing.T, bin, workspace string) {
	t.Helper()
	res := runGobble(t, bin, "inspect", "remaining", "--workspace", workspace)
	requireSuccess(t, res)
	if len(res.stdout) != 0 {
		t.Fatalf("inspect remaining stdout = %q, want empty", res.stdout)
	}
}

func requireReuseIdentityMatched(t *testing.T, bin, workspace string) {
	t.Helper()
	res := runGobble(t, bin, "inspect", "reuse", "--workspace", workspace)
	requireSuccess(t, res)
	recs := decodeJSONL(t, res.stdout)
	if len(recs) == 0 {
		t.Fatalf("inspect reuse empty, want reused identities")
	}
	for _, rec := range recs {
		if rec["decision"] != "reused" || rec["reason"] != "reused-identity-matched" {
			t.Fatalf("inspect reuse got %#v, want reused / reused-identity-matched", rec)
		}
	}
}

func requireOccupiedWorkspace(t *testing.T, res gobbleResult) {
	t.Helper()
	if len(res.stdout) != 0 {
		t.Fatalf("stdout = %q, want empty", res.stdout)
	}
	if res.code != 1 {
		t.Fatalf("exit = %d, want 1\nstderr: %s", res.code, res.stderr)
	}
	ge := decodeCLIError(t, res)
	if ge.Op != "run" {
		t.Fatalf("op = %q, want run", ge.Op)
	}
	for _, d := range ge.Defects {
		if d.Code == gobble.DefectOccupiedWorkspace {
			return
		}
	}
	t.Fatalf("defects = %#v, want occupied-workspace", ge.Defects)
}

func occupancyActive(data []byte) bool {
	var run struct {
		Occupancy *struct {
			Active bool `json:"active"`
		} `json:"occupancy"`
	}
	if json.Unmarshal(data, &run) != nil || run.Occupancy == nil {
		return false
	}
	return run.Occupancy.Active
}

func decodeJSONL(t *testing.T, data []byte) []map[string]any {
	t.Helper()
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	var out []map[string]any
	dec := json.NewDecoder(bytes.NewReader(data))
	for dec.More() {
		var rec map[string]any
		if err := dec.Decode(&rec); err != nil {
			t.Fatalf("JSONL: %v\n%s", err, data)
		}
		out = append(out, rec)
	}
	return out
}

func decodeCLIError(t *testing.T, res gobbleResult) gobble.Error {
	t.Helper()
	var ge gobble.Error
	if err := json.Unmarshal(res.stderr, &ge); err != nil {
		t.Fatalf("stderr JSON: %v\n%s", err, res.stderr)
	}
	return ge
}

func releaseWorkspace(t *testing.T, bin, workspace string) {
	t.Helper()
	res := runGobble(t, bin, "release", "--workspace", workspace)
	if res.code == 0 {
		if len(res.stderr) != 0 {
			t.Fatalf("release stderr = %q, want empty", res.stderr)
		}
		if string(res.stdout) != "{\"op\":\"release\"}\n" {
			t.Fatalf("release stdout = %q, want {\"op\":\"release\"}\\n", res.stdout)
		}
		return
	}
	ge := decodeCLIError(t, res)
	if !hasDefect(ge, gobble.DefectLiveOccupancy) {
		t.Fatalf("release exit = %d, want 0\nstderr: %s", res.code, res.stderr)
	}
	forceDeadOwner(t, workspace)
	res = runGobble(t, bin, "release", "--workspace", workspace)
	requireSuccess(t, res)
	if string(res.stdout) != "{\"op\":\"release\"}\n" {
		t.Fatalf("release stdout = %q, want {\"op\":\"release\"}\\n", res.stdout)
	}
}

func hasDefect(ge gobble.Error, code gobble.DefectCode) bool {
	for _, d := range ge.Defects {
		if d.Code == code {
			return true
		}
	}
	return false
}

func forceDeadOwner(t *testing.T, workspace string) {
	t.Helper()
	path := filepath.Join(workspace, ".gobble", "run.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	var run map[string]any
	if err := json.Unmarshal(data, &run); err != nil {
		t.Fatalf("Unmarshal run.json: %v", err)
	}
	occ, _ := run["occupancy"].(map[string]any)
	if occ == nil {
		occ = map[string]any{"active": true}
		run["occupancy"] = occ
	}
	host, err := os.Hostname()
	if err != nil {
		t.Fatalf("Hostname() error = %v", err)
	}
	occ["active"] = true
	occ["host"] = host
	occ["pid"] = deadPID(t)
	out, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
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
