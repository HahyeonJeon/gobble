package gobble

import (
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/internal/engine"
)

func testDefectCodes(e *Error) []DefectCode {
	if e == nil {
		return nil
	}
	codes := make([]DefectCode, len(e.Defects))
	for i, d := range e.Defects {
		codes[i] = d.Code
	}
	return codes
}

func TestRestageSpecClearsLiteralOpacity(t *testing.T) {
	from := Literal("sample.html").WithDir(Dir("work"))
	spec := PathSpec{Base: "report", Ext: ".txt"}
	got := classifySpec(spec, from, DeriveAppend)
	rendered, err := got.Render()
	if err != nil {
		t.Fatalf("case restage-literal: classifySpec().Render() error = %v, want nil", err)
	}
	if rendered != "work/report.txt" {
		t.Fatalf("case restage-literal: classifySpec().Render() got %q, want %q", rendered, "work/report.txt")
	}
}

func TestComposeRestageFromLiteralRendersNewFields(t *testing.T) {
	p := NewPipeline("restage")
	src := p.AddTask(TaskSpec{
		Name:    "src",
		Command: []string{"echo"},
		Outputs: []Bind{{
			Name: "html",
			Spec: Literal("sample.html").WithDir(Dir("work")),
		}},
	})
	p.AddTask(TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Inputs:  []Bind{{Name: "in", From: src.Out("html")}},
		Outputs: []Bind{{
			Name: "out",
			From: src.Out("html"),
			Spec: PathSpec{Base: "report", Ext: ".txt"},
		}},
	})
	g, err := Compose(p)
	if err != nil {
		t.Fatalf("case compose-restage: Compose() error = %v, want nil", err)
	}
	var spec PathSpec
	found := false
	for _, task := range g.tasks {
		if task.id != "copy" {
			continue
		}
		for _, b := range task.outputs {
			if b.name == "out" {
				spec = b.spec
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("case compose-restage: Graph missing copy.out")
	}
	got, err := spec.Render()
	if err != nil {
		t.Fatalf("case compose-restage: Graph copy.out.Render() error = %v, want nil", err)
	}
	if got != "work/report.txt" {
		t.Fatalf("case compose-restage: Graph copy.out.Render() got %q, want %q (opaque parent would be work/sample.html)", got, "work/report.txt")
	}
	if err := Validate(g); err != nil {
		t.Fatalf("case compose-restage: Validate() error = %v, want nil", err)
	}
}

func TestComposeValidatePlanRestagedPipelineInput(t *testing.T) {
	p := NewPipeline("input-restage")
	in := p.AddInput("reads", PathSpec{Dir: Dir("in"), Base: "sample", Ext: ".txt"})
	p.AddTask(TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Inputs: []Bind{{
			Name: "src",
			From: in,
			Spec: PathSpec{Dir: Dir("work")},
		}},
		Outputs: []Bind{{
			Name: "dst",
			Spec: PathSpec{Dir: Dir("out"), Base: "sample", Ext: ".txt"},
		}},
	})
	g, err := Compose(p)
	if err != nil {
		t.Fatalf("case input-restage: Compose() error = %v, want nil", err)
	}
	if err := Validate(g); err != nil {
		t.Fatalf("case input-restage: Validate() error = %v, want nil", err)
	}
	plan, err := BuildPlan(g)
	if err != nil {
		t.Fatalf("case input-restage: BuildPlan() error = %v, want nil", err)
	}
	if plan == nil {
		t.Fatalf("case input-restage: BuildPlan() plan = nil, want non-nil")
	}
	doc, err := planDocument(g)
	if err != nil {
		t.Fatalf("case input-restage: planDocument() error = %v, want nil", err)
	}
	if len(doc.Tasks) != 1 || len(doc.Tasks[0].Inputs) != 1 {
		t.Fatalf("case input-restage: document tasks/inputs got %#v", doc.Tasks)
	}
	got := doc.Tasks[0].Inputs[0]
	if got.Path != "work/sample.txt" {
		t.Fatalf("case input-restage: dest path got %q, want %q", got.Path, "work/sample.txt")
	}
	if got.Source != "in/sample.txt" {
		t.Fatalf("case input-restage: source path got %q, want %q", got.Source, "in/sample.txt")
	}
	if len(doc.Edges) != 1 {
		t.Fatalf("case input-restage: edges got %d, want 1", len(doc.Edges))
	}
	wait := doc.Edges[0].Wait
	if len(wait) != 1 || wait[0] != "in/sample.txt" {
		t.Fatalf("case input-restage: wait got %#v, want [in/sample.txt]", wait)
	}
}

func TestRelatedFilePredicate(t *testing.T) {
	if !isRelatedFile(PathSpec{Ext: ".bai"}) {
		t.Fatalf("isRelatedFile(Ext only) got false, want true")
	}
	if !isRelatedFile(PathSpec{Dir: Dir("out"), Ext: ".bai"}) {
		t.Fatalf("isRelatedFile(Dir+Ext) got false, want true")
	}
	if isRelatedFile(PathSpec{Base: "aln", Ext: ".bam"}) {
		t.Fatalf("isRelatedFile(Base+Ext) got true, want false")
	}
	if !isZeroSpec(PathSpec{}) {
		t.Fatalf("isZeroSpec(zero) got false, want true")
	}
	if isZeroSpec(PathSpec{Ext: ".bai"}) {
		t.Fatalf("isZeroSpec(related) got true, want false")
	}
}

func TestClassifyAgreesWithInternalPath(t *testing.T) {
	bam := PathSpec{Base: "aln", Ext: ".bam"}
	from := PathSpec{Dir: Dir("work"), Base: "sample", Suffixes: []string{"sorted"}, Ext: ".bam"}
	fromLit := Literal("aln.bam").WithDir(Dir("work"))
	tests := []struct {
		name string
		spec PathSpec
		from PathSpec
		rule DeriveRule
	}{
		{name: "zero inherit", spec: PathSpec{}, from: bam, rule: DeriveAppend},
		{name: "related append", spec: PathSpec{Ext: ".bai"}, from: bam, rule: DeriveAppend},
		{name: "related replace", spec: PathSpec{Ext: ".bai"}, from: bam, rule: DeriveReplaceExt},
		{name: "related with dir", spec: PathSpec{Dir: Dir("out"), Ext: ".bai"}, from: bam, rule: DeriveAppend},
		{name: "dir only restage", spec: PathSpec{Dir: Dir("out")}, from: from, rule: DeriveAppend},
		{name: "steps only restage", spec: PathSpec{Suffixes: []string{"markdup"}}, from: from, rule: DeriveAppend},
		{name: "literal parent restage", spec: PathSpec{Base: "report", Ext: ".txt"}, from: fromLit, rule: DeriveAppend},
		{name: "literal spec restage", spec: Literal("out.bam"), from: bam, rule: DeriveAppend},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifySpec(tt.spec, tt.from, tt.rule)
			eng := engine.Classify(snapshotPath(tt.spec), snapshotPath(tt.from), engine.DeriveRule(tt.rule))
			rec := snapshotPath(got)
			if rec.Dir != eng.Dir || rec.Prefix != eng.Prefix || rec.Base != eng.Base || rec.Ext != eng.Ext ||
				rec.Literal != eng.Literal || rec.Opaque != eng.Opaque || rec.BadLit != eng.BadLit {
				t.Fatalf("case %s: classifySpec %+v, Classify %+v", tt.name, rec, eng)
			}
			if len(rec.Suffixes) != len(eng.Suffixes) {
				t.Fatalf("case %s: suffixes got %v, want %v", tt.name, rec.Suffixes, eng.Suffixes)
			}
			for i := range rec.Suffixes {
				if rec.Suffixes[i] != eng.Suffixes[i] {
					t.Fatalf("case %s: suffixes got %v, want %v", tt.name, rec.Suffixes, eng.Suffixes)
				}
			}
		})
	}
}

func TestValidateRejectDeclarations(t *testing.T) {
	tests := []struct {
		name    string
		graph   *Graph
		code    DefectCode
		unit    string
		message string
	}{
		{
			name: "group and spec both set",
			graph: &Graph{
				name: "xor",
				tasks: []graphTask{{
					id:      "index",
					name:    "index",
					command: []string{"bwa"},
					outputs: []graphBind{{
						name:    "idx",
						spec:    PathSpec{Base: "ref", Ext: ".amb"},
						members: []graphMember{{name: "amb", spec: PathSpec{Base: "ref", Ext: ".amb"}}},
					}},
				}},
			},
			code:    DefectInvalidValue,
			unit:    "index.idx",
			message: "group and spec both set",
		},
		{
			name: "command and script both set",
			graph: &Graph{
				name: "cmd-script",
				tasks: []graphTask{{
					id:      "copy",
					name:    "copy",
					command: []string{"cp"},
					script:  "echo hi",
					outputs: []graphBind{{name: "out", spec: PathSpec{Base: "out", Ext: ".txt"}}},
				}},
			},
			code:    DefectInvalidValue,
			unit:    "copy",
			message: "command and script both set",
		},
		{
			name: "empty env key",
			graph: &Graph{
				name: "env",
				tasks: []graphTask{{
					id:      "copy",
					name:    "copy",
					command: []string{"cp"},
					env:     map[string]string{"": "x"},
					outputs: []graphBind{{name: "out", spec: PathSpec{Base: "out", Ext: ".txt"}}},
				}},
			},
			code:    DefectInvalidValue,
			unit:    "copy",
			message: "empty env key",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.graph)
			var ge *Error
			if !errors.As(err, &ge) {
				t.Fatalf("case %s: Validate() error = %v, want *Error", tt.name, err)
			}
			if ge.Op != "validate" {
				t.Fatalf("case %s: Error.Op got %q, want %q", tt.name, ge.Op, "validate")
			}
			found := false
			for _, d := range ge.Defects {
				if d.Code == tt.code && d.Unit == tt.unit && (tt.message == "" || d.Message == tt.message) {
					found = true
				}
			}
			if !found {
				t.Fatalf("case %s: Validate() defects got %v, want code %s unit %q message %q",
					tt.name, testDefectCodes(ge), tt.code, tt.unit, tt.message)
			}
		})
	}
}

func TestBuildPlanNeverReady(t *testing.T) {
	g := &Graph{
		name: "never-ready",
		tasks: []graphTask{{
			id:      "copy",
			name:    "copy",
			command: []string{"cp"},
			outputs: []graphBind{{name: "out", spec: PathSpec{Base: "out", Ext: ".txt"}}},
		}},
		edges: []graphEdge{{
			fromTask: "ghost",
			fromPort: "out",
			toTask:   "copy",
			toPort:   "out",
		}},
	}
	plan, err := BuildPlan(g)
	if plan != nil {
		t.Fatalf("case never-ready: BuildPlan() plan != nil, want nil")
	}
	var ge *Error
	if !errors.As(err, &ge) {
		t.Fatalf("case never-ready: BuildPlan() error = %v, want *Error", err)
	}
	if ge.Op != "plan" {
		t.Fatalf("case never-ready: Error.Op got %q, want %q", ge.Op, "plan")
	}
	found := false
	for _, d := range ge.Defects {
		if d.Code == DefectNeverReady && d.Unit == "copy.out" {
			found = true
		}
	}
	if !found {
		t.Fatalf("case never-ready: defects got %v, want code %s unit %q", testDefectCodes(ge), DefectNeverReady, "copy.out")
	}
}

func TestEngineImportBan(t *testing.T) {
	cmd := exec.Command("go", "list", "-f", "{{range .Imports}}{{println .}}{{end}}{{range .TestImports}}{{println .}}{{end}}", "./internal/engine")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list ./internal/engine: %v", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if line == "github.com/HahyeonJeon/gobble" {
			t.Fatalf("internal/engine imports gobble")
		}
	}
}

func TestPlanDocumentRenderFailure(t *testing.T) {
	g := &Graph{
		name: "bad-render",
		tasks: []graphTask{{
			id:      "copy",
			name:    "copy",
			command: []string{"cp"},
			outputs: []graphBind{{
				name: "out",
				spec: PathSpec{Base: "sample"}.AppendSuffix(""),
			}},
		}},
	}
	_, err := planDocument(g)
	var ge *Error
	if !errors.As(err, &ge) {
		t.Fatalf("case plan-render: planDocument() error = %v, want *Error", err)
	}
	if ge.Op != "plan" {
		t.Fatalf("case plan-render: Error.Op got %q, want %q", ge.Op, "plan")
	}
	found := false
	for _, d := range ge.Defects {
		if d.Code == DefectInvalidPath && d.Unit == "copy.out" {
			found = true
		}
	}
	if !found {
		t.Fatalf("case plan-render: defects got %v, want code %s unit %q", testDefectCodes(ge), DefectInvalidPath, "copy.out")
	}
}
