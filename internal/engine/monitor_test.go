package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMonitorUsesGlobalLatestSnapshotWithSelectedLiveLogs(t *testing.T) {
	dir := t.TempDir()
	doc := Document{Name: "monitor", Tasks: []TaskPlan{
		{ID: "a", Name: "align", Display: &Display{Samples: []string{"S01"}}, Command: []string{"true"}},
		{ID: "b", Name: "report", Command: []string{"true"}},
	}, Edges: []Edge{{FromTask: "a", FromPort: "out", ToTask: "b", ToPort: "in"}}}
	plan, err := marshalControlPlan(doc, "")
	if err != nil {
		t.Fatal(err)
	}
	writeCheckFile(t, filepath.Join(dir, ControlDir, PlanFile), string(plan))
	writeOccupancy(t, dir, jsonOccupancy{Active: false})
	states := []jsonTaskState{
		{ID: "a", Attempt: 1, Status: StatusFailed, Executor: executorProcess, Command: []string{"true"}},
		{ID: "a", Attempt: 2, Status: StatusRunning, Executor: executorProcess, Command: []string{"true"}, Started: "2026-01-01T00:00:00Z"},
		{ID: "b", Attempt: 1, Status: StatusNotStarted, Executor: executorProcess, Command: []string{"true"}},
	}
	data, err := json.Marshal(jsonTasksFile{SchemaVersion: SchemaVersion, Tasks: states})
	if err != nil {
		t.Fatal(err)
	}
	writeCheckFile(t, filepath.Join(dir, ControlDir, TasksFile), string(data))
	stdout, _ := taskLogPaths(states[1])
	writeCheckFile(t, filepath.Join(dir, stdout), strings.Repeat("x", 5000)+"live output")
	before := snapshotDir(t, dir)
	raw, defects := Inspect(dir, viewMonitor, "a", testInstallIdentity())
	if len(defects) != 0 {
		t.Fatal(defects)
	}
	var got monitorDoc
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Pipeline != "monitor" || len(got.Tasks) != 2 || got.Tasks[0].Attempt != 2 || len(got.Edges) != 1 {
		t.Fatalf("snapshot: %+v", got)
	}
	if got.Tasks[0].Display == nil || got.Tasks[0].Display.Samples[0] != "S01" {
		t.Fatal("display lost in plan round trip")
	}
	if len(got.Logs) != 1 || len(got.Logs[0].StdoutTail) != inspectLogTail || !strings.HasSuffix(got.Logs[0].StdoutTail, "live output") {
		t.Fatalf("live tail: %+v", got.Logs)
	}
	if snapshotDir(t, dir) != before {
		t.Fatal("monitor mutated workspace")
	}
	raw, defects = Inspect(dir, viewMonitor, "", testInstallIdentity())
	if len(defects) != 0 {
		t.Fatal(defects)
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Logs) != 0 {
		t.Fatal("global refresh read all logs")
	}
	identity := *testInstallIdentity()
	identity.GobbleExecutableSHA256 = strings.Repeat("b", 64)
	if _, defects := Inspect(dir, viewMonitor, "", &identity); len(defects) == 0 {
		t.Fatal("identity mismatch accepted")
	}
	if _, defects := Inspect(dir, viewMonitor, "missing", testInstallIdentity()); len(defects) == 0 {
		t.Fatal("unknown instance accepted")
	}
	// Derived live pointers must honor the same symlink containment gate.
	if err := os.Remove(filepath.Join(dir, stdout)); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	writeCheckFile(t, outside, "private")
	if err := os.Symlink(outside, filepath.Join(dir, stdout)); err != nil {
		t.Fatal(err)
	}
	if _, defects := Inspect(dir, viewMonitor, "a", testInstallIdentity()); len(defects) == 0 {
		t.Fatal("outside live log accepted")
	}
}

func TestMonitorRejectsMixedControlRevisions(t *testing.T) {
	dir := t.TempDir()
	writeOccupancy(t, dir, jsonOccupancy{Active: false})
	data, err := json.Marshal(jsonTasksFile{SchemaVersion: SchemaVersion, Snapshot: "different", Tasks: []jsonTaskState{{ID: "a", Status: StatusNotStarted}}})
	if err != nil {
		t.Fatal(err)
	}
	writeCheckFile(t, filepath.Join(dir, ControlDir, TasksFile), string(data))
	before := snapshotDir(t, dir)
	if _, defects := Inspect(dir, viewMonitor, "", testInstallIdentity()); len(defects) == 0 {
		t.Fatal("mixed revisions accepted")
	}
	if snapshotDir(t, dir) != before {
		t.Fatal("failed read mutated workspace")
	}
}
