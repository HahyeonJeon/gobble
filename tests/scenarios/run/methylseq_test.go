package run_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/methylseqscenario"
)

func TestMethylRunCandidateDeclaresRequiredFinalArtifacts(t *testing.T) {
	raw := methylseqscenario.Plan(t, methylseq.DefaultConfig())
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "Ecoli_10K_methylated.bismark_deduplicate").Outputs, "deduplicated_bam", "results/methylseq/bismark/Ecoli_10K_methylated/Ecoli_10K_methylated_pe.deduplicated.bam")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "Ecoli_10K_methylated.bismark_methylation_extractor").Outputs, "cpg", "results/methylseq/methylation-calls/Ecoli_10K_methylated/CpG_context_Ecoli_10K_methylated_pe.deduplicated.txt.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Outputs, "html", "results/methylseq/multiqc/multiqc_report.html")
	if !methylseq.Lifecycle.Run {
		t.Fatal("Methyl run participation is false")
	}
}

func TestMethylRunExecutesOfficialGraphAndPublishesFinals(t *testing.T) {
	runtime := methylseqscenario.NewRuntime(t, methylseq.DefaultConfig())
	if err := runtime.Run(t.Context()); err != nil {
		t.Fatalf("Run(Methyl graph): %v\nerrors: %#v", err, runtime.InspectRecords(gobble.ViewErrors))
	}
	if remaining := runtime.InspectRecords(gobble.ViewRemaining); len(remaining) != 0 {
		t.Fatalf("remaining = %#v, want empty", remaining)
	}
	for _, rel := range []string{
		"results/methylseq/bismark/Ecoli_10K_methylated/Ecoli_10K_methylated_pe.deduplicated.bam",
		"results/methylseq/methylation-calls/Ecoli_10K_methylated/CpG_context_Ecoli_10K_methylated_pe.deduplicated.txt.gz",
		"results/methylseq/reports/Ecoli_10K_methylated/Ecoli_10K_methylated.bismark_report.html",
		"results/methylseq/summary/bismark_summary_report.html",
		"results/methylseq/multiqc/multiqc_report.html",
	} {
		info, err := os.Stat(filepath.Join(runtime.Workspace(), filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("published %s: %v", rel, err)
		}
	}
}
