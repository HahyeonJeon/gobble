package assets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestBismarkAlignStandaloneComposeBuildPlan(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "genome", Ext: ".fasta"}
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_2", Ext: ".fastq.gz"}
	opts := BismarkAlignOptions{
		ExtraArgs: []string{"--quiet"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := BismarkAlignPipeline(fasta, r1, r2, opts)
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "bismark_align")
	if task.Name != "bismark_align" {
		t.Fatalf("task name = %q, want bismark_align", task.Name)
	}
	if task.Image != bismarkImage {
		t.Fatalf("image = %q, want %q", task.Image, bismarkImage)
	}
	if commandHasSamtools(task.Command) {
		t.Fatalf("command = %#v, must not contain samtools", task.Command)
	}
	if !containsAll(task.Command,
		"bismark",
		"--genome", "in",
		"--bam",
		"--output_dir", "work/bismark-align",
		"--basename", "aligned",
		"-p", "2",
		"-1", "in/test_1.fastq.gz",
		"-2", "in/test_2.fastq.gz",
		"--quiet",
	) {
		t.Fatalf("command = %#v, want named flags then extra-args", task.Command)
	}
	n := len(task.Command)
	if n < 1 || task.Command[n-1] != "--quiet" {
		t.Fatalf("command tail = %#v, want [--quiet]", task.Command)
	}
	assertUniqueParamNames(t, task.Params)
	assertIOPath(t, task.Inputs, "fasta", "in/genome.fasta")
	assertIOPath(t, task.Outputs, "bam", "work/bismark-align/aligned_pe.bam")
	assertGroupMembers(t, task.Inputs, "index", wantBismarkGenomeMembers("in"))
}

func TestBismarkAlignNestedModule(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "genome", Ext: ".fasta"}
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_2", Ext: ".fastq.gz"}
	p := gobble.NewPipeline("assay")
	hf := p.AddInput("fasta", fasta)
	h1 := p.AddInput("r1", r1)
	h2 := p.AddInput("r2", r2)
	mod := AddModule(p, "align")
	idx := AddBismarkGenome(mod, hf, BismarkGenomeOptions{ExtraArgs: []string{"--verbose"}})
	ports := AddBismarkAlign(mod, hf, idx.Index, h1, h2, BismarkAlignOptions{ExtraArgs: []string{"--quiet"}})
	if ports.BAM.IsZero() {
		t.Fatalf("ports.BAM IsZero = true, want false")
	}
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "align.bismark_align")
	if task.Name != "bismark_align" || task.Module != "align" {
		t.Fatalf("nested task = %+v, want name bismark_align module align", task)
	}
	if commandHasSamtools(task.Command) {
		t.Fatalf("command = %#v, must not contain samtools", task.Command)
	}
	if !containsAll(task.Command, "--quiet") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
	assertIOPath(t, task.Outputs, "bam", "work/bismark-align/aligned_pe.bam")
	assertGroupMembers(t, task.Inputs, "index", wantBismarkGenomeMembers("in"))
}

func TestBismarkAlignNestedRun(t *testing.T) {
	requireDocker(t)
	srcFASTA := cachePin(t, PinWGSGenomeFASTA)
	srcR1 := cachePin(t, PinWGSTest1FASTQ)
	srcR2 := cachePin(t, PinWGSTest2FASTQ)
	dir := t.TempDir()
	stageFile(t, dir, "in/genome.fasta", srcFASTA)
	stageFile(t, dir, "in/test_1.fastq.gz", srcR1)
	stageFile(t, dir, "in/test_2.fastq.gz", srcR2)
	p := gobble.NewPipeline("methyl")
	hf := p.AddInput("fasta", gobble.PathSpec{Dir: gobble.Dir("in"), Name: "genome", Ext: ".fasta"})
	h1 := p.AddInput("r1", gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_1", Ext: ".fastq.gz"})
	h2 := p.AddInput("r2", gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_2", Ext: ".fastq.gz"})
	idx := AddBismarkGenome(p, hf, BismarkGenomeOptions{Resources: gobble.Resources{CPU: 1}})
	AddBismarkAlign(p, hf, idx.Index, h1, h2, BismarkAlignOptions{Resources: gobble.Resources{CPU: 1}})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, filepath.FromSlash("work/bismark-align/aligned_pe.bam")))
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("published BAM: %v", err)
	}
}
