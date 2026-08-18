package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestCheckClosedOccupancyNotOccupied(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	writeCheckFile(t, filepath.Join(dir, ControlDir, RunIdentityFile), closedRunJSON("run-1"))
	req := Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	}
	if defects := Check(req); len(defects) != 0 {
		t.Fatalf("closed occupancy Check() defects %v, want none", defects)
	}
}

func TestCheckClosedOccupancyOutputExists(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	writeCheckFile(t, filepath.Join(dir, "out", "sample.txt"), "leftover")
	writeCheckFile(t, filepath.Join(dir, ControlDir, RunIdentityFile), closedRunJSON("run-1"))
	defects := Check(Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	})
	if !hasDefect(defects, DefectOutputExists, "copy.out") {
		t.Fatalf("closed+output: Check() defects %v, want output-exists", defects)
	}
	if hasDefect(defects, DefectOccupiedWorkspace, "") {
		t.Fatalf("closed+output: Check() reported occupied-workspace")
	}
}

func TestOccupyClosedWorkspaceOneOwner(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	writeCheckFile(t, filepath.Join(dir, ControlDir, RunIdentityFile), closedRunJSON("run-1"))
	req := Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	}
	type result struct {
		ok  bool
		pid int
	}
	got := make(chan result, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, defects := occupy(req)
			if len(defects) > 0 {
				if !hasDefect(defects, DefectOccupiedWorkspace, "") {
					t.Errorf("occupy defects %v, want occupied-workspace or success", defects)
				}
				got <- result{}
				return
			}
			pid := 0
			if s != nil && s.run.Occupancy != nil {
				pid = s.run.Occupancy.PID
			}
			got <- result{ok: true, pid: pid}
		}()
	}
	wg.Wait()
	close(got)
	owners := 0
	var ownerPID int
	for r := range got {
		if r.ok {
			owners++
			ownerPID = r.pid
		}
	}
	if owners != 1 {
		t.Fatalf("closed occupy owners got %d, want 1", owners)
	}
	run, exists, err := readRunIdentity(dir)
	if err != nil || !exists {
		t.Fatalf("read run.json exists=%v err=%v", exists, err)
	}
	if !occupancyIsActive(run) || run.Occupancy == nil || run.Occupancy.PID != ownerPID {
		t.Fatalf("run occupancy got %#v, want one owner pid %d", run.Occupancy, ownerPID)
	}
	if _, err := os.Stat(filepath.Join(dir, ControlDir, RunIdentityFile)); err != nil {
		t.Fatalf("run.json missing after occupy: %v", err)
	}
}

func TestRunPersistsIdentityFacts(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	defects := Run(Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	})
	if len(defects) != 0 {
		t.Fatalf("Run() defects %v, want none", defects)
	}
	run, exists, err := readRunIdentity(dir)
	if err != nil || !exists {
		t.Fatalf("run.json exists=%v err=%v", exists, err)
	}
	if run.SchemaVersion != SchemaVersion {
		t.Fatalf("run schema_version got %d, want %d", run.SchemaVersion, SchemaVersion)
	}
	if run.Occupancy == nil || !run.Occupancy.Active || run.Occupancy.PID != os.Getpid() {
		t.Fatalf("run occupancy got %#v, want active this pid", run.Occupancy)
	}
	host, err := currentHost()
	if err != nil {
		t.Fatal(err)
	}
	if run.Occupancy.Host != host {
		t.Fatalf("occupancy host got %q, want %q", run.Occupancy.Host, host)
	}
	if run.Occupancy.Started == "" {
		t.Fatal("occupancy started empty")
	}
	if _, err := time.Parse(time.RFC3339Nano, run.Occupancy.Started); err != nil {
		t.Fatalf("occupancy started %q: %v", run.Occupancy.Started, err)
	}
	planVer, planOK, err := readSchemaFile(filepath.Join(dir, ControlDir, PlanFile))
	if err != nil || !planOK || planVer != SchemaVersion {
		t.Fatalf("plan schema exists=%v version=%d err=%v, want %d", planOK, planVer, err, SchemaVersion)
	}
	st := taskStates(t, dir)["copy"]
	if st.Instance != "" || st.ShardIndex != DefaultShardIndex || st.ShardCount != DefaultShardCount || st.Attempt != DefaultAttempt {
		t.Fatalf("reserved slots got instance %q shard %d/%d attempt %d", st.Instance, st.ShardIndex, st.ShardCount, st.Attempt)
	}
	if st.Started == "" || st.Ended == "" {
		t.Fatalf("task timing started %q ended %q, want both set", st.Started, st.Ended)
	}
	if len(st.Fingerprints) != 1 || st.Fingerprints[0].Path != "in/sample.txt" || st.Fingerprints[0].SHA256 == "" {
		t.Fatalf("fingerprints got %#v", st.Fingerprints)
	}
	if len(st.Checksums) != 1 || st.Checksums[0].Path != "out/sample.txt" || st.Checksums[0].SHA256 == "" {
		t.Fatalf("checksums got %#v", st.Checksums)
	}
	if st.Fingerprints[0].SHA256 != st.Checksums[0].SHA256 {
		t.Fatalf("copy fingerprint %s != checksum %s", st.Fingerprints[0].SHA256, st.Checksums[0].SHA256)
	}
	if len(st.Lineage) < 2 {
		t.Fatalf("lineage got %#v, want input and output edges", st.Lineage)
	}
}

func TestIsolatePathsDistinct(t *testing.T) {
	a := isolateRel(TaskPlan{ID: "copy"})
	b := isolateRel(TaskPlan{ID: "copy", Instance: "s1"})
	c := isolateRel(TaskPlan{ID: "copy", ShardIndex: 1, ShardCount: 2})
	d := isolateRel(TaskPlan{ID: "copy", Attempt: 2, ShardCount: 1})
	if a != ControlDir+"/tasks/copy/_/0/1" {
		t.Fatalf("first-horizon isolate got %q", a)
	}
	seen := map[string]string{"default": a, "instance": b, "shard": c, "attempt": d}
	for name, path := range seen {
		for other, opath := range seen {
			if name != other && path == opath {
				t.Fatalf("%s and %s share isolate %q", name, other, path)
			}
		}
	}
}

func TestRunAfterReleaseHitsOutputExists(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	req := Request{Workspace: dir, Document: sampleDoc("", "", "in/sample.txt", "out/sample.txt")}
	if defects := Run(req); len(defects) != 0 {
		t.Fatalf("first Run() defects %v, want none", defects)
	}
	before, err := os.ReadFile(filepath.Join(dir, "out", "sample.txt"))
	if err != nil {
		t.Fatal(err)
	}
	forceDeadOwner(t, dir)
	if defects := Release(dir); len(defects) != 0 {
		t.Fatalf("Release() defects %v, want none", defects)
	}
	defects := Run(req)
	if !hasDefect(defects, DefectOutputExists, "copy.out") {
		t.Fatalf("Run after Release defects %v, want output-exists", defects)
	}
	if hasDefect(defects, DefectOccupiedWorkspace, "") {
		t.Fatalf("Run after Release reported occupied-workspace")
	}
	after, err := os.ReadFile(filepath.Join(dir, "out", "sample.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("Run after Release changed leftover dest")
	}
}

func closedRunJSON(id string) string {
	return `{
  "schema_version": 1,
  "id": "` + id + `",
  "status": "succeeded",
  "started": "2026-01-01T00:00:00Z",
  "ended": "2026-01-01T00:00:01Z",
  "occupancy": {
    "active": false,
    "host": "old",
    "pid": 1,
    "started": "2026-01-01T00:00:00Z",
    "closed": "2026-01-01T00:00:01Z"
  }
}
`
}

func forceDeadOwner(t *testing.T, workspace string) {
	t.Helper()
	path := filepath.Join(workspace, ControlDir, RunIdentityFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var run jsonRun
	if err := json.Unmarshal(data, &run); err != nil {
		t.Fatal(err)
	}
	if run.Occupancy == nil {
		run.Occupancy = &jsonOccupancy{Active: true}
	}
	host, err := currentHost()
	if err != nil {
		t.Fatal(err)
	}
	run.Occupancy.Active = true
	run.Occupancy.Host = host
	run.Occupancy.PID = deadPID(t)
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
