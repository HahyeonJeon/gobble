package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDocumentCarriesExecutionFields(t *testing.T) {
	doc := sampleDoc("alpine:3.19.1", "local", "in/sample.txt", "out/sample.txt")
	task := doc.Tasks[0]
	if task.ID != "copy" || task.Image != "alpine:3.19.1" || task.Backend != "local" {
		t.Fatalf("TaskPlan identity got id %q image %q backend %q", task.ID, task.Image, task.Backend)
	}
	if len(task.Command) != 3 || task.Command[0] != "cp" {
		t.Fatalf("TaskPlan.Command got %#v, want cp argv", task.Command)
	}
	if task.Resources.CPU != 1 || task.Resources.Memory != "512m" {
		t.Fatalf("TaskPlan.Resources got cpu %v memory %q", task.Resources.CPU, task.Resources.Memory)
	}
	if len(task.Params) != 1 || task.Params[0] != (ParamPlan{Name: "mode", Value: "fast"}) {
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
	if len(doc.Edges) != 1 || doc.Edges[0].FromTask != "" || doc.Edges[0].ToTask != "copy" {
		t.Fatalf("Document.Edges got %#v", doc.Edges)
	}
}

func TestCheckAccept(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	req := Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	}
	if defects := Check(req); len(defects) != 0 {
		t.Fatalf("Check() defects %v, want none", defects)
	}
	if _, err := os.Stat(filepath.Join(dir, ControlDir)); !os.IsNotExist(err) {
		t.Fatalf("Check() created %s, want untouched workspace", ControlDir)
	}
}

func TestCheckSameBasenameDifferentDirs(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "a", "sample.txt"), "a")
	writeCheckFile(t, filepath.Join(dir, "in", "b", "sample.txt"), "b")
	req := Request{
		Workspace: dir,
		Document: Document{
			Name: "samebase",
			Tasks: []TaskPlan{{
				ID:      "join",
				Name:    "join",
				Command: []string{"cat", "in/a/sample.txt", "in/b/sample.txt"},
				Inputs: []IO{
					{Name: "a", Path: "in/a/sample.txt"},
					{Name: "b", Path: "in/b/sample.txt"},
				},
				Outputs: []IO{
					{Name: "oa", Path: "out/a/sample.txt"},
					{Name: "ob", Path: "out/b/sample.txt"},
				},
			}},
			Edges: []Edge{
				{FromPort: "a", ToTask: "join", ToPort: "a"},
				{FromPort: "b", ToTask: "join", ToPort: "b"},
			},
		},
	}
	if defects := Check(req); len(defects) != 0 {
		t.Fatalf("same-basename Check() defects %v, want none", defects)
	}
}

func TestCheckRefuse(t *testing.T) {
	type setupFunc func(t *testing.T) (Request, string)

	tests := []struct {
		name string
		code string
		unit string
		prep setupFunc
	}{
		{
			name: "missing workspace",
			code: DefectInvalidPath,
			prep: func(t *testing.T) (Request, string) {
				missing := filepath.Join(t.TempDir(), "absent")
				return Request{
					Workspace: missing,
					Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
				}, missing
			},
		},
		{
			name: "non-directory workspace",
			code: DefectInvalidPath,
			prep: func(t *testing.T) (Request, string) {
				dir := t.TempDir()
				file := filepath.Join(dir, "file")
				writeCheckFile(t, file, "not-a-dir")
				return Request{
					Workspace: file,
					Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
				}, file
			},
		},
		{
			name: "occupied workspace",
			code: DefectOccupiedWorkspace,
			prep: func(t *testing.T) (Request, string) {
				dir := t.TempDir()
				writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
				writeCheckFile(t, filepath.Join(dir, ControlDir, RunIdentityFile), `{
  "schema_version": 2,
  "id": "run-1",
  "status": "running",
  "occupancy": {"active": true, "host": "h", "pid": 1, "lease": "lease"}
}
`)
				return Request{
					Workspace: dir,
					Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
				}, dir
			},
		},
		{
			name: "schema 0 control document",
			code: DefectUnsupportedSchema,
			prep: func(t *testing.T) (Request, string) {
				dir := t.TempDir()
				writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
				writeCheckFile(t, filepath.Join(dir, ControlDir, RunIdentityFile), `{"id":"run-1"}`)
				return Request{
					Workspace: dir,
					Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
				}, dir
			},
		},
		{
			name: "schema 1 control document",
			code: DefectUnsupportedSchema,
			prep: func(t *testing.T) (Request, string) {
				dir := t.TempDir()
				writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
				writeCheckFile(t, filepath.Join(dir, ControlDir, RunIdentityFile), `{
  "schema_version": 1,
  "id": "run-1",
  "status": "succeeded",
  "occupancy": {"active": false}
}
`)
				return Request{
					Workspace: dir,
					Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
				}, dir
			},
		},
		{
			name: "cap below 1",
			code: DefectInvalidValue,
			prep: func(t *testing.T) (Request, string) {
				dir := t.TempDir()
				writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
				return Request{
					Workspace: dir,
					Cap:       -1,
					Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
				}, dir
			},
		},
		{
			name: "cap above 64",
			code: DefectInvalidValue,
			prep: func(t *testing.T) (Request, string) {
				dir := t.TempDir()
				writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
				return Request{
					Workspace: dir,
					Cap:       65,
					Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
				}, dir
			},
		},
		{
			name: "unreadable occupancy",
			code: DefectInvalidPath,
			prep: func(t *testing.T) (Request, string) {
				dir := t.TempDir()
				writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
				writeCheckFile(t, filepath.Join(dir, ControlDir), "not-a-dir")
				return Request{
					Workspace: dir,
					Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
				}, dir
			},
		},
		{
			name: "absolute plan path",
			code: DefectInvalidPath,
			unit: "copy.out",
			prep: func(t *testing.T) (Request, string) {
				dir := t.TempDir()
				writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
				return Request{
					Workspace: dir,
					Document:  sampleDoc("", "", "in/sample.txt", "/tmp/out.txt"),
				}, dir
			},
		},
		{
			name: "escaping plan path",
			code: DefectInvalidPath,
			unit: "copy.out",
			prep: func(t *testing.T) (Request, string) {
				dir := t.TempDir()
				writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
				return Request{
					Workspace: dir,
					Document:  sampleDoc("", "", "in/sample.txt", "../out.txt"),
				}, dir
			},
		},
		{
			name: ".gobble plan path",
			code: DefectInvalidPath,
			unit: "copy.out",
			prep: func(t *testing.T) (Request, string) {
				dir := t.TempDir()
				writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
				return Request{
					Workspace: dir,
					Document:  sampleDoc("", "", "in/sample.txt", ".gobble/out.txt"),
				}, dir
			},
		},
		{
			name: ".git plan path",
			code: DefectInvalidPath,
			unit: "copy.out",
			prep: func(t *testing.T) (Request, string) {
				dir := t.TempDir()
				writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
				return Request{
					Workspace: dir,
					Document:  sampleDoc("", "", "in/sample.txt", ".git/hooks/pre-commit"),
				}, dir
			},
		},
		{
			name: "image starts with dash",
			code: DefectInvalidValue,
			unit: "copy",
			prep: func(t *testing.T) (Request, string) {
				dir := t.TempDir()
				writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
				return Request{
					Workspace: dir,
					Document:  sampleDoc("--privileged", "", "in/sample.txt", "out/sample.txt"),
				}, dir
			},
		},
		{
			name: "image contains whitespace",
			code: DefectInvalidValue,
			unit: "copy",
			prep: func(t *testing.T) (Request, string) {
				dir := t.TempDir()
				writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
				return Request{
					Workspace: dir,
					Document:  sampleDoc("alpine:3.21 latest", "", "in/sample.txt", "out/sample.txt"),
				}, dir
			},
		},
		{
			name: "directory input",
			code: DefectMissingInput,
			unit: "copy.in",
			prep: func(t *testing.T) (Request, string) {
				dir := t.TempDir()
				if err := os.MkdirAll(filepath.Join(dir, "in", "sample.txt"), 0o755); err != nil {
					t.Fatal(err)
				}
				return Request{
					Workspace: dir,
					Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
				}, dir
			},
		},
		{
			name: "non-local backend",
			code: DefectUnsupportedBackend,
			unit: "copy",
			prep: func(t *testing.T) (Request, string) {
				dir := t.TempDir()
				writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
				return Request{
					Workspace: dir,
					Document:  sampleDoc("", "slurm", "in/sample.txt", "out/sample.txt"),
				}, dir
			},
		},
		{
			name: "missing required input",
			code: DefectMissingInput,
			unit: "copy.in",
			prep: func(t *testing.T) (Request, string) {
				dir := t.TempDir()
				return Request{
					Workspace: dir,
					Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
				}, dir
			},
		},
		{
			name: "pre-existing output",
			code: DefectOutputExists,
			unit: "copy.out",
			prep: func(t *testing.T) (Request, string) {
				dir := t.TempDir()
				writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
				writeCheckFile(t, filepath.Join(dir, "out", "sample.txt"), "leftover")
				return Request{
					Workspace: dir,
					Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
				}, dir
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, root := tt.prep(t)
			before := snapshotDir(t, root)
			defects := Check(req)
			if !hasDefect(defects, tt.code, tt.unit) {
				t.Fatalf("case %s: Check() defects %v, want code %s unit %q", tt.name, defects, tt.code, tt.unit)
			}
			after := snapshotDir(t, root)
			if before != after {
				t.Fatalf("case %s: workspace changed\nbefore:\n%s\nafter:\n%s", tt.name, before, after)
			}
		})
	}
}

func TestCheckDanglingOutputSymlink(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	if err := os.MkdirAll(filepath.Join(dir, "out"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "missing-target"), filepath.Join(dir, "out", "sample.txt")); err != nil {
		t.Fatal(err)
	}
	defects := Check(Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	})
	if !hasDefect(defects, DefectOutputExists, "copy.out") {
		t.Fatalf("dangling output symlink: Check() defects %v, want output-exists", defects)
	}
	if _, err := os.Stat(filepath.Join(dir, ControlDir)); !os.IsNotExist(err) {
		t.Fatalf("dangling output symlink: Check created %s", ControlDir)
	}
}

func TestCheckOccupiedNotOutputExists(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	writeCheckFile(t, filepath.Join(dir, "out", "sample.txt"), "leftover")
	writeCheckFile(t, filepath.Join(dir, ControlDir, RunIdentityFile), `{
  "schema_version": 2,
  "id": "run-1",
  "status": "running",
  "occupancy": {"active": true, "host": "h", "pid": 1, "lease": "lease"}
}
`)
	defects := Check(Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	})
	if !hasDefect(defects, DefectOccupiedWorkspace, "") {
		t.Fatalf("occupied+output: Check() defects %v, want occupied-workspace", defects)
	}
	if hasDefect(defects, DefectOutputExists, "copy.out") {
		t.Fatalf("occupied+output: Check() also reported output-exists, want occupied first")
	}
}

func TestCheckGroupEmptyPathOK(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "ref.amb"), "a")
	writeCheckFile(t, filepath.Join(dir, "ref.ann"), "n")
	req := Request{
		Workspace: dir,
		Document: Document{
			Name: "group",
			Tasks: []TaskPlan{{
				ID:      "idx",
				Name:    "idx",
				Command: []string{"true"},
				Inputs: []IO{{
					Name: "idx",
					Members: []IOMember{
						{Name: "amb", Path: "ref.amb"},
						{Name: "ann", Path: "ref.ann"},
					},
				}},
				Outputs: []IO{{
					Name: "out",
					Members: []IOMember{
						{Name: "amb", Path: "out.amb"},
						{Name: "ann", Path: "out.ann"},
					},
				}},
			}},
			Edges: []Edge{{FromPort: "ref", ToTask: "idx", ToPort: "idx"}},
		},
	}
	if defects := Check(req); len(defects) != 0 {
		t.Fatalf("group empty path Check() defects %v, want none", defects)
	}
}

func TestCheckTaskExceedsCapacity(t *testing.T) {
	orig := readHostCapacity
	t.Cleanup(func() { readHostCapacity = orig })
	readHostCapacity = func() hostCapacity {
		return hostCapacity{CPU: 1, CPUKnown: true, Memory: 1024, MemKnown: true}
	}
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc("", "", "in/sample.txt", "out/sample.txt")
	doc.Tasks[0].Resources = ResourcePlan{CPU: 2, Memory: ""}
	before := snapshotDir(t, dir)
	defects := Check(Request{Workspace: dir, Document: doc})
	if !hasDefect(defects, DefectInvalidValue, "copy") {
		t.Fatalf("cpu over capacity Check() defects %v, want invalid-value copy", defects)
	}
	after := snapshotDir(t, dir)
	if before != after {
		t.Fatalf("cpu over capacity changed workspace")
	}

	doc.Tasks[0].Resources = ResourcePlan{CPU: 0, Memory: "512m"}
	defects = Check(Request{Workspace: dir, Document: doc})
	if !hasDefect(defects, DefectInvalidValue, "copy") {
		t.Fatalf("memory over capacity Check() defects %v, want invalid-value copy", defects)
	}
	if _, err := os.Stat(filepath.Join(dir, ControlDir)); !os.IsNotExist(err) {
		t.Fatalf("over capacity Check occupied workspace")
	}
}

func sampleDoc(image, backend, inPath, outPath string) Document {
	return Document{
		Name: "copy",
		Tasks: []TaskPlan{{
			ID:      "copy",
			Name:    "copy",
			Command: []string{"cp", inPath, outPath},
			Image:   image,
			Backend: backend,
			Resources: ResourcePlan{
				CPU:    1,
				Memory: "512m",
			},
			Params:  []ParamPlan{{Name: "mode", Value: "fast"}},
			Inputs:  []IO{{Name: "in", Path: inPath}},
			Outputs: []IO{{Name: "out", Path: outPath}},
		}},
		Edges: []Edge{{FromPort: "reads", ToTask: "copy", ToPort: "in"}},
	}
}

func writeCheckFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func snapshotDir(t *testing.T, root string) string {
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

func TestCheckAncestorSymlinkInvalidPath(t *testing.T) {
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "sentinel")
	if err := os.WriteFile(sentinel, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "in"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "out")); err != nil {
		t.Fatal(err)
	}
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	defects := Check(Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	})
	if !hasDefect(defects, DefectInvalidPath, "copy.out") {
		t.Fatalf("ancestor symlink Check() defects %v, want invalid-path", defects)
	}
	got, err := os.ReadFile(sentinel)
	if err != nil || string(got) != "outside" {
		t.Fatalf("sentinel got %q, want outside", got)
	}
	if _, err := os.Stat(filepath.Join(dir, ControlDir)); !os.IsNotExist(err) {
		t.Fatalf("Check created %s", ControlDir)
	}
}

func TestCheckControlSymlinkInvalidPath(t *testing.T) {
	outside := t.TempDir()
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	if err := os.Symlink(outside, filepath.Join(dir, ControlDir)); err != nil {
		t.Fatal(err)
	}
	defects := Check(Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	})
	if !hasDefect(defects, DefectInvalidPath, "") && !hasDefect(defects, DefectInvalidPath, ControlDir) {
		found := false
		for _, d := range defects {
			if d.Code == DefectInvalidPath {
				found = true
			}
		}
		if !found {
			t.Fatalf("control symlink Check() defects %v, want invalid-path", defects)
		}
	}
}

func TestCheckControlChildSymlinkInvalidPath(t *testing.T) {
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "sentinel")
	if err := os.WriteFile(sentinel, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	if err := os.Mkdir(filepath.Join(dir, ControlDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sentinel, filepath.Join(dir, ControlDir, RunIdentityFile)); err != nil {
		t.Fatal(err)
	}
	defects := Check(Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	})
	found := false
	for _, d := range defects {
		if d.Code == DefectInvalidPath {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("control child symlink Check() defects %v, want invalid-path", defects)
	}
	got, err := os.ReadFile(sentinel)
	if err != nil || string(got) != "outside" {
		t.Fatalf("sentinel got %q, want outside", got)
	}
}

func TestCheckMissingDestContainedAncestorOK(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	if defects := Check(Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/nested/sample.txt"),
	}); len(defects) != 0 {
		t.Fatalf("missing nested dest Check() defects %v, want none", defects)
	}
}
