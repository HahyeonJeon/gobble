package stargenomegenerate_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	stargenomegenerate "github.com/HahyeonJeon/gobble/assets/modules/star-genome-generate"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestSelectedSTARGenomeGenerateStandalone(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	gtf := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genes", Ext: ".gtf"}
	p := stargenomegenerate.Pipeline(fasta, gtf, stargenomegenerate.Options{
		Options:             modules.Options{Resources: gobble.Resources{CPU: 2}},
		SJDBOverhang:        100,
		GenomeSAIndexNBases: 7,
	})
	task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "star_genome_generate")
	if task.Image != string(stargenomegenerate.DefaultImage) {
		t.Fatalf("image = %q, want selected immutable image %q", task.Image, stargenomegenerate.DefaultImage)
	}
	if !pc.ContainsAll(task.Command,
		"STAR", "--runMode", "genomeGenerate",
		"--genomeDir", "work/star-genome",
		"--genomeFastaFiles", "in/genome.fasta",
		"--sjdbGTFfile", "in/genes.gtf",
		"--runThreadN", "2",
		"--sjdbOverhang", "100",
		"--genomeSAindexNbases", "7",
	) {
		t.Fatalf("command = %#v, want selected STAR genomeGenerate argv", task.Command)
	}
	pc.AssertIOPath(t, task.Inputs, "fasta", "in/genome.fasta")
	pc.AssertIOPath(t, task.Inputs, "gtf", "in/genes.gtf")
	pc.AssertTreeIO(t, task.Outputs, "index", "work/star-genome")
}

func TestSelectedSTARGenomeGenerateNestedModule(t *testing.T) {
	p := gobble.NewPipeline("assay")
	hf := p.AddInput("fasta", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"})
	hg := p.AddInput("gtf", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genes", Ext: ".gtf"})
	ports, err := stargenomegenerate.Add(p.AddModule("reference"), hf, hg, stargenomegenerate.Options{GenomeSAIndexNBases: 7})
	if err != nil || ports.Index.IsZero() {
		t.Fatalf("Add selected STAR genomeGenerate = (%+v, %v)", ports, err)
	}
	task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "reference.star_genome_generate")
	if task.Module != "reference" || task.Image != string(stargenomegenerate.DefaultImage) {
		t.Fatalf("nested selected task = %+v", task)
	}
}
