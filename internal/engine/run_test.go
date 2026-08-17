package engine

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEngineDoesNotImportGobble(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("ParseDir() error = %v", err)
	}
	for _, p := range pkgs {
		for name, f := range p.Files {
			for _, imp := range f.Imports {
				path, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					t.Fatalf("%s: import %s", name, imp.Path.Value)
				}
				if path == "github.com/HahyeonJeon/gobble" {
					t.Fatalf("%s imports %s", name, path)
				}
			}
		}
	}
}

func TestRunProcessPublishesAndOccupies(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	defects := Run(Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	})
	if len(defects) != 0 {
		t.Fatalf("Run() defects %v, want none", defects)
	}
	got, err := os.ReadFile(filepath.Join(dir, "out", "sample.txt"))
	if err != nil {
		t.Fatalf("published output: %v", err)
	}
	if string(got) != "reads" {
		t.Fatalf("published output got %q, want reads", got)
	}
	isolate := filepath.Join(dir, ControlDir, "tasks", "copy", "work")
	if _, err := os.Stat(filepath.Join(isolate, "in", "sample.txt")); err != nil {
		t.Fatalf("staged input: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "work")); !os.IsNotExist(err) {
		t.Fatalf("created workspace work/, want isolation under %s", ControlDir)
	}
	mustJSONFile(t, filepath.Join(dir, ControlDir, RunIdentityFile))
	mustJSONFile(t, filepath.Join(dir, ControlDir, PlanFile))
	tasks := mustJSONFile(t, filepath.Join(dir, ControlDir, TasksFile))
	if _, err := os.Stat(filepath.Join(dir, ControlDir, "tasks", "copy", "stdout")); err != nil {
		t.Fatalf("stdout: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ControlDir, "tasks", "copy", "stderr")); err != nil {
		t.Fatalf("stderr: %v", err)
	}
	var doc jsonTasksFile
	if err := json.Unmarshal(tasks, &doc); err != nil {
		t.Fatalf("tasks.json: %v", err)
	}
	if len(doc.Tasks) != 1 {
		t.Fatalf("tasks.json tasks got %d, want 1", len(doc.Tasks))
	}
	st := doc.Tasks[0]
	if st.Status != StatusSucceeded || st.Executor != executorProcess {
		t.Fatalf("task state got status %q executor %q", st.Status, st.Executor)
	}
	if st.Resources.CPU != 1 || st.Resources.Memory != "512m" {
		t.Fatalf("recorded resources got cpu %v memory %q", st.Resources.CPU, st.Resources.Memory)
	}
	if len(st.Params) != 1 || st.Params[0].Name != "mode" || st.Params[0].Value != "fast" {
		t.Fatalf("recorded params got %#v", st.Params)
	}
}

func TestRunOccupiedSecondStart(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	req := Request{Workspace: dir, Document: sampleDoc("", "", "in/sample.txt", "out/sample.txt")}
	if defects := Run(req); len(defects) != 0 {
		t.Fatalf("first Run() defects %v, want none", defects)
	}
	before, err := os.ReadFile(filepath.Join(dir, "out", "sample.txt"))
	if err != nil {
		t.Fatalf("published output: %v", err)
	}
	defects := Run(req)
	if !hasDefect(defects, DefectOccupiedWorkspace, "") {
		t.Fatalf("second Run() defects %v, want occupied-workspace", defects)
	}
	after, err := os.ReadFile(filepath.Join(dir, "out", "sample.txt"))
	if err != nil {
		t.Fatalf("published output after second Run: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("second Run changed published output")
	}
}

func TestRunRefuseDoesNotOccupy(t *testing.T) {
	dir := t.TempDir()
	defects := Run(Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	})
	if !hasDefect(defects, DefectMissingInput, "copy.in") {
		t.Fatalf("Run() defects %v, want missing-input", defects)
	}
	if _, err := os.Stat(filepath.Join(dir, ControlDir)); !os.IsNotExist(err) {
		t.Fatalf("refused Run created %s", ControlDir)
	}
}

func TestRunMissingOutputUnpublished(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc("", "", "in/sample.txt", "out/sample.txt")
	doc.Tasks[0].Command = []string{"true"}
	defects := Run(Request{Workspace: dir, Document: doc})
	if !hasDefect(defects, DefectFailed, "copy") {
		t.Fatalf("Run() defects %v, want failed copy", defects)
	}
	if _, err := os.Stat(filepath.Join(dir, "out", "sample.txt")); !os.IsNotExist(err) {
		t.Fatalf("missing output was published")
	}
	if _, err := os.Stat(filepath.Join(dir, ControlDir, "tasks", "copy", "work")); err != nil {
		t.Fatalf("work directory after failure: %v", err)
	}
}

func TestRunPublishRollback(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	writeCheckFile(t, filepath.Join(dir, "out", "blocked"), "not-a-dir")
	doc := Document{
		Name: "rollback",
		Tasks: []TaskPlan{{
			ID:      "copy",
			Name:    "copy",
			Command: []string{"sh", "-c", "cp in/sample.txt out/first.txt && cp in/sample.txt out/blocked/second.txt"},
			Inputs:  []IO{{Name: "in", Path: "in/sample.txt"}},
			Outputs: []IO{
				{Name: "first", Path: "out/first.txt"},
				{Name: "second", Path: "out/blocked/second.txt"},
			},
		}},
		Edges: []Edge{{FromPort: "reads", ToTask: "copy", ToPort: "in"}},
	}
	defects := Run(Request{Workspace: dir, Document: doc})
	if !hasDefect(defects, DefectFailed, "copy") {
		t.Fatalf("Run() defects %v, want failed copy", defects)
	}
	if _, err := os.Stat(filepath.Join(dir, "out", "first.txt")); !os.IsNotExist(err) {
		t.Fatalf("partial publish left out/first.txt")
	}
}

func TestRunUnparseableMemory(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc("", "", "in/sample.txt", "out/sample.txt")
	doc.Tasks[0].Resources.Memory = "not-a-size"
	defects := Run(Request{Workspace: dir, Document: doc})
	if len(defects) != 0 {
		t.Fatalf("unparseable memory Run() defects %v, want none", defects)
	}
	raw := mustJSONFile(t, filepath.Join(dir, ControlDir, TasksFile))
	var file jsonTasksFile
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("tasks.json: %v", err)
	}
	if file.Tasks[0].Resources.Memory != "not-a-size" {
		t.Fatalf("recorded memory got %q, want not-a-size", file.Tasks[0].Resources.Memory)
	}
}

func TestScheduleCapOneIsSerial(t *testing.T) {
	orig := execTask
	t.Cleanup(func() { execTask = orig })
	var mu sync.Mutex
	current, max := 0, 0
	execTask = func(workspace string, task TaskPlan) report {
		mu.Lock()
		current++
		if current > max {
			max = current
		}
		mu.Unlock()
		time.Sleep(20 * time.Millisecond)
		for _, out := range task.Outputs {
			writeCheckFile(t, workspaceFile(workspace, out.Path), task.ID)
		}
		mu.Lock()
		current--
		mu.Unlock()
		return report{ID: task.ID, Published: true}
	}
	dir := t.TempDir()
	defects := Run(Request{Workspace: dir, Cap: 1, Document: twoIndependentDoc()})
	if len(defects) != 0 {
		t.Fatalf("Run() defects %v, want none", defects)
	}
	if max != 1 {
		t.Fatalf("max in flight got %d, want 1", max)
	}
}

func TestScheduleCapTwoLaunchesTogether(t *testing.T) {
	orig := execTask
	t.Cleanup(func() { execTask = orig })
	started := make(chan string, 2)
	release := make(chan struct{})
	var inFlight int32
	execTask = func(workspace string, task TaskPlan) report {
		atomic.AddInt32(&inFlight, 1)
		started <- task.ID
		<-release
		for _, out := range task.Outputs {
			writeCheckFile(t, workspaceFile(workspace, out.Path), task.ID)
		}
		atomic.AddInt32(&inFlight, -1)
		return report{ID: task.ID, Published: true}
	}
	dir := t.TempDir()
	done := make(chan []Defect, 1)
	go func() {
		done <- Run(Request{Workspace: dir, Cap: 2, Document: twoIndependentDoc()})
	}()
	seen := map[string]bool{}
	for i := 0; i < 2; i++ {
		select {
		case id := <-started:
			seen[id] = true
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for launch %d; seen %v", i+1, seen)
		}
	}
	if atomic.LoadInt32(&inFlight) != 2 {
		t.Fatalf("in flight got %d, want 2", atomic.LoadInt32(&inFlight))
	}
	close(release)
	select {
	case defects := <-done:
		if len(defects) != 0 {
			t.Fatalf("Run() defects %v, want none", defects)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Run")
	}
	if !seen["left"] || !seen["right"] {
		t.Fatalf("launched %v, want left and right", seen)
	}
	mustJSONFile(t, filepath.Join(dir, ControlDir, TasksFile))
}

func TestScheduleBlocksDependents(t *testing.T) {
	orig := execTask
	t.Cleanup(func() { execTask = orig })
	var launched []string
	var mu sync.Mutex
	execTask = func(workspace string, task TaskPlan) report {
		mu.Lock()
		launched = append(launched, task.ID)
		mu.Unlock()
		if task.ID == "fail" {
			return report{ID: task.ID, Exit: 1, Message: "exit 1"}
		}
		for _, out := range task.Outputs {
			writeCheckFile(t, workspaceFile(workspace, out.Path), task.ID)
		}
		return report{ID: task.ID, Published: true}
	}
	dir := t.TempDir()
	defects := Run(Request{Workspace: dir, Cap: 2, Document: failAndIndependentDoc()})
	if !hasDefect(defects, DefectFailed, "fail") {
		t.Fatalf("Run() defects %v, want failed fail", defects)
	}
	mu.Lock()
	got := append([]string(nil), launched...)
	mu.Unlock()
	for _, id := range got {
		if id == "dep" {
			t.Fatalf("launched blocked dependent: %v", got)
		}
	}
	raw := mustJSONFile(t, filepath.Join(dir, ControlDir, TasksFile))
	var file jsonTasksFile
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("tasks.json: %v", err)
	}
	byID := map[string]jsonTaskState{}
	for _, st := range file.Tasks {
		byID[st.ID] = st
	}
	if byID["fail"].Status != StatusFailed {
		t.Fatalf("fail status got %q, want failed", byID["fail"].Status)
	}
	if byID["dep"].Status != StatusBlocked {
		t.Fatalf("dep status got %q, want blocked", byID["dep"].Status)
	}
	if byID["ok"].Status != StatusSucceeded {
		t.Fatalf("ok status got %q, want succeeded", byID["ok"].Status)
	}
}

func twoIndependentDoc() Document {
	return Document{
		Name: "fanout",
		Tasks: []TaskPlan{
			{ID: "left", Name: "left", Outputs: []IO{{Name: "out", Path: "out/left.txt"}}},
			{ID: "right", Name: "right", Outputs: []IO{{Name: "out", Path: "out/right.txt"}}},
		},
	}
}

func failAndIndependentDoc() Document {
	return Document{
		Name: "contain",
		Tasks: []TaskPlan{
			{ID: "fail", Name: "fail", Outputs: []IO{{Name: "out", Path: "out/fail.txt"}}},
			{ID: "dep", Name: "dep", Inputs: []IO{{Name: "in", Path: "out/fail.txt"}}, Outputs: []IO{{Name: "out", Path: "out/dep.txt"}}},
			{ID: "ok", Name: "ok", Outputs: []IO{{Name: "out", Path: "out/ok.txt"}}},
		},
		Edges: []Edge{
			{FromTask: "fail", FromPort: "out", ToTask: "dep", ToPort: "in"},
		},
	}
}

func mustJSONFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("%s is not complete JSON: %v\n%s", path, err, data)
	}
	return data
}
