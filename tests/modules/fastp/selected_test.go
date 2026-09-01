package fastpevidence_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	"github.com/HahyeonJeon/gobble/assets/modules/fastp"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestSelectedFastPStandalone(t *testing.T) {
	read1 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	read2 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"}
	p := fastp.ProductPipeline(read1, read2, fastp.Options{
		Options: modules.Options{Resources: gobble.Resources{CPU: 2}, ExtraArgs: []string{"--disable_quality_filtering"}},
		Prefix:  "sample",
	})
	task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "fastp")
	if task.Image != string(fastp.DefaultImage) {
		t.Fatalf("image = %q, want selected immutable image %q", task.Image, fastp.DefaultImage)
	}
	if !pc.ContainsAll(task.Command,
		"fastp", "--in1", "in/test_1.fastq.gz", "--in2", "in/test_2.fastq.gz",
		"--out1", "work/fastp/sample_R1.fastp.fastq.gz", "--out2", "work/fastp/sample_R2.fastp.fastq.gz",
		"--thread", "2", "--disable_quality_filtering",
	) {
		t.Fatalf("command = %#v, want selected FastP argv", task.Command)
	}
	pc.AssertIOPath(t, task.Outputs, "prepared_read1", "work/fastp/sample_R1.fastp.fastq.gz")
	pc.AssertIOPath(t, task.Outputs, "prepared_read2", "work/fastp/sample_R2.fastp.fastq.gz")
}

func TestSelectedFastPNestedModule(t *testing.T) {
	p := gobble.NewPipeline("assay")
	h1 := p.AddInput("read1", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"})
	h2 := p.AddInput("read2", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"})
	ports, err := fastp.Add(p.AddModule("reads"), h1, h2, fastp.Options{})
	if err != nil || ports.Read1.IsZero() || ports.Read2.IsZero() {
		t.Fatalf("Add selected FastP = (%+v, %v)", ports, err)
	}
	task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "reads.fastp")
	if task.Module != "reads" || task.Image != string(fastp.DefaultImage) {
		t.Fatalf("nested selected task = %+v", task)
	}
}
