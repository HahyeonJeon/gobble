//go:build live

package run_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/scrnaseq"
	scrnaseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/scrnaseq"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/scrnaseqscenario"
)

func TestOfficialSCRNAFixtureRunsManifestOwnedProductWithoutImages(t *testing.T) {
	workspace := t.TempDir()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	cache := filepath.Join(filepath.Dir(file), "..", "..", "pipelines", "scrnaseq", "testdata", "cache")
	if _, err := scrnaseqevidence.StageOfficial(cache, workspace); err != nil {
		t.Fatalf("StageOfficial: %v", err)
	}
	productRuntime := scrnaseqscenario.NewRuntimeWithWorkspaceInputs(t, scrnaseq.DefaultConfig(), workspace)
	if err := productRuntime.Run(t.Context()); err != nil {
		t.Fatalf("Run exact official scRNA fixture: %v\nerrors: %#v", err, productRuntime.InspectRecords(gobble.ViewErrors))
	}
	for _, rel := range []string{
		"results/scrnaseq/samples/Sample_X/qcatch/filtered_quants.h5ad",
		"results/scrnaseq/matrices/Sample_X/Sample_X_raw_matrix.sce.rds",
		"results/scrnaseq/matrices/combined_raw_matrix.h5ad",
		"results/scrnaseq/multiqc/multiqc_report.html",
	} {
		if info, err := os.Stat(filepath.Join(workspace, filepath.FromSlash(rel))); err != nil || !info.Mode().IsRegular() {
			t.Errorf("official operation artifact %s: %v", rel, err)
		}
	}
}
