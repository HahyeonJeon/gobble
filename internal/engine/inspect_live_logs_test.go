package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/internal/testutil"
)

func TestInspectLogsShowsLiveTailOfStartedAttempt(t *testing.T) {
	dir := t.TempDir()
	doc := Document{Name: "live-logs", Tasks: []TaskPlan{
		{ID: "a", Name: "align", Command: []string{"true"}},
		{ID: "b", Name: "report", Command: []string{"true"}},
	}, Edges: []Edge{{FromTask: "a", FromPort: "out", ToTask: "b", ToPort: "in"}}}
	plan, err := marshalControlPlan(doc, "")
	if err != nil {
		t.Fatal(err)
	}
	writeCheckFile(t, testutil.ControlPath(t, filepath.Join(dir, ControlDir, PlanFile)), string(plan))
	writeOccupancy(t, dir, jsonOccupancy{Active: false})
	states := []jsonTaskState{
		{ID: "a", Attempt: 1, Status: StatusRunning, Executor: executorProcess, Command: []string{"true"}, Started: "2026-01-01T00:00:00Z"},
		{ID: "b", Attempt: 1, Status: StatusNotStarted, Executor: executorProcess, Command: []string{"true"}},
	}
	data, err := json.Marshal(jsonTasksFile{SchemaVersion: SchemaVersion, Tasks: states})
	if err != nil {
		t.Fatal(err)
	}
	writeCheckFile(t, testutil.ControlPath(t, filepath.Join(dir, ControlDir, TasksFile)), string(data))
	stdout, stderr := taskLogPaths(states[0])
	if stdout == "" || stderr == "" {
		t.Fatalf("started attempt has no derived log paths: %q %q", stdout, stderr)
	}
	if out, errPath := taskLogPaths(states[1]); out != "" || errPath != "" {
		t.Fatalf("unstarted attempt got log paths: %q %q", out, errPath)
	}
	writeCheckFile(t, filepath.Join(dir, stdout), strings.Repeat("x", 5000)+"live output")
	before := snapshotDir(t, dir)

	raw, defects := Inspect(dir, viewLogs, "", testInstallIdentity())
	if len(defects) != 0 {
		t.Fatal(defects)
	}
	var got inspectLogsDoc
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Logs) != 2 {
		t.Fatalf("logs: %+v", got.Logs)
	}
	live := got.Logs[0]
	if live.Stdout != stdout || live.StdoutSize != 5011 || len(live.StdoutTail) != inspectLogTail || !strings.HasSuffix(live.StdoutTail, "live output") {
		t.Fatalf("live tail: %+v", live)
	}
	if live.StderrSize != 0 || live.StderrTail != "" {
		t.Fatalf("absent stderr should stay empty: %+v", live)
	}
	if idle := got.Logs[1]; idle.Stdout != "" || idle.StdoutSize != 0 {
		t.Fatalf("unstarted attempt: %+v", idle)
	}
	if snapshotDir(t, dir) != before {
		t.Fatal("inspect logs mutated workspace")
	}

	// Derived live pointers honor the same symlink containment gate as published ones.
	if err := os.Remove(filepath.Join(dir, stdout)); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	writeCheckFile(t, outside, "private")
	if err := os.Symlink(outside, filepath.Join(dir, stdout)); err != nil {
		t.Fatal(err)
	}
	if _, defects := Inspect(dir, viewLogs, "", testInstallIdentity()); len(defects) == 0 {
		t.Fatal("outside live log accepted")
	}
}
