package engine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble/internal/engine/exec"
)

func TestClassifyResumeChangeClasses(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "a.txt"), "reads")
	writeCheckFile(t, filepath.Join(dir, "in", "other.txt"), "other")
	writeCheckFile(t, filepath.Join(dir, "out", "a.txt"), "reads")
	writeCheckFile(t, filepath.Join(dir, "out", "b.txt"), "reads")
	inRec := mustFileRecord(t, filepath.Join(dir, "in", "a.txt"), "in/a.txt")
	outA := mustFileRecord(t, filepath.Join(dir, "out", "a.txt"), "out/a.txt")
	outB := mustFileRecord(t, filepath.Join(dir, "out", "b.txt"), "out/b.txt")

	copyPlan := Document{
		Tasks: []TaskPlan{{
			ID:      "copy",
			Command: []string{"cp", "in/a.txt", "out/a.txt"},
			Inputs:  []IO{{Name: "in", Path: "in/a.txt"}},
			Outputs: []IO{{Name: "out", Path: "out/a.txt"}},
		}},
		Edges: []Edge{{FromPort: "reads", ToTask: "copy", ToPort: "in", Wait: []string{"in/a.txt"}}},
	}
	copyState := jsonTaskState{
		ID:           "copy",
		Status:       StatusSucceeded,
		Command:      []string{"cp", "in/a.txt", "out/a.txt"},
		Attempt:      1,
		Fingerprints: []jsonFileHash{inRec},
		Checksums:    []jsonFileHash{outA},
		Lineage:      []jsonLineage{{Producer: "copy", Path: "out/a.txt", Checksum: outA.SHA256}},
	}
	extraPlan := Document{
		Tasks: append(append([]TaskPlan(nil), copyPlan.Tasks...), TaskPlan{
			ID:      "extra",
			Command: []string{"true"},
			Outputs: []IO{{Name: "out", Path: "out/extra.txt"}},
		}),
		Edges: copyPlan.Edges,
	}
	extraState := jsonTaskState{
		ID:      "extra",
		Status:  StatusSucceeded,
		Command: []string{"true"},
		Attempt: 1,
	}
	chain := Document{
		Tasks: []TaskPlan{
			{
				ID:      "a",
				Command: []string{"cp", "in/a.txt", "out/a.txt"},
				Inputs:  []IO{{Name: "in", Path: "in/a.txt"}},
				Outputs: []IO{{Name: "out", Path: "out/a.txt"}},
			},
			{
				ID:      "b",
				Command: []string{"cp", "out/a.txt", "out/b.txt"},
				Inputs:  []IO{{Name: "in", Path: "out/a.txt"}},
				Outputs: []IO{{Name: "out", Path: "out/b.txt"}},
			},
		},
		Edges: []Edge{
			{FromPort: "reads", ToTask: "a", ToPort: "in", Wait: []string{"in/a.txt"}},
			{FromTask: "a", FromPort: "out", ToTask: "b", ToPort: "in", Wait: []string{"out/a.txt"}},
		},
	}
	rewired := Document{
		Tasks: chain.Tasks,
		Edges: []Edge{
			{FromPort: "reads", ToTask: "a", ToPort: "in", Wait: []string{"in/a.txt"}},
			{FromPort: "reads", ToTask: "b", ToPort: "in", Wait: []string{"in/a.txt"}},
		},
	}
	chainStates := []jsonTaskState{
		{
			ID:           "a",
			Status:       StatusSucceeded,
			Command:      []string{"cp", "in/a.txt", "out/a.txt"},
			Attempt:      1,
			Fingerprints: []jsonFileHash{inRec},
			Checksums:    []jsonFileHash{outA},
			Lineage:      []jsonLineage{{Producer: "a", Path: "out/a.txt", Checksum: outA.SHA256}},
		},
		{
			ID:           "b",
			Status:       StatusSucceeded,
			Command:      []string{"cp", "out/a.txt", "out/b.txt"},
			Attempt:      1,
			Fingerprints: []jsonFileHash{outA},
			Checksums:    []jsonFileHash{outB},
			Lineage:      []jsonLineage{{Producer: "b", Path: "out/b.txt", Checksum: outB.SHA256}},
		},
	}
	waitChanged := copyPlan
	waitChanged.Edges = []Edge{{FromPort: "reads", ToTask: "copy", ToPort: "in", Wait: []string{"in/other.txt"}}}
	destRenamed := copyPlan
	destRenamed.Tasks = append([]TaskPlan(nil), copyPlan.Tasks...)
	destRenamed.Tasks[0].Outputs = []IO{{Name: "out", Path: "out/renamed.txt"}}
	cmdChanged := copyPlan
	cmdChanged.Tasks = append([]TaskPlan(nil), copyPlan.Tasks...)
	cmdChanged.Tasks[0].Command = []string{"true"}
	resourcesOnly := copyPlan
	resourcesOnly.Tasks = append([]TaskPlan(nil), copyPlan.Tasks...)
	resourcesOnly.Tasks[0].Resources = ResourcePlan{CPU: 2, Memory: "1g"}

	tests := []struct {
		name         string
		recorded     Document
		supplied     Document
		tasks        []jsonTaskState
		ident        string
		wantChange   string
		wantDecision string
	}{
		{"Added", copyPlan, extraPlan, []jsonTaskState{copyState}, "extra", changeAdded, reuseRerun},
		{"Removed", extraPlan, copyPlan, []jsonTaskState{copyState, extraState}, "extra", changeRemoved, ""},
		{"Rewired", chain, rewired, chainStates, "b", changeRewired, reuseRerun},
		{"Repathed dest", copyPlan, destRenamed, []jsonTaskState{copyState}, "copy", changeRepathed, reuseRerun},
		{"Repathed wait-only", copyPlan, waitChanged, []jsonTaskState{copyState}, "copy", changeRepathed, reuseRerun},
		{"IdentityChanged", copyPlan, cmdChanged, []jsonTaskState{copyState}, "copy", changeIdentityChanged, reuseRerun},
		{"Unchanged", copyPlan, copyPlan, []jsonTaskState{copyState}, "copy", changeUnchanged, reuseReused},
		{"resources-only Unchanged", copyPlan, resourcesOnly, []jsonTaskState{copyState}, "copy", changeUnchanged, reuseReused},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyResume(dir, tc.recorded, tc.supplied, tc.tasks)
			dec := got.Decision[tc.ident]
			if dec.Change != tc.wantChange {
				t.Fatalf("change got %q, want %q (%#v)", dec.Change, tc.wantChange, dec)
			}
			if dec.Decision != tc.wantDecision {
				t.Fatalf("decision got %q, want %q (%#v)", dec.Decision, tc.wantDecision, dec)
			}
		})
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
	wantPerm := filePerm(t, src)
	if err := stagedReplace(src, dst); err != nil {
		t.Fatalf("stagedReplace() error = %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil || string(got) != "next" {
		t.Fatalf("dest got %q, want next", got)
	}
	if srcPerm := filePerm(t, src); srcPerm != wantPerm {
		t.Fatalf("source mode got %o, want %o", srcPerm, wantPerm)
	}
	info, err := os.Lstat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() {
		t.Fatal("replaced dest is not regular")
	}
	writeCheckFile(t, src, "again")
	if err := stagedReplace(src, dst); err != nil {
		t.Fatalf("stagedReplace() second error = %v", err)
	}
	got, err = os.ReadFile(dst)
	if err != nil || string(got) != "again" {
		t.Fatalf("dest after second replace got %q, want again", got)
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
	if defects := Run(t.Context(), Request{Workspace: dir, Document: doc}); len(defects) != 0 {
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
	if defects := Resume(t.Context(), Request{Workspace: dir, Document: next}); len(defects) != 0 {
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
	if defects := Run(t.Context(), Request{Workspace: dir, Document: doc}); len(defects) != 0 {
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
	if defects := Run(t.Context(), Request{Workspace: dir, Document: doc}); len(defects) != 0 {
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
	if defects := Resume(t.Context(), Request{Workspace: dir, Document: next}); len(defects) != 0 {
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
	if defects := Run(t.Context(), Request{Workspace: dir, Document: doc}); len(defects) != 0 {
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
	inner := schedulerExecutor()
	useExec(t, &fnExec{
		submit: func(ctx context.Context, job exec.Job) (exec.Handle, exec.Report, error) {
			if job.Identity == "b" {
				got, err := os.ReadFile(filepath.Join(isolateWorkspace(job.Isolate), "out", "a.txt"))
				if err != nil {
					t.Errorf("b started before a dest exists: %v", err)
				} else if string(got) != "new" {
					t.Errorf("b saw a dest %q, want new", got)
				}
			}
			return inner.Submit(ctx, job)
		},
		poll:      inner.Poll,
		cancel:    inner.Cancel,
		reconcile: inner.Reconcile,
	})
	if defects := Resume(t.Context(), Request{Workspace: dir, Document: next}); len(defects) != 0 {
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
	if defects := Run(t.Context(), Request{Workspace: dir, Document: doc}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	forceDeadOwner(t, dir)
	if defects := Release(dir); len(defects) != 0 {
		t.Fatalf("Release() defects %v", defects)
	}
	next := sampleDoc("", "", "in/sample.txt", "out/renamed.txt")
	next.Tasks[0].Command = doc.Tasks[0].Command
	defects := Resume(t.Context(), Request{Workspace: dir, Document: next})
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
