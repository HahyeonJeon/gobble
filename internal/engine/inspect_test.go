package engine

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectNoFingerprintsAffectsDownstream(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "a.txt"), "a")
	writeCheckFile(t, filepath.Join(dir, "out", "a.txt"), "a")
	writeCheckFile(t, filepath.Join(dir, "out", "b.txt"), "b")
	depRec := mustFileRecord(t, filepath.Join(dir, "out", "a.txt"), "out/a.txt")
	doc := Document{
		Name: "pipe",
		Tasks: []TaskPlan{
			{
				ID:      "copy",
				Name:    "copy",
				Command: []string{"true"},
				Inputs:  []IO{{Name: "in", Path: "in/a.txt"}},
				Outputs: []IO{{Name: "out", Path: "out/a.txt"}},
			},
			{
				ID:      "dep",
				Name:    "dep",
				Command: []string{"true"},
				Inputs:  []IO{{Name: "in", Path: "out/a.txt"}},
				Outputs: []IO{{Name: "out", Path: "out/b.txt"}},
			},
		},
		Edges: []Edge{{
			FromTask: "copy",
			FromPort: "out",
			ToTask:   "dep",
			ToPort:   "in",
			Wait:     []string{"out/a.txt"},
		}},
	}
	plan, err := marshalControlPlan(doc)
	if err != nil {
		t.Fatal(err)
	}
	writeCheckFile(t, filepath.Join(dir, ControlDir, PlanFile), string(plan))
	writeOccupancy(t, dir, jsonOccupancy{Active: false, Closed: "2026-01-01T00:00:01Z"})
	tasks := jsonTasksFile{
		SchemaVersion: SchemaVersion,
		Tasks: []jsonTaskState{
			{
				ID:       "copy",
				Status:   StatusSucceeded,
				Executor: executorProcess,
				Command:  []string{"true"},
				Attempt:  1,
				Params:   []jsonParam{},
			},
			{
				ID:           "dep",
				Status:       StatusSucceeded,
				Executor:     executorProcess,
				Command:      []string{"true"},
				Attempt:      1,
				Params:       []jsonParam{},
				Fingerprints: []jsonFileHash{depRec},
			},
		},
	}
	taskBytes, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeCheckFile(t, filepath.Join(dir, ControlDir, TasksFile), string(append(taskBytes, '\n')))
	before := snapshotDir(t, dir)
	raw, defects := Inspect(dir, viewRemaining, "")
	if len(defects) != 0 {
		t.Fatalf("Inspect(remaining) defects %v", defects)
	}
	recs := decodeInspectJSONL(t, raw)
	byID := map[string]map[string]any{}
	for _, rec := range recs {
		byID[rec["identity"].(string)] = rec
	}
	if byID["copy"]["remaining"] != false || byID["copy"]["affected"] != true {
		t.Fatalf("copy remaining got %#v", byID["copy"])
	}
	if byID["copy"]["reason"] != reasonIdentityChanged {
		t.Fatalf("copy reason got %#v, want identity-changed", byID["copy"]["reason"])
	}
	differ, _ := byID["copy"]["differing"].([]any)
	if len(differ) != 1 || differ[0] != fingerprintsAbsent {
		t.Fatalf("copy differing got %#v", byID["copy"]["differing"])
	}
	if byID["dep"]["remaining"] != false || byID["dep"]["affected"] != true {
		t.Fatalf("dep remaining got %#v", byID["dep"])
	}
	if byID["dep"]["reason"] != reasonDownstreamOfRerun {
		t.Fatalf("dep reason got %#v, want downstream-of-rerun", byID["dep"]["reason"])
	}
	after := snapshotDir(t, dir)
	if before != after {
		t.Fatalf("Inspect remaining mutated workspace")
	}
}

func TestInspectRemainingInstanceUsesFullSet(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "a.txt"), "a")
	writeCheckFile(t, filepath.Join(dir, "out", "b.txt"), "b")
	inRec := mustFileRecord(t, filepath.Join(dir, "in", "a.txt"), "in/a.txt")
	doc := Document{
		Name: "pipe",
		Tasks: []TaskPlan{
			{
				ID:      "a",
				Name:    "a",
				Command: []string{"false"},
				Inputs:  []IO{{Name: "in", Path: "in/a.txt"}},
				Outputs: []IO{{Name: "out", Path: "out/a.txt"}},
			},
			{
				ID:      "b",
				Name:    "b",
				Command: []string{"true"},
				Inputs:  []IO{{Name: "in", Path: "in/a.txt"}},
				Outputs: []IO{{Name: "out", Path: "out/b.txt"}},
			},
		},
		Edges: []Edge{{
			FromTask: "a",
			FromPort: "out",
			ToTask:   "b",
			ToPort:   "in",
			Wait:     []string{"out/a.txt"},
		}},
	}
	plan, err := marshalControlPlan(doc)
	if err != nil {
		t.Fatal(err)
	}
	writeCheckFile(t, filepath.Join(dir, ControlDir, PlanFile), string(plan))
	writeOccupancy(t, dir, jsonOccupancy{Active: false, Closed: "2026-01-01T00:00:01Z"})
	tasks := jsonTasksFile{
		SchemaVersion: SchemaVersion,
		Tasks: []jsonTaskState{
			{
				ID:       "a",
				Status:   StatusFailed,
				Executor: executorProcess,
				Command:  []string{"false"},
				Attempt:  1,
				Params:   []jsonParam{},
			},
			{
				ID:           "b",
				Status:       StatusSucceeded,
				Executor:     executorProcess,
				Command:      []string{"true"},
				Attempt:      1,
				Params:       []jsonParam{},
				Fingerprints: []jsonFileHash{inRec},
			},
		},
	}
	taskBytes, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeCheckFile(t, filepath.Join(dir, ControlDir, TasksFile), string(append(taskBytes, '\n')))

	allRaw, defects := Inspect(dir, viewRemaining, "")
	if len(defects) != 0 {
		t.Fatalf("Inspect(remaining) defects %v", defects)
	}
	all := remainingByID(t, allRaw)
	if all["a"]["remaining"] != true || all["a"]["affected"] != true {
		t.Fatalf("unfiltered a got %#v", all["a"])
	}
	if all["b"]["remaining"] != false || all["b"]["affected"] != true {
		t.Fatalf("unfiltered b got %#v", all["b"])
	}
	if all["b"]["reason"] != reasonDownstreamOfRerun {
		t.Fatalf("unfiltered b reason got %#v, want downstream-of-rerun", all["b"]["reason"])
	}

	oneRaw, defects := Inspect(dir, viewRemaining, "b")
	if len(defects) != 0 {
		t.Fatalf("Inspect(remaining, b) defects %v", defects)
	}
	one := remainingByID(t, oneRaw)
	if len(one) != 1 || one["b"] == nil {
		t.Fatalf("filtered remaining got %#v, want only b", one)
	}
	if one["b"]["remaining"] != all["b"]["remaining"] || one["b"]["affected"] != all["b"]["affected"] || one["b"]["reason"] != all["b"]["reason"] {
		t.Fatalf("filtered b got %#v, want %#v", one["b"], all["b"])
	}
}

func TestInspectSuccessfulRunNotAffected(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	if defects := Run(t.Context(), Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	raw, defects := Inspect(dir, viewRemaining, "")
	if len(defects) != 0 {
		t.Fatalf("Inspect(remaining) defects %v", defects)
	}
	if len(bytes.TrimSpace(raw)) != 0 {
		t.Fatalf("remaining got %s, want empty", raw)
	}
	timing, defects := Inspect(dir, viewTiming, "")
	if len(defects) != 0 {
		t.Fatalf("Inspect(timing) defects %v", defects)
	}
	var doc inspectTimingDoc
	if err := json.Unmarshal(timing, &doc); err != nil {
		t.Fatalf("timing JSON: %v", err)
	}
	if doc.Started == "" || doc.Ended == "" {
		t.Fatalf("run timing missing: %#v", doc)
	}
	if len(doc.Instances) != 1 || doc.Instances[0].Identity != "copy" || doc.Instances[0].Started == "" || doc.Instances[0].Ended == "" {
		t.Fatalf("instance timing got %#v", doc.Instances)
	}
}

func TestInspectReuseViewReadsDecisions(t *testing.T) {
	dir := t.TempDir()
	writeOccupancy(t, dir, jsonOccupancy{Active: false, Closed: "2026-01-01T00:00:01Z"})
	writeCheckFile(t, filepath.Join(dir, ControlDir, TasksFile), `{
  "schema_version": 2,
  "tasks": [
    {
      "id": "copy",
      "instance": "",
      "shard_index": 0,
      "shard_count": 1,
      "attempt": 1,
      "status": "succeeded",
      "executor": "process",
      "image": "",
      "command": ["true"],
      "resources": {"cpu": 0, "memory": ""},
      "params": [],
      "decision": "reused",
      "reuse_reason": "reused-identity-matched"
    }
  ]
}
`)
	empty, defects := Inspect(dir, viewReuse, "")
	if len(defects) != 0 {
		t.Fatalf("Inspect(reuse) defects %v", defects)
	}
	recs := decodeInspectJSONL(t, empty)
	if len(recs) != 1 || recs[0]["identity"] != "copy" || recs[0]["decision"] != reuseReused {
		t.Fatalf("reuse got %#v", recs)
	}
	if recs[0]["reason"] != reasonReusedIdentityMatched {
		t.Fatalf("reuse reason got %#v", recs[0]["reason"])
	}
}

func TestInspectReuseViewEmptyWithoutDecisions(t *testing.T) {
	dir := t.TempDir()
	writeOccupancy(t, dir, jsonOccupancy{Active: false, Closed: "2026-01-01T00:00:01Z"})
	writeCheckFile(t, filepath.Join(dir, ControlDir, TasksFile), `{
  "schema_version": 2,
  "tasks": [
    {
      "id": "copy",
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
	raw, defects := Inspect(dir, viewReuse, "")
	if len(defects) != 0 {
		t.Fatalf("Inspect(reuse) defects %v", defects)
	}
	if len(bytes.TrimSpace(raw)) != 0 {
		t.Fatalf("reuse got %s, want empty", raw)
	}
}

func TestInspectLogTailBounded(t *testing.T) {
	dir := t.TempDir()
	writeOccupancy(t, dir, jsonOccupancy{Active: true, Host: "h", PID: 1})
	rel := isolateRel(TaskPlan{ID: "copy", Attempt: 1})
	body := strings.Repeat("x", inspectLogTail+64)
	writeCheckFile(t, filepath.Join(dir, filepath.FromSlash(rel), "stdout"), body)
	writeCheckFile(t, filepath.Join(dir, ControlDir, TasksFile), `{
  "schema_version": 2,
  "tasks": [
    {
      "id": "copy",
      "instance": "",
      "shard_index": 0,
      "shard_count": 1,
      "attempt": 1,
      "status": "failed",
      "executor": "process",
      "image": "",
      "command": ["true"],
      "resources": {"cpu": 0, "memory": ""},
      "params": [],
      "stdout": "`+rel+`/stdout",
      "stderr": "`+rel+`/stderr"
    }
  ]
}
`)
	raw, defects := Inspect(dir, viewLogs, "")
	if len(defects) != 0 {
		t.Fatalf("Inspect(logs) defects %v", defects)
	}
	var doc inspectLogsDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("logs JSON: %v", err)
	}
	if len(doc.Logs) != 1 {
		t.Fatalf("logs got %#v", doc.Logs)
	}
	if doc.Logs[0].StdoutSize != int64(len(body)) {
		t.Fatalf("stdout_size got %d, want %d", doc.Logs[0].StdoutSize, len(body))
	}
	if int64(len(doc.Logs[0].StdoutTail)) > inspectLogTail {
		t.Fatalf("stdout_tail length %d exceeds bound %d", len(doc.Logs[0].StdoutTail), inspectLogTail)
	}
	if !strings.HasSuffix(body, doc.Logs[0].StdoutTail) {
		t.Fatalf("stdout_tail is not a suffix of the log file")
	}
}

func TestInspectInstanceSelector(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	if defects := Run(t.Context(), Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	raw, defects := Inspect(dir, viewInstances, "copy")
	if len(defects) != 0 {
		t.Fatalf("Inspect(instances, copy) defects %v", defects)
	}
	recs := decodeInspectJSONL(t, raw)
	if len(recs) != 1 || recs[0]["identity"] != "copy" {
		t.Fatalf("instances copy got %#v", recs)
	}
	_, defects = Inspect(dir, viewRun, "missing")
	if !hasDefect(defects, DefectNotFound, "missing") {
		t.Fatalf("unknown instance defects %v, want not-found", defects)
	}
}

// TestClassifyReuseReasons covers the unexported classifyReuse seam.
// Public reuse-reason contract rows live in package gobble tests.
func TestClassifyReuseReasons(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "a.txt"), "reads")
	writeCheckFile(t, filepath.Join(dir, "out", "a.txt"), "reads")
	inRec := mustFileRecord(t, filepath.Join(dir, "in", "a.txt"), "in/a.txt")
	outRec := mustFileRecord(t, filepath.Join(dir, "out", "a.txt"), "out/a.txt")
	base := jsonTaskState{
		ID:           "copy",
		Status:       StatusSucceeded,
		Command:      []string{"cp", "in/a.txt", "out/a.txt"},
		Image:        "alpine:3.19.1",
		Params:       []jsonParam{{Name: "mode", Value: "fast"}},
		Env:          map[string]string{"HOME": "/tmp"},
		Fingerprints: []jsonFileHash{inRec},
		Checksums:    []jsonFileHash{outRec},
		Lineage:      []jsonLineage{{Producer: "copy", Path: "out/a.txt", Checksum: outRec.SHA256}},
	}
	plan := TaskPlan{
		ID:      "copy",
		Command: []string{"cp", "in/a.txt", "out/a.txt"},
		Image:   "alpine:3.19.1",
		Params:  []ParamPlan{{Name: "mode", Value: "fast"}},
		Env:     map[string]string{"HOME": "/tmp"},
		Inputs:  []IO{{Name: "in", Path: "in/a.txt"}},
		Outputs: []IO{{Name: "out", Path: "out/a.txt"}},
	}
	got := classifyReuse(dir, base, plan, plan)
	if got.Decision != reuseReused || got.Reason != reasonReusedIdentityMatched {
		t.Fatalf("match got %#v", got)
	}

	failed := base
	failed.Status = StatusFailed
	got = classifyReuse(dir, failed, plan, plan)
	if got.Decision != reuseRerun || got.Reason != reasonPreviousUnsuccessful {
		t.Fatalf("failed got %#v", got)
	}

	inc := base
	inc.Status = StatusIncomplete
	got = classifyReuse(dir, inc, plan, plan)
	if got.Reason != reasonPreviousIncomplete {
		t.Fatalf("incomplete got %#v", got)
	}

	changed := plan
	changed.Command = []string{"true"}
	got = classifyReuse(dir, base, plan, changed)
	if got.Reason != reasonCommandOrScriptChanged {
		t.Fatalf("command got %#v", got)
	}

	scripted := base
	scripted.Script = "cp in/a.txt out/a.txt"
	sameCmd := plan
	sameCmd.Script = scripted.Script
	got = classifyReuse(dir, scripted, sameCmd, sameCmd)
	if got.Decision != reuseReused {
		t.Fatalf("script match got %#v", got)
	}
	changedScript := sameCmd
	changedScript.Script = "cp in/a.txt out/a.txt\n# v2"
	got = classifyReuse(dir, scripted, sameCmd, changedScript)
	if got.Reason != reasonCommandOrScriptChanged {
		t.Fatalf("script got %#v", got)
	}

	changedEnv := plan
	changedEnv.Env = map[string]string{"HOME": "/other"}
	got = classifyReuse(dir, base, plan, changedEnv)
	if got.Reason != reasonEnvChanged {
		t.Fatalf("env got %#v", got)
	}

	renamed := plan
	renamed.Outputs = []IO{{Name: "out", Path: "out/renamed.txt"}}
	writeCheckFile(t, filepath.Join(dir, "out", "renamed.txt"), "reads")
	got = classifyReuse(dir, base, plan, renamed)
	if got.Reason != reasonOutputMissing {
		t.Fatalf("dest rename got %#v", got)
	}

	absent := base
	absent.Fingerprints = nil
	got = classifyReuse(dir, absent, plan, plan)
	if got.Reason != reasonIdentityChanged || len(got.Differing) != 1 || got.Differing[0] != fingerprintsAbsent {
		t.Fatalf("absent fingerprints got %#v", got)
	}

	if err := os.Remove(filepath.Join(dir, "out", "a.txt")); err != nil {
		t.Fatal(err)
	}
	got = classifyReuse(dir, base, plan, plan)
	if got.Reason != reasonOutputMissing {
		t.Fatalf("missing output got %#v", got)
	}
}

func decodeInspectJSONL(t *testing.T, data []byte) []map[string]any {
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

func remainingByID(t *testing.T, data []byte) map[string]map[string]any {
	t.Helper()
	out := map[string]map[string]any{}
	for _, rec := range decodeInspectJSONL(t, data) {
		id, _ := rec["identity"].(string)
		out[id] = rec
	}
	return out
}
