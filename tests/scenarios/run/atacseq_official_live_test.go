//go:build live

package run_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/atacseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	atacseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/atacseq"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/atacseqscenario"
)

func TestOfficialFixtureRunsEverySelectedOperationWithoutDocker(t *testing.T) {
	workspace := t.TempDir()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	cache := filepath.Join(filepath.Dir(file), "..", "..", "pipelines", "atacseq", "testdata", "cache")
	staged, err := atacseqevidence.StageOfficial(cache, workspace)
	if err != nil {
		t.Fatalf("StageOfficial: %v", err)
	}
	productRuntime := atacseqscenario.NewRuntimeWithWorkspaceInputs(t, atacseq.DefaultConfig(), workspace)
	if err := productRuntime.Run(t.Context()); err != nil {
		t.Fatalf("Run exact official ATAC fixture: %v", err)
	}

	consumed := productRuntime.ConsumedInputs()
	samples, _ := atacseqscenario.Samples(t)
	raw := pc.MustPlanJSON(t, atacseq.Build(samples, atacseq.DefaultConfig()))
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
		"results/atacseq/consensus/replicates/featurecounts/consensus.featureCounts.txt",
		"results/atacseq/ataqv/html/fixture.txt",
		"results/atacseq/igv/igv_session.xml",
		"results/atacseq/multiqc/multiqc_report.html",
	} {
		if info, err := os.Stat(filepath.Join(workspace, filepath.FromSlash(rel))); err != nil || !info.Mode().IsRegular() {
			t.Errorf("official operation artifact %s: %v", rel, err)
		}
	}
}
