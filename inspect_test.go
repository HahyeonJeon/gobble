package gobble_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/internal/engine"
)

func TestInspectMissingWorkspace(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent")
	data, err := gobble.Inspect(missing, gobble.ViewRun, "")
	if data != nil {
		t.Fatalf("Inspect() data = %q, want nil", data)
	}
	requireInspectError(t, "missing workspace", err, gobble.DefectNotFound, "")
	if _, statErr := os.Stat(missing); !os.IsNotExist(statErr) {
		t.Fatalf("Inspect created missing workspace")
	}
}

func TestInspectMissingRunDoesNotCreate(t *testing.T) {
	dir := t.TempDir()
	before := snapshotWorkspace(t, dir)
	data, err := gobble.Inspect(dir, gobble.ViewRun, "")
	if data != nil {
		t.Fatalf("Inspect() data = %q, want nil", data)
	}
	requireInspectError(t, "missing run", err, gobble.DefectNotFound, "")
	after := snapshotWorkspace(t, dir)
	if before != after {
		t.Fatalf("Inspect mutated missing-run workspace\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if _, statErr := os.Stat(filepath.Join(dir, engine.ControlDir)); !os.IsNotExist(statErr) {
		t.Fatalf("Inspect created %s", engine.ControlDir)
	}
}

func TestInspectUnknownViewAndInstance(t *testing.T) {
	dir := readyRunWorkspace(t)
	g := mustCompose(processEnvCopyPipeline)(t)
	if err := gobble.Run(t.Context(), g, dir, 0); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	before := snapshotWorkspace(t, dir)
	_, err := gobble.Inspect(dir, gobble.View("events"), "")
	requireInspectError(t, "unknown view", err, gobble.DefectNotFound, "events")
	_, err = gobble.Inspect(dir, gobble.ViewInstances, "nope")
	requireInspectError(t, "unknown instance", err, gobble.DefectNotFound, "nope")
	after := snapshotWorkspace(t, dir)
	if before != after {
		t.Fatalf("Inspect unknown selectors mutated workspace")
	}
}

func TestInspectUnsupportedSchema(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, engine.ControlDir, engine.RunIdentityFile), `{
  "schema_version": 0,
  "id": "run-1",
  "status": "running",
  "started": "2026-01-01T00:00:00Z",
  "occupancy": {"active": true, "host": "h", "pid": 1}
}
`)
	before := snapshotWorkspace(t, dir)
	_, err := gobble.Inspect(dir, gobble.ViewRun, "")
	requireInspectError(t, "unsupported schema 0", err, gobble.DefectUnsupportedSchema, "")
	after := snapshotWorkspace(t, dir)
	if before != after {
		t.Fatalf("unsupported-schema Inspect mutated workspace")
	}

	dir = t.TempDir()
	writeRunFile(t, filepath.Join(dir, engine.ControlDir, engine.RunIdentityFile), `{
  "schema_version": 1,
  "id": "run-1",
  "status": "running",
  "started": "2026-01-01T00:00:00Z",
  "occupancy": {"active": true, "host": "h", "pid": 1}
}
`)
	before = snapshotWorkspace(t, dir)
	_, err = gobble.Inspect(dir, gobble.ViewRun, "")
	requireInspectError(t, "unsupported schema", err, gobble.DefectUnsupportedSchema, "")
	after = snapshotWorkspace(t, dir)
	if before != after {
		t.Fatalf("unsupported-schema Inspect mutated workspace")
	}
}

func TestInspectOp(t *testing.T) {
	_, err := gobble.Inspect(t.TempDir(), gobble.ViewRun, "")
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		t.Fatalf("error = %v, want *Error", err)
	}
	if ge.Op != "inspect" {
		t.Fatalf("Error.Op got %q, want inspect", ge.Op)
	}
}

func TestInspectViewsAfterSuccessfulRun(t *testing.T) {
	dir := readyRunWorkspace(t)
	g := mustCompose(processEnvCopyPipeline)(t)
	if err := gobble.Run(t.Context(), g, dir, 0); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	before := snapshotWorkspace(t, dir)

	runRaw := mustInspectObject(t, dir, "run", "")
	if runRaw["id"] != "copy" {
		t.Fatalf("run.id got %#v, want copy", runRaw["id"])
	}
	if runRaw["status"] != engine.StatusSucceeded {
		t.Fatalf("run.status got %#v, want succeeded", runRaw["status"])
	}
	if runRaw["schema_version"] != float64(engine.SchemaVersion) {
		t.Fatalf("run.schema_version got %#v, want %d", runRaw["schema_version"], engine.SchemaVersion)
	}
	occ, _ := runRaw["occupancy"].(map[string]any)
	if occ == nil || occ["active"] != true {
		t.Fatalf("run.occupancy got %#v, want active", runRaw["occupancy"])
	}
	if occ["live"] != true {
		t.Fatalf("run.occupancy.live got %#v, want true", occ["live"])
	}
	if runRaw["started"] == "" || runRaw["ended"] == "" {
		t.Fatalf("run timing missing: %#v", runRaw)
	}

	instances := mustInspectJSONL(t, dir, "instances", "")
	if len(instances) != 1 {
		t.Fatalf("instances got %d records, want 1", len(instances))
	}
	if instances[0]["schema_version"] != float64(engine.SchemaVersion) {
		t.Fatalf("instance schema_version got %#v, want %d", instances[0]["schema_version"], engine.SchemaVersion)
	}
	if instances[0]["identity"] != "copy" || instances[0]["status"] != engine.StatusSucceeded {
		t.Fatalf("instance record got %#v", instances[0])
	}
	env, _ := instances[0]["env"].(map[string]any)
	if env["HOME"] != "/tmp/gobble-home" {
		t.Fatalf("instance env got %#v, want HOME=/tmp/gobble-home", instances[0]["env"])
	}
	if instances[0]["shard_index"] != float64(0) || instances[0]["attempt"] != float64(1) {
		t.Fatalf("instance slots got %#v", instances[0])
	}

	errorsView := mustInspectObject(t, dir, "errors", "")
	errs, _ := errorsView["errors"].([]any)
	if len(errs) != 0 {
		t.Fatalf("errors got %#v, want empty", errorsView["errors"])
	}

	logsView := mustInspectObject(t, dir, "logs", "")
	logs, _ := logsView["logs"].([]any)
	if len(logs) != 1 {
		t.Fatalf("logs got %#v", logsView["logs"])
	}
	log0, _ := logs[0].(map[string]any)
	stdout, _ := log0["stdout"].(string)
	if !strings.HasPrefix(stdout, engine.ControlDir+"/tasks/copy/") {
		t.Fatalf("log stdout pointer got %#v", log0["stdout"])
	}

	timing := mustInspectObject(t, dir, "timing", "")
	if timing["started"] == "" || timing["ended"] == "" {
		t.Fatalf("timing run times missing: %#v", timing)
	}
	instTimes, _ := timing["instances"].([]any)
	if len(instTimes) != 1 {
		t.Fatalf("timing instances got %#v", timing["instances"])
	}
	one, _ := instTimes[0].(map[string]any)
	if one["identity"] != "copy" || one["started"] == "" || one["ended"] == "" {
		t.Fatalf("timing instance got %#v, want start and end", one)
	}

	dag := mustInspectObject(t, dir, string(gobble.ViewDAG), "")
	if dag["schema_version"] != float64(engine.SchemaVersion) {
		t.Fatalf("dag.schema_version got %#v, want %d", dag["schema_version"], engine.SchemaVersion)
	}
	nodes, _ := dag["nodes"].([]any)
	if len(nodes) != 1 || nodes[0] != "copy" {
		t.Fatalf("DAG nodes got %#v", dag["nodes"])
	}
	_, err := gobble.Inspect(dir, gobble.View("DAG"), "")
	requireInspectError(t, "uppercase DAG view", err, gobble.DefectNotFound, "DAG")

	lineage := mustInspectObject(t, dir, "lineage", "")
	if _, ok := lineage["lineage"]; !ok {
		t.Fatalf("lineage missing key: %#v", lineage)
	}

	remaining := mustInspectJSONL(t, dir, "remaining", "")
	if len(remaining) != 0 {
		t.Fatalf("remaining got %#v, want empty after successful recorded run", remaining)
	}
	reuse := mustInspectJSONL(t, dir, "reuse", "")
	if len(reuse) != 0 {
		t.Fatalf("reuse got %#v, want empty before Resume", reuse)
	}

	after := snapshotWorkspace(t, dir)
	if before != after {
		t.Fatalf("Inspect rewrote workspace files")
	}
}

func TestInspectRemainingAfterFailure(t *testing.T) {
	dir := readyRunWorkspace(t)
	err := gobble.Run(t.Context(), mustCompose(processContainPipeline)(t), dir, 2)
	requireRunError(t, "contained failure", err, gobble.DefectFailed, "fail")
	before := snapshotWorkspace(t, dir)
	remaining := mustInspectJSONL(t, dir, "remaining", "")
	byID := map[string]map[string]any{}
	for _, rec := range remaining {
		id, _ := rec["identity"].(string)
		byID[id] = rec
	}
	if byID["fail"] == nil || byID["fail"]["remaining"] != true || byID["fail"]["affected"] != true {
		t.Fatalf("fail remaining got %#v", byID["fail"])
	}
	if byID["dep"] == nil || byID["dep"]["remaining"] != true || byID["dep"]["affected"] != true {
		t.Fatalf("dep remaining got %#v", byID["dep"])
	}
	if rec := byID["ok"]; rec != nil && (rec["remaining"] == true || rec["affected"] == true) {
		t.Fatalf("ok should not be remaining or affected: %#v", rec)
	}
	if _, ok := byID["ok"]; ok {
		t.Fatalf("ok listed in remaining view: %#v", byID["ok"])
	}
	errorsView := mustInspectObject(t, dir, "errors", "")
	errs, _ := errorsView["errors"].([]any)
	if len(errs) < 2 {
		t.Fatalf("errors got %#v, want unsuccessful instances", errorsView["errors"])
	}
	after := snapshotWorkspace(t, dir)
	if before != after {
		t.Fatalf("Inspect remaining mutated workspace")
	}
}

func TestInspectLiveOccupancyDoesNotBlock(t *testing.T) {
	dir := readyRunWorkspace(t)
	if err := gobble.Run(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	before := snapshotWorkspace(t, dir)
	raw := mustInspectObject(t, dir, "run", "")
	occ, _ := raw["occupancy"].(map[string]any)
	if occ["active"] != true || occ["live"] != true {
		t.Fatalf("live inspect occupancy got %#v", occ)
	}
	after := snapshotWorkspace(t, dir)
	if before != after {
		t.Fatalf("live Inspect rewrote workspace")
	}
	err := gobble.Run(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0)
	requireRunError(t, "still occupied", err, gobble.DefectOccupiedWorkspace, "")
}

func TestInspectReleasedWorkspace(t *testing.T) {
	dir := readyRunWorkspace(t)
	if err := gobble.Run(t.Context(), mustCompose(processCopyPipeline)(t), dir, 0); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	forcePublicDeadOwner(t, dir)
	if err := gobble.Release(dir); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	before := snapshotWorkspace(t, dir)
	raw := mustInspectObject(t, dir, "run", "")
	occ, _ := raw["occupancy"].(map[string]any)
	if occ["active"] != false || occ["live"] != false {
		t.Fatalf("released occupancy got %#v", occ)
	}
	if occ["closed"] == "" {
		t.Fatalf("released occupancy missing closed: %#v", occ)
	}
	after := snapshotWorkspace(t, dir)
	if before != after {
		t.Fatalf("Inspect of released workspace rewrote files")
	}
}

func processEnvCopyPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("copy")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
	p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"sh", "-c", "pwd > out/pwd.txt && cp in/sample.txt out/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{
			{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "sample", Ext: ".txt"}},
			{Name: "pwd", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Base: "pwd", Ext: ".txt"}},
		},
		Params:    []gobble.Param{{Name: "mode", Value: "fast"}},
		Env:       map[string]string{"HOME": "/tmp/gobble-home"},
		Resources: gobble.Resources{CPU: 1, Memory: "512m"},
	})
	return p
}

func requireInspectError(t *testing.T, name string, err error, code gobble.DefectCode, unit string) *gobble.Error {
	t.Helper()
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		t.Fatalf("case %s: error = %v, want *Error", name, err)
	}
	if ge.Op != "inspect" {
		t.Fatalf("case %s: Error.Op got %q, want inspect", name, ge.Op)
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

func mustInspectObject(t *testing.T, workspace, view, instance string) map[string]any {
	t.Helper()
	data, err := gobble.Inspect(workspace, gobble.View(view), instance)
	if err != nil {
		t.Fatalf("Inspect(%s) error = %v", view, err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Inspect(%s) JSON: %v\n%s", view, err, data)
	}
	return out
}

func mustInspectJSONL(t *testing.T, workspace, view, instance string) []map[string]any {
	t.Helper()
	data, err := gobble.Inspect(workspace, gobble.View(view), instance)
	if err != nil {
		t.Fatalf("Inspect(%s) error = %v", view, err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	var out []map[string]any
	dec := json.NewDecoder(bytes.NewReader(data))
	for dec.More() {
		var rec map[string]any
		if err := dec.Decode(&rec); err != nil {
			t.Fatalf("Inspect(%s) JSONL: %v\n%s", view, err, data)
		}
		out = append(out, rec)
	}
	return out
}
