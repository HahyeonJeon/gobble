package gobble_test

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

// workflowCaseGolden is the Ideation part 03 Bind/PathSpec table encoded
// as plan JSON. workflowCasePipeline in compose_test.go is that table.
const workflowCaseGolden = "testdata/workflow-case/plan.json"

func TestWorkflowCaseBuildPlan(t *testing.T) {
	g, err := gobble.Compose(workflowCasePipeline())
	if err != nil {
		t.Fatalf("case workflow-case: Compose() error = %v, want nil", err)
	}
	var buf bytes.Buffer
	plan, err := gobble.BuildPlan(g, gobble.WriteTo(&buf))
	if err != nil {
		t.Fatalf("case workflow-case: BuildPlan() error = %v, want nil", err)
	}
	if plan == nil {
		t.Fatalf("case workflow-case: BuildPlan() plan = nil, want non-nil")
	}

	want, err := os.ReadFile(workflowCaseGolden)
	if err != nil {
		t.Fatalf("case workflow-case: ReadFile(%s) error = %v", workflowCaseGolden, err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Fatalf("case workflow-case: BuildPlan JSON != %s\ngot:\n%s\nwant:\n%s", workflowCaseGolden, buf.Bytes(), want)
	}

	var decoded workflowCasePlan
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("case workflow-case: Unmarshal() error = %v", err)
	}
	assertWorkflowCasePlan(t, decoded)
}

func assertWorkflowCasePlan(t *testing.T, got workflowCasePlan) {
	t.Helper()
	if got.Pipeline != "workflow-case" {
		t.Fatalf("case workflow-case: pipeline got %q, want %q", got.Pipeline, "workflow-case")
	}
	wantIDs := []string{"prep.fastp", "call.align.bwa", "call.qc.fastqc", "call.join.report"}
	if !reflect.DeepEqual(got.DAG.Nodes, wantIDs) {
		t.Fatalf("case workflow-case: dag.nodes got %v, want %v", got.DAG.Nodes, wantIDs)
	}
	wantEdges := []workflowCaseEdge{
		{From: "prep.fastp.clean_r1", To: "call.align.bwa.r1"},
		{From: "prep.fastp.clean_r2", To: "call.align.bwa.r2"},
		{From: "prep.fastp.clean_r1", To: "call.qc.fastqc.r1"},
		{From: "prep.fastp.clean_r2", To: "call.qc.fastqc.r2"},
		{From: "call.align.bwa.bam", To: "call.join.report.bam"},
		{From: "call.align.bwa.bai", To: "call.join.report.bai"},
		{From: "call.qc.fastqc.html", To: "call.join.report.html"},
	}
	if !reflect.DeepEqual(got.DAG.Edges, wantEdges) {
		t.Fatalf("case workflow-case: dag.edges got %#v, want %#v", got.DAG.Edges, wantEdges)
	}

	byID := make(map[string]workflowCaseTask, len(got.Tasks))
	ids := make([]string, 0, len(got.Tasks))
	for _, task := range got.Tasks {
		ids = append(ids, task.ID)
		byID[task.ID] = task
		if task.Backend != "local" {
			t.Fatalf("case workflow-case: %s backend got %q, want %q", task.ID, task.Backend, "local")
		}
		if task.Params == nil {
			t.Fatalf("case workflow-case: %s params = null, want []", task.ID)
		}
		if len(task.Params) != 0 {
			t.Fatalf("case workflow-case: %s params got %#v, want empty", task.ID, task.Params)
		}
		if task.Resources.CPU != 0 || task.Resources.Memory != "" {
			t.Fatalf("case workflow-case: %s resources got cpu %v memory %q, want 0 and empty", task.ID, task.Resources.CPU, task.Resources.Memory)
		}
	}
	if !reflect.DeepEqual(ids, wantIDs) {
		t.Fatalf("case workflow-case: task ids got %v, want %v", ids, wantIDs)
	}

	assertTaskMeta(t, byID["prep.fastp"], "fastp", "prep", "", "", "example/fastp:0")
	assertTaskMeta(t, byID["call.align.bwa"], "bwa", "call", "align", "", "example/bwa:0")
	assertTaskMeta(t, byID["call.qc.fastqc"], "fastqc", "call", "qc", "", "example/fastqc:0")
	assertTaskMeta(t, byID["call.join.report"], "report", "call", "", "join", "")

	assertIO(t, "prep.fastp.r1", byID["prep.fastp"].Inputs, "r1", "in/sample_S1_L001_R1_001.fastq.gz", workflowCaseSpec{
		Dir: "in", Lead: "sample_S1_L001_R1_", Name: "001", Steps: []string{}, Ext: ".fastq.gz",
	})
	assertIO(t, "prep.fastp.r2", byID["prep.fastp"].Inputs, "r2", "in/sample_S1_L001_R2_001.fastq.gz", workflowCaseSpec{
		Dir: "in", Lead: "sample_S1_L001_R2_", Name: "001", Steps: []string{}, Ext: ".fastq.gz",
	})
	assertIO(t, "prep.fastp.clean_r1", byID["prep.fastp"].Outputs, "clean_r1", "work/prep/sample_S1_L001_R1_001.clean.fastq.gz", workflowCaseSpec{
		Dir: "work/prep", Lead: "sample_S1_L001_R1_", Name: "001", Steps: []string{"clean"}, Ext: ".fastq.gz",
	})
	assertIO(t, "prep.fastp.clean_r2", byID["prep.fastp"].Outputs, "clean_r2", "work/prep/sample_S1_L001_R2_001.clean.fastq.gz", workflowCaseSpec{
		Dir: "work/prep", Lead: "sample_S1_L001_R2_", Name: "001", Steps: []string{"clean"}, Ext: ".fastq.gz",
	})
	assertIO(t, "call.align.bwa.bam", byID["call.align.bwa"].Outputs, "bam", "work/align/sample.sorted.bam", workflowCaseSpec{
		Dir: "work/align", Name: "sample", Steps: []string{"sorted"}, Ext: ".bam",
	})
	assertIO(t, "call.align.bwa.bai", byID["call.align.bwa"].Outputs, "bai", "work/align/sample.sorted.bam.bai", workflowCaseSpec{
		Dir: "work/align", Name: "sample", Steps: []string{"sorted"}, Ext: ".bam.bai",
	})
	assertIO(t, "call.qc.fastqc.html", byID["call.qc.fastqc"].Outputs, "html", "work/qc/sample_clean_fastqc.html", workflowCaseSpec{
		Dir: "work/qc", Steps: []string{}, Literal: true,
	})
	assertIO(t, "call.join.report.summary", byID["call.join.report"].Outputs, "summary", "out/report.json", workflowCaseSpec{
		Dir: "out", Name: "report", Steps: []string{}, Ext: ".json",
	})
}

func assertTaskMeta(t *testing.T, task workflowCaseTask, name, module, branch, merge, image string) {
	t.Helper()
	if task.Name != name || task.Module != module || task.Branch != branch || task.Merge != merge {
		t.Fatalf("case workflow-case: %s meta name/module/branch/merge got %q %q %q %q, want %q %q %q %q",
			task.ID, task.Name, task.Module, task.Branch, task.Merge, name, module, branch, merge)
	}
	if task.Image != image {
		t.Fatalf("case workflow-case: %s image got %q, want %q (string only; never pulled)", task.ID, task.Image, image)
	}
}

func assertIO(t *testing.T, unit string, binds []workflowCaseIO, name, path string, spec workflowCaseSpec) {
	t.Helper()
	for _, b := range binds {
		if b.Name != name {
			continue
		}
		if b.Path != path {
			t.Fatalf("case workflow-case: %s path got %q, want %q", unit, b.Path, path)
		}
		if spec.Steps == nil {
			spec.Steps = []string{}
		}
		if b.Spec.Steps == nil {
			b.Spec.Steps = []string{}
		}
		if !reflect.DeepEqual(b.Spec, spec) {
			t.Fatalf("case workflow-case: %s spec got %#v, want %#v", unit, b.Spec, spec)
		}
		return
	}
	t.Fatalf("case workflow-case: missing bind %s", unit)
}

type workflowCasePlan struct {
	Pipeline string             `json:"pipeline"`
	Tasks    []workflowCaseTask `json:"tasks"`
	DAG      workflowCaseDAG    `json:"dag"`
}

type workflowCaseTask struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	Module    string              `json:"module"`
	Branch    string              `json:"branch"`
	Merge     string              `json:"merge"`
	Command   []string            `json:"command"`
	Image     string              `json:"image"`
	Backend   string              `json:"backend"`
	Resources workflowCaseRes     `json:"resources"`
	Params    []workflowCaseParam `json:"params"`
	Inputs    []workflowCaseIO    `json:"inputs"`
	Outputs   []workflowCaseIO    `json:"outputs"`
}

type workflowCaseRes struct {
	CPU    float64 `json:"cpu"`
	Memory string  `json:"memory"`
}

type workflowCaseParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type workflowCaseIO struct {
	Name string           `json:"name"`
	Path string           `json:"path"`
	Spec workflowCaseSpec `json:"spec"`
}

type workflowCaseSpec struct {
	Dir     string   `json:"dir"`
	Lead    string   `json:"lead"`
	Name    string   `json:"name"`
	Steps   []string `json:"steps"`
	Ext     string   `json:"ext"`
	Literal bool     `json:"literal"`
}

type workflowCaseDAG struct {
	Nodes []string           `json:"nodes"`
	Edges []workflowCaseEdge `json:"edges"`
}

type workflowCaseEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}
