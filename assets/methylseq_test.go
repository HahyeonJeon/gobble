package assets

import "testing"

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

	assertIOPath(t, planTask(t, raw, "fastp").Inputs, "r1", "in/test_1.fastq.gz")
	assertIOPath(t, planTask(t, raw, "bismark_genome").Inputs, "fasta", "in/genome.fasta")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_0", "work/raw/fastqc/test_1_fastqc.html")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_4", "work/fastp/test_1.fastp.json")

	assertNoTaskName(t, tasks, "bwa_index", "bwa_mem", "star_align", "star_genome_generate")
}

func TestMethylSeqOmitsRawAddTask(t *testing.T) {
	assertNoCall(t, "methylseq.go", "AddTask")
}
