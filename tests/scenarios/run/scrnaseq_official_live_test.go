//go:build live

package run_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/scrnaseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	scrnaseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/scrnaseq"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/scrnaseqscenario"
)

func TestOfficialSCRNAFixtureRunsEverySelectedOperationWithoutDocker(t *testing.T) {
	workspace := t.TempDir()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	cache := filepath.Join(filepath.Dir(file), "..", "..", "pipelines", "scrnaseq", "testdata", "cache")
	staged, err := scrnaseqevidence.StageOfficial(cache, workspace)
	if err != nil {
		t.Fatalf("StageOfficial: %v", err)
	}
	productRuntime := scrnaseqscenario.NewRuntimeWithWorkspaceInputs(t, scrnaseq.DefaultConfig(), workspace)
	if err := productRuntime.Run(t.Context()); err != nil {
		t.Fatalf("Run exact official scRNA fixture: %v", err)
	}

	consumed := productRuntime.ConsumedInputs()
	samples, _ := scrnaseqscenario.Samples(t)
	raw := pc.MustPlanJSON(t, scrnaseq.Build(samples, scrnaseq.DefaultConfig()))
	for _, task := range pc.AllTasks(t, raw) {
		if len(consumed[task.ID]) == 0 {
			t.Errorf("selected operation %s (%s) opened no declared input bytes", task.ID, task.Name)
		}
	}
	for _, input := range staged {
		matched := false
		for _, records := range consumed {
			for _, record := range records {
				if record.Path == input.Destination && record.SHA256 == input.Pin.SHA256 {
					matched = true
				}
			}
		}
		if !matched {
			t.Errorf("official identity %s %s at %s was not opened by a selected operation", input.Pin.Name, input.Pin.SHA256, input.Destination)
		}
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
