package wgsevidence_test

import (
	"crypto/sha256"
	"fmt"
	"testing"

	. "github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/internal/sourcecheck"
)

func TestWGSComposeBuildPlan(t *testing.T) {
	raw := pc.MustPlanJSON(t, Pipeline())
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != "fd762650d4fcfb4f14b862a67cc123777e98a3b2cd291b76196d94472295e2f1" {
		t.Fatalf("graph snapshot = %s, want pre-move WGS identity", got)
	}
	tasks := pc.AllTasks(t, raw)

	if got := pc.CountTasksNamed(tasks, "bwa_index"); got != 1 {
		t.Fatalf("bwa_index count = %d, want 1", got)
	}
	if got := pc.CountTasksNamed(tasks, "bwa_mem"); got != 2 {
		t.Fatalf("bwa_mem count = %d, want 2", got)
	}
	if got := pc.CountTasksNamed(tasks, "fastp"); got != 2 {
		t.Fatalf("fastp count = %d, want 2", got)
	}
	if got := pc.CountTasksNamed(tasks, "samtools_sort"); got != 2 {
		t.Fatalf("samtools_sort count = %d, want 2", got)
	}
	if got := pc.CountTasksNamed(tasks, "samtools_index"); got != 2 {
		t.Fatalf("samtools_index count = %d, want 2", got)
	}
	pc.MustHaveTaskID(t, tasks, "raw.fastqc")
	pc.MustHaveTaskID(t, tasks, "clean.fastqc")
	pc.MustHaveTaskID(t, tasks, "sample1.fastp")
	pc.MustHaveTaskID(t, tasks, "sample2.fastp")
	pc.MustHaveTaskID(t, tasks, "sample1.bwa_mem")
	pc.MustHaveTaskID(t, tasks, "sample2.bwa_mem")
	pc.MustHaveTaskID(t, tasks, "bwa_index")
	pc.MustHaveTaskID(t, tasks, "multiqc")

	rawQC := pc.TaskByID(t, raw, "raw.fastqc")
	if rawQC.Module != "raw" {
		t.Fatalf("raw.fastqc module = %q, want raw", rawQC.Module)
	}
	cleanQC := pc.TaskByID(t, raw, "clean.fastqc")
	if cleanQC.Module != "clean" {
		t.Fatalf("clean.fastqc module = %q, want clean", cleanQC.Module)
	}

	pc.AssertIOPath(t, pc.TaskByID(t, raw, "sample1.fastp").Inputs, "r1", "in/test_1.fastq.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "sample2.fastp").Inputs, "r1", "in/test_1.fastq.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Inputs, "report_0", "work/raw/fastqc/test_1_fastqc.html")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Inputs, "report_4", "work/sample1/fastp/test_1.fastp.json")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Inputs, "report_6", "work/sample2/fastp/test_1.fastp.json")

	pc.AssertNoTaskName(t, tasks, "star_align", "star_genome_generate", "bismark_align", "bismark_genome", "bismark_methylation_extractor")
}

func TestWGSOmitsRawAddTask(t *testing.T) {
	sourcecheck.AssertNoCall(t, "../../../assets/pipelines/wgs/wgs.go", "AddTask")
}

func TestWGSAddsPinnedFAI(t *testing.T) {
	raw := pc.MustPlanJSON(t, Pipeline())
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "bwa_index").Inputs, "fasta", "in/genome.fasta")
	sourcecheck.AssertCalls(t, "../../../assets/pipelines/wgs/wgs.go", "AddInput")
}
