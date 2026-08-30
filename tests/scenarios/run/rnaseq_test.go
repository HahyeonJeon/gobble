package run_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
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

func TestRNARunExecutesOfficialGraphAndPublishesFinals(t *testing.T) {
	runtime := rnaseqscenario.NewRuntime(t, rnaseq.DefaultConfig())
	if err := runtime.Run(t.Context()); err != nil {
		t.Fatalf("Run(RNA graph): %v\nerrors: %#v", err, runtime.InspectRecords(gobble.ViewErrors))
	}
	if remaining := runtime.InspectRecords(gobble.ViewRemaining); len(remaining) != 0 {
		t.Fatalf("remaining = %#v, want empty after RNA success", remaining)
	}
	for _, rel := range []string{
		"results/rnaseq/bam/WT_REP1/WT_REP1.marked.bam",
		"results/rnaseq/salmon/WT_REP1/quant.sf",
		"results/rnaseq/matrices/gene_counts.tsv",
		"results/rnaseq/multiqc/multiqc_report.html",
	} {
		info, err := os.Stat(filepath.Join(runtime.Workspace(), filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("published %s: %v", rel, err)
		}
	}
}
