package local_e2e_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

type gobbleResult struct {
	stdout []byte
	stderr []byte
	code   int
}

func buildGobble(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gobble")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/gobble")
	cmd.Dir = moduleRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build gobble: %v\n%s", err, out)
	}
	return bin
}

func gobbleCommand(t *testing.T, bin, dir string, args ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(bin, args...)
	if dir == "" {
		dir = moduleRoot(t)
	}
	cmd.Dir = dir
	tmp := t.TempDir()
	env := os.Environ()
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		if strings.HasPrefix(e, "TMPDIR=") {
			continue
		}
		out = append(out, e)
	}
	cmd.Env = append(out, "TMPDIR="+tmp)
	return cmd
}

func runGobble(t *testing.T, bin string, args ...string) gobbleResult {
	t.Helper()
	return runGobbleDir(t, bin, "", args...)
}

func runGobbleDir(t *testing.T, bin, dir string, args ...string) gobbleResult {
	t.Helper()
	cmd := gobbleCommand(t, bin, dir, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return gobbleResult{
		stdout: stdout.Bytes(),
		stderr: stderr.Bytes(),
		code:   exitCode(err),
	}
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return -1
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
	requireSuccess(t, res)
	if string(res.stdout) != "{\"op\":\"release\"}\n" {
		t.Fatalf("release stdout = %q, want {\"op\":\"release\"}\\n", res.stdout)
	}
}

func recoverAfterSuccessCLI(t *testing.T, bin, pkg, workspace string, extra ...string) {
	t.Helper()
	requireRemainingEmpty(t, bin, workspace)

	occupiedArgs := append([]string{"run", pkg, "--workspace", workspace}, extra...)
	occupied := runGobble(t, bin, occupiedArgs...)
	requireOccupiedWorkspace(t, occupied)

	inspectRun := runGobble(t, bin, "inspect", "run", "--workspace", workspace)
	requireSuccess(t, inspectRun)
	if !occupancyActive(inspectRun.stdout) {
		t.Fatalf("inspect run occupancy inactive: %s", inspectRun.stdout)
	}

	releaseWorkspace(t, bin, workspace)

	resumeArgs := append([]string{"resume", pkg, "--workspace", workspace}, extra...)
	resume := runGobble(t, bin, resumeArgs...)
	requireSuccess(t, resume)
	if string(resume.stdout) != "{\"op\":\"resume\"}\n" {
		t.Fatalf("resume stdout = %q, want {\"op\":\"resume\"}\\n", resume.stdout)
	}

	requireRemainingEmpty(t, bin, workspace)
	requireReuseIdentityMatched(t, bin, workspace)
}

func requireCLIOp(t *testing.T, res gobbleResult, want string) {
	t.Helper()
	requireSuccess(t, res)
	if string(res.stdout) != want {
		t.Fatalf("stdout = %q, want %q", res.stdout, want)
	}
}
