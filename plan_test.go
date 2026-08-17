package gobble_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestBuildPlanReject(t *testing.T) {
	tests := []struct {
		name     string
		graph    func(t *testing.T) *gobble.Graph
		code     gobble.DefectCode
		unit     string
		nilGraph bool
	}{
		{
			name:     "nil graph",
			nilGraph: true,
			code:     gobble.DefectInvalidName,
		},
		{
			name:  "derived-path conflict",
			graph: mustCompose(derivedRelatedCollisionPipeline),
			code:  gobble.DefectConflict,
			unit:  "collide.out",
		},
		{
			name:  "derived index path equals another output",
			graph: mustCompose(derivedIndexCollisionPipeline),
			code:  gobble.DefectConflict,
			unit:  "collide.out",
		},
		{
			name:  "same-task input/output conflict",
			graph: mustCompose(sameTaskIOPipeline),
			code:  gobble.DefectConflict,
			unit:  "copy.out",
		},
		{
			name:  "unsupported-backend",
			graph: mustCompose(unsupportedBackendPipeline),
			code:  gobble.DefectUnsupportedBackend,
			unit:  "copy",
		},
		{
			name:  "NaN CPU",
			graph: mustCompose(nanCPUPipeline),
			code:  gobble.DefectInvalidName,
			unit:  "copy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var g *gobble.Graph
			if !tt.nilGraph {
				g = tt.graph(t)
			}
			var buf bytes.Buffer
			plan, err := gobble.BuildPlan(g, gobble.WriteTo(&buf))
			if plan != nil {
				t.Fatalf("case %s: BuildPlan() plan != nil, want nil", tt.name)
			}
			if buf.Len() != 0 {
				t.Fatalf("case %s: WriteTo wrote %d bytes, want 0", tt.name, buf.Len())
			}
			var ge *gobble.Error
			if !errors.As(err, &ge) {
				t.Fatalf("case %s: BuildPlan() error = %v, want *Error", tt.name, err)
			}
			if ge.Op != "plan" {
				t.Fatalf("case %s: Error.Op got %q, want %q", tt.name, ge.Op, "plan")
			}
			found := false
			codes := make([]gobble.DefectCode, len(ge.Defects))
			units := make([]string, len(ge.Defects))
			for i, d := range ge.Defects {
				codes[i] = d.Code
				units[i] = d.Unit
				if d.Code == tt.code && (tt.unit == "" || d.Unit == tt.unit) {
					found = true
				}
			}
			if !found {
				t.Fatalf("case %s: BuildPlan() defects codes got %v units %v, want code %s unit %q", tt.name, codes, units, tt.code, tt.unit)
			}
		})
	}
}

func TestBuildPlanWriteTo(t *testing.T) {
	g, err := gobble.Compose(oneTask("local-backend", gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{fileOut("out")},
	}))
	if err != nil {
		t.Fatalf("case write-to: Compose() error = %v, want nil", err)
	}

	var mem bytes.Buffer
	plan, err := gobble.BuildPlan(g, gobble.WriteTo(&mem))
	if err != nil {
		t.Fatalf("case write-to: BuildPlan() error = %v, want nil", err)
	}
	if plan == nil {
		t.Fatalf("case write-to: BuildPlan() plan = nil, want non-nil")
	}

	var direct bytes.Buffer
	if err := plan.WriteJSON(&direct); err != nil {
		t.Fatalf("case write-to: WriteJSON() error = %v, want nil", err)
	}
	if mem.String() != direct.String() {
		t.Fatalf("case write-to: WriteTo JSON != WriteJSON JSON")
	}

	var again bytes.Buffer
	if err := plan.WriteJSON(&again); err != nil {
		t.Fatalf("case write-to: second WriteJSON() error = %v, want nil", err)
	}
	if again.String() != direct.String() {
		t.Fatalf("case write-to: repeated WriteJSON() mismatch")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "plan.json")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("case write-to: Create() error = %v", err)
	}
	_, err = gobble.BuildPlan(g, gobble.WriteTo(f))
	if cerr := f.Close(); cerr != nil && err == nil {
		err = cerr
	}
	if err != nil {
		t.Fatalf("case write-to: BuildPlan(WriteTo file) error = %v, want nil", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("case write-to: ReadFile() error = %v", err)
	}
	if string(got) != direct.String() {
		t.Fatalf("case write-to: TempDir JSON != WriteJSON JSON")
	}

	encoded, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("case write-to: json.Marshal() error = %v, want nil", err)
	}
	var fromWrite, fromMarshal any
	if err := json.Unmarshal(direct.Bytes(), &fromWrite); err != nil {
		t.Fatalf("case write-to: Unmarshal(WriteJSON) error = %v", err)
	}
	if err := json.Unmarshal(encoded, &fromMarshal); err != nil {
		t.Fatalf("case write-to: Unmarshal(json.Marshal) error = %v", err)
	}
	if !jsonEqual(fromWrite, fromMarshal) {
		t.Fatalf("case write-to: json.Marshal() document != WriteJSON document")
	}
}

func jsonEqual(a, b any) bool {
	got, err := json.Marshal(a)
	if err != nil {
		return false
	}
	want, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return bytes.Equal(got, want)
}

func TestBuildPlanLocalBackendAndProcess(t *testing.T) {
	g, err := gobble.Compose(oneTask("process", gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{fileOut("out")},
	}))
	if err != nil {
		t.Fatalf("case process: Compose() error = %v, want nil", err)
	}
	var buf bytes.Buffer
	plan, err := gobble.BuildPlan(g, gobble.WriteTo(&buf))
	if err != nil {
		t.Fatalf("case process: BuildPlan() error = %v, want nil", err)
	}
	if plan == nil {
		t.Fatalf("case process: BuildPlan() plan = nil, want non-nil")
	}
	var decoded struct {
		Tasks []struct {
			Image     string `json:"image"`
			Backend   string `json:"backend"`
			Params    []any  `json:"params"`
			Resources struct {
				CPU    float64 `json:"cpu"`
				Memory string  `json:"memory"`
			} `json:"resources"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("case process: Unmarshal() error = %v", err)
	}
	if len(decoded.Tasks) != 1 {
		t.Fatalf("case process: tasks got %d, want 1", len(decoded.Tasks))
	}
	task := decoded.Tasks[0]
	if task.Image != "" {
		t.Fatalf("case process: image got %q, want empty", task.Image)
	}
	if task.Backend != "local" {
		t.Fatalf("case process: backend got %q, want %q", task.Backend, "local")
	}
	if task.Params == nil {
		t.Fatalf("case process: params = null, want []")
	}
	if len(task.Params) != 0 {
		t.Fatalf("case process: params got %#v, want empty", task.Params)
	}
	if task.Resources.CPU != 0 || task.Resources.Memory != "" {
		t.Fatalf("case process: resources got cpu %v memory %q, want 0 and empty", task.Resources.CPU, task.Resources.Memory)
	}
}

func TestBuildPlanWriteToKeepsPlan(t *testing.T) {
	g, err := gobble.Compose(oneTask("write-fail", gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{fileOut("out")},
	}))
	if err != nil {
		t.Fatalf("case write-fail: Compose() error = %v, want nil", err)
	}
	plan, err := gobble.BuildPlan(g, gobble.WriteTo(failWriter{}))
	if plan == nil {
		t.Fatalf("case write-fail: BuildPlan() plan = nil, want built plan")
	}
	if err == nil {
		t.Fatalf("case write-fail: BuildPlan() error = nil, want write error")
	}
	var ge *gobble.Error
	if errors.As(err, &ge) {
		t.Fatalf("case write-fail: BuildPlan() error = %v, want raw write error not *Error", err)
	}
	var direct bytes.Buffer
	if werr := plan.WriteJSON(&direct); werr != nil {
		t.Fatalf("case write-fail: kept plan WriteJSON() error = %v, want nil", werr)
	}
	if direct.Len() == 0 {
		t.Fatalf("case write-fail: kept plan WriteJSON() wrote 0 bytes")
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func mustCompose(pipe func() *gobble.Pipeline) func(t *testing.T) *gobble.Graph {
	return func(t *testing.T) *gobble.Graph {
		t.Helper()
		g, err := gobble.Compose(pipe())
		if err != nil {
			t.Fatalf("Compose() error = %v, want nil", err)
		}
		if g == nil {
			t.Fatalf("Compose() graph = nil, want compose-valid graph")
		}
		return g
	}
}
