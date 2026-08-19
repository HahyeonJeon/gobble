package assets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestRNASeqComposeBuildPlan(t *testing.T) {
	raw := mustPlanJSON(t, RNASeq())
	tasks := planAllTasks(t, raw)

	mustHaveTaskID(t, tasks, "raw.fastqc")
	mustHaveTaskID(t, tasks, "clean.fastqc")
	mustHaveTaskID(t, tasks, "fastp")
	mustHaveTaskID(t, tasks, "star_genome_generate")
	mustHaveTaskID(t, tasks, "star_align")
	mustHaveTaskID(t, tasks, "samtools_sort")
	mustHaveTaskID(t, tasks, "samtools_index")
	mustHaveTaskID(t, tasks, "multiqc")

	if got := countTasksNamed(tasks, "star_genome_generate"); got != 1 {
		t.Fatalf("star_genome_generate count = %d, want 1", got)
	}
	if got := countTasksNamed(tasks, "star_align"); got != 1 {
		t.Fatalf("star_align count = %d, want 1", got)
	}

	rawQC := planTask(t, raw, "raw.fastqc")
	if rawQC.Module != "raw" {
		t.Fatalf("raw.fastqc module = %q, want raw", rawQC.Module)
	}
	cleanQC := planTask(t, raw, "clean.fastqc")
	if cleanQC.Module != "clean" {
		t.Fatalf("clean.fastqc module = %q, want clean", cleanQC.Module)
	}

	gg := planTask(t, raw, "star_genome_generate")
	if !containsAll(gg.Command,
		"--sjdbGTFfile", "in/genes.gtf",
		"--runThreadN", "2",
		"--genomeSAindexNbases", "7",
		"--sjdbOverhang", "100",
	) {
		t.Fatalf("genomeGenerate command = %#v, want GTF, threads, and extra-args", gg.Command)
	}
	assertIOPath(t, gg.Inputs, "fasta", "in/genome.fasta")
	assertIOPath(t, gg.Inputs, "gtf", "in/genes.gtf")
	assertGroupMembers(t, gg.Outputs, "index", wantSTARGenomeSJDBMembers("work/star-genome"))

	align := planTask(t, raw, "star_align")
	if !containsAll(align.Command, "--runThreadN", "2") {
		t.Fatalf("align command = %#v, want --runThreadN 2", align.Command)
	}
	assertIOPath(t, align.Outputs, "bam", "work/star-align/Aligned.out.bam")
	assertIOPath(t, align.Outputs, "log_final", "work/star-align/Log.final.out")
	assertGroupMembers(t, align.Inputs, "index", wantSTARGenomeSJDBMembers("work/star-genome"))

	assertIOPath(t, planTask(t, raw, "fastp").Inputs, "r1", "in/SRR6357072_1.fastq.gz")
	assertIOPath(t, planTask(t, raw, "fastp").Inputs, "r2", "in/SRR6357072_2.fastq.gz")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_0", "work/raw/fastqc/SRR6357072_1_fastqc.html")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_4", "work/fastp/SRR6357072_1.fastp.json")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_6", "work/star-align/Aligned.out.bam")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_7", "work/star-align/Log.final.out")

	assertNoTaskName(t, tasks, "bwa_index", "bwa_mem", "bismark_align", "bismark_genome", "bismark_methylation_extractor")
}

func TestRNASeqOmitsRawAddTask(t *testing.T) {
	assertNoCall(t, "rnaseq.go", "AddTask")
}

func TestRNASeqRun(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	stageFile(t, dir, "in/genome.fasta", cachePin(t, PinRNAGenomeFASTA))
	stageFile(t, dir, "in/genes.gtf", cachePin(t, PinRNAGTF))
	stageFile(t, dir, "in/SRR6357072_1.fastq.gz", cachePin(t, PinRNATest1FASTQ))
	stageFile(t, dir, "in/SRR6357072_2.fastq.gz", cachePin(t, PinRNATest2FASTQ))
	g, err := gobble.Compose(RNASeq())
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(g, dir, 2); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	logPath := filepath.Join(dir, filepath.FromSlash("work/star-align/Log.final.out"))
	assertUniquelyMappedAbove(t, logPath, 10)
	for _, rel := range []string{
		"work/star-align/Aligned.out.bam",
		"work/multiqc/multiqc_report.html",
	} {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("published %s: %v", rel, err)
		}
	}
}
