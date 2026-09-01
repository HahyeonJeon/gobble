package staralign_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	staralign "github.com/HahyeonJeon/gobble/assets/modules/star-align"
	stargenomegenerate "github.com/HahyeonJeon/gobble/assets/modules/star-genome-generate"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestSelectedSTARAlignStandalone(t *testing.T) {
	index := gobble.DeclareTree(gobble.Dir("in/star-index"))
	gtf := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genes", Ext: ".gtf"}
	read1 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	read2 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"}
	p := staralign.Pipeline(index, gtf, read1, read2, staralign.Options{
		Options:   modules.Options{Resources: gobble.Resources{CPU: 2}, ExtraArgs: []string{"--outFilterMultimapNmax", "1"}},
		Sample:    "sample",
		ReadGroup: "run1",
	})
	task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "star_align")
	if task.Image != string(staralign.DefaultImage) {
		t.Fatalf("image = %q, want selected immutable image %q", task.Image, staralign.DefaultImage)
	}
	if !pc.ContainsAll(task.Command,
		"STAR", "--genomeDir", "in/star-index", "--readFilesIn", "in/test_1.fastq.gz", "in/test_2.fastq.gz",
		"--sjdbGTFfile", "in/genes.gtf", "--quantMode", "TranscriptomeSAM", "GeneCounts",
		"--runThreadN", "2", "--outFilterMultimapNmax", "1",
	) {
		t.Fatalf("command = %#v, want selected STAR alignment argv", task.Command)
	}
	pc.AssertIOPath(t, task.Outputs, "genome_bam", "work/star-align/Aligned.out.bam")
	pc.AssertIOPath(t, task.Outputs, "transcript_bam", "work/star-align/Aligned.toTranscriptome.out.bam")
	pc.AssertTreeIO(t, task.Inputs, "index", "in/star-index")
}

func TestSelectedSTARAlignConsumesSelectedGenomeIndex(t *testing.T) {
	p := gobble.NewPipeline("assay")
	hf := p.AddInput("fasta", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"})
	hg := p.AddInput("gtf", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genes", Ext: ".gtf"})
	h1 := p.AddInput("read1", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"})
	h2 := p.AddInput("read2", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"})
	parent := p.AddModule("alignment")
	index, err := stargenomegenerate.Add(parent, hf, hg, stargenomegenerate.Options{GenomeSAIndexNBases: 7})
	if err != nil {
		t.Fatal(err)
	}
	ports, err := staralign.Add(parent, index.Index, hg, h1, h2, staralign.Options{})
	if err != nil || ports.GenomeBAM.IsZero() || ports.TranscriptBAM.IsZero() {
		t.Fatalf("Add selected STAR align = (%+v, %v)", ports, err)
	}
	task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "alignment.star_align")
	if task.Module != "alignment" || task.Image != string(staralign.DefaultImage) {
		t.Fatalf("nested selected task = %+v", task)
	}
}
