package gobble

import (
	"errors"
	"testing"
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
