package gobble_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestComposeWorkflowCase(t *testing.T) {
	p := workflowCasePipeline()
	var zero gobble.Handle
	if !zero.IsZero() {
		t.Fatalf("case workflow-case: zero Handle.IsZero() got false, want true")
	}

	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("case workflow-case: Compose() error = %v, want nil", err)
	}
	if g == nil {
		t.Fatalf("case workflow-case: Compose() graph = nil, want non-nil")
	}
}

func TestComposeHandles(t *testing.T) {
	p := gobble.NewPipeline("handles")
	in := p.AddInput("reads", gobble.PathSpec{Name: "sample", Ext: ".fastq.gz"})
	if in.IsZero() {
		t.Fatalf("case handles: AddInput().IsZero() got true, want false")
	}
	if in.Name() != "reads" {
		t.Fatalf("case handles: AddInput().Name() got %q, want %q", in.Name(), "reads")
	}
	if !in.Spec().Equal(gobble.PathSpec{Name: "sample", Ext: ".fastq.gz"}) {
		t.Fatalf("case handles: AddInput().Spec() got %+v, want Name=sample Ext=.fastq.gz", in.Spec())
	}

	task := p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Inputs:  []gobble.Bind{{Name: "src", From: in}},
		Outputs: []gobble.Bind{{Name: "dst", Spec: gobble.PathSpec{Name: "out", Ext: ".txt"}}},
	})
	out := task.Out("dst")
	inPort := task.In("src")
	if out.IsZero() || inPort.IsZero() {
		t.Fatalf("case handles: Out/In IsZero() got true, want false")
	}
	if out.Name() != "dst" || inPort.Name() != "src" {
		t.Fatalf("case handles: port names got %q %q, want dst src", out.Name(), inPort.Name())
	}
	missing := gobble.NewPipeline("missing-port").AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{fileOut("out")},
	}).Out("nope")
	if missing.IsZero() {
		t.Fatalf("case handles: Out(%q).IsZero() got true, want false", "nope")
	}
	if missing.Name() != "nope" {
		t.Fatalf("case handles: Out(%q).Name() got %q, want %q", "nope", missing.Name(), "nope")
	}

	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("case handles: Compose() error = %v, want nil", err)
	}
	if g == nil {
		t.Fatalf("case handles: Compose() graph = nil, want non-nil")
	}
}

func TestComposeReject(t *testing.T) {
	tests := []struct {
		name string
		pipe *gobble.Pipeline
		code gobble.DefectCode
		unit string
	}{
		{
			name: "cycle",
			pipe: cyclePipeline(),
			code: gobble.DefectCycle,
			unit: "loop",
		},
		{
			name: "missing-input zero From",
			pipe: missingInputPipeline(),
			code: gobble.DefectMissingInput,
			unit: "copy.reads",
		},
		{
			name: "missing-input In request",
			pipe: missingInRequestPipeline(),
			code: gobble.DefectMissingInput,
			unit: "copy.nope",
		},
		{
			name: "missing-output no binds",
			pipe: missingOutputPipeline(),
			code: gobble.DefectMissingOutput,
			unit: "copy",
		},
		{
			name: "missing-output Out request",
			pipe: missingOutRequestPipeline(),
			code: gobble.DefectMissingOutput,
			unit: "copy.nope",
		},
		{
			name: "missing-command",
			pipe: missingCommandPipeline(),
			code: gobble.DefectMissingCommand,
			unit: "copy",
		},
		{
			name: "invalid-name empty pipeline",
			pipe: invalidNameEmptyPipeline(),
			code: gobble.DefectInvalidName,
			unit: "pipeline",
		},
		{
			name: "invalid-name empty task",
			pipe: invalidNameEmptyTaskPipeline(),
			code: gobble.DefectInvalidName,
			unit: "prep",
		},
		{
			name: "invalid-name spelling",
			pipe: invalidNameSpellingPipeline(),
			code: gobble.DefectInvalidName,
			unit: "1copy",
		},
		{
			name: "invalid-name duplicate sibling",
			pipe: invalidNameDuplicatePipeline(),
			code: gobble.DefectInvalidName,
			unit: "copy",
		},
		{
			name: "invalid-path slash name",
			pipe: invalidPathSlashPipeline(),
			code: gobble.DefectInvalidPath,
			unit: "copy.out",
		},
		{
			name: "invalid-path empty lead and name",
			pipe: invalidPathEmptyPipeline(),
			code: gobble.DefectInvalidPath,
			unit: "copy.out",
		},
		{
			name: "invalid-path dir escape",
			pipe: invalidPathEscapePipeline(),
			code: gobble.DefectInvalidPath,
			unit: "copy.out",
		},
		{
			name: "invalid-path literal method",
			pipe: invalidPathLiteralPipeline(),
			code: gobble.DefectInvalidPath,
			unit: "copy.out",
		},
		{
			name: "foreign From colliding task id",
			pipe: foreignFromCollidingIDPipeline(),
			code: gobble.DefectMissingInput,
			unit: "use.in",
		},
		{
			name: "foreign output From colliding task id",
			pipe: foreignOutputFromCollidingIDPipeline(),
			code: gobble.DefectMissingInput,
			unit: "use.out",
		},
		{
			name: "foreign output From non-colliding task id",
			pipe: foreignOutputFromNonCollidingIDPipeline(),
			code: gobble.DefectMissingInput,
			unit: "use.out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := gobble.Compose(tt.pipe)
			if g != nil {
				t.Fatalf("case %s: Compose() graph != nil, want nil", tt.name)
			}
			var ge *gobble.Error
			if !errors.As(err, &ge) {
				t.Fatalf("case %s: Compose() error = %v, want *Error", tt.name, err)
			}
			if ge.Op != "compose" {
				t.Fatalf("case %s: Error.Op got %q, want %q", tt.name, ge.Op, "compose")
			}
			found := false
			codes := make([]gobble.DefectCode, len(ge.Defects))
			units := make([]string, len(ge.Defects))
			for i, d := range ge.Defects {
				codes[i] = d.Code
				units[i] = d.Unit
				if d.Code == tt.code && d.Unit == tt.unit {
					found = true
				}
			}
			if !found {
				t.Fatalf("case %s: Compose() defects codes got %v units %v, want code %s unit %q", tt.name, codes, units, tt.code, tt.unit)
			}
		})
	}
}

func TestComposeForeignFromCollidingID(t *testing.T) {
	g, err := gobble.Compose(foreignFromCollidingIDPipeline())
	if g != nil {
		t.Fatalf("case foreign-from: Compose() graph != nil, want nil")
	}
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		t.Fatalf("case foreign-from: Compose() error = %v, want *Error", err)
	}
	if ge.Op != "compose" {
		t.Fatalf("case foreign-from: Error.Op got %q, want %q", ge.Op, "compose")
	}
	found := false
	codes := make([]gobble.DefectCode, len(ge.Defects))
	units := make([]string, len(ge.Defects))
	for i, d := range ge.Defects {
		codes[i] = d.Code
		units[i] = d.Unit
		if d.Code == gobble.DefectMissingInput && d.Unit == "use.in" {
			found = true
		}
	}
	if !found {
		t.Fatalf("case foreign-from: Compose() defects codes got %v units %v, want code %s unit %q", codes, units, gobble.DefectMissingInput, "use.in")
	}
}

func TestComposeForeignOutputFrom(t *testing.T) {
	tests := []struct {
		name string
		pipe *gobble.Pipeline
		unit string
	}{
		{
			name: "colliding task id",
			pipe: foreignOutputFromCollidingIDPipeline(),
			unit: "use.out",
		},
		{
			name: "non-colliding task id",
			pipe: foreignOutputFromNonCollidingIDPipeline(),
			unit: "use.out",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := gobble.Compose(tt.pipe)
			if g != nil {
				t.Fatalf("case foreign-output-from %s: Compose() graph != nil, want nil (no DAG edge)", tt.name)
			}
			var ge *gobble.Error
			if !errors.As(err, &ge) {
				t.Fatalf("case foreign-output-from %s: Compose() error = %v, want *Error", tt.name, err)
			}
			if ge.Op != "compose" {
				t.Fatalf("case foreign-output-from %s: Error.Op got %q, want %q", tt.name, ge.Op, "compose")
			}
			found := false
			codes := make([]gobble.DefectCode, len(ge.Defects))
			units := make([]string, len(ge.Defects))
			for i, d := range ge.Defects {
				codes[i] = d.Code
				units[i] = d.Unit
				if d.Code == gobble.DefectMissingInput && d.Unit == tt.unit {
					found = true
				}
			}
			if !found {
				t.Fatalf("case foreign-output-from %s: Compose() defects codes got %v units %v, want code %s unit %q", tt.name, codes, units, gobble.DefectMissingInput, tt.unit)
			}
		})
	}
}

func TestComposePipelineBranchMerge(t *testing.T) {
	p := gobble.NewPipeline("root-branch")
	align := p.Branch("align")
	qc := p.Branch("qc")
	join := p.Merge("join", align, qc)
	src := align.AddTask(gobble.TaskSpec{
		Name:    "bwa",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{{Name: "bam", Spec: gobble.PathSpec{Name: "aln", Ext: ".bam"}}},
	})
	qc.AddTask(gobble.TaskSpec{
		Name:    "fastqc",
		Command: []string{"fastqc"},
		Outputs: []gobble.Bind{{Name: "html", Spec: gobble.PathSpec{Name: "qc", Ext: ".html"}}},
	})
	join.AddTask(gobble.TaskSpec{
		Name:    "report",
		Command: []string{"report"},
		Inputs:  []gobble.Bind{{Name: "bam", From: src.Out("bam")}},
		Outputs: []gobble.Bind{{Name: "summary", Spec: gobble.PathSpec{Name: "report", Ext: ".json"}}},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("case pipeline-branch-merge: Compose() error = %v, want nil", err)
	}
	if g == nil {
		t.Fatalf("case pipeline-branch-merge: Compose() graph = nil, want non-nil")
	}
	if err := gobble.Validate(g); err != nil {
		t.Fatalf("case pipeline-branch-merge: Validate() error = %v, want nil", err)
	}
}

func TestComposeNestedModuleID(t *testing.T) {
	p := gobble.NewPipeline("nested")
	p.AddModule("a").AddModule("b").AddTask(gobble.TaskSpec{
		Name:    "task",
		Command: []string{"echo"},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	raw := mustBuildPlanJSON(t, p)
	var decoded struct {
		Tasks []struct {
			ID     string `json:"id"`
			Module string `json:"module"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("case nested-module: Unmarshal() error = %v", err)
	}
	if len(decoded.Tasks) != 1 {
		t.Fatalf("case nested-module: tasks got %d, want 1", len(decoded.Tasks))
	}
	if decoded.Tasks[0].ID != "a.b.task" {
		t.Fatalf("case nested-module: task id got %q, want %q", decoded.Tasks[0].ID, "a.b.task")
	}
	if decoded.Tasks[0].Module != "b" {
		t.Fatalf("case nested-module: module got %q, want %q", decoded.Tasks[0].Module, "b")
	}
}

func TestComposeDeriveReplaceExt(t *testing.T) {
	p := gobble.NewPipeline("picard-bai")
	align := p.AddTask(gobble.TaskSpec{
		Name:    "align",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{{Name: "bam", Spec: gobble.PathSpec{Name: "aln", Ext: ".bam"}}},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "index",
		Command: []string{"picard"},
		Inputs:  []gobble.Bind{{Name: "bam", From: align.Out("bam")}},
		Outputs: []gobble.Bind{{
			Name: "bai",
			Spec: gobble.PathSpec{Ext: ".bai"},
			From: align.Out("bam"),
			Rule: gobble.DeriveReplaceExt,
		}},
	})
	raw := mustBuildPlanJSON(t, p)
	got := planOutputPath(t, raw, "index", "bai")
	if got != "aln.bai" {
		t.Fatalf("case derive-replace-ext: index.bai path got %q, want %q", got, "aln.bai")
	}
}

func TestComposeDirOnlyRestage(t *testing.T) {
	p := gobble.NewPipeline("dir-restage")
	src := p.AddTask(gobble.TaskSpec{
		Name:    "align",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{{
			Name: "bam",
			Spec: gobble.PathSpec{Dir: gobble.Dir("work/align"), Name: "sample", Steps: []string{"sorted"}, Ext: ".bam"},
		}},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Inputs:  []gobble.Bind{{Name: "in", From: src.Out("bam")}},
		Outputs: []gobble.Bind{{
			Name: "out",
			From: src.Out("bam"),
			Spec: gobble.PathSpec{Dir: gobble.Dir("out")},
		}},
	})
	raw := mustBuildPlanJSON(t, p)
	got := planOutputPath(t, raw, "copy", "out")
	if got != "out/sample.sorted.bam" {
		t.Fatalf("case dir-only-restage: copy.out path got %q, want %q", got, "out/sample.sorted.bam")
	}
}

func TestComposeStepsOnlyRestage(t *testing.T) {
	p := gobble.NewPipeline("steps-restage")
	src := p.AddTask(gobble.TaskSpec{
		Name:    "align",
		Command: []string{"bwa"},
		Outputs: []gobble.Bind{{
			Name: "bam",
			Spec: gobble.PathSpec{Dir: gobble.Dir("work"), Name: "sample", Steps: []string{"sorted"}, Ext: ".bam"},
		}},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "mark",
		Command: []string{"picard"},
		Inputs:  []gobble.Bind{{Name: "in", From: src.Out("bam")}},
		Outputs: []gobble.Bind{{
			Name: "out",
			From: src.Out("bam"),
			Spec: gobble.PathSpec{Steps: []string{"markdup"}},
		}},
	})
	raw := mustBuildPlanJSON(t, p)
	got := planOutputPath(t, raw, "mark", "out")
	if got != "work/sample.markdup.bam" {
		t.Fatalf("case steps-only-restage: mark.out path got %q, want %q", got, "work/sample.markdup.bam")
	}
}

func workflowCasePipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("workflow-case")
	readsR1 := p.AddInput("reads_r1", gobble.PathSpec{
		Dir:  gobble.Dir("in"),
		Lead: "sample_S1_L001_R1_",
		Name: "001",
		Ext:  ".fastq.gz",
	})
	readsR2 := p.AddInput("reads_r2", gobble.PathSpec{
		Dir:  gobble.Dir("in"),
		Lead: "sample_S1_L001_R2_",
		Name: "001",
		Ext:  ".fastq.gz",
	})

	prep := p.AddModule("prep")
	fastp := addFastp(prep, readsR1, readsR2)

	call := p.AddModule("call")
	align := call.Branch("align")
	qc := call.Branch("qc")
	join := call.Merge("join", align, qc)

	bam := gobble.PathSpec{
		Dir:   gobble.Dir("work/align"),
		Name:  "sample",
		Steps: []string{"sorted"},
		Ext:   ".bam",
	}
	bwa := align.AddTask(gobble.TaskSpec{
		Name:    "bwa",
		Command: []string{"bwa"},
		Image:   "example/bwa:0",
		Inputs: []gobble.Bind{
			{Name: "r1", From: fastp.Out("clean_r1")},
			{Name: "r2", From: fastp.Out("clean_r2")},
		},
		Outputs: []gobble.Bind{
			{Name: "bam", Spec: bam},
			{Name: "bai", Spec: bam.Append(".bai")},
		},
	})

	fastqc := qc.AddTask(gobble.TaskSpec{
		Name:    "fastqc",
		Command: []string{"fastqc"},
		Image:   "example/fastqc:0",
		Inputs: []gobble.Bind{
			{Name: "r1", From: fastp.Out("clean_r1")},
			{Name: "r2", From: fastp.Out("clean_r2")},
		},
		Outputs: []gobble.Bind{
			{Name: "html", Spec: gobble.Literal("sample_clean_fastqc.html").WithDir(gobble.Dir("work/qc"))},
		},
	})

	join.AddTask(gobble.TaskSpec{
		Name:    "report",
		Command: []string{"report"},
		Inputs: []gobble.Bind{
			{Name: "bam", From: bwa.Out("bam")},
			{Name: "bai", From: bwa.Out("bai")},
			{Name: "html", From: fastqc.Out("html")},
		},
		Outputs: []gobble.Bind{
			{Name: "summary", Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Name: "report", Ext: ".json"}},
		},
	})
	return p
}

func addFastp(prep *gobble.Module, r1, r2 gobble.Handle) *gobble.Task {
	return prep.AddTask(gobble.TaskSpec{
		Name:    "fastp",
		Command: []string{"fastp"},
		Image:   "example/fastp:0",
		Inputs: []gobble.Bind{
			{Name: "r1", From: r1},
			{Name: "r2", From: r2},
		},
		Outputs: []gobble.Bind{
			{
				Name: "clean_r1",
				Spec: gobble.PathSpec{
					Dir:   gobble.Dir("work/prep"),
					Lead:  "sample_S1_L001_R1_",
					Name:  "001",
					Steps: []string{"clean"},
					Ext:   ".fastq.gz",
				},
			},
			{
				Name: "clean_r2",
				Spec: gobble.PathSpec{
					Dir:   gobble.Dir("work/prep"),
					Lead:  "sample_S1_L001_R2_",
					Name:  "001",
					Steps: []string{"clean"},
					Ext:   ".fastq.gz",
				},
			},
		},
	})
}

func oneTask(name string, spec gobble.TaskSpec) *gobble.Pipeline {
	p := gobble.NewPipeline(name)
	p.AddTask(spec)
	return p
}

func fileOut(name string) gobble.Bind {
	return gobble.Bind{Name: name, Spec: gobble.PathSpec{Name: "out", Ext: ".txt"}}
}

func cyclePipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("cycle")
	inputs := []gobble.Bind{{Name: "in"}}
	t := p.AddTask(gobble.TaskSpec{
		Name:    "loop",
		Command: []string{"echo"},
		Inputs:  inputs,
		Outputs: []gobble.Bind{fileOut("out")},
	})
	inputs[0].From = t.Out("out")
	return p
}

func missingInputPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("missing-input")
	p.AddInput("reads", gobble.PathSpec{Name: "sample", Ext: ".fastq.gz"})
	p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Inputs:  []gobble.Bind{{Name: "reads"}},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	return p
}

func missingInRequestPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("missing-in")
	in := p.AddInput("reads", gobble.PathSpec{Name: "sample", Ext: ".fastq.gz"})
	t := p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Inputs:  []gobble.Bind{{Name: "src", From: in}},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	t.In("nope")
	return p
}

func missingOutputPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("missing-output")
	in := p.AddInput("reads", gobble.PathSpec{Name: "sample", Ext: ".fastq.gz"})
	p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Inputs:  []gobble.Bind{{Name: "src", From: in}},
	})
	return p
}

func missingOutRequestPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("missing-out")
	in := p.AddInput("reads", gobble.PathSpec{Name: "sample", Ext: ".fastq.gz"})
	t := p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Inputs:  []gobble.Bind{{Name: "src", From: in}},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	t.Out("nope")
	return p
}

func missingCommandPipeline() *gobble.Pipeline {
	return oneTask("missing-command", gobble.TaskSpec{
		Name:    "copy",
		Outputs: []gobble.Bind{fileOut("out")},
	})
}

func invalidNameEmptyPipeline() *gobble.Pipeline {
	return oneTask("", gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{fileOut("out")},
	})
}

func invalidNameEmptyTaskPipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("empty-task")
	p.AddModule("prep").AddTask(gobble.TaskSpec{
		Name:    "",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	return p
}

func invalidNameSpellingPipeline() *gobble.Pipeline {
	return oneTask("invalid-spelling", gobble.TaskSpec{
		Name:    "1copy",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{fileOut("out")},
	})
}

func invalidNameDuplicatePipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("dup")
	spec := gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{fileOut("out")},
	}
	p.AddTask(spec)
	p.AddTask(spec)
	return p
}

func invalidPathSlashPipeline() *gobble.Pipeline {
	return oneTask("slash", gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Name: "a/b", Ext: ".txt"}}},
	})
}

func invalidPathEmptyPipeline() *gobble.Pipeline {
	return oneTask("empty-path", gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Ext: ".txt"}}},
	})
}

func invalidPathEscapePipeline() *gobble.Pipeline {
	return oneTask("escape", gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("../out"), Name: "x", Ext: ".txt"}}},
	})
}

func invalidPathLiteralPipeline() *gobble.Pipeline {
	return oneTask("literal", gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"cp"},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.Literal("aln.bam").AppendStep("sorted")}},
	})
}

func foreignFromCollidingIDPipeline() *gobble.Pipeline {
	a := gobble.NewPipeline("run-a")
	ta := a.AddModule("prep").AddTask(gobble.TaskSpec{
		Name:    "fastp",
		Command: []string{"fastp"},
		Outputs: []gobble.Bind{{Name: "clean", Spec: gobble.PathSpec{Name: "foreign", Ext: ".fq"}}},
	})
	b := gobble.NewPipeline("run-b")
	b.AddModule("prep").AddTask(gobble.TaskSpec{
		Name:    "fastp",
		Command: []string{"fastp"},
		Outputs: []gobble.Bind{{Name: "clean", Spec: gobble.PathSpec{Name: "local", Ext: ".fq"}}},
	})
	b.AddTask(gobble.TaskSpec{
		Name:    "use",
		Command: []string{"use"},
		Inputs:  []gobble.Bind{{Name: "in", From: ta.Out("clean")}},
		Outputs: []gobble.Bind{fileOut("out")},
	})
	return b
}

func foreignOutputFromCollidingIDPipeline() *gobble.Pipeline {
	a := gobble.NewPipeline("run-a")
	ta := a.AddModule("prep").AddTask(gobble.TaskSpec{
		Name:    "fastp",
		Command: []string{"fastp"},
		Outputs: []gobble.Bind{{Name: "clean", Spec: gobble.PathSpec{Name: "foreign", Ext: ".fq"}}},
	})
	b := gobble.NewPipeline("run-b")
	b.AddModule("prep").AddTask(gobble.TaskSpec{
		Name:    "fastp",
		Command: []string{"fastp"},
		Outputs: []gobble.Bind{{Name: "clean", Spec: gobble.PathSpec{Name: "local", Ext: ".fq"}}},
	})
	b.AddTask(gobble.TaskSpec{
		Name:    "use",
		Command: []string{"use"},
		Outputs: []gobble.Bind{{
			Name: "out",
			From: ta.Out("clean"),
			Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Name: "use", Ext: ".txt"},
		}},
	})
	return b
}

func foreignOutputFromNonCollidingIDPipeline() *gobble.Pipeline {
	a := gobble.NewPipeline("run-a")
	ta := a.AddTask(gobble.TaskSpec{
		Name:    "src",
		Command: []string{"echo"},
		Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Name: "foreign", Ext: ".txt"}}},
	})
	b := gobble.NewPipeline("run-b")
	b.AddTask(gobble.TaskSpec{
		Name:    "use",
		Command: []string{"use"},
		Outputs: []gobble.Bind{{
			Name: "out",
			From: ta.Out("out"),
			Spec: gobble.PathSpec{Dir: gobble.Dir("out"), Name: "use", Ext: ".txt"},
		}},
	})
	return b
}

func mustBuildPlanJSON(t *testing.T, p *gobble.Pipeline) []byte {
	t.Helper()
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v, want nil", err)
	}
	if g == nil {
		t.Fatalf("Compose() graph = nil, want non-nil")
	}
	var buf bytes.Buffer
	plan, err := gobble.BuildPlan(g, gobble.WriteTo(&buf))
	if err != nil {
		t.Fatalf("BuildPlan() error = %v, want nil", err)
	}
	if plan == nil {
		t.Fatalf("BuildPlan() plan = nil, want non-nil")
	}
	return buf.Bytes()
}

func planOutputPath(t *testing.T, raw []byte, taskID, port string) string {
	t.Helper()
	var decoded struct {
		Tasks []struct {
			ID      string `json:"id"`
			Outputs []struct {
				Name string `json:"name"`
				Path string `json:"path"`
			} `json:"outputs"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, task := range decoded.Tasks {
		if task.ID != taskID {
			continue
		}
		for _, b := range task.Outputs {
			if b.Name == port {
				return b.Path
			}
		}
	}
	t.Fatalf("plan missing %s.%s", taskID, port)
	return ""
}
