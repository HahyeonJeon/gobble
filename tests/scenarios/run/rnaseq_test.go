package run_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/rnaseqscenario"
)

func TestRNARunCandidateDeclaresRequiredFinalArtifacts(t *testing.T) {
	raw := rnaseqscenario.Plan(t, rnaseq.DefaultConfig())
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "WT_REP1.picard_markduplicates").Outputs, "marked_bam", "results/rnaseq/bam/WT_REP1/WT_REP1.marked.bam")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "cohort.tximport").Outputs, "gene_counts", "results/rnaseq/matrices/gene_counts.tsv")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Outputs, "html", "results/rnaseq/multiqc/multiqc_report.html")
	if !rnaseq.Lifecycle.Run {
		t.Fatal("RNA run participation is false")
	}
}
