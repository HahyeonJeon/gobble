package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReleaseMissingRun(t *testing.T) {
	dir := t.TempDir()
	defects := Release(dir)
	if !hasDefect(defects, DefectNotFound, "") {
		t.Fatalf("Release() defects %v, want not-found", defects)
	}
}

func TestReleaseAlreadyReleased(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, ControlDir, RunIdentityFile), closedRunJSON("run-1"))
	before := snapshotDir(t, dir)
	defects := Release(dir)
	if !hasDefect(defects, DefectAlreadyReleased, "") {
		t.Fatalf("Release() defects %v, want already-released", defects)
	}
	after := snapshotDir(t, dir)
	if before != after {
		t.Fatalf("already-released mutated workspace")
	}
}

func TestReleaseLiveOccupancy(t *testing.T) {
	dir := t.TempDir()
	host, err := currentHost()
	if err != nil {
		t.Fatal(err)
	}
	writeOccupancy(t, dir, jsonOccupancy{Active: true, Host: host, PID: os.Getpid(), Started: "2026-01-01T00:00:00Z"})
	before := snapshotDir(t, dir)
	defects := Release(dir)
	if !hasDefect(defects, DefectLiveOccupancy, "") {
		t.Fatalf("Release() defects %v, want live-occupancy", defects)
	}
	after := snapshotDir(t, dir)
	if before != after {
		t.Fatalf("live-occupancy mutated workspace")
	}
}

func TestReleaseForeignHost(t *testing.T) {
	dir := t.TempDir()
	writeOccupancy(t, dir, jsonOccupancy{Active: true, Host: "other-host", PID: deadPID(t), Started: "2026-01-01T00:00:00Z"})
	before := snapshotDir(t, dir)
	defects := Release(dir)
	if !hasDefect(defects, DefectForeignHost, "") {
		t.Fatalf("Release() defects %v, want foreign-host", defects)
	}
	after := snapshotDir(t, dir)
	if before != after {
		t.Fatalf("foreign-host mutated workspace")
	}
}

func TestReleaseUnsupportedSchema(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, ControlDir, RunIdentityFile), `{
  "schema_version": 1,
  "id": "run-1",
  "status": "running",
  "started": "2026-01-01T00:00:00Z",
  "occupancy": {"active": true, "host": "h", "pid": 1}
}
`)
	before := snapshotDir(t, dir)
	defects := Release(dir)
	if !hasDefect(defects, DefectUnsupportedSchema, "") {
		t.Fatalf("Release() defects %v, want unsupported-schema", defects)
	}
	after := snapshotDir(t, dir)
	if before != after {
		t.Fatalf("unsupported-schema mutated workspace")
	}
}

func TestReleaseDeadOwnerMarksIncomplete(t *testing.T) {
	dir := t.TempDir()
	host, err := currentHost()
	if err != nil {
		t.Fatal(err)
	}
	writeOccupancy(t, dir, jsonOccupancy{Active: true, Host: host, PID: deadPID(t), Started: "2026-01-01T00:00:00Z"})
	writeCheckFile(t, filepath.Join(dir, "out", "keep.txt"), "keep")
	writeCheckFile(t, filepath.Join(dir, ControlDir, TasksFile), `{
  "schema_version": 2,
  "tasks": [
    {
      "id": "copy",
      "instance": "",
      "shard_index": 0,
      "shard_count": 1,
      "attempt": 1,
      "status": "running",
      "executor": "process",
      "image": "",
      "command": ["cp"],
      "resources": {"cpu": 0, "memory": ""},
      "params": [],
      "reason": "ready"
    }
  ]
}
`)
	if defects := Release(dir); len(defects) != 0 {
		t.Fatalf("Release() defects %v, want none", defects)
	}
	if _, err := os.Stat(filepath.Join(dir, ControlDir, RunIdentityFile)); err != nil {
		t.Fatalf("Release deleted run.json: %v", err)
	}
	if _, err := os.ReadFile(filepath.Join(dir, "out", "keep.txt")); err != nil {
		t.Fatalf("Release deleted artifact: %v", err)
	}
	run, exists, err := readRunIdentity(dir)
	if err != nil || !exists {
		t.Fatalf("run.json exists=%v err=%v", exists, err)
	}
	if run.SchemaVersion != SchemaVersion {
		t.Fatalf("released schema_version got %d, want %d", run.SchemaVersion, SchemaVersion)
	}
	if occupancyIsActive(run) {
		t.Fatalf("occupancy still active after Release")
	}
	if run.Occupancy == nil || run.Occupancy.Closed == "" {
		t.Fatalf("missing close audit: %#v", run.Occupancy)
	}
	if run.Status != StatusFailed {
		t.Fatalf("run status got %q, want failed", run.Status)
	}
	st := taskStates(t, dir)["copy"]
	if st.Status != StatusIncomplete {
		t.Fatalf("copy status got %q, want incomplete", st.Status)
	}
	if st.Ended == "" {
		t.Fatal("incomplete task end empty")
	}
}

func TestReleaseSchema0And1Unsupported(t *testing.T) {
	for _, ver := range []int{0, 1} {
		dir := t.TempDir()
		body := `{"id":"run-1"}`
		if ver == 1 {
			body = `{
  "schema_version": 1,
  "id": "run-1",
  "status": "running",
  "occupancy": {"active": true, "host": "h", "pid": 1}
}
`
		}
		writeCheckFile(t, filepath.Join(dir, ControlDir, RunIdentityFile), body)
		before := snapshotDir(t, dir)
		defects := Release(dir)
		if !hasDefect(defects, DefectUnsupportedSchema, "") {
			t.Fatalf("schema %d: Release() defects %v, want unsupported-schema", ver, defects)
		}
		after := snapshotDir(t, dir)
		if before != after {
			t.Fatalf("schema %d: Release mutated workspace", ver)
		}
	}
}

func TestReleaseMarksIncompleteWithSlots(t *testing.T) {
	dir := t.TempDir()
	host, err := currentHost()
	if err != nil {
		t.Fatal(err)
	}
	writeOccupancy(t, dir, jsonOccupancy{Active: true, Host: host, PID: deadPID(t), Started: "2026-01-01T00:00:00Z"})
	writeCheckFile(t, filepath.Join(dir, ControlDir, TasksFile), `{
  "schema_version": 2,
  "tasks": [
    {
      "id": "copy",
      "instance": "",
      "shard_index": 0,
      "shard_count": 1,
      "attempt": 1,
      "status": "running",
      "executor": "process",
      "image": "",
      "command": ["true"],
      "resources": {"cpu": 0, "memory": ""},
      "params": []
    },
    {
      "id": "ok",
      "instance": "",
      "shard_index": 0,
      "shard_count": 1,
      "attempt": 1,
      "status": "succeeded",
      "executor": "process",
      "image": "",
      "command": ["true"],
      "resources": {"cpu": 0, "memory": ""},
      "params": []
    }
  ]
}
`)
	if defects := Release(dir); len(defects) != 0 {
		t.Fatalf("Release() defects %v, want none", defects)
	}
	raw := mustJSONFile(t, filepath.Join(dir, ControlDir, TasksFile))
	var file jsonTasksFile
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("tasks.json: %v", err)
	}
	if file.SchemaVersion != SchemaVersion {
		t.Fatalf("schema_version got %d, want %d", file.SchemaVersion, SchemaVersion)
	}
	if len(file.Tasks) != 2 {
		t.Fatalf("tasks got %d, want 2", len(file.Tasks))
	}
	byID := map[string]jsonTaskState{}
	for _, st := range file.Tasks {
		byID[st.ID] = st
		if st.Instance != "" || st.ShardIndex != DefaultShardIndex || st.ShardCount != DefaultShardCount || st.Attempt != DefaultAttempt {
			t.Fatalf("%s slots got instance %q shard %d/%d attempt %d, want empty/0/1/1",
				st.ID, st.Instance, st.ShardIndex, st.ShardCount, st.Attempt)
		}
	}
	if byID["copy"].Status != StatusIncomplete {
		t.Fatalf("copy status got %q, want incomplete", byID["copy"].Status)
	}
	if byID["ok"].Status != StatusSucceeded {
		t.Fatalf("ok status got %q, want succeeded", byID["ok"].Status)
	}
}

func TestReleaseSucceededRunKeepsSucceeded(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	if defects := Run(Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	}); len(defects) != 0 {
		t.Fatalf("Run() defects %v, want none", defects)
	}
	forceDeadOwner(t, dir)
	if defects := Release(dir); len(defects) != 0 {
		t.Fatalf("Release() defects %v, want none", defects)
	}
	run, exists, err := readRunIdentity(dir)
	if err != nil || !exists {
		t.Fatalf("run.json exists=%v err=%v", exists, err)
	}
	if occupancyIsActive(run) {
		t.Fatal("occupancy still active")
	}
	if run.Status != StatusSucceeded {
		t.Fatalf("run status got %q, want succeeded", run.Status)
	}
	if taskStates(t, dir)["copy"].Status != StatusSucceeded {
		t.Fatalf("copy status got %q, want succeeded", taskStates(t, dir)["copy"].Status)
	}
}

func writeOccupancy(t *testing.T, workspace string, occ jsonOccupancy) {
	t.Helper()
	run := jsonRun{
		SchemaVersion: SchemaVersion,
		ID:            "run-1",
		Status:        StatusRunning,
		Started:       "2026-01-01T00:00:00Z",
		Occupancy:     &occ,
	}
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeCheckFile(t, filepath.Join(workspace, ControlDir, RunIdentityFile), string(append(data, '\n')))
}
