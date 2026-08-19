package assets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestSTARAlignStandaloneComposeBuildPlan(t *testing.T) {
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_2", Ext: ".fastq.gz"}
	opts := STARAlignOptions{
		ExtraArgs: []string{"--outFilterMultimapNmax", "1"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := STARAlignPipeline(r1, r2, opts)
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "star_align")
	if task.Name != "star_align" {
		t.Fatalf("task name = %q, want star_align", task.Name)
	}
	if task.Image != starImage {
		t.Fatalf("image = %q, want %q", task.Image, starImage)
	}
	if commandHasSamtools(task.Command) {
		t.Fatalf("command = %#v, must not contain samtools", task.Command)
	}
	if !containsAll(task.Command,
		"STAR",
		"--genomeDir", "work/star-genome",
		"--readFilesIn", "in/test_1.fastq.gz", "in/test_2.fastq.gz",
		"--readFilesCommand", "zcat",
		"--runThreadN", "2",
		"--outFileNamePrefix", "work/star-align/",
		"--outSAMtype", "BAM", "Unsorted",
		"--outFilterMultimapNmax", "1",
	) {
		t.Fatalf("command = %#v, want named flags then extra-args", task.Command)
	}
	n := len(task.Command)
	if n < 2 || task.Command[n-2] != "--outFilterMultimapNmax" || task.Command[n-1] != "1" {
		t.Fatalf("command tail = %#v, want [--outFilterMultimapNmax 1]", task.Command)
	}
	assertUniqueParamNames(t, task.Params)
	assertIOPath(t, task.Outputs, "bam", "work/star-align/Aligned.out.bam")
	assertGroupMembers(t, task.Inputs, "index", wantSTARGenomeMembers("work/star-genome"))
}

func TestSTARAlignNestedModule(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "genome", Ext: ".fasta"}
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_2", Ext: ".fastq.gz"}
	p := gobble.NewPipeline("assay")
	hf := p.AddInput("fasta", fasta)
	h1 := p.AddInput("r1", r1)
	h2 := p.AddInput("r2", r2)
	mod := AddModule(p, "align")
	idx := AddSTARGenomeGenerate(mod, hf, STARGenomeGenerateOptions{ExtraArgs: []string{"--genomeSAindexNbases", "7"}})
	ports := AddSTARAlign(mod, idx.Index, h1, h2, STARAlignOptions{ExtraArgs: []string{"--outFilterMultimapNmax", "1"}})
	if ports.BAM.IsZero() {
		t.Fatalf("ports.BAM IsZero = true, want false")
	}
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "align.star_align")
	if task.Name != "star_align" || task.Module != "align" {
		t.Fatalf("nested task = %+v, want name star_align module align", task)
	}
	if commandHasSamtools(task.Command) {
		t.Fatalf("command = %#v, must not contain samtools", task.Command)
	}
	if !containsAll(task.Command, "--outFilterMultimapNmax", "1") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
	assertIOPath(t, task.Outputs, "bam", "work/star-align/Aligned.out.bam")
	assertGroupMembers(t, task.Inputs, "index", wantSTARGenomeMembers("work/star-genome"))
}

func TestSTARAlignNestedRun(t *testing.T) {
	requireDocker(t)
	srcFASTA := cachePin(t, PinWGSGenomeFASTA)
	srcR1 := cachePin(t, PinWGSTest1FASTQ)
	srcR2 := cachePin(t, PinWGSTest2FASTQ)
	dir := t.TempDir()
	stageFile(t, dir, "in/genome.fasta", srcFASTA)
	stageFile(t, dir, "in/test_1.fastq.gz", srcR1)
	stageFile(t, dir, "in/test_2.fastq.gz", srcR2)
	p := gobble.NewPipeline("rna")
	hf := p.AddInput("fasta", gobble.PathSpec{Dir: gobble.Dir("in"), Name: "genome", Ext: ".fasta"})
	h1 := p.AddInput("r1", gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_1", Ext: ".fastq.gz"})
	h2 := p.AddInput("r2", gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_2", Ext: ".fastq.gz"})
	idx := AddSTARGenomeGenerate(p, hf, STARGenomeGenerateOptions{
		ExtraArgs: []string{"--genomeSAindexNbases", "7"},
		Resources: gobble.Resources{CPU: 1},
	})
	AddSTARAlign(p, idx.Index, h1, h2, STARAlignOptions{Resources: gobble.Resources{CPU: 1}})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, filepath.FromSlash("work/star-align/Aligned.out.bam")))
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("published BAM: %v", err)
	}
}
