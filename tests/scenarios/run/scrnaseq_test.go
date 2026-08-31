package run_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/scrnaseq"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/scrnaseqscenario"
)

func TestSCRNARunCompletesRequiredArtifactsHermetically(t *testing.T) {
	runtime := scrnaseqscenario.NewRuntime(t, scrnaseq.DefaultConfig())
	if err := runtime.Run(t.Context()); err != nil {
		t.Fatalf("Run(scRNA graph): %v", err)
	}
	for _, rel := range []string{
		"results/scrnaseq/reference/simpleaf_index/index/.gobble-tree.json",
		"results/scrnaseq/samples/Sample_X/simpleaf/af_map/.gobble-tree.json",
		"results/scrnaseq/samples/Sample_X/simpleaf/af_quant/.gobble-tree.json",
		"results/scrnaseq/samples/Sample_X/qcatch/QCatch_report.html",
		"results/scrnaseq/samples/Sample_X/qcatch/filtered_quants.h5ad",
		"results/scrnaseq/matrices/Sample_X/Sample_X_raw_matrix.h5ad",
		"results/scrnaseq/matrices/Sample_X/Sample_X_raw_matrix.seurat.rds",
		"results/scrnaseq/matrices/combined_raw_matrix.h5ad",
		"results/scrnaseq/multiqc/multiqc_report.html",
	} {
		if info, err := os.Stat(filepath.Join(runtime.Workspace(), filepath.FromSlash(rel))); err != nil || !info.Mode().IsRegular() {
			t.Errorf("required artifact %s: %v", rel, err)
		}
	}
	if remaining := runtime.InspectRecords(gobble.ViewRemaining); len(remaining) != 0 || !scrnaseq.Lifecycle.Run {
		t.Fatalf("scRNA remaining = %#v, run participation %t", remaining, scrnaseq.Lifecycle.Run)
	}
}
