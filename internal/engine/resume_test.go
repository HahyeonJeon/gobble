package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPlanDriftTaskSetAndEdges(t *testing.T) {
	base := Document{
		Tasks: []TaskPlan{{ID: "a"}, {ID: "b"}},
		Edges: []Edge{{FromTask: "a", FromPort: "out", ToTask: "b", ToPort: "in"}},
	}
	if d := planDrift(base, base); len(d) != 0 {
		t.Fatalf("same plan: %v", d)
	}
	added := Document{
		Tasks: []TaskPlan{{ID: "a"}, {ID: "b"}, {ID: "c"}},
		Edges: base.Edges,
	}
	if !hasDefect(planDrift(base, added), DefectPlanDrift, "") {
		t.Fatalf("added task: want plan-drift")
	}
	changed := Document{
		Tasks: []TaskPlan{{ID: "a"}, {ID: "b"}},
		Edges: []Edge{{FromTask: "b", FromPort: "out", ToTask: "a", ToPort: "in"}},
	}
	if !hasDefect(planDrift(base, changed), DefectPlanDrift, "") {
		t.Fatalf("changed edges: want plan-drift")
	}
	identity := Document{
		Tasks: []TaskPlan{
			{ID: "a", Command: []string{"echo"}},
			{ID: "b", Env: map[string]string{"K": "v"}},
		},
		Edges: base.Edges,
	}
	if d := planDrift(base, identity); len(d) != 0 {
		t.Fatalf("identity-only change: %v", d)
	}
	waitOnly := Document{
		Tasks: []TaskPlan{{ID: "a"}, {ID: "b"}},
		Edges: []Edge{{FromPort: "reads", ToTask: "a", ToPort: "in", Wait: []string{"in/sample.txt"}}},
	}
	if d := planDrift(waitOnly, waitOnly); len(d) != 0 {
		t.Fatalf("same wait-only plan: %v", d)
	}
	waitChanged := Document{
		Tasks: waitOnly.Tasks,
		Edges: []Edge{{FromPort: "reads", ToTask: "a", ToPort: "in", Wait: []string{"in/other.txt"}}},
	}
	if !hasDefect(planDrift(waitOnly, waitChanged), DefectPlanDrift, "") {
		t.Fatalf("changed wait-only edge: want plan-drift")
	}
}

func TestCheckResumeOutputsUnattributed(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "out", "a.txt"), "foreign")
	writeCheckFile(t, filepath.Join(dir, "out", "b.txt"), "stray")
	writeCheckFile(t, filepath.Join(dir, "out", "ok.txt"), "ok")
	doc := Document{
		Tasks: []TaskPlan{
			{ID: "a", Outputs: []IO{{Name: "out", Path: "out/a.txt"}}},
			{ID: "b", Outputs: []IO{{Name: "out", Path: "out/b.txt"}}},
			{ID: "ok", Outputs: []IO{{Name: "out", Path: "out/ok.txt"}}},
		},
	}
	tasks := []jsonTaskState{{
		ID:      "a",
		Status:  StatusFailed,
		Attempt: 1,
	}, {
		ID:      "b",
		Status:  StatusBlocked,
		Attempt: 1,
	}, {
		ID:        "ok",
		Status:    StatusSucceeded,
		Attempt:   1,
		Checksums: []jsonFileHash{{Path: "out/ok.txt", SHA256: "abc"}},
		Lineage:   []jsonLineage{{Producer: "ok", Path: "out/ok.txt", Checksum: "abc"}},
	}}
	class := remainingClass{
		Decision: map[string]reuseDecision{
			"a":  {Identity: "a", Decision: reuseRerun},
			"b":  {Identity: "b", Decision: reuseRerun},
			"ok": {Identity: "ok", Decision: reuseRerun},
		},
	}
	defects := checkResumeOutputs(dir, doc, tasks, class)
	if !hasDefect(defects, DefectOutputExists, "a.out") {
		t.Fatalf("defects %v, want output-exists a.out", defects)
	}
	if !hasDefect(defects, DefectOutputExists, "b.out") {
		t.Fatalf("defects %v, want output-exists b.out", defects)
	}
	if hasDefect(defects, DefectOutputExists, "ok.out") {
		t.Fatalf("published dest treated as output-exists")
	}
}

func TestStagedReplaceLeavesDestOnFailedWrite(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "out", "sample.txt")
	writeCheckFile(t, dst, "prior")
	src := filepath.Join(dir, "work", "sample.txt")
	writeCheckFile(t, src, "next")
	if err := stagedReplace(src, dst); err != nil {
		t.Fatalf("stagedReplace() error = %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil || string(got) != "next" {
		t.Fatalf("dest got %q, want next", got)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("dest mode got %o, want 0644", info.Mode().Perm())
	}
	if err := os.Chmod(dst, 0o600); err != nil {
		t.Fatal(err)
	}
	writeCheckFile(t, src, "again")
	if err := stagedReplace(src, dst); err != nil {
		t.Fatalf("stagedReplace() second error = %v", err)
	}
	info, err = os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("dest mode after match got %o, want 0600", info.Mode().Perm())
	}
}

func TestPublishAllStillUnlinksPartial(t *testing.T) {
	dir := t.TempDir()
	isolate := filepath.Join(dir, "iso")
	writeCheckFile(t, filepath.Join(isolate, "out", "first.txt"), "one")
	task := TaskPlan{
		ID: "copy",
		Outputs: []IO{
			{Name: "first", Path: "out/first.txt"},
			{Name: "second", Path: "out/missing.txt"},
		},
	}
	if err := publishAll(dir, isolate, task); err == nil {
		t.Fatal("publishAll() error = nil, want missing source")
	}
	if _, err := os.Stat(filepath.Join(dir, "out", "first.txt")); !os.IsNotExist(err) {
		t.Fatal("publishAll left a dest after rollback")
	}
}

func TestResumeRerunsWhenScriptChanges(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc("", "", "in/sample.txt", "out/sample.txt")
	doc.Tasks[0].Command = nil
	doc.Tasks[0].Script = "cp in/sample.txt out/sample.txt"
	if defects := Run(Request{Workspace: dir, Document: doc}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	st := taskStates(t, dir)["copy"]
	if st.Script != doc.Tasks[0].Script {
		t.Fatalf("persisted script got %q, want %q", st.Script, doc.Tasks[0].Script)
	}
	forceDeadOwner(t, dir)
	if defects := Release(dir); len(defects) != 0 {
		t.Fatalf("Release() defects %v", defects)
	}
	next := doc
	next.Tasks = append([]TaskPlan(nil), doc.Tasks...)
	next.Tasks[0].Script = "cp in/sample.txt out/sample.txt\n# v2"
	if defects := Resume(Request{Workspace: dir, Document: next}); len(defects) != 0 {
		t.Fatalf("Resume() defects %v", defects)
	}
	after := taskStates(t, dir)["copy"]
	if after.Attempt != 2 || after.Decision != reuseRerun || after.ReuseReason != reasonCommandOrScriptChanged {
		t.Fatalf("script resume got attempt=%d decision=%q reason=%q", after.Attempt, after.Decision, after.ReuseReason)
	}
}

func TestInspectRerunsWhenPlanScriptChanges(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc("", "", "in/sample.txt", "out/sample.txt")
	doc.Tasks[0].Command = nil
	doc.Tasks[0].Script = "cp in/sample.txt out/sample.txt"
	if defects := Run(Request{Workspace: dir, Document: doc}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	planPath := filepath.Join(dir, ControlDir, PlanFile)
	raw, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	var plan jsonPlan
	if err := json.Unmarshal(raw, &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Tasks) != 1 || plan.Tasks[0].Script == "" {
		t.Fatalf("control plan script got %#v", plan.Tasks)
	}
	plan.Tasks[0].Script = plan.Tasks[0].Script + "\n# v2"
	rewritten, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeCheckFile(t, planPath, string(append(rewritten, '\n')))
	remaining, defects := Inspect(dir, viewRemaining, "")
	if len(defects) != 0 {
		t.Fatalf("Inspect(remaining) defects %v", defects)
	}
	recs := remainingByID(t, remaining)
	if recs["copy"]["affected"] != true || recs["copy"]["reason"] != reasonCommandOrScriptChanged {
		t.Fatalf("plan script overwrite remaining got %#v", recs["copy"])
	}
}

func TestResumeRerunsWhenEnvChanges(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc("", "", "in/sample.txt", "out/sample.txt")
	doc.Tasks[0].Env = map[string]string{"HOME": "/tmp/gobble-home"}
	if defects := Run(Request{Workspace: dir, Document: doc}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	st := taskStates(t, dir)["copy"]
	if st.Env["HOME"] != "/tmp/gobble-home" {
		t.Fatalf("persisted env got %#v", st.Env)
	}
	forceDeadOwner(t, dir)
	if defects := Release(dir); len(defects) != 0 {
		t.Fatalf("Release() defects %v", defects)
	}
	next := doc
	next.Tasks = append([]TaskPlan(nil), doc.Tasks...)
	next.Tasks[0].Env = map[string]string{"HOME": "/tmp/gobble-home-2"}
	if defects := Resume(Request{Workspace: dir, Document: next}); len(defects) != 0 {
		t.Fatalf("Resume() defects %v", defects)
	}
	after := taskStates(t, dir)["copy"]
	if after.Attempt != 2 || after.Decision != reuseRerun || after.ReuseReason != reasonEnvChanged {
		t.Fatalf("env resume got attempt=%d decision=%q reason=%q", after.Attempt, after.Decision, after.ReuseReason)
	}
}

func TestResumeSequentialRerunWaitsForUpstreamDest(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "old")
	doc := sequentialCopyDoc(
		[]string{"cp", "in/sample.txt", "out/a.txt"},
		[]string{"cp", "out/a.txt", "out/b.txt"},
	)
	if defects := Run(Request{Workspace: dir, Document: doc}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	forceDeadOwner(t, dir)
	if defects := Release(dir); len(defects) != 0 {
		t.Fatalf("Release() defects %v", defects)
	}
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "new")
	next := sequentialCopyDoc(
		[]string{"sh", "-c", "cp in/sample.txt out/a.txt"},
		[]string{"sh", "-c", "cp out/a.txt out/b.txt"},
	)
	orig := execTask
	t.Cleanup(func() { execTask = orig })
	execTask = func(workspace string, task TaskPlan) report {
		if task.ID == "b" {
			got, err := os.ReadFile(filepath.Join(workspace, "out", "a.txt"))
			if err != nil {
				t.Errorf("b started before a dest exists: %v", err)
			} else if string(got) != "new" {
				t.Errorf("b saw a dest %q, want new", got)
			}
		}
		return orig(workspace, task)
	}
	if defects := Resume(Request{Workspace: dir, Document: next}); len(defects) != 0 {
		t.Fatalf("Resume() defects %v", defects)
	}
	got, err := os.ReadFile(filepath.Join(dir, "out", "b.txt"))
	if err != nil || string(got) != "new" {
		t.Fatalf("b dest got %q err=%v, want new", got, err)
	}
}

func TestResumeDestRenameDoesNotReuse(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc("", "", "in/sample.txt", "out/sample.txt")
	if defects := Run(Request{Workspace: dir, Document: doc}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	forceDeadOwner(t, dir)
	if defects := Release(dir); len(defects) != 0 {
		t.Fatalf("Release() defects %v", defects)
	}
	next := sampleDoc("", "", "in/sample.txt", "out/renamed.txt")
	next.Tasks[0].Command = doc.Tasks[0].Command
	defects := Resume(Request{Workspace: dir, Document: next})
	after := taskStates(t, dir)["copy"]
	if after.Decision == reuseReused {
		t.Fatalf("dest rename reused: %#v defects=%v", after, defects)
	}
	if after.ReuseReason != reasonOutputMissing && after.Decision != reuseRerun {
		t.Fatalf("dest rename got decision=%q reason=%q defects=%v", after.Decision, after.ReuseReason, defects)
	}
}

func sequentialCopyDoc(aCmd, bCmd []string) Document {
	return Document{
		Name: "chain",
		Tasks: []TaskPlan{
			{
				ID:      "a",
				Name:    "a",
				Command: aCmd,
				Inputs:  []IO{{Name: "in", Path: "in/sample.txt"}},
				Outputs: []IO{{Name: "out", Path: "out/a.txt"}},
			},
			{
				ID:      "b",
				Name:    "b",
				Command: bCmd,
				Inputs:  []IO{{Name: "in", Path: "out/a.txt"}},
				Outputs: []IO{{Name: "out", Path: "out/b.txt"}},
			},
		},
		Edges: []Edge{
			{FromPort: "reads", ToTask: "a", ToPort: "in", Wait: []string{"in/sample.txt"}},
			{FromTask: "a", FromPort: "out", ToTask: "b", ToPort: "in", Wait: []string{"out/a.txt"}},
		},
	}
}
