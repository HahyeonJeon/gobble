package fastpevidence_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	. "github.com/HahyeonJeon/gobble/assets/modules/fastp"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestFastpStandaloneComposeBuildPlan(t *testing.T) {
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"}
	opts := FastpOptions{
		ExtraArgs: []string{"--qualified_quality_phred", "15"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := FastpPipeline(r1, r2, opts)
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "fastp")
	if task.Name != "fastp" {
		t.Fatalf("task name = %q, want fastp", task.Name)
	}
	if task.Image != "quay.io/biocontainers/fastp:1.3.6--h43da1c4_0" {
		t.Fatalf("image = %q, want locked fastp pin", task.Image)
	}
	if !pc.ContainsAll(task.Command,
		"fastp",
		"--in1", "in/test_1.fastq.gz",
		"--in2", "in/test_2.fastq.gz",
		"--out1", "work/fastp/test_1.clean.fastq.gz",
		"--out2", "work/fastp/test_2.clean.fastq.gz",
		"--json", "work/fastp/test_1.fastp.json",
		"--html", "work/fastp/test_1.fastp.html",
		"--detect_adapter_for_pe",
		"--thread", "2",
		"--qualified_quality_phred", "15",
	) {
		t.Fatalf("command = %#v, want named flags then extra-args", task.Command)
	}
	n := len(task.Command)
	if n < 2 || task.Command[n-2] != "--qualified_quality_phred" || task.Command[n-1] != "15" {
		t.Fatalf("extra-args last tokens = %#v, want [--qualified_quality_phred 15]", task.Command[n-2:])
	}
	pc.AssertUniqueParamNames(t, task.Params)
	pc.AssertIOPath(t, task.Outputs, "clean_r1", "work/fastp/test_1.clean.fastq.gz")
	pc.AssertIOPath(t, task.Outputs, "clean_r2", "work/fastp/test_2.clean.fastq.gz")
	pc.AssertIOPath(t, task.Outputs, "json", "work/fastp/test_1.fastp.json")
	pc.AssertIOPath(t, task.Outputs, "html", "work/fastp/test_1.fastp.html")
}

func TestFastpNestedModule(t *testing.T) {
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"}
	p := gobble.NewPipeline("assay")
	h1 := p.AddInput("r1", r1)
	h2 := p.AddInput("r2", r2)
	mod := p.AddModule("prep")
	ports := AddFastp(mod, h1, h2, FastpOptions{ExtraArgs: []string{"--disable_quality_filtering"}})
	if ports.CleanR1.IsZero() || ports.JSON.IsZero() {
		t.Fatalf("ports IsZero, want handles")
	}
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "prep.fastp")
	if task.Name != "fastp" || task.Module != "prep" {
		t.Fatalf("nested task = %+v, want name fastp module prep", task)
	}
	if !pc.ContainsAll(task.Command, "--disable_quality_filtering") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
}
