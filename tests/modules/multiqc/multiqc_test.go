package multiqcevidence_test

import (
	"errors"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	. "github.com/HahyeonJeon/gobble/assets/modules/multiqc"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestMultiQCStandaloneComposeBuildPlan(t *testing.T) {
	report := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_fastqc", Ext: ".zip"}
	opts := Options{
		Options: modules.Options{
			ExtraArgs: []string{"--title", "qc"},
			Resources: gobble.Resources{CPU: 1},
		},
	}
	p := Pipeline([]gobble.PathSpec{report}, opts)
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "multiqc")
	if task.Name != "multiqc" {
		t.Fatalf("task name = %q, want multiqc", task.Name)
	}
	if task.Image != string(DefaultImage) {
		t.Fatalf("image = %q, want locked MultiQC pin", task.Image)
	}
	if !pc.ContainsAll(task.Command, "multiqc", "--force", "--outdir", "results/multiqc", ".", "--title", "qc") {
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
	pc.AssertIOPath(t, task.Outputs, "html", "results/multiqc/multiqc_report.html")
	pc.AssertTreeIO(t, task.Outputs, "data", "results/multiqc/multiqc_data")
}

func TestMultiQCNestedModule(t *testing.T) {
	report := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_fastqc", Ext: ".zip"}
	p := gobble.NewPipeline("assay")
	h := p.AddInput("zip", report)
	mod := p.AddModule("qc")
	ports, err := Add(mod, []gobble.Handle{h}, Options{Options: modules.Options{ExtraArgs: []string{"--fullnames"}}})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
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

func TestMultiQCProductRejectsPathBearingExtraArgs(t *testing.T) {
	tests := []struct {
		name  string
		extra string
	}{
		{name: "output long", extra: "--outdir=elsewhere"},
		{name: "output short", extra: "-oelsewhere"},
		{name: "filename long", extra: "--filename=other.html"},
		{name: "filename short", extra: "-nother.html"},
		{name: "file list", extra: "--file-list=in/reports.txt"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pipeline := Pipeline(
				[]gobble.PathSpec{{Dir: gobble.Dir("in"), Base: "fastqc", Ext: ".zip"}},
				Options{Options: modules.Options{ExtraArgs: []string{test.extra}}},
			)
			graph, err := gobble.Compose(pipeline)
			var composeErr *gobble.Error
			if graph != nil || !errors.As(err, &composeErr) || len(composeErr.Defects) != 1 || composeErr.Defects[0].Code != gobble.DefectInvalidValue || composeErr.Defects[0].Unit != "multiqc" {
				t.Fatalf("Compose() with ExtraArgs %q returned graph=%t, error=%v; want one multiqc invalid-value defect", test.extra, graph != nil, err)
			}
		})
	}
}
