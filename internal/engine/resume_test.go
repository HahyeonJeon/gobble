package engine

import (
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
}

func TestCheckResumeOutputsUnattributed(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "out", "b.txt"), "stray")
	doc := Document{
		Tasks: []TaskPlan{
			{ID: "a", Outputs: []IO{{Name: "out", Path: "out/a.txt"}}},
			{ID: "b", Outputs: []IO{{Name: "out", Path: "out/b.txt"}}},
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
	}}
	class := remainingClass{
		Decision: map[string]reuseDecision{
			"a": {Identity: "a", Decision: reuseRerun},
			"b": {Identity: "b", Decision: reuseRerun},
		},
	}
	defects := checkResumeOutputs(dir, doc, tasks, class)
	if !hasDefect(defects, DefectOutputExists, "b.out") {
		t.Fatalf("defects %v, want output-exists b.out", defects)
	}
	if hasDefect(defects, DefectOutputExists, "a.out") {
		t.Fatalf("authorized rerun dest treated as output-exists")
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
