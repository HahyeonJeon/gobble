package design

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/internal/sourcecheck"
)

func TestLinkedQCComposeBuildPlan(t *testing.T) {
	raw := pc.MustPlanJSON(t, LinkedQC())
	tasks := pc.AllTasks(t, raw)

	if got := pc.CountTasksNamed(tasks, "fastqc"); got != 2 {
		t.Fatalf("fastqc count = %d, want 2", got)
	}
	if got := pc.CountTasksNamed(tasks, "multiqc"); got != 1 {
		t.Fatalf("multiqc count = %d, want 1", got)
	}
	if len(tasks) != 3 {
		ids := make([]string, 0, len(tasks))
		for _, task := range tasks {
			ids = append(ids, task.ID)
		}
		t.Fatalf("task count = %d ids %v, want 3 (FastQC+MultiQC only)", len(tasks), ids)
	}

	pc.MustHaveTaskID(t, tasks, "rna.fastqc")
	pc.MustHaveTaskID(t, tasks, "methyl.fastqc")
	pc.MustHaveTaskID(t, tasks, "multiqc")

	pc.AssertIOPath(t, pc.TaskByID(t, raw, "rna.fastqc").Inputs, "reads", "in/SRR6357072_1.fastq.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "methyl.fastqc").Inputs, "reads", "in/Ecoli_10K_methylated_R1.fastq.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Inputs, "report_0", "work/rna/fastqc/SRR6357072_1_fastqc.html")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Inputs, "report_1", "work/rna/fastqc/SRR6357072_1_fastqc.zip")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Inputs, "report_2", "work/methyl/fastqc/Ecoli_10K_methylated_R1_fastqc.html")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Inputs, "report_3", "work/methyl/fastqc/Ecoli_10K_methylated_R1_fastqc.zip")

	pc.AssertNoTaskName(t, tasks,
		"fastp",
		"bwa_index", "bwa_mem",
		"samtools_sort", "samtools_index",
		"star_align", "star_genome_generate",
		"bismark_align", "bismark_genome", "bismark_methylation_extractor",
	)
}

func TestLinkedQCSharesPinPathSpecs(t *testing.T) {
	wgsPlan := pc.MustPlanJSON(t, wgs.Pipeline())
	prev := gobble.SampleSheetPath()
	t.Cleanup(func() { gobble.SetSampleSheetPath(prev) })
	gobble.SetSampleSheetPath("../../pipelines/rnaseq/testdata/rnaseq-samplesheet.csv")
	rnaPlan := pc.MustPlanJSON(t, rnaseq.Pipeline())
	gobble.SetSampleSheetPath("../../pipelines/methylseq/testdata/methylseq-samplesheet.csv")
	methylPlan := pc.MustPlanJSON(t, methylseq.Pipeline())
	qc := pc.MustPlanJSON(t, LinkedQC())

	pc.AssertIOPath(t, pc.TaskByID(t, wgsPlan, "sample1.fastp").Inputs, "r1", "in/test_1.fastq.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, wgsPlan, "sample1.fastp").Inputs, "r2", "in/test_2.fastq.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, rnaPlan, "ctrl1.fastp").Inputs, "r1", "in/SRR6357072_1.fastq.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, rnaPlan, "ctrl1.fastp").Inputs, "r2", "in/SRR6357072_2.fastq.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, methylPlan, "sample1.fastp").Inputs, "r1", "in/Ecoli_10K_methylated_R1.fastq.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, methylPlan, "sample1.fastp").Inputs, "r2", "in/Ecoli_10K_methylated_R2.fastq.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, qc, "rna.fastqc").Inputs, "reads", "in/SRR6357072_1.fastq.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, qc, "methyl.fastqc").Inputs, "reads", "in/Ecoli_10K_methylated_R1.fastq.gz")
}

func TestLinkedQCOmitsRawAddTask(t *testing.T) {
	sourcecheck.AssertNoCall(t, "linked_qc_fixture_test.go", "AddTask", "WGS", "RNASeq", "MethylSeq")
}
