package assets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestBismarkMethylationExtractorStandaloneComposeBuildPlan(t *testing.T) {
	bam := gobble.PathSpec{Dir: gobble.Dir("work/bismark-align"), Name: "aligned_pe", Ext: ".bam"}
	opts := BismarkMethylationExtractorOptions{
		ExtraArgs: []string{"--no_overlap"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := BismarkMethylationExtractorPipeline(bam, opts)
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "bismark_methylation_extractor")
	if task.Name != "bismark_methylation_extractor" {
		t.Fatalf("task name = %q, want bismark_methylation_extractor", task.Name)
	}
	if task.Image != bismarkImage {
		t.Fatalf("image = %q, want %q", task.Image, bismarkImage)
	}
	if containsAll(task.Command, "--multicore") || containsAll(task.Command, "--parallel") {
		t.Fatalf("command = %#v, extractor must not copy Resources.CPU", task.Command)
	}
	if !containsAll(task.Command,
		"bismark_methylation_extractor",
		"--bedGraph", "--counts", "--gzip", "--report", "--comprehensive", "-p",
		"--output_dir", "work/bismark-extractor",
		"--no_overlap",
		"work/bismark-align/aligned_pe.bam",
	) {
		t.Fatalf("command = %#v, want named flags then extra-args then BAM", task.Command)
	}
	n := len(task.Command)
	if n < 2 || task.Command[n-2] != "--no_overlap" || task.Command[n-1] != "work/bismark-align/aligned_pe.bam" {
		t.Fatalf("command tail = %#v, want [--no_overlap work/bismark-align/aligned_pe.bam]", task.Command)
	}
	assertUniqueParamNames(t, task.Params)
	assertIOPath(t, task.Outputs, "bedgraph", "work/bismark-extractor/aligned_pe.bedGraph.gz")
	assertIOPath(t, task.Outputs, "coverage", "work/bismark-extractor/aligned_pe.bismark.cov.gz")
	assertIOPath(t, task.Outputs, "report", "work/bismark-extractor/aligned_pe_splitting_report.txt")
	assertIOPath(t, task.Outputs, "mbias", "work/bismark-extractor/aligned_pe.M-bias.txt")
	assertIOPath(t, task.Outputs, "cpg", "work/bismark-extractor/CpG_context_aligned_pe.txt.gz")
}

func TestBismarkMethylationExtractorNestedModule(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "genome", Ext: ".fasta"}
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_2", Ext: ".fastq.gz"}
	p := gobble.NewPipeline("assay")
	hf := p.AddInput("fasta", fasta)
	h1 := p.AddInput("r1", r1)
	h2 := p.AddInput("r2", r2)
	mod := AddModule(p, "methyl")
	idx := AddBismarkGenome(mod, hf, BismarkGenomeOptions{})
	aln := AddBismarkAlign(mod, hf, idx.Index, h1, h2, BismarkAlignOptions{})
	ports := AddBismarkMethylationExtractor(mod, aln.BAM, BismarkMethylationExtractorOptions{ExtraArgs: []string{"--no_overlap"}})
	if ports.BedGraph.IsZero() || ports.Coverage.IsZero() || ports.Report.IsZero() || ports.Mbias.IsZero() || ports.CpG.IsZero() {
		t.Fatalf("ports IsZero, want false")
	}
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "methyl.bismark_methylation_extractor")
	if task.Name != "bismark_methylation_extractor" || task.Module != "methyl" {
		t.Fatalf("nested task = %+v, want name bismark_methylation_extractor module methyl", task)
	}
	if !containsAll(task.Command, "--no_overlap") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
	assertIOPath(t, task.Outputs, "bedgraph", "work/bismark-extractor/aligned_pe.bedGraph.gz")
	assertIOPath(t, task.Outputs, "cpg", "work/bismark-extractor/CpG_context_aligned_pe.txt.gz")
}

func TestBismarkMethylationExtractorNestedRun(t *testing.T) {
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
	aln := AddBismarkAlign(p, hf, idx.Index, h1, h2, BismarkAlignOptions{Resources: gobble.Resources{CPU: 1}})
	AddBismarkMethylationExtractor(p, aln.BAM, BismarkMethylationExtractorOptions{})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, rel := range []string{
		"work/bismark-extractor/aligned_pe.bedGraph.gz",
		"work/bismark-extractor/aligned_pe.bismark.cov.gz",
		"work/bismark-extractor/aligned_pe_splitting_report.txt",
		"work/bismark-extractor/aligned_pe.M-bias.txt",
		"work/bismark-extractor/CpG_context_aligned_pe.txt.gz",
	} {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("published %s: %v", rel, err)
		}
	}
}
