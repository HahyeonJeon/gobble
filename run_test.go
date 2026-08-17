package gobble_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/internal/engine"
)

func TestPreflightNilGraph(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent")
	err := gobble.Preflight(nil, missing, 0)
	ge := requireRunError(t, "nil graph", err, gobble.DefectInvalidName, "")
	if ge.Op != "run" {
		t.Fatalf("nil graph: Error.Op got %q, want run", ge.Op)
	}
	if _, statErr := os.Stat(missing); !os.IsNotExist(statErr) {
		t.Fatalf("nil graph: missing workspace was created")
	}
}

func TestPreflightRefuse(t *testing.T) {
	tests := []struct {
		name string
		code gobble.DefectCode
		unit string
		prep func(t *testing.T) (g *gobble.Graph, workspace string, cap int)
	}{
		{
			name: "invalid graph",
			code: gobble.DefectConflict,
			unit: "copy.out",
			prep: func(t *testing.T) (*gobble.Graph, string, int) {
				dir := t.TempDir()
				return mustCompose(sameTaskIOPipeline)(t), dir, 0
			},
		},
		{
			name: "missing workspace",
			code: gobble.DefectInvalidPath,
			prep: func(t *testing.T) (*gobble.Graph, string, int) {
				return mustCompose(runCopyPipeline)(t), filepath.Join(t.TempDir(), "absent"), 0
			},
		},
		{
			name: "non-directory workspace",
			code: gobble.DefectInvalidPath,
			prep: func(t *testing.T) (*gobble.Graph, string, int) {
				dir := t.TempDir()
				file := filepath.Join(dir, "file")
				writeRunFile(t, file, "not-a-dir")
				return mustCompose(runCopyPipeline)(t), file, 0
			},
		},
		{
			name: "occupied workspace",
			code: gobble.DefectOccupiedWorkspace,
			prep: func(t *testing.T) (*gobble.Graph, string, int) {
				dir := readyRunWorkspace(t)
				writeRunFile(t, filepath.Join(dir, engine.ControlDir, engine.RunIdentityFile), `{"id":"run-1"}`)
				return mustCompose(runCopyPipeline)(t), dir, 0
			},
		},
		{
			name: "cap below 1",
			code: gobble.DefectInvalidName,
			prep: func(t *testing.T) (*gobble.Graph, string, int) {
				return mustCompose(runCopyPipeline)(t), readyRunWorkspace(t), -1
			},
		},
		{
			name: "absolute plan path",
			code: gobble.DefectInvalidPath,
			unit: "copy.out",
			prep: func(t *testing.T) (*gobble.Graph, string, int) {
				return mustCompose(absoluteOutputPipeline)(t), readyRunWorkspace(t), 0
			},
		},
		{
			name: ".gobble plan path",
			code: gobble.DefectInvalidPath,
			unit: "copy.out",
			prep: func(t *testing.T) (*gobble.Graph, string, int) {
				return mustCompose(gobbleOutputPipeline)(t), readyRunWorkspace(t), 0
			},
		},
		{
			name: "non-local backend",
			code: gobble.DefectUnsupportedBackend,
			unit: "copy",
			prep: func(t *testing.T) (*gobble.Graph, string, int) {
				return mustCompose(unsupportedBackendPipeline)(t), readyRunWorkspace(t), 0
			},
		},
		{
			name: "missing required input",
			code: gobble.DefectMissingInput,
			unit: "copy.in",
			prep: func(t *testing.T) (*gobble.Graph, string, int) {
				return mustCompose(runCopyPipeline)(t), t.TempDir(), 0
			},
		},
		{
			name: "pre-existing output",
			code: gobble.DefectOutputExists,
			unit: "copy.out",
			prep: func(t *testing.T) (*gobble.Graph, string, int) {
				dir := readyRunWorkspace(t)
				writeRunFile(t, filepath.Join(dir, "out", "sample.txt"), "leftover")
				return mustCompose(runCopyPipeline)(t), dir, 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, workspace, cap := tt.prep(t)
			before := snapshotWorkspace(t, workspace)
			err := gobble.Preflight(g, workspace, cap)
			requireRunError(t, tt.name, err, tt.code, tt.unit)
			after := snapshotWorkspace(t, workspace)
			if before != after {
				t.Fatalf("case %s: workspace changed\nbefore:\n%s\nafter:\n%s", tt.name, before, after)
			}
			info, statErr := os.Stat(workspace)
			if statErr == nil && info.IsDir() && tt.name != "occupied workspace" {
				if _, statErr := os.Stat(filepath.Join(workspace, engine.ControlDir)); !os.IsNotExist(statErr) {
					t.Fatalf("case %s: created %s", tt.name, engine.ControlDir)
				}
			}
		})
	}
}

func TestPreflightOccupiedNotOutputExists(t *testing.T) {
	dir := readyRunWorkspace(t)
	writeRunFile(t, filepath.Join(dir, "out", "sample.txt"), "leftover")
	writeRunFile(t, filepath.Join(dir, engine.ControlDir, engine.RunIdentityFile), `{"id":"run-1"}`)
	err := gobble.Preflight(mustCompose(runCopyPipeline)(t), dir, 0)
	requireRunError(t, "occupied+output", err, gobble.DefectOccupiedWorkspace, "")
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		t.Fatalf("occupied+output: error = %v, want *Error", err)
	}
	for _, d := range ge.Defects {
		if d.Code == gobble.DefectOutputExists {
			t.Fatalf("occupied+output: also reported output-exists, want occupied first")
		}
		if d.Code == gobble.DefectMissingInput {
			t.Fatalf("occupied+output: also reported missing-input, want occupied first")
		}
	}
}

func TestPreflightSameBasenameDifferentDirs(t *testing.T) {
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, "in", "a", "sample.txt"), "a")
	writeRunFile(t, filepath.Join(dir, "in", "b", "sample.txt"), "b")
	err := gobble.Preflight(mustCompose(sameBasenamePipeline)(t), dir, 0)
	if err != nil {
		t.Fatalf("same-basename: Preflight() error = %v, want nil", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, engine.ControlDir)); !os.IsNotExist(statErr) {
		t.Fatalf("same-basename: created %s", engine.ControlDir)
	}
}

func TestPreflightAcceptDefaultCap(t *testing.T) {
	dir := readyRunWorkspace(t)
	err := gobble.Preflight(mustCompose(runCopyPipeline)(t), dir, 0)
	if err != nil {
		t.Fatalf("default cap: Preflight() error = %v, want nil", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, engine.ControlDir)); !os.IsNotExist(statErr) {
		t.Fatalf("default cap: created %s", engine.ControlDir)
	}
}

func TestPlanDocumentExecutionView(t *testing.T) {
	g := mustCompose(runCopyPipeline)(t)
	doc, err := gobble.PlanDocument(g)
	if err != nil {
		t.Fatalf("PlanDocument() error = %v, want nil", err)
	}
	if len(doc.Tasks) != 1 {
		t.Fatalf("PlanDocument() tasks got %d, want 1", len(doc.Tasks))
	}
	task := doc.Tasks[0]
	if task.ID != "copy" {
		t.Fatalf("TaskPlan.ID got %q, want copy", task.ID)
	}
	if task.Image != "alpine:3.19.1" {
		t.Fatalf("TaskPlan.Image got %q, want alpine:3.19.1", task.Image)
	}
	if task.Backend != "local" {
		t.Fatalf("TaskPlan.Backend got %q, want local", task.Backend)
	}
	wantCmd := []string{"cp", "in/sample.txt", "out/sample.txt"}
	if len(task.Command) != len(wantCmd) {
		t.Fatalf("TaskPlan.Command got %#v, want %#v", task.Command, wantCmd)
	}
	for i, arg := range wantCmd {
		if task.Command[i] != arg {
			t.Fatalf("TaskPlan.Command got %#v, want %#v", task.Command, wantCmd)
		}
	}
	if task.Resources.CPU != 1 || task.Resources.Memory != "512m" {
		t.Fatalf("TaskPlan.Resources got cpu %v memory %q", task.Resources.CPU, task.Resources.Memory)
	}
	if len(task.Params) != 1 || task.Params[0].Name != "mode" || task.Params[0].Value != "fast" {
		t.Fatalf("TaskPlan.Params got %#v", task.Params)
	}
	for _, arg := range task.Command {
		if arg == "fast" {
			t.Fatalf("Params interpolated into Command: %#v", task.Command)
		}
	}
	if len(task.Inputs) != 1 || task.Inputs[0].Path != "in/sample.txt" {
		t.Fatalf("TaskPlan.Inputs got %#v", task.Inputs)
	}
	if len(task.Outputs) != 1 || task.Outputs[0].Path != "out/sample.txt" {
		t.Fatalf("TaskPlan.Outputs got %#v", task.Outputs)
	}
	if len(doc.Edges) != 1 || doc.Edges[0].FromTask != "" || doc.Edges[0].ToPort != "in" {
		t.Fatalf("Document.Edges got %#v", doc.Edges)
	}
}

func runCopyPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("copy")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Name: "sample", Ext: ".txt"})
	p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp", "in/sample.txt", "out/sample.txt"},
		Image:   "alpine:3.19.1",
		Backend: "local",
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Name: "sample", Ext: ".txt"},
		}},
		Params:    []gobble.Param{{Name: "mode", Value: "fast"}},
		Resources: gobble.Resources{CPU: 1, Memory: "512m"},
	})
	return p
}

func sameBasenamePipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("samebase")
	a := p.AddInput("a", gobble.PathSpec{Dir: gobble.Dir("in/a"), Name: "sample", Ext: ".txt"})
	b := p.AddInput("b", gobble.PathSpec{Dir: gobble.Dir("in/b"), Name: "sample", Ext: ".txt"})
	p.AddTask(gobble.TaskSpec{
		Name:    "join",
		Command: []string{"cat", "in/a/sample.txt", "in/b/sample.txt"},
		Inputs: []gobble.Bind{
			{Name: "a", From: a},
			{Name: "b", From: b},
		},
		Outputs: []gobble.Bind{
			{Name: "oa", Spec: gobble.PathSpec{Dir: gobble.Dir("out/a"), Name: "sample", Ext: ".txt"}},
			{Name: "ob", Spec: gobble.PathSpec{Dir: gobble.Dir("out/b"), Name: "sample", Ext: ".txt"}},
		},
	})
	return p
}

func absoluteOutputPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("absolute")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Name: "sample", Ext: ".txt"})
	p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.Literal("/tmp/out.txt")}},
	})
	return p
}

func gobbleOutputPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("reserved")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Name: "sample", Ext: ".txt"})
	p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Dir: gobble.Dir(".gobble"), Name: "out", Ext: ".txt"},
		}},
	})
	return p
}

func readyRunWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeRunFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	return dir
}

func writeRunFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func requireRunError(t *testing.T, name string, err error, code gobble.DefectCode, unit string) *gobble.Error {
	t.Helper()
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		t.Fatalf("case %s: error = %v, want *Error", name, err)
	}
	if ge.Op != "run" {
		t.Fatalf("case %s: Error.Op got %q, want run", name, ge.Op)
	}
	found := false
	codes := make([]gobble.DefectCode, len(ge.Defects))
	units := make([]string, len(ge.Defects))
	for i, d := range ge.Defects {
		codes[i] = d.Code
		units[i] = d.Unit
		if d.Code == code && (unit == "" || d.Unit == unit) {
			found = true
		}
	}
	if !found {
		t.Fatalf("case %s: defects codes got %v units %v, want code %s unit %q", name, codes, units, code, unit)
	}
	return ge
}

func snapshotWorkspace(t *testing.T, root string) string {
	t.Helper()
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return "ABSENT\n"
	}
	if err != nil {
		t.Fatalf("Lstat(%s) error = %v", root, err)
	}
	if !info.IsDir() {
		data, err := os.ReadFile(root)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", root, err)
		}
		return "FILE " + string(data) + "\n"
	}
	var out []byte
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if info.IsDir() {
			out = append(out, []byte(rel+"/\n")...)
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out = append(out, []byte(rel+" "+string(data)+"\n")...)
		return nil
	})
	if err != nil {
		t.Fatalf("Walk(%s) error = %v", root, err)
	}
	return string(out)
}
