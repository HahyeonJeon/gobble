package bismarkalign_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	bismarkalign "github.com/HahyeonJeon/gobble/assets/modules/bismark-align"
	bismarkgenome "github.com/HahyeonJeon/gobble/assets/modules/bismark-genome"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestSelectedBismarkAlignStandalone(t *testing.T) {
	index := gobble.DeclareTree(gobble.Dir("in/bismark-index"))
	read1 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	read2 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"}
	p := bismarkalign.Pipeline(index, read1, read2, bismarkalign.Options{
		Options:       modules.Options{Resources: gobble.Resources{CPU: 6, Memory: "15g"}},
		Prefix:        "sample",
		ScoreMinSlope: 0.2,
		MinInsert:     20,
		MaxInsert:     500,
	})
	task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "bismark_align")
	if task.Image != string(bismarkalign.DefaultImage) {
		t.Fatalf("image = %q, want selected immutable image %q", task.Image, bismarkalign.DefaultImage)
	}
	if !pc.ContainsAll(task.Command,
		"bismark", "--bowtie2", "--genome", "in/bismark-index", "--basename", "sample",
		"--multicore", "2", "--score_min", "L,0,-0.2", "--minins", "20", "--maxins", "500",
		"-1", "in/test_1.fastq.gz", "-2", "in/test_2.fastq.gz",
	) {
		t.Fatalf("command = %#v, want selected Bismark alignment argv", task.Command)
	}
	pc.AssertIOPath(t, task.Outputs, "bam", "work/bismark-align/sample_pe.bam")
	pc.AssertIOPath(t, task.Outputs, "report", "work/bismark-align/sample_PE_report.txt")
	pc.AssertTreeIO(t, task.Inputs, "index", "in/bismark-index")
}

func TestSelectedBismarkAlignConsumesSelectedGenomeTree(t *testing.T) {
	p := gobble.NewPipeline("assay")
	hf := p.AddInput("fasta", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fa"})
	h1 := p.AddInput("read1", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"})
	h2 := p.AddInput("read2", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"})
	parent := p.AddModule("methylation")
	index, err := bismarkgenome.Add(parent, hf, bismarkgenome.Options{})
	if err != nil {
		t.Fatal(err)
	}
	ports, err := bismarkalign.Add(parent, index.Index, h1, h2, bismarkalign.Options{})
	if err != nil || ports.BAM.IsZero() {
		t.Fatalf("Add selected Bismark align = (%+v, %v)", ports, err)
	}
	task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "methylation.bismark_align")
	if task.Module != "methylation" || task.Image != string(bismarkalign.DefaultImage) {
		t.Fatalf("nested selected task = %+v", task)
	}
}
