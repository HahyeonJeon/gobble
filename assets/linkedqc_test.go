package assets

import "testing"

func TestLinkedQCComposeBuildPlan(t *testing.T) {
	raw := mustPlanJSON(t, LinkedQC())
	tasks := planAllTasks(t, raw)

	if got := countTasksNamed(tasks, "fastqc"); got != 2 {
		t.Fatalf("fastqc count = %d, want 2", got)
	}
	if got := countTasksNamed(tasks, "multiqc"); got != 1 {
		t.Fatalf("multiqc count = %d, want 1", got)
	}
	if len(tasks) != 3 {
		ids := make([]string, 0, len(tasks))
		for _, task := range tasks {
			ids = append(ids, task.ID)
		}
		t.Fatalf("task count = %d ids %v, want 3 (FastQC+MultiQC only)", len(tasks), ids)
	}

	mustHaveTaskID(t, tasks, "r1.fastqc")
	mustHaveTaskID(t, tasks, "r2.fastqc")
	mustHaveTaskID(t, tasks, "multiqc")

	assertIOPath(t, planTask(t, raw, "r1.fastqc").Inputs, "reads", "in/test_1.fastq.gz")
	assertIOPath(t, planTask(t, raw, "r2.fastqc").Inputs, "reads", "in/test_2.fastq.gz")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_0", "work/r1/fastqc/test_1_fastqc.html")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_1", "work/r1/fastqc/test_1_fastqc.zip")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_2", "work/r2/fastqc/test_2_fastqc.html")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_3", "work/r2/fastqc/test_2_fastqc.zip")

	assertNoTaskName(t, tasks,
		"fastp",
		"bwa_index", "bwa_mem",
		"samtools_sort", "samtools_index",
		"star_align", "star_genome_generate",
		"bismark_align", "bismark_genome", "bismark_methylation_extractor",
	)
}

func TestLinkedQCSharesPinPathSpecs(t *testing.T) {
	wgs := mustPlanJSON(t, WGS())
	rna := mustPlanJSON(t, RNASeq())
	methyl := mustPlanJSON(t, MethylSeq())
	qc := mustPlanJSON(t, LinkedQC())

	assertIOPath(t, planTask(t, wgs, "sample1.fastp").Inputs, "r1", "in/test_1.fastq.gz")
	assertIOPath(t, planTask(t, wgs, "sample1.fastp").Inputs, "r2", "in/test_2.fastq.gz")
	assertIOPath(t, planTask(t, rna, "fastp").Inputs, "r1", "in/test_1.fastq.gz")
	assertIOPath(t, planTask(t, rna, "fastp").Inputs, "r2", "in/test_2.fastq.gz")
	assertIOPath(t, planTask(t, methyl, "fastp").Inputs, "r1", "in/test_1.fastq.gz")
	assertIOPath(t, planTask(t, methyl, "fastp").Inputs, "r2", "in/test_2.fastq.gz")
	assertIOPath(t, planTask(t, qc, "r1.fastqc").Inputs, "reads", "in/test_1.fastq.gz")
	assertIOPath(t, planTask(t, qc, "r2.fastqc").Inputs, "reads", "in/test_2.fastq.gz")
}

func TestLinkedQCOmitsRawAddTask(t *testing.T) {
	assertNoCall(t, "linkedqc.go", "AddTask", "WGS", "RNASeq", "MethylSeq")
}
