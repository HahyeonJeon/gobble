package run_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/atacseq"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/atacseqscenario"
)

func TestATACRunCompletesRequiredArtifactsHermetically(t *testing.T) {
	runtime := atacseqscenario.NewRuntime(t, atacseq.DefaultConfig())
	if err := runtime.Run(t.Context()); err != nil {
		t.Fatalf("Run(ATAC graph): %#v", err)
	}
	for _, rel := range []string{
		"results/atacseq/samples/OSMOTIC_STRESS_T0_PE/replicate_1/alignment/OSMOTIC_STRESS_T0_PE_R1.filtered.bam",
		"results/atacseq/samples/OSMOTIC_STRESS_T0_PE/replicate_1/peaks/OSMOTIC_STRESS_T0_PE_R1_peaks.broadPeak",
		"results/atacseq/qc/peaks/replicates/macs2_peak.plots.pdf",
		"results/atacseq/qc/peaks/replicates/homer_annotation.plots.pdf",
		"results/atacseq/qc/peaks/replicates/homer_annotation.summary_mqc.tsv",
		"results/atacseq/consensus/replicates/consensus.bed",
		"results/atacseq/consensus/replicates/featurecounts/consensus.featureCounts.txt",
		"results/atacseq/igv/igv_session.xml",
		"results/atacseq/multiqc/multiqc_report.html",
	} {
		if info, err := os.Stat(filepath.Join(runtime.Workspace(), filepath.FromSlash(rel))); err != nil || !info.Mode().IsRegular() {
			t.Errorf("required artifact %s: %v", rel, err)
		}
	}
	if remaining := runtime.InspectRecords(gobble.ViewRemaining); len(remaining) != 0 || !atacseq.Lifecycle().Run {
		t.Fatalf("ATAC remaining = %#v, run participation %t", remaining, atacseq.Lifecycle().Run)
	}
}
