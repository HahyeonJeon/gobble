package assets

import "testing"

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

	assertIOPath(t, planTask(t, raw, "fastp").Inputs, "r1", "in/test_1.fastq.gz")
	assertIOPath(t, planTask(t, raw, "star_genome_generate").Inputs, "fasta", "in/genome.fasta")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_0", "work/raw/fastqc/test_1_fastqc.html")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_4", "work/fastp/test_1.fastp.json")

	assertNoTaskName(t, tasks, "bwa_index", "bwa_mem", "bismark_align", "bismark_genome", "bismark_methylation_extractor")
}

func TestRNASeqOmitsRawAddTask(t *testing.T) {
	assertNoCall(t, "rnaseq.go", "AddTask")
}
