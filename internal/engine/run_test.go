package engine

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	isolate := filepath.Join(dir, ControlDir, "tasks", "copy", "_", "0", "1", "work")
	if _, err := os.Stat(filepath.Join(isolate, "in", "sample.txt")); err != nil {
		t.Fatalf("staged input: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "work")); !os.IsNotExist(err) {
		t.Fatalf("created workspace work/, want isolation under %s", ControlDir)
	}
	mustJSONFile(t, filepath.Join(dir, ControlDir, RunIdentityFile))
	mustJSONFile(t, filepath.Join(dir, ControlDir, PlanFile))
	tasks := mustJSONFile(t, filepath.Join(dir, ControlDir, TasksFile))
	if _, err := os.Stat(filepath.Join(dir, ControlDir, "tasks", "copy", "_", "0", "1", "stdout")); err != nil {
		t.Fatalf("stdout: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ControlDir, "tasks", "copy", "_", "0", "1", "stderr")); err != nil {
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

func TestRunRestagedInputCopiesFromSource(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	if _, err := os.Stat(filepath.Join(dir, "work", "sample.txt")); !os.IsNotExist(err) {
		t.Fatalf("workspace dest existed before Run")
	}
	defects := Run(Request{
		Workspace: dir,
		Document: Document{
			Name: "restage",
			Tasks: []TaskPlan{{
				ID:      "copy",
				Name:    "copy",
				Command: []string{"cp", "work/sample.txt", "out/sample.txt"},
				Inputs:  []IO{{Name: "in", Path: "work/sample.txt", Source: "in/sample.txt"}},
				Outputs: []IO{{Name: "out", Path: "out/sample.txt"}},
			}},
			Edges: []Edge{{FromPort: "reads", ToTask: "copy", ToPort: "in", Wait: []string{"in/sample.txt"}}},
		},
	})
	if len(defects) != 0 {
		t.Fatalf("Run() defects %v, want none", defects)
	}
	isolateDest := filepath.Join(dir, ControlDir, "tasks", "copy", "_", "0", "1", "work", "work", "sample.txt")
	got, err := os.ReadFile(isolateDest)
	if err != nil {
		t.Fatalf("isolate dest: %v", err)
	}
	if string(got) != "reads" {
		t.Fatalf("isolate dest got %q, want reads", got)
	}
	published, err := os.ReadFile(filepath.Join(dir, "out", "sample.txt"))
	if err != nil {
		t.Fatalf("published output: %v", err)
	}
	if string(published) != "reads" {
		t.Fatalf("published output got %q, want reads", published)
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
	if _, err := os.Stat(filepath.Join(dir, ControlDir, "tasks", "copy", "_", "0", "1", "work")); err != nil {
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
	if !hasDefect(defects, DefectInvalidMemory, "copy") {
		t.Fatalf("unparseable memory Run() defects %v, want invalid-memory copy", defects)
	}
	if _, err := os.Stat(filepath.Join(dir, ControlDir)); !os.IsNotExist(err) {
		t.Fatalf("unparseable memory Run occupied workspace")
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

func TestRunFromInPort(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "sample.fq"), "reads")
	doc := Document{
		Name: "from-in",
		Tasks: []TaskPlan{
			{
				ID:      "prep",
				Name:    "prep",
				Command: []string{"sh", "-c", "cp sample.fq prep.fq"},
				Inputs:  []IO{{Name: "src", Path: "sample.fq"}},
				Outputs: []IO{{Name: "out", Path: "prep.fq"}},
			},
			{
				ID:      "copy",
				Name:    "copy",
				Command: []string{"cp", "sample.fq", "copy.fq"},
				Inputs:  []IO{{Name: "in", Path: "sample.fq"}},
				Outputs: []IO{{Name: "out", Path: "copy.fq"}},
			},
		},
		Edges: []Edge{
			{FromPort: "reads", ToTask: "prep", ToPort: "src", Wait: []string{"sample.fq"}},
			{FromTask: "prep", FromPort: "src", ToTask: "copy", ToPort: "in", Wait: []string{"sample.fq"}},
		},
	}
	defects := Run(Request{Workspace: dir, Document: doc})
	if len(defects) != 0 {
		t.Fatalf("FromIn Run() defects %v, want none", defects)
	}
	got, err := os.ReadFile(filepath.Join(dir, "copy.fq"))
	if err != nil {
		t.Fatalf("published copy.fq: %v", err)
	}
	if string(got) != "reads" {
		t.Fatalf("copy.fq got %q, want reads", got)
	}
	if _, err := os.ReadFile(filepath.Join(dir, "prep.fq")); err != nil {
		t.Fatalf("published prep.fq: %v", err)
	}
}

func TestRunRelatedFileOutputFrom(t *testing.T) {
	t.Run("starts after published bam", func(t *testing.T) {
		dir := t.TempDir()
		defects := Run(Request{Workspace: dir, Document: relatedFileOutputFromDoc(
			[]string{"sh", "-c", "printf bam > aln.bam"},
			[]string{"cp", "aln.bam", "aln.bam.bai"},
		)})
		if len(defects) != 0 {
			t.Fatalf("related-file From Run() defects %v, want none", defects)
		}
		got, err := os.ReadFile(filepath.Join(dir, "aln.bam.bai"))
		if err != nil {
			t.Fatalf("published bai: %v", err)
		}
		if string(got) != "bam" {
			t.Fatalf("published bai got %q, want bam", got)
		}
		byID := taskStates(t, dir)
		if byID["align"].Status != StatusSucceeded {
			t.Fatalf("align status got %q, want succeeded", byID["align"].Status)
		}
		if byID["index"].Status != StatusSucceeded {
			t.Fatalf("index status got %q, want succeeded", byID["index"].Status)
		}
	})

	t.Run("missing from-path stays not-started", func(t *testing.T) {
		orig := execTask
		t.Cleanup(func() { execTask = orig })
		var launched []string
		var mu sync.Mutex
		execTask = func(workspace string, task TaskPlan) report {
			mu.Lock()
			launched = append(launched, task.ID)
			mu.Unlock()
			return report{ID: task.ID, Published: true}
		}
		dir := t.TempDir()
		defects := Run(Request{Workspace: dir, Document: relatedFileOutputFromDoc(
			[]string{"true"},
			[]string{"true"},
		)})
		if !hasDefect(defects, DefectFailed, "index") {
			t.Fatalf("missing from-path Run() defects %v, want failed index", defects)
		}
		mu.Lock()
		got := append([]string(nil), launched...)
		mu.Unlock()
		for _, id := range got {
			if id == "index" {
				t.Fatalf("launched index without published bam: %v", got)
			}
		}
		byID := taskStates(t, dir)
		if byID["align"].Status != StatusSucceeded {
			t.Fatalf("align status got %q, want succeeded", byID["align"].Status)
		}
		if byID["index"].Status != StatusNotStarted {
			t.Fatalf("index status got %q, want not-started", byID["index"].Status)
		}
	})

	t.Run("failed upstream stays blocked", func(t *testing.T) {
		dir := t.TempDir()
		defects := Run(Request{Workspace: dir, Document: relatedFileOutputFromDoc(
			[]string{"false"},
			[]string{"cp", "aln.bam", "aln.bam.bai"},
		)})
		if !hasDefect(defects, DefectFailed, "align") {
			t.Fatalf("failed upstream Run() defects %v, want failed align", defects)
		}
		if hasDefect(defects, DefectFailed, "index") {
			t.Fatalf("failed upstream Run() defects %v, want index blocked not failed", defects)
		}
		byID := taskStates(t, dir)
		if byID["align"].Status != StatusFailed {
			t.Fatalf("align status got %q, want failed", byID["align"].Status)
		}
		if byID["index"].Status != StatusBlocked {
			t.Fatalf("index status got %q, want blocked", byID["index"].Status)
		}
		if _, err := os.Stat(filepath.Join(dir, ControlDir, "tasks", "index", "_", "0", "1", "work")); !os.IsNotExist(err) {
			t.Fatalf("launched blocked index")
		}
	})
}

func relatedFileOutputFromDoc(alignCmd, indexCmd []string) Document {
	return Document{
		Name: "related-bai",
		Tasks: []TaskPlan{
			{
				ID:      "align",
				Name:    "align",
				Command: alignCmd,
				Outputs: []IO{{Name: "bam", Path: "aln.bam"}},
			},
			{
				ID:      "index",
				Name:    "index",
				Command: indexCmd,
				Inputs:  []IO{{Name: "bam", Path: "aln.bam"}},
				Outputs: []IO{{Name: "bai", Path: "aln.bam.bai"}},
			},
		},
		Edges: []Edge{
			{FromTask: "align", FromPort: "bam", ToTask: "index", ToPort: "bam", Wait: []string{"aln.bam"}},
			{FromTask: "align", FromPort: "bam", ToTask: "index", ToPort: "bai", Wait: []string{"aln.bam"}},
		},
	}
}

func taskStates(t *testing.T, workspace string) map[string]jsonTaskState {
	t.Helper()
	raw := mustJSONFile(t, filepath.Join(workspace, ControlDir, TasksFile))
	var file jsonTasksFile
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("tasks.json: %v", err)
	}
	byID := make(map[string]jsonTaskState, len(file.Tasks))
	for _, st := range file.Tasks {
		byID[st.ID] = st
	}
	return byID
}

func TestRunNotStartedIsFailed(t *testing.T) {
	orig := execTask
	t.Cleanup(func() { execTask = orig })
	execTask = func(workspace string, task TaskPlan) report {
		return report{ID: task.ID, Published: true}
	}
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := Document{
		Name: "stall",
		Tasks: []TaskPlan{
			{
				ID:      "prep",
				Name:    "prep",
				Command: []string{"true"},
				Inputs:  []IO{{Name: "src", Path: "in/sample.txt"}},
				Outputs: []IO{{Name: "out", Path: "out/prep.txt"}},
			},
			{
				ID:      "copy",
				Name:    "copy",
				Command: []string{"true"},
				Inputs:  []IO{{Name: "in", Path: "out/prep.txt"}},
				Outputs: []IO{{Name: "out", Path: "out/copy.txt"}},
			},
		},
		Edges: []Edge{
			{FromPort: "reads", ToTask: "prep", ToPort: "src", Wait: []string{"in/sample.txt"}},
			{FromTask: "prep", FromPort: "out", ToTask: "copy", ToPort: "in", Wait: []string{"out/prep.txt"}},
		},
	}
	defects := Run(Request{Workspace: dir, Document: doc})
	if !hasDefect(defects, DefectFailed, "copy") {
		t.Fatalf("not-started Run() defects %v, want failed copy", defects)
	}
}

func TestRunPersistError(t *testing.T) {
	orig := execTask
	t.Cleanup(func() { execTask = orig })
	execTask = func(workspace string, task TaskPlan) report {
		p := filepath.Join(workspace, ControlDir, TasksFile)
		if err := os.Remove(p); err != nil {
			t.Errorf("remove tasks.json: %v", err)
		}
		if err := os.Mkdir(p, 0o755); err != nil {
			t.Errorf("mkdir tasks.json: %v", err)
		}
		for _, out := range task.Outputs {
			writeCheckFile(t, workspaceFile(workspace, out.Path), task.ID)
		}
		return report{ID: task.ID, Published: true}
	}
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	defects := Run(Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	})
	if !hasDefect(defects, DefectInvalidPath, "") {
		t.Fatalf("persist Run() defects %v, want persist invalid-path", defects)
	}
}

func TestAdmitResources(t *testing.T) {
	origCap := readHostCapacity
	origExec := execTask
	t.Cleanup(func() {
		readHostCapacity = origCap
		execTask = origExec
	})

	t.Run("cpu remaining is serial", func(t *testing.T) {
		readHostCapacity = func() hostCapacity {
			return hostCapacity{CPU: 1, CPUKnown: true, Memory: 1 << 30, MemKnown: true}
		}
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
		doc := twoIndependentDoc()
		doc.Tasks[0].Resources.CPU = 1
		doc.Tasks[1].Resources.CPU = 1
		defects := Run(Request{Workspace: dir, Cap: 2, Document: doc})
		if len(defects) != 0 {
			t.Fatalf("cpu admit Run() defects %v, want none", defects)
		}
		if max != 1 {
			t.Fatalf("cpu admit max in flight got %d, want 1", max)
		}
	})

	t.Run("memory remaining is serial", func(t *testing.T) {
		readHostCapacity = func() hostCapacity {
			return hostCapacity{CPU: 8, CPUKnown: true, Memory: 512 << 20, MemKnown: true}
		}
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
		doc := twoIndependentDoc()
		doc.Tasks[0].Resources.Memory = "512m"
		doc.Tasks[1].Resources.Memory = "512m"
		defects := Run(Request{Workspace: dir, Cap: 2, Document: doc})
		if len(defects) != 0 {
			t.Fatalf("memory admit Run() defects %v, want none", defects)
		}
		if max != 1 {
			t.Fatalf("memory admit max in flight got %d, want 1", max)
		}
	})

	t.Run("zero request does not consume", func(t *testing.T) {
		readHostCapacity = func() hostCapacity {
			return hostCapacity{CPU: 1, CPUKnown: true, Memory: 1024, MemKnown: true}
		}
		started := make(chan string, 2)
		release := make(chan struct{})
		execTask = func(workspace string, task TaskPlan) report {
			started <- task.ID
			<-release
			for _, out := range task.Outputs {
				writeCheckFile(t, workspaceFile(workspace, out.Path), task.ID)
			}
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
				t.Fatalf("zero request timeout waiting for launch %d; seen %v", i+1, seen)
			}
		}
		close(release)
		select {
		case defects := <-done:
			if len(defects) != 0 {
				t.Fatalf("zero request Run() defects %v, want none", defects)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("zero request timeout waiting for Run")
		}
	})

	t.Run("unspecified axis binds count only", func(t *testing.T) {
		readHostCapacity = func() hostCapacity {
			return hostCapacity{}
		}
		started := make(chan string, 2)
		release := make(chan struct{})
		execTask = func(workspace string, task TaskPlan) report {
			started <- task.ID
			<-release
			for _, out := range task.Outputs {
				writeCheckFile(t, workspaceFile(workspace, out.Path), task.ID)
			}
			return report{ID: task.ID, Published: true}
		}
		dir := t.TempDir()
		doc := twoIndependentDoc()
		doc.Tasks[0].Resources = ResourcePlan{CPU: 8, Memory: "8g"}
		doc.Tasks[1].Resources = ResourcePlan{CPU: 8, Memory: "8g"}
		done := make(chan []Defect, 1)
		go func() {
			done <- Run(Request{Workspace: dir, Cap: 2, Document: doc})
		}()
		for i := 0; i < 2; i++ {
			select {
			case <-started:
			case <-time.After(2 * time.Second):
				t.Fatalf("unspecified axis timeout waiting for launch %d", i+1)
			}
		}
		close(release)
		select {
		case defects := <-done:
			if len(defects) != 0 {
				t.Fatalf("unspecified axis Run() defects %v, want none", defects)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("unspecified axis timeout waiting for Run")
		}
	})
}

func TestRunScriptWrapper(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc("", "", "in/sample.txt", "out/sample.txt")
	doc.Tasks[0].Command = nil
	doc.Tasks[0].Script = "cp in/sample.txt out/sample.txt\nprintf '%s\\n' \"$-\" > out/flags.txt"
	doc.Tasks[0].Outputs = []IO{
		{Name: "out", Path: "out/sample.txt"},
		{Name: "flags", Path: "out/flags.txt"},
	}
	defects := Run(Request{Workspace: dir, Document: doc})
	if len(defects) != 0 {
		t.Fatalf("script Run() defects %v, want none", defects)
	}
	got, err := os.ReadFile(filepath.Join(dir, "out", "sample.txt"))
	if err != nil {
		t.Fatalf("script output: %v", err)
	}
	if string(got) != "reads" {
		t.Fatalf("script output got %q, want reads", got)
	}
	flags, err := os.ReadFile(filepath.Join(dir, "out", "flags.txt"))
	if err != nil {
		t.Fatalf("script flags: %v", err)
	}
	body := string(flags)
	if !strings.Contains(body, "e") || !strings.Contains(body, "u") {
		t.Fatalf("script $- got %q, want e and u from set -eu", body)
	}

	dir = t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	fail := sampleDoc("", "", "in/sample.txt", "out/sample.txt")
	fail.Tasks[0].Command = nil
	fail.Tasks[0].Script = "echo $UNSET_GOBBLE_VAR > out/sample.txt"
	defects = Run(Request{Workspace: dir, Document: fail})
	if !hasDefect(defects, DefectFailed, "copy") {
		t.Fatalf("script nounset Run() defects %v, want failed copy", defects)
	}
}

func TestRunProcessEnvDeclared(t *testing.T) {
	t.Setenv("SECRET", "should-not-leak")
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc("", "", "in/sample.txt", "out/sample.txt")
	doc.Tasks[0].Env = map[string]string{"HOME": "/tmp/gobble-home", "FOO": "bar"}
	doc.Tasks[0].Command = []string{"sh", "-c", "env > out/env.txt && cp in/sample.txt out/sample.txt"}
	doc.Tasks[0].Outputs = []IO{
		{Name: "out", Path: "out/sample.txt"},
		{Name: "env", Path: "out/env.txt"},
	}
	defects := Run(Request{Workspace: dir, Document: doc})
	if len(defects) != 0 {
		t.Fatalf("declared env Run() defects %v, want none", defects)
	}
	got, err := os.ReadFile(filepath.Join(dir, "out", "env.txt"))
	if err != nil {
		t.Fatalf("env output: %v", err)
	}
	body := string(got)
	if strings.Contains(body, "SECRET") {
		t.Fatalf("process inherited SECRET: %q", body)
	}
	if !strings.Contains(body, "HOME=/tmp/gobble-home") || !strings.Contains(body, "FOO=bar") {
		t.Fatalf("process env got %q, want declared HOME and FOO", body)
	}
	if !strings.Contains(body, "PATH=/usr/bin:/bin") {
		t.Fatalf("process env got %q, want default PATH", body)
	}

	dir = t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc = sampleDoc("", "", "in/sample.txt", "out/sample.txt")
	doc.Tasks[0].Env = map[string]string{"PATH": "/opt/bin"}
	doc.Tasks[0].Command = []string{"/bin/sh", "-c", "/usr/bin/env > out/env.txt && /bin/cp in/sample.txt out/sample.txt"}
	doc.Tasks[0].Outputs = []IO{
		{Name: "out", Path: "out/sample.txt"},
		{Name: "env", Path: "out/env.txt"},
	}
	defects = Run(Request{Workspace: dir, Document: doc})
	if len(defects) != 0 {
		t.Fatalf("author PATH Run() defects %v, want none", defects)
	}
	got, err = os.ReadFile(filepath.Join(dir, "out", "env.txt"))
	if err != nil {
		t.Fatalf("author PATH env output: %v", err)
	}
	body = string(got)
	if !strings.Contains(body, "PATH=/opt/bin") {
		t.Fatalf("author PATH env got %q, want PATH=/opt/bin", body)
	}
	if strings.Contains(body, "PATH=/usr/bin:/bin") {
		t.Fatalf("author PATH env kept default PATH: %q", body)
	}
}

func TestRunGroupStagePublishByName(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "ref.amb"), "amb")
	writeCheckFile(t, filepath.Join(dir, "ref.ann"), "ann")
	doc := Document{
		Name: "group",
		Tasks: []TaskPlan{{
			ID:      "copy",
			Name:    "copy",
			Command: []string{"sh", "-c", "cp ref.amb out.amb && cp ref.ann out.ann"},
			Inputs: []IO{{
				Name: "idx",
				Members: []IOMember{
					{Name: "amb", Path: "ref.amb"},
					{Name: "ann", Path: "ref.ann"},
				},
			}},
			Outputs: []IO{{
				Name: "idx",
				Members: []IOMember{
					{Name: "amb", Path: "out.amb"},
					{Name: "ann", Path: "out.ann"},
				},
			}},
		}},
		Edges: []Edge{{FromPort: "ref", ToTask: "copy", ToPort: "idx"}},
	}
	defects := Run(Request{Workspace: dir, Document: doc})
	if len(defects) != 0 {
		t.Fatalf("group Run() defects %v, want none", defects)
	}
	for _, name := range []string{"out.amb", "out.ann"} {
		if _, err := os.ReadFile(filepath.Join(dir, name)); err != nil {
			t.Fatalf("published group member %s: %v", name, err)
		}
	}
	isolate := filepath.Join(dir, ControlDir, "tasks", "copy", "_", "0", "1", "work")
	for _, name := range []string{"ref.amb", "ref.ann"} {
		info, err := os.Stat(filepath.Join(isolate, name))
		if err != nil {
			t.Fatalf("staged group member %s: %v", name, err)
		}
		if info.Mode().Perm() != 0o444 {
			t.Fatalf("staged %s mode %o, want 0444", name, info.Mode().Perm())
		}
	}

	dir = t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "ref.amb"), "amb")
	writeCheckFile(t, filepath.Join(dir, "ref.ann"), "ann")
	doc.Tasks[0].Command = []string{"sh", "-c", "cp ref.amb out.amb"}
	defects = Run(Request{Workspace: dir, Document: doc})
	if !hasDefect(defects, DefectFailed, "copy") {
		t.Fatalf("missing group member Run() defects %v, want failed copy", defects)
	}
	if _, err := os.Stat(filepath.Join(dir, "out.amb")); !os.IsNotExist(err) {
		t.Fatalf("partial group publish left out.amb")
	}
}

func TestRunWaitUsesPlanPathsOnly(t *testing.T) {
	orig := execTask
	t.Cleanup(func() { execTask = orig })
	var launched []string
	var mu sync.Mutex
	execTask = func(workspace string, task TaskPlan) report {
		mu.Lock()
		launched = append(launched, task.ID)
		mu.Unlock()
		if task.ID == "align" {
			writeCheckFile(t, filepath.Join(workspace, "aln.bam"), "bam")
		}
		return report{ID: task.ID, Published: true}
	}
	dir := t.TempDir()
	doc := Document{
		Name: "wait-only",
		Tasks: []TaskPlan{
			{
				ID:      "align",
				Name:    "align",
				Command: []string{"true"},
				Outputs: []IO{{Name: "bam", Path: "aln.bam"}},
			},
			{
				ID:      "index",
				Name:    "index",
				Command: []string{"true"},
				Inputs:  []IO{{Name: "in", Path: "missing-to-port.txt"}},
				Outputs: []IO{{Name: "bai", Path: "aln.bam.bai"}},
			},
		},
		Edges: []Edge{{
			FromTask: "align",
			FromPort: "bam",
			ToTask:   "index",
			ToPort:   "in",
			Wait:     []string{"aln.bam"},
		}},
	}
	defects := Run(Request{Workspace: dir, Document: doc})
	if len(defects) != 0 {
		t.Fatalf("wait-only Run() defects %v, want none", defects)
	}
	mu.Lock()
	got := append([]string(nil), launched...)
	mu.Unlock()
	if len(got) != 2 {
		t.Fatalf("wait-only launched %v, want align and index", got)
	}

	execTask = func(workspace string, task TaskPlan) report {
		mu.Lock()
		launched = append(launched, task.ID)
		mu.Unlock()
		return report{ID: task.ID, Published: true}
	}
	launched = nil
	dir = t.TempDir()
	doc.Edges[0].Wait = []string{"never-written.bam"}
	defects = Run(Request{Workspace: dir, Document: doc})
	if !hasDefect(defects, DefectFailed, "index") {
		t.Fatalf("missing wait Run() defects %v, want failed index", defects)
	}
	mu.Lock()
	got = append([]string(nil), launched...)
	mu.Unlock()
	for _, id := range got {
		if id == "index" {
			t.Fatalf("launched index without wait path: %v", got)
		}
	}
	if taskStates(t, dir)["index"].Status != StatusNotStarted {
		t.Fatalf("index status got %q, want not-started", taskStates(t, dir)["index"].Status)
	}
}

func TestRunProcessEnvIsFixed(t *testing.T) {
	t.Setenv("SECRET", "should-not-leak")
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc("", "", "in/sample.txt", "out/sample.txt")
	doc.Tasks[0].Command = []string{"sh", "-c", "env > out/env.txt && cp in/sample.txt out/sample.txt"}
	doc.Tasks[0].Outputs = []IO{
		{Name: "out", Path: "out/sample.txt"},
		{Name: "env", Path: "out/env.txt"},
	}
	defects := Run(Request{Workspace: dir, Document: doc})
	if len(defects) != 0 {
		t.Fatalf("env Run() defects %v, want none", defects)
	}
	got, err := os.ReadFile(filepath.Join(dir, "out", "env.txt"))
	if err != nil {
		t.Fatalf("env output: %v", err)
	}
	body := string(got)
	if strings.Contains(body, "SECRET") {
		t.Fatalf("process inherited SECRET: %q", body)
	}
	if !strings.Contains(body, "PATH=/usr/bin:/bin") {
		t.Fatalf("process env got %q, want PATH=/usr/bin:/bin", body)
	}
}

func TestCopyFileRefusesSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "secret")
	writeCheckFile(t, target, "secret")
	src := filepath.Join(dir, "link")
	if err := os.Symlink(target, src); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "out.txt")
	if err := copyFile(src, dst); err == nil {
		t.Fatal("copyFile followed symlink")
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatal("copied through symlink")
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
