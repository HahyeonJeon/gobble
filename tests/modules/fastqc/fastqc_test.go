package fastqcevidence_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	. "github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestFastQCStandaloneComposeBuildPlan(t *testing.T) {
	reads := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	opts := FastQCOptions{
		ExtraArgs: []string{"--kmers", "7"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := FastQCPipeline(reads, opts)
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "fastqc")
	if task.Name != "fastqc" {
		t.Fatalf("task name = %q, want fastqc", task.Name)
	}
	if task.Image != "quay.io/biocontainers/fastqc:0.12.1--hdfd78af_0" {
		t.Fatalf("image = %q, want locked FastQC pin", task.Image)
	}
	if !pc.ContainsAll(task.Command, "fastqc", "--outdir", "work/fastqc", "--noextract", "--threads", "2", "in/test_1.fastq.gz", "--kmers", "7") {
		t.Fatalf("command = %#v, want named flags, input, extra-args", task.Command)
	}
	if lastN := task.Command[len(task.Command)-2:]; lastN[0] != "--kmers" || lastN[1] != "7" {
		t.Fatalf("extra-args last tokens = %#v, want [--kmers 7]", lastN)
	}
	pc.AssertUniqueParamNames(t, task.Params)
	pc.AssertIOPath(t, task.Outputs, "html", "work/fastqc/test_1_fastqc.html")
	pc.AssertIOPath(t, task.Outputs, "zip", "work/fastqc/test_1_fastqc.zip")
}

func TestFastQCNestedModule(t *testing.T) {
	reads := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	p := gobble.NewPipeline("assay")
	h := p.AddInput("reads", reads)
	mod := p.AddModule("raw")
	ports := AddFastQC(mod, h, FastQCOptions{ExtraArgs: []string{"--quiet"}})
	if ports.HTML.IsZero() || ports.Zip.IsZero() {
		t.Fatalf("ports HTML/Zip IsZero = %v/%v, want false", ports.HTML.IsZero(), ports.Zip.IsZero())
	}
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "raw.fastqc")
	if task.Name != "fastqc" {
		t.Fatalf("nested name = %q, want fastqc", task.Name)
	}
	if task.Module != "raw" {
		t.Fatalf("nested module = %q, want raw", task.Module)
	}
	if !pc.ContainsAll(task.Command, "--quiet") {
		t.Fatalf("command = %#v, want extra-args --quiet", task.Command)
	}
}
