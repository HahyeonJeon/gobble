//go:build live

package install_e2e_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type asyncCommand struct {
	cmd    *exec.Cmd
	done   chan commandResult
	mu     sync.Mutex
	result *commandResult
}

type identityDocument struct {
	SchemaVersion int            `json:"schema_version"`
	View          string         `json:"view"`
	Match         bool           `json:"match"`
	Required      identityFields `json:"required"`
	Have          identityFields `json:"have"`
}

type identityFields struct {
	GobbleVCSRevision      string `json:"gobble_vcs_revision"`
	GobbleExecutableSHA256 string `json:"gobble_executable_sha256"`
	PipelineVCSRevision    string `json:"pipeline_vcs_revision"`
	GOOS                   string `json:"goos"`
	GOARCH                 string `json:"goarch"`
	InstallKind            string `json:"install_kind"`
	IdentityMode           string `json:"identity_mode"`
}

type instanceView struct {
	Identity string `json:"identity"`
	Status   string `json:"status"`
	Executor string `json:"executor"`
}

type remainingView struct {
	Identity  string `json:"identity"`
	Remaining bool   `json:"remaining"`
}

func startCommand(t *testing.T, executable, cwd string, env []string, args ...string) *asyncCommand {
	t.Helper()
	cmd := exec.Command(executable, args...)
	cmd.Dir = cwd
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start %s %v: %v", executable, args, err)
	}
	async := &asyncCommand{cmd: cmd, done: make(chan commandResult, 1)}
	go func() {
		err := cmd.Wait()
		async.done <- commandResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), Code: exitCode(err), Err: err}
	}()
	t.Cleanup(func() { async.cleanup() })
	return async
}

func (a *asyncCommand) poll() (commandResult, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.result != nil {
		return *a.result, true
	}
	select {
	case result := <-a.done:
		a.result = &result
		return result, true
	default:
		return commandResult{}, false
	}
}

func (a *asyncCommand) wait(t *testing.T, timeout time.Duration) commandResult {
	t.Helper()
	if result, ok := a.poll(); ok {
		return result
	}
	select {
	case result := <-a.done:
		a.mu.Lock()
		a.result = &result
		a.mu.Unlock()
		return result
	case <-time.After(timeout):
		_ = a.cmd.Process.Kill()
		t.Fatalf("timed out waiting for pid %d", a.cmd.Process.Pid)
		return commandResult{}
	}
}

func (a *asyncCommand) cleanup() {
	if _, ok := a.poll(); ok {
		return
	}
	_ = a.cmd.Process.Signal(os.Interrupt)
	select {
	case result := <-a.done:
		a.mu.Lock()
		a.result = &result
		a.mu.Unlock()
	case <-time.After(20 * time.Second):
		_ = a.cmd.Process.Kill()
		select {
		case result := <-a.done:
			a.mu.Lock()
			a.result = &result
			a.mu.Unlock()
		case <-time.After(5 * time.Second):
		}
	}
}

func waitForRunningDocker(t *testing.T, running *asyncCommand, inspect func() commandResult) string {
	t.Helper()
	deadline := time.Now().Add(4 * time.Minute)
	var last commandResult
	for time.Now().Before(deadline) {
		if result, done := running.poll(); done {
			t.Fatalf("run ended before Docker was observed running: code=%d err=%v\nstdout: %s\nstderr: %s\nlast inspect: code=%d stdout=%s stderr=%s", result.Code, result.Err, result.Stdout, result.Stderr, last.Code, last.Stdout, last.Stderr)
		}
		last = inspect()
		if last.Code == 0 {
			for _, record := range decodeJSONL[instanceView](t, last.Stdout, "inspect instances") {
				if record.Executor == "docker" && record.Status == "running" {
					return record.Identity
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for running Docker task; last inspect code=%d stdout=%s stderr=%s", last.Code, last.Stdout, last.Stderr)
	return ""
}

func interruptRunAndRequireRemaining(t *testing.T, running *asyncCommand, inspect func(string) commandResult) []string {
	t.Helper()
	if err := running.cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("signal running command: %v", err)
	}
	result := running.wait(t, 90*time.Second)
	requireStructuredFailure(t, result, "canceled Run", 1, "canceled", "")
	remainingResult := requireCommand(t, inspect("remaining"), "inspect remaining after cancellation")
	records := decodeJSONL[remainingView](t, remainingResult.Stdout, "inspect remaining after cancellation")
	var remaining []string
	for _, record := range records {
		if record.Remaining {
			remaining = append(remaining, record.Identity)
		}
	}
	if len(remaining) == 0 {
		t.Fatalf("canceled Run has empty remaining work: %s", remainingResult.Stdout)
	}
	return remaining
}

func requireRemainingEmpty(t *testing.T, inspect func(string) commandResult) {
	t.Helper()
	result := requireCommand(t, inspect("remaining"), "inspect remaining after Resume")
	for _, record := range decodeJSONL[remainingView](t, result.Stdout, "inspect remaining after Resume") {
		if record.Remaining {
			t.Fatalf("Resume left remaining identity %s: %s", record.Identity, result.Stdout)
		}
	}
}

func requireSuccessfulDocker(t *testing.T, inspect func(string) commandResult) {
	t.Helper()
	result := requireCommand(t, inspect("instances"), "inspect instances after Resume")
	found := false
	for _, record := range decodeJSONL[instanceView](t, result.Stdout, "inspect instances after Resume") {
		if record.Executor == "docker" && record.Status == "succeeded" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Resume has no successful Docker instance: %s", result.Stdout)
	}
}

func decodeJSONL[T any](t *testing.T, data []byte, operation string) []T {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(data))
	var records []T
	for {
		var record T
		err := decoder.Decode(&record)
		if errors.Is(err, io.EOF) {
			return records
		}
		if err != nil {
			t.Fatalf("%s JSONL: %v\n%s", operation, err, data)
		}
		records = append(records, record)
	}
}

func inspectIdentity(t *testing.T, result commandResult, operation string) identityDocument {
	t.Helper()
	requireCommand(t, result, operation)
	var identity identityDocument
	if err := json.Unmarshal(result.Stdout, &identity); err != nil {
		t.Fatalf("%s JSON: %v\n%s", operation, err, result.Stdout)
	}
	if identity.View != "identity" || identity.Required.GOOS == "" || identity.Required.GOARCH == "" || identity.Required.IdentityMode == "" || identity.Have.GOOS == "" || identity.Have.GOARCH == "" || identity.Have.IdentityMode == "" {
		t.Fatalf("%s omits identity match fields: %#v", operation, identity)
	}
	return identity
}

func requireNoGoInvocation(t *testing.T, marker, operation string) {
	t.Helper()
	if data, err := os.ReadFile(marker); err == nil {
		t.Fatalf("%s invoked go stub: %s", operation, data)
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read go stub marker after %s: %v", operation, err)
	}
}

func requireWGSOutputs(t *testing.T, workspace string) {
	t.Helper()
	for _, rel := range []string{
		"results/wgs/multiqc/multiqc_report.html",
		"results/wgs/samples/patient1/testN/alignment/testN.recalibrated.bam",
		"results/wgs/samples/patient2/testT/alignment/testT.recalibrated.bam",
		"results/wgs/joint/joint_germline.vcf.gz",
	} {
		path := filepath.Join(workspace, filepath.FromSlash(rel))
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			t.Fatalf("published WGS output %s: info=%v err=%v", path, info, err)
		}
	}
}

func runInstalledWGSRecovery(t *testing.T, a *assay, workspace string) []string {
	t.Helper()
	stageWGSWorkspace(t, a, workspace)
	running := startCommand(t, a.gobble, a.consumer, a.agentEnv, "run", "./wgs", "--workspace", workspace, "--cap", "1")
	neutralDir := t.TempDir()
	inspect := func(view string) commandResult {
		return runCommand(a.gobble, neutralDir, a.neutralEnv, "inspect", view, "--workspace", workspace)
	}
	waitForRunningDocker(t, running, func() commandResult { return inspect("instances") })
	remaining := interruptRunAndRequireRemaining(t, running, inspect)
	release := requireCommand(t, runCommand(a.gobble, neutralDir, a.neutralEnv, "release", "--workspace", workspace), "installed Release")
	if string(release.Stdout) != "{\"op\":\"release\"}\n" {
		t.Fatalf("installed Release stdout = %q", release.Stdout)
	}
	requireNoGoInvocation(t, a.goStubMarker, "installed Inspect/Release")
	resume := requireCommand(t, runCommand(a.gobble, a.consumer, a.agentEnv, "resume", "./wgs", "--workspace", workspace, "--cap", "1"), "installed Resume")
	if string(resume.Stdout) != "{\"op\":\"resume\"}\n" {
		t.Fatalf("installed Resume stdout = %q", resume.Stdout)
	}
	requireRemainingEmpty(t, inspect)
	requireSuccessfulDocker(t, inspect)
	requireWGSOutputs(t, workspace)
	requireNoGoInvocation(t, a.goStubMarker, "installed final Inspect")
	return remaining
}

func runPackedWGSRecovery(t *testing.T, a *assay, workspace string) []string {
	t.Helper()
	stageWGSWorkspace(t, a, workspace)
	neutralDir := t.TempDir()
	running := startCommand(t, a.packedWGS, neutralDir, a.neutralEnv, "run", "--workspace", workspace, "--cap", "1")
	inspect := func(view string) commandResult {
		return runCommand(a.packedWGS, neutralDir, a.neutralEnv, "inspect", view, "--workspace", workspace)
	}
	waitForRunningDocker(t, running, func() commandResult { return inspect("instances") })
	remaining := interruptRunAndRequireRemaining(t, running, inspect)
	return remaining
}

func finishPackedWGSRecovery(t *testing.T, a *assay, workspace string, inspect func(string) commandResult) {
	t.Helper()
	release := requireCommand(t, runCommand(a.packedWGS, t.TempDir(), a.neutralEnv, "release", "--workspace", workspace), "packed Release")
	if string(release.Stdout) != "{\"op\":\"release\"}\n" {
		t.Fatalf("packed Release stdout = %q", release.Stdout)
	}
	resume := requireCommand(t, runCommand(a.packedWGS, t.TempDir(), a.neutralEnv, "resume", "--workspace", workspace, "--cap", "1"), "packed Resume")
	if string(resume.Stdout) != "{\"op\":\"resume\"}\n" {
		t.Fatalf("packed Resume stdout = %q", resume.Stdout)
	}
	requireRemainingEmpty(t, inspect)
	requireSuccessfulDocker(t, inspect)
	requireWGSOutputs(t, workspace)
	requireNoGoInvocation(t, a.goStubMarker, "packed runtime")
}
