package multiqcevidence_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	. "github.com/HahyeonJeon/gobble/assets/modules/multiqc"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestMultiQCStandaloneComposeBuildPlan(t *testing.T) {
	report := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_fastqc", Ext: ".zip"}
	opts := MultiQCOptions{
		ExtraArgs: []string{"--title", "qc"},
		Resources: gobble.Resources{CPU: 1},
	}
	p := MultiQCPipeline([]gobble.PathSpec{report}, opts)
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "multiqc")
	if task.Name != "multiqc" {
		t.Fatalf("task name = %q, want multiqc", task.Name)
	}
	if task.Image != "quay.io/biocontainers/multiqc:1.35--pyhdfd78af_0" {
		t.Fatalf("image = %q, want locked MultiQC pin", task.Image)
	}
	if !pc.ContainsAll(task.Command, "multiqc", "--force", "--outdir", "work/multiqc", "--zip-data-dir", "in/test_fastqc.zip", "--title", "qc") {
		t.Fatalf("command = %#v, want named flags, reports, extra-args", task.Command)
	}
	n := len(task.Command)
	if n < 2 || task.Command[n-2] != "--title" || task.Command[n-1] != "qc" {
		t.Fatalf("extra-args last tokens = %#v, want [--title qc]", task.Command[n-2:])
	}
	if pc.ContainsAll(task.Command, "--threads") || pc.ContainsAll(task.Command, "--thread") {
		t.Fatalf("command = %#v, MultiQC must not copy Resources.CPU into a thread flag", task.Command)
	}
	pc.AssertUniqueParamNames(t, task.Params)
	pc.AssertIOPath(t, task.Inputs, "report_0", "in/test_fastqc.zip")
	pc.AssertIOPath(t, task.Outputs, "html", "work/multiqc/multiqc_report.html")
	pc.AssertIOPath(t, task.Outputs, "data", "work/multiqc/multiqc_data.zip")
}

func TestMultiQCNestedModule(t *testing.T) {
	report := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_fastqc", Ext: ".zip"}
	p := gobble.NewPipeline("assay")
	h := p.AddInput("zip", report)
	mod := p.AddModule("qc")
	ports := AddMultiQC(mod, []gobble.Handle{h}, MultiQCOptions{ExtraArgs: []string{"--fullnames"}})
	if ports.HTML.IsZero() || ports.Data.IsZero() {
		t.Fatalf("ports HTML/Data IsZero = %v/%v, want false", ports.HTML.IsZero(), ports.Data.IsZero())
	}
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "qc.multiqc")
	if task.Name != "multiqc" || task.Module != "qc" {
		t.Fatalf("nested task = %+v, want name multiqc module qc", task)
	}
	if !pc.ContainsAll(task.Command, "--fullnames") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
}
