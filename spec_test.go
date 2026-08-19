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
	spec := PathSpec{Name: "report", Ext: ".txt"}
	got := restageSpec(spec, from)
	rendered, err := got.Render()
	if err != nil {
		t.Fatalf("case restage-literal: restageSpec().Render() error = %v, want nil", err)
	}
	if rendered != "work/report.txt" {
		t.Fatalf("case restage-literal: restageSpec().Render() got %q, want %q", rendered, "work/report.txt")
	}
	classified := classifySpec(spec, from, DeriveAppend)
	classRendered, err := classified.Render()
	if err != nil {
		t.Fatalf("case restage-literal: classifySpec().Render() error = %v, want nil", err)
	}
	if classRendered != "work/report.txt" {
		t.Fatalf("case restage-literal: classifySpec().Render() got %q, want %q", classRendered, "work/report.txt")
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
			Spec: PathSpec{Name: "report", Ext: ".txt"},
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
	in := p.AddInput("reads", PathSpec{Dir: Dir("in"), Name: "sample", Ext: ".txt"})
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
			Spec: PathSpec{Dir: Dir("out"), Name: "sample", Ext: ".txt"},
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

func TestRenderAgreesWithSnapshot(t *testing.T) {
	bam := PathSpec{Name: "aln", Ext: ".bam"}
	r1 := PathSpec{Lead: "samplename_S1_L001_R1_", Name: "001", Ext: ".fastq.gz"}
	chain := PathSpec{Name: "sample", Steps: []string{"sorted", "markdup"}, Ext: ".bam"}
	accept := []struct {
		name string
		spec PathSpec
		want string
	}{
		{name: "gzipped FASTQ", spec: PathSpec{Name: "sample", Ext: ".fastq.gz"}, want: "sample.fastq.gz"},
		{name: "Illumina R1", spec: r1, want: "samplename_S1_L001_R1_001.fastq.gz"},
		{name: "Illumina R2", spec: r1.WithLead("samplename_S1_L001_R2_"), want: "samplename_S1_L001_R2_001.fastq.gz"},
		{name: "htslib BAI", spec: bam.Append(".bai"), want: "aln.bam.bai"},
		{name: "Picard BAI", spec: bam.ReplaceExtension(".bai"), want: "aln.bai"},
		{name: "processing chain", spec: chain, want: "sample.sorted.markdup.bam"},
		{name: "literal", spec: Literal("sample.Aligned.sortedByCoord.out.bam"), want: "sample.Aligned.sortedByCoord.out.bam"},
		{name: "directory", spec: PathSpec{Dir: Dir("work/align"), Name: "sample", Steps: []string{"sorted"}, Ext: ".bam"}, want: "work/align/sample.sorted.bam"},
		{name: "empty Steps and Ext", spec: PathSpec{Name: "LICENSE"}, want: "LICENSE"},
		{name: ".fastq stays on name", spec: PathSpec{Name: "sample", Ext: ".fastq"}, want: "sample.fastq"},
		{name: "internal dotdot", spec: PathSpec{Dir: Dir("work/align/../lane"), Name: "x", Ext: ".txt"}, want: "work/lane/x.txt"},
	}
	for _, tt := range accept {
		t.Run("accept/"+tt.name, func(t *testing.T) {
			got, err := tt.spec.Render()
			eng, ed := snapshotPath(tt.spec).Render()
			if err != nil {
				t.Fatalf("case %s: PathSpec.Render() error = %v, want nil", tt.name, err)
			}
			if ed != nil {
				t.Fatalf("case %s: snapshotPath.Render() defect = %v, want nil", tt.name, ed)
			}
			if got != tt.want || eng != tt.want || got != eng {
				t.Fatalf("case %s: PathSpec.Render()=%q snapshotPath.Render()=%q want %q", tt.name, got, eng, tt.want)
			}
		})
	}

	lit := Literal("aln.bam")
	invalid := []struct {
		name string
		spec PathSpec
	}{
		{name: "empty lead and name", spec: PathSpec{Ext: ".bam"}},
		{name: "zero PathSpec", spec: PathSpec{}},
		{name: "step is dot", spec: PathSpec{Name: "x", Steps: []string{"."}}},
		{name: "empty step token", spec: PathSpec{Name: "sample", Steps: []string{""}}},
		{name: "AppendStep empty", spec: PathSpec{Name: "sample"}.AppendStep("")},
		{name: "AppendStep dot", spec: PathSpec{Name: "sample", Ext: ".bam"}.AppendStep(".")},
		{name: "sample. hole", spec: PathSpec{Name: "sample"}.AppendStep("")},
		{name: ".fastq hole empty step", spec: PathSpec{Name: "sample", Steps: []string{""}, Ext: ".fastq"}},
		{name: "ext is dot", spec: PathSpec{Name: "x", Ext: "."}},
		{name: "ext is dotdot", spec: PathSpec{Name: "x", Ext: ".."}},
		{name: "dir escape parent", spec: PathSpec{Dir: Dir("work/../out"), Name: "x"}},
		{name: "literal AppendStep", spec: lit.AppendStep("sorted")},
		{name: "name slash", spec: PathSpec{Name: "a/b"}},
		{name: "name is dot", spec: PathSpec{Name: "."}},
		{name: "name is dotdot", spec: PathSpec{Name: ".."}},
		{name: "Append empty extra", spec: PathSpec{Name: "sample", Ext: ".bam"}.Append("")},
		{name: "Append dot extra", spec: PathSpec{Name: "sample", Ext: ".bam"}.Append(".")},
		{name: "ext trailing dot", spec: PathSpec{Name: "sample", Ext: ".bam."}},
		{name: "literal Append empty extra", spec: Literal("aln.bam").Append("")},
	}
	for _, tt := range invalid {
		t.Run("invalid/"+tt.name, func(t *testing.T) {
			got, err := tt.spec.Render()
			eng, ed := snapshotPath(tt.spec).Render()
			var ge *Error
			if !errors.As(err, &ge) {
				t.Fatalf("case %s: PathSpec.Render() got path %q error = %v, want *Error", tt.name, got, err)
			}
			if ed == nil {
				t.Fatalf("case %s: snapshotPath.Render() path %q defect = nil, want invalid-path", tt.name, eng)
			}
			if ge.Op != "render" {
				t.Fatalf("case %s: PathSpec.Render() Op got %q, want render", tt.name, ge.Op)
			}
			if ed.Code != string(DefectInvalidPath) {
				t.Fatalf("case %s: snapshotPath.Render() code got %q, want %s", tt.name, ed.Code, DefectInvalidPath)
			}
			found := false
			for _, d := range ge.Defects {
				if d.Code == DefectInvalidPath {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("case %s: PathSpec.Render() codes got %v, want %s", tt.name, testDefectCodes(ge), DefectInvalidPath)
			}
			if got != "" || eng != "" {
				t.Fatalf("case %s: Render paths got PathSpec %q snapshot %q, want empty", tt.name, got, eng)
			}
		})
	}
}

func TestClassifyAgreesWithSnapshot(t *testing.T) {
	bam := PathSpec{Name: "aln", Ext: ".bam"}
	from := PathSpec{Dir: Dir("work"), Name: "sample", Steps: []string{"sorted"}, Ext: ".bam"}
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
		{name: "steps only restage", spec: PathSpec{Steps: []string{"markdup"}}, from: from, rule: DeriveAppend},
		{name: "literal parent restage", spec: PathSpec{Name: "report", Ext: ".txt"}, from: fromLit, rule: DeriveAppend},
		{name: "literal spec restage", spec: Literal("out.bam"), from: bam, rule: DeriveAppend},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifySpec(tt.spec, tt.from, tt.rule)
			eng := engine.Classify(snapshotPath(tt.spec), snapshotPath(tt.from), engine.DeriveRule(tt.rule))
			snap := snapshotPath(got)
			if snap.Dir != eng.Dir || snap.Lead != eng.Lead || snap.Name != eng.Name || snap.Ext != eng.Ext ||
				snap.Literal != eng.Literal || snap.Opaque != eng.Opaque || snap.BadLit != eng.BadLit {
				t.Fatalf("case %s: classifySpec snapshot %+v, Classify %+v", tt.name, snap, eng)
			}
			if len(snap.Steps) != len(eng.Steps) {
				t.Fatalf("case %s: steps got %v, want %v", tt.name, snap.Steps, eng.Steps)
			}
			for i := range snap.Steps {
				if snap.Steps[i] != eng.Steps[i] {
					t.Fatalf("case %s: steps got %v, want %v", tt.name, snap.Steps, eng.Steps)
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
						spec:    PathSpec{Name: "ref", Ext: ".amb"},
						members: []graphMember{{name: "amb", spec: PathSpec{Name: "ref", Ext: ".amb"}}},
					}},
				}},
			},
			code:    DefectInvalidName,
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
					outputs: []graphBind{{name: "out", spec: PathSpec{Name: "out", Ext: ".txt"}}},
				}},
			},
			code:    DefectInvalidName,
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
					outputs: []graphBind{{name: "out", spec: PathSpec{Name: "out", Ext: ".txt"}}},
				}},
			},
			code:    DefectInvalidName,
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
			outputs: []graphBind{{name: "out", spec: PathSpec{Name: "out", Ext: ".txt"}}},
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
				spec: PathSpec{Name: "sample"}.AppendStep(""),
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
