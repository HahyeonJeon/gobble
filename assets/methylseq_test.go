package assets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestMethylSeqComposeBuildPlan(t *testing.T) {
	raw := mustPlanJSON(t, MethylSeq())
	tasks := planAllTasks(t, raw)

	mustHaveTaskID(t, tasks, "raw.fastqc")
	mustHaveTaskID(t, tasks, "clean.fastqc")
	mustHaveTaskID(t, tasks, "fastp")
	mustHaveTaskID(t, tasks, "bismark_genome")
	mustHaveTaskID(t, tasks, "bismark_align")
	mustHaveTaskID(t, tasks, "bismark_methylation_extractor")
	mustHaveTaskID(t, tasks, "multiqc")

	if got := countTasksNamed(tasks, "bismark_genome"); got != 1 {
		t.Fatalf("bismark_genome count = %d, want 1", got)
	}
	if got := countTasksNamed(tasks, "bismark_align"); got != 1 {
		t.Fatalf("bismark_align count = %d, want 1", got)
	}

	rawQC := planTask(t, raw, "raw.fastqc")
	if rawQC.Module != "raw" {
		t.Fatalf("raw.fastqc module = %q, want raw", rawQC.Module)
	}
	cleanQC := planTask(t, raw, "clean.fastqc")
	if cleanQC.Module != "clean" {
		t.Fatalf("clean.fastqc module = %q, want clean", cleanQC.Module)
	}

	genome := planTask(t, raw, "bismark_genome")
	if genome.Image != wantBismarkImage {
		t.Fatalf("bismark_genome image = %q, want %q", genome.Image, wantBismarkImage)
	}
	if !containsAll(genome.Command, "work/bismark-genome") {
		t.Fatalf("bismark_genome command = %#v, want restaged dest", genome.Command)
	}
	assertIOPath(t, genome.Inputs, "fasta", "work/bismark-genome/genome.fa")
	assertIOSource(t, genome.Inputs, "fasta", "in/genome.fa")
	assertGroupMembers(t, genome.Outputs, "index", wantBismarkGenomeMembers("work/bismark-genome"))

	align := planTask(t, raw, "bismark_align")
	if align.Image != wantBismarkImage {
		t.Fatalf("bismark_align image = %q, want %q", align.Image, wantBismarkImage)
	}
	if !containsAll(align.Command, "--genome", "work/bismark-genome", "--output_dir", "work/bismark-align") {
		t.Fatalf("bismark_align command = %#v, want restaged genome and --output_dir", align.Command)
	}
	assertIOPath(t, align.Outputs, "report", "work/bismark-align/aligned_PE_report.txt")
	assertGroupMembers(t, align.Inputs, "index", wantBismarkGenomeMembers("work/bismark-genome"))

	extractor := planTask(t, raw, "bismark_methylation_extractor")
	if extractor.Image != wantBismarkImage {
		t.Fatalf("extractor image = %q, want %q", extractor.Image, wantBismarkImage)
	}
	if !containsAll(extractor.Command, "--output_dir", "work/bismark-extractor", "-p") {
		t.Fatalf("extractor command = %#v, want --output_dir and paired-end -p", extractor.Command)
	}

	assertIOPath(t, planTask(t, raw, "fastp").Inputs, "r1", "in/Ecoli_10K_methylated_R1.fastq.gz")
	assertIOPath(t, planTask(t, raw, "fastp").Inputs, "r2", "in/Ecoli_10K_methylated_R2.fastq.gz")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_0", "work/raw/fastqc/Ecoli_10K_methylated_R1_fastqc.html")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_4", "work/fastp/Ecoli_10K_methylated_R1.fastp.json")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_6", "work/bismark-align/aligned_PE_report.txt")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_7", "work/bismark-extractor/aligned_pe_splitting_report.txt")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_8", "work/bismark-extractor/aligned_pe.M-bias.txt")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_9", "work/bismark-extractor/aligned_pe.bismark.cov.gz")

	assertNoTaskName(t, tasks, "bwa_index", "bwa_mem", "star_align", "star_genome_generate", "index_files")
}

func TestMethylSeqCPUFlagsCompose(t *testing.T) {
	p := gobble.NewPipeline("methyl-cpu")
	hf := p.AddInput("fasta", pinnedMethylFASTA())
	h1 := p.AddInput("r1", pinnedMethylFASTQ1())
	h2 := p.AddInput("r2", pinnedMethylFASTQ2())
	idx := AddBismarkGenome(p, hf, BismarkGenomeOptions{Resources: gobble.Resources{CPU: 2}})
	AddBismarkAlign(p, hf, idx.Index, h1, h2, BismarkAlignOptions{Resources: gobble.Resources{CPU: 2}})
	raw := mustPlanJSON(t, p)
	genome := planTask(t, raw, "bismark_genome")
	if !containsAll(genome.Command, "--parallel", "2") {
		t.Fatalf("bismark_genome command = %#v, want --parallel 2", genome.Command)
	}
	align := planTask(t, raw, "bismark_align")
	if !containsAll(align.Command, "-p", "2") {
		t.Fatalf("bismark_align command = %#v, want -p 2", align.Command)
	}
}

func TestMethylSeqRun(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	stageMethylPins(t, dir)
	g, err := gobble.Compose(MethylSeq())
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash("in/Bisulfite_Genome"))); !os.IsNotExist(err) {
		t.Fatalf("Bisulfite_Genome written into in/: %v", err)
	}
	unique := uniquePEAlignments(t, filepath.Join(dir, filepath.FromSlash("work/bismark-align/aligned_PE_report.txt")))
	t.Logf("unique paired-end alignments = %d", unique)
	assertUniqueAlignmentFloor(t, unique)
	assertMethylationCallRows(t, unique,
		filepath.Join(dir, filepath.FromSlash("work/bismark-extractor/CpG_context_aligned_pe.txt.gz")),
		filepath.Join(dir, filepath.FromSlash("work/bismark-extractor/aligned_pe.bismark.cov.gz")),
	)
}

func TestMethylSeqOmitsRawAddTask(t *testing.T) {
	assertNoCall(t, "methylseq.go", "AddTask")
}
