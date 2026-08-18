package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSessionProofTwoReservedKeysIsolate(t *testing.T) {
	left := TaskPlan{
		ID:       "left",
		Name:     "left",
		Instance: "s1",
		Command:  []string{"cp", "in/a.txt", "out/a.txt"},
		Inputs:   []IO{{Name: "in", Path: "in/a.txt"}},
		Outputs:  []IO{{Name: "out", Path: "out/a.txt"}},
	}
	right := TaskPlan{
		ID:       "right",
		Name:     "right",
		Instance: "s2",
		Command:  []string{"cp", "in/b.txt", "out/b.txt"},
		Inputs:   []IO{{Name: "in", Path: "in/b.txt"}},
		Outputs:  []IO{{Name: "out", Path: "out/b.txt"}},
	}
	applyReservedDefaults(&left)
	applyReservedDefaults(&right)
	leftIdent := reservedIdentity(left)
	rightIdent := reservedIdentity(right)
	if leftIdent == rightIdent {
		t.Fatalf("reserved keys collided: %s", leftIdent)
	}
	leftIso := isolateRel(left)
	rightIso := isolateRel(right)
	if leftIso == rightIso {
		t.Fatalf("reserved keys share isolate %q", leftIso)
	}
	if leftIso != ControlDir+"/tasks/left/s1/0/1" || rightIso != ControlDir+"/tasks/right/s2/0/1" {
		t.Fatalf("isolate paths got %q and %q", leftIso, rightIso)
	}

	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "a.txt"), "left-reads")
	writeCheckFile(t, filepath.Join(dir, "in", "b.txt"), "right-reads")
	doc := Document{Name: "pair", Tasks: []TaskPlan{left, right}}
	if defects := Run(Request{Workspace: dir, Document: doc, Cap: 2}); len(defects) != 0 {
		t.Fatalf("Run() defects %v", defects)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(leftIso), "work")); err != nil {
		t.Fatalf("left work dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rightIso), "work")); err != nil {
		t.Fatalf("right work dir: %v", err)
	}

	instances := inspectJSONLByID(t, dir, viewInstances)
	if instances[leftIdent]["instance"] != "s1" || instances[rightIdent]["instance"] != "s2" {
		t.Fatalf("instances got left %#v right %#v", instances[leftIdent], instances[rightIdent])
	}
	if instances[leftIdent]["status"] != StatusSucceeded || instances[rightIdent]["status"] != StatusSucceeded {
		t.Fatalf("run statuses got left %#v right %#v", instances[leftIdent], instances[rightIdent])
	}
	lineage := inspectLineageByProducer(t, dir)
	if len(lineage[leftIdent]) == 0 || len(lineage[rightIdent]) == 0 {
		t.Fatalf("lineage missing a reserved key: %#v", lineage)
	}
	for _, lin := range lineage[leftIdent] {
		if lin.Consumer == rightIdent || lin.Producer == rightIdent {
			t.Fatalf("left lineage mixed with right: %#v", lin)
		}
	}

	forceDeadOwner(t, dir)
	if defects := Release(dir); len(defects) != 0 {
		t.Fatalf("Release() defects %v", defects)
	}
	rerun := doc
	rerun.Tasks[0].Command = []string{"sh", "-c", "exit 1"}
	if defects := Resume(Request{Workspace: dir, Document: rerun, Cap: 2}); !hasDefect(defects, DefectFailed, "left") {
		t.Fatalf("Resume() defects %v, want failed left", defects)
	}

	reuse := inspectJSONLByID(t, dir, viewReuse)
	if reuse[leftIdent]["decision"] != reuseRerun || reuse[rightIdent]["decision"] != reuseReused {
		t.Fatalf("reuse records shared or wrong: %#v", reuse)
	}
	if reuse[leftIdent]["reason"] == "" || reuse[rightIdent]["reason"] == "" {
		t.Fatalf("reuse missing reason: %#v", reuse)
	}
	if reuse[leftIdent]["identity"] == reuse[rightIdent]["identity"] {
		t.Fatalf("reserved keys share reuse record %#v", reuse[leftIdent])
	}
	after := inspectJSONLByID(t, dir, viewInstances)
	if after[leftIdent]["attempt"] != float64(2) {
		t.Fatalf("left attempt got %#v, want 2", after[leftIdent])
	}
	if after[rightIdent]["attempt"] != float64(1) {
		t.Fatalf("right executed on left failure: %#v", after[rightIdent])
	}
	if _, err := os.Stat(filepath.Join(dir, ControlDir, "tasks", "right", "s2", "0", "2", "work")); !os.IsNotExist(err) {
		t.Fatalf("right grew a new attempt isolate")
	}
	remaining := inspectJSONLByID(t, dir, viewRemaining)
	if remaining[leftIdent]["remaining"] != true {
		t.Fatalf("left remaining got %#v", remaining[leftIdent])
	}
	if remaining[rightIdent] != nil {
		t.Fatalf("right listed as remaining: %#v", remaining[rightIdent])
	}
}

func inspectJSONLByID(t *testing.T, workspace, view string) map[string]map[string]any {
	t.Helper()
	raw, defects := Inspect(workspace, view, "")
	if len(defects) != 0 {
		t.Fatalf("Inspect(%s) defects %v", view, defects)
	}
	return remainingByID(t, raw)
}

func inspectLineageByProducer(t *testing.T, workspace string) map[string][]jsonLineage {
	t.Helper()
	raw, defects := Inspect(workspace, viewLineage, "")
	if len(defects) != 0 {
		t.Fatalf("Inspect(lineage) defects %v", defects)
	}
	var doc inspectLineageDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("lineage JSON: %v\n%s", err, raw)
	}
	out := map[string][]jsonLineage{}
	for _, lin := range doc.Lineage {
		out[lin.Producer] = append(out[lin.Producer], lin)
	}
	return out
}
