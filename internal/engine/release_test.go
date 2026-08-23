package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
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
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	if defects := Run(t.Context(), Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	}); len(defects) != 0 {
		t.Fatalf("Run() defects %v, want none", defects)
	}
	ForgetHeldLease(dir)
	before := snapshotDir(t, dir)
	defects := Release(dir)
	if !hasDefect(defects, DefectLiveOccupancy, "") {
		t.Fatalf("Release() defects %v, want live-occupancy", defects)
	}
	after := snapshotDir(t, dir)
	if before != after {
		t.Fatalf("live-occupancy mutated workspace")
	}
	DropHeldLease(dir)
}

func TestReleaseOccupyingProcessAfterRun(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	if defects := Run(t.Context(), Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	}); len(defects) != 0 {
		t.Fatalf("Run() defects %v, want none", defects)
	}
	if defects := Release(dir); len(defects) != 0 {
		t.Fatalf("occupying-process Release() defects %v, want none", defects)
	}
	run, exists, err := readRunIdentity(dir)
	if err != nil || !exists {
		t.Fatalf("run.json exists=%v err=%v", exists, err)
	}
	if occupancyIsActive(run) {
		t.Fatal("occupancy still active after occupying-process Release")
	}
}

func TestReleasePIDOnlyUnsupported(t *testing.T) {
	dir := t.TempDir()
	host, err := currentHost()
	if err != nil {
		t.Fatal(err)
	}
	writeOccupancy(t, dir, jsonOccupancy{Active: true, Host: host, PID: os.Getpid(), Started: "2026-01-01T00:00:00Z"})
	defects := Release(dir)
	if !hasDefect(defects, DefectUnsupportedSchema, "") {
		t.Fatalf("PID-only Release() defects %v, want unsupported-schema", defects)
	}
}

func TestReleaseForeignHost(t *testing.T) {
	dir := t.TempDir()
	writeOccupancy(t, dir, jsonOccupancy{Active: true, Host: "other-host", PID: deadPID(t), Lease: "lease", Started: "2026-01-01T00:00:00Z"})
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

func TestReleaseDeadOwnerEmptyRuntimeIDBecomesIncomplete(t *testing.T) {
	dir := t.TempDir()
	host, err := currentHost()
	if err != nil {
		t.Fatal(err)
	}
	writeOccupancy(t, dir, jsonOccupancy{Active: true, Host: host, PID: deadPID(t), Lease: "lease", Started: "2026-01-01T00:00:00Z"})
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
	defects := Release(dir)
	if len(defects) != 0 {
		t.Fatalf("Release() defects %v, want none", defects)
	}
	if _, err := os.ReadFile(filepath.Join(dir, "out", "keep.txt")); err != nil {
		t.Fatalf("Release deleted artifact: %v", err)
	}
	run, exists, err := readRunIdentity(dir)
	if err != nil || !exists {
		t.Fatalf("run.json exists=%v err=%v", exists, err)
	}
	if occupancyIsActive(run) {
		t.Fatal("later-process Release kept occupancy for unproved process")
	}
	st := taskStates(t, dir)["copy"]
	if st.Status != StatusIncomplete {
		t.Fatalf("copy status got %q, want incomplete", st.Status)
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
	writeOccupancy(t, dir, jsonOccupancy{Active: true, Host: host, PID: deadPID(t), Lease: "lease", Started: "2026-01-01T00:00:00Z"})
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
	defects := Release(dir)
	if len(defects) != 0 {
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
	if defects := Run(t.Context(), Request{
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

func TestLaterProcessFileDestCompletePublishedUnfinalized(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc("", "", "in/sample.txt", "out/sample.txt")
	if defects := Run(t.Context(), Request{Workspace: dir, Document: doc}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	patchAttempt(t, dir, func(st *jsonTaskState) {
		st.Status = StatusRunning
		st.RuntimeID = "unproved-pid"
		st.Reason = "ready"
		st.Ended = ""
		st.Fingerprints = nil
		st.Checksums = nil
		st.Lineage = nil
	})
	forceDeadOwner(t, dir)
	if defects := Release(dir); len(defects) != 0 {
		t.Fatalf("later-process Release() defects %v", defects)
	}
	state := taskStates(t, dir)["copy"]
	if state.Status != StatusPublishedUnfinalized {
		t.Fatalf("status got %q, want published-unfinalized", state.Status)
	}
	if len(state.Checksums) != 0 || len(state.Fingerprints) != 0 {
		t.Fatalf("published-unfinalized invented identity: fingerprints=%#v checksums=%#v", state.Fingerprints, state.Checksums)
	}
	raw, defects := Inspect(dir, viewRemaining, "")
	if len(defects) != 0 {
		t.Fatalf("Inspect(remaining) defects %v", defects)
	}
	if len(raw) != 0 {
		t.Fatalf("published-unfinalized remaining got %s, want empty", raw)
	}

	next := cloneDocument(doc)
	next.Tasks[0].Command = []string{"sh", "-c", "exit 99"}
	next.Tasks = append(next.Tasks, TaskPlan{
		ID:      "after",
		Name:    "after",
		Command: []string{"cp", "out/sample.txt", "out/after.txt"},
		Inputs:  []IO{{Name: "in", Path: "out/sample.txt"}},
		Outputs: []IO{{Name: "out", Path: "out/after.txt"}},
	})
	next.Edges = append(next.Edges, Edge{
		FromTask: "copy",
		FromPort: "out",
		ToTask:   "after",
		ToPort:   "in",
		Wait:     []string{"out/sample.txt"},
	})
	defects = Resume(t.Context(), Request{Workspace: dir, Document: next})
	if len(defects) != 0 {
		t.Fatalf("Resume() defects %v, want none for dest-complete skip", defects)
	}
	states := taskStates(t, dir)
	if states["copy"].Status != StatusPublishedUnfinalized || states["copy"].Attempt != state.Attempt {
		t.Fatalf("published-unfinalized copy reran: %#v", states["copy"])
	}
	if states["after"].Status != StatusSucceeded {
		t.Fatalf("downstream state got %q, want succeeded", states["after"].Status)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "out", "after.txt")); err != nil || string(got) != "reads" {
		t.Fatalf("downstream output got %q err=%v, want reads", got, err)
	}
	run, exists, err := readRunIdentity(dir)
	if err != nil || !exists || run.Status != StatusSucceeded {
		t.Fatalf("published-unfinalized Resume run exists=%v err=%v status=%q, want succeeded", exists, err, run.Status)
	}
}

func TestLaterProcessTreeDirectoryOnlyIncomplete(t *testing.T) {
	dir := t.TempDir()
	doc := Document{
		Name: "tree",
		Tasks: []TaskPlan{{
			ID:      "tree",
			Name:    "tree",
			Command: []string{"sh", "-c", "mkdir -p out/tree; printf member > out/tree/member.txt"},
			Outputs: []IO{{Name: "tree", Kind: ArtifactTree, Path: "out/tree"}},
		}},
	}
	if defects := Run(t.Context(), Request{Workspace: dir, Document: doc}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	if err := os.Remove(filepath.Join(dir, "out", "tree", treeManifestName)); err != nil {
		t.Fatal(err)
	}
	patchAttempt(t, dir, func(st *jsonTaskState) {
		st.Status = StatusRunning
		st.RuntimeID = "unproved-pid"
		st.Reason = "ready"
		st.Ended = ""
		st.Fingerprints = nil
		st.Checksums = nil
		st.Lineage = nil
	})
	forceDeadOwner(t, dir)
	if defects := Release(dir); len(defects) != 0 {
		t.Fatalf("later-process Release() defects %v", defects)
	}
	released := taskStates(t, dir)["tree"]
	if released.Status != StatusIncomplete {
		t.Fatalf("tree directory-only status got %q, want incomplete", released.Status)
	}
	if defects := Resume(t.Context(), Request{Workspace: dir, Document: doc}); len(defects) != 0 {
		t.Fatalf("Resume() tree directory-only defects %v, want rerun", defects)
	}
	after := taskStates(t, dir)["tree"]
	if after.Status != StatusSucceeded || after.Attempt != released.Attempt+1 {
		t.Fatalf("resumed Tree state got status=%q attempt=%d, want succeeded attempt %d", after.Status, after.Attempt, released.Attempt+1)
	}
	if !regularFile(filepath.Join(dir, "out", "tree", treeManifestName)) {
		t.Fatal("resumed Tree missing regular dest manifest")
	}
}

func TestLaterProcessFileSymlinkIncompleteReruns(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc("", "", "in/sample.txt", "out/sample.txt")
	if defects := Run(t.Context(), Request{Workspace: dir, Document: doc}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	dest := filepath.Join(dir, "out", "sample.txt")
	if err := os.Remove(dest); err != nil {
		t.Fatal(err)
	}
	writeCheckFile(t, filepath.Join(dir, "foreign.txt"), "foreign")
	if err := os.Symlink("../foreign.txt", dest); err != nil {
		t.Fatal(err)
	}
	patchAttempt(t, dir, func(st *jsonTaskState) {
		st.Status = StatusRunning
		st.RuntimeID = "unproved-pid"
		st.Reason = "ready"
		st.Ended = ""
		st.Fingerprints = nil
		st.Checksums = nil
		st.Lineage = nil
	})
	forceDeadOwner(t, dir)
	if defects := Release(dir); len(defects) != 0 {
		t.Fatalf("later-process Release() defects %v", defects)
	}
	released := taskStates(t, dir)["copy"]
	if released.Status != StatusIncomplete {
		t.Fatalf("File symlink status got %q, want incomplete", released.Status)
	}
	if defects := Resume(t.Context(), Request{Workspace: dir, Document: doc}); len(defects) != 0 {
		t.Fatalf("Resume() File symlink defects %v, want rerun", defects)
	}
	info, err := os.Lstat(dest)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("resumed File dest mode=%v err=%v, want regular", info, err)
	}
	if got, err := os.ReadFile(dest); err != nil || string(got) != "reads" {
		t.Fatalf("resumed File dest got %q err=%v, want reads", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "foreign.txt")); err != nil || string(got) != "foreign" {
		t.Fatalf("symlink target got %q err=%v, want unchanged foreign", got, err)
	}
}

func TestLaterProcessGroupPartialIncompleteReruns(t *testing.T) {
	dir := t.TempDir()
	doc := Document{
		Name: "group",
		Tasks: []TaskPlan{{
			ID:      "group",
			Name:    "group",
			Command: []string{"sh", "-c", "mkdir -p out; printf a > out/a.txt; printf b > out/b.txt"},
			Outputs: []IO{{
				Name: "pair",
				Kind: ArtifactGroup,
				Members: []IOMember{
					{Name: "a", Path: "out/a.txt"},
					{Name: "b", Path: "out/b.txt"},
				},
			}},
		}},
	}
	if defects := Run(t.Context(), Request{Workspace: dir, Document: doc}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	if err := os.Remove(filepath.Join(dir, "out", "b.txt")); err != nil {
		t.Fatal(err)
	}
	patchAttempt(t, dir, func(st *jsonTaskState) {
		st.Status = StatusRunning
		st.RuntimeID = "unproved-pid"
		st.Reason = "ready"
		st.Ended = ""
		st.Fingerprints = nil
		st.Checksums = nil
		st.Lineage = nil
	})
	forceDeadOwner(t, dir)
	if defects := Release(dir); len(defects) != 0 {
		t.Fatalf("later-process Release() defects %v", defects)
	}
	released := taskStates(t, dir)["group"]
	if released.Status != StatusIncomplete {
		t.Fatalf("partial Group status got %q, want incomplete", released.Status)
	}
	if defects := Resume(t.Context(), Request{Workspace: dir, Document: doc}); len(defects) != 0 {
		t.Fatalf("Resume() partial Group defects %v, want rerun", defects)
	}
	after := taskStates(t, dir)["group"]
	if after.Status != StatusSucceeded || after.Attempt != released.Attempt+1 {
		t.Fatalf("resumed Group state got status=%q attempt=%d, want succeeded attempt %d", after.Status, after.Attempt, released.Attempt+1)
	}
	for path, want := range map[string]string{"out/a.txt": "a", "out/b.txt": "b"} {
		got, err := os.ReadFile(filepath.Join(dir, path))
		if err != nil || string(got) != want {
			t.Fatalf("resumed Group member %s got %q err=%v, want %q", path, got, err, want)
		}
	}
}

func TestLaterProcessDockerUnknownKeepsOccupancy(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc("", "", "in/sample.txt", "out/sample.txt")
	if defects := Run(t.Context(), Request{Workspace: dir, Document: doc}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	patchAttempt(t, dir, func(st *jsonTaskState) {
		st.Status = StatusRunning
		st.Executor = executorDocker
		st.Image = "example.invalid/image:latest"
		st.RuntimeID = ""
		st.Reason = "ready"
		st.Ended = ""
	})
	forceDeadOwner(t, dir)
	defects := Release(dir)
	if !hasDefect(defects, DefectUnknownBackend, "copy") {
		t.Fatalf("later-process Docker Release() defects %v, want unknown-backend", defects)
	}
	run, exists, err := readRunIdentity(dir)
	if err != nil || !exists || !occupancyIsActive(run) {
		t.Fatalf("Docker unknown occupancy exists=%v err=%v run=%#v", exists, err, run.Occupancy)
	}
}

func TestDestCompleteByBindKind(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "file.txt"), "file")
	writeCheckFile(t, filepath.Join(dir, "group", "a.txt"), "a")
	if err := os.MkdirAll(filepath.Join(dir, "tree"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeCheckFile(t, filepath.Join(dir, "tree", treeManifestName), `{}`)
	if !destComplete(dir, []IO{{Name: "file", Kind: ArtifactFile, Path: "file.txt"}}) {
		t.Fatal("regular File dest not complete")
	}
	if !destComplete(dir, []IO{{Name: "group", Kind: ArtifactGroup, Members: []IOMember{{Name: "a", Path: "group/a.txt"}}}}) {
		t.Fatal("complete Group not complete")
	}
	if destComplete(dir, []IO{{Name: "group", Kind: ArtifactGroup, Members: []IOMember{{Name: "a", Path: "group/a.txt"}, {Name: "b", Path: "group/b.txt"}}}}) {
		t.Fatal("Group with missing member reported complete")
	}
	if !destComplete(dir, []IO{{Name: "tree", Kind: ArtifactTree, Path: "tree"}}) {
		t.Fatal("Tree directory plus manifest not complete")
	}
	if err := os.Symlink("file.txt", filepath.Join(dir, "link.txt")); err != nil {
		t.Fatal(err)
	}
	if destComplete(dir, []IO{{Name: "file", Kind: ArtifactFile, Path: "link.txt"}}) {
		t.Fatal("File symlink reported complete")
	}
}

func TestReleaseMixedSnapshotRefused(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	if defects := Run(t.Context(), Request{Workspace: dir, Document: sampleDoc("", "", "in/sample.txt", "out/sample.txt")}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	tamperTasksSnapshot(t, dir, "mixed")
	before := snapshotDir(t, dir)
	if defects := Release(dir); !hasDefect(defects, DefectInvalidPath, "") {
		t.Fatalf("Release mixed snapshot defects %v, want invalid-path", defects)
	}
	if after := snapshotDir(t, dir); after != before {
		t.Fatal("Release mixed snapshot mutated workspace")
	}
}

func TestConcurrentOccupyingProcessReleaseSerialized(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	if defects := Run(t.Context(), Request{Workspace: dir, Document: sampleDoc("", "", "in/sample.txt", "out/sample.txt")}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	start := make(chan struct{})
	results := make(chan []Defect, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- Release(dir)
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	succeeded := 0
	rejected := 0
	for defects := range results {
		switch {
		case len(defects) == 0:
			succeeded++
		case hasDefect(defects, DefectAlreadyReleased, "") || hasDefect(defects, DefectLiveOccupancy, ""):
			rejected++
		default:
			t.Fatalf("concurrent Release defects %v", defects)
		}
	}
	if succeeded != 1 || rejected != 1 {
		t.Fatalf("concurrent Release got success=%d rejected=%d, want 1/1", succeeded, rejected)
	}
	if _, _, _, _, _, defects := readCoherentControl(dir); len(defects) != 0 {
		t.Fatalf("final control snapshot defects %v", defects)
	}
}

func TestInspectAndReleaseZeroAttemptsNotSuccess(t *testing.T) {
	dir := t.TempDir()
	host, err := currentHost()
	if err != nil {
		t.Fatal(err)
	}
	writeOccupancy(t, dir, jsonOccupancy{Active: true, Host: host, PID: deadPID(t), Lease: "lease"})
	writeCheckFile(t, filepath.Join(dir, ControlDir, TasksFile), `{"schema_version":2,"tasks":[]}`)
	if _, defects := Inspect(dir, viewRun, ""); !hasDefect(defects, DefectInvalidValue, "") {
		t.Fatalf("Inspect zero attempts defects %v, want invalid-value", defects)
	}
	if defects := Release(dir); !hasDefect(defects, DefectInvalidValue, "") {
		t.Fatalf("Release zero attempts defects %v, want invalid-value", defects)
	}
}

func tamperTasksSnapshot(t *testing.T, workspace, snapshot string) {
	t.Helper()
	path := filepath.Join(workspace, ControlDir, TasksFile)
	var file jsonTasksFile
	if err := json.Unmarshal(mustJSONFile(t, path), &file); err != nil {
		t.Fatal(err)
	}
	file.Snapshot = snapshot
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeCheckFile(t, path, string(append(data, '\n')))
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
