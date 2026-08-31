//go:build live

package run_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/scrnaseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	scrnaseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/scrnaseq"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/scrnaseqscenario"
)

func TestOfficialSCRNAFixtureProvesEverySelectedCommandWithoutImages(t *testing.T) {
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
	officialInputs := make([]scrnaseqscenario.OfficialInput, 0, len(staged))
	for _, input := range staged {
		officialInputs = append(officialInputs, scrnaseqscenario.OfficialInput{Name: input.Pin.Name, Path: input.Destination, SHA256: input.Pin.SHA256})
	}
	productRuntime := scrnaseqscenario.NewRuntimeWithOfficialInputs(t, scrnaseq.DefaultConfig(), workspace, officialInputs)
	if err := productRuntime.Run(t.Context()); err != nil {
		t.Fatalf("Run exact official scRNA command evidence: %v\nerrors: %#v\nlogs: %#v", err, productRuntime.InspectRecords(gobble.ViewErrors), productRuntime.InspectRecords(gobble.ViewLogs))
	}

	operations := productRuntime.OperationEvidence()
	samples, _ := scrnaseqscenario.Samples(t)
	raw := pc.MustPlanJSON(t, scrnaseq.Build(samples, scrnaseq.DefaultConfig()))
	selectedCommands := map[string]bool{
		"cat_fastq": true, "fastqc": true, "gtf_gene_filter": true,
		"gffread_transcriptome": true, "gtf_to_t2g": true,
		"simpleaf_index": true, "simpleaf_quant": true, "qcatch": true,
		"matrix_to_h5ad": true, "anndatar_convert": true,
		"h5ad_concat": true, "multiqc": true,
	}
	seenCommands := make(map[string]bool, len(selectedCommands))
	wantByPath := make(map[string]string, len(staged))
	consumedOfficial := make(map[string]bool, len(staged))
	for _, input := range staged {
		wantByPath[input.Destination] = input.Pin.SHA256
	}
	for _, task := range pc.AllTasks(t, raw) {
		operation, ok := operations[task.ID]
		if !ok {
			t.Errorf("selected command %s (%s) has no command-specific evidence", task.ID, task.Name)
			continue
		}
		if operation.TaskName != task.Name || len(operation.Argv) == 0 || len(operation.StagedInputs) == 0 {
			t.Errorf("selected command %s evidence = %+v, want exact argv and staged bytes", task.ID, operation)
		}
		if !selectedCommands[task.Name] {
			t.Errorf("selected command %s uses unrecognized contract %q", task.ID, task.Name)
		}
		seenCommands[task.Name] = true
		for _, input := range operation.StagedInputs {
			if want := wantByPath[input.Path]; want == "" || input.SHA256 != want {
				t.Errorf("selected command %s staged input = %+v, want exact manifest identity", task.ID, input)
			}
			consumedOfficial[input.Path] = true
		}
	}
	for command := range selectedCommands {
		if !seenCommands[command] {
			t.Errorf("official evidence omitted selected command contract %s", command)
		}
	}
	for _, input := range staged {
		if !consumedOfficial[input.Destination] {
			t.Errorf("official identity %s %s at %s was not consumed by selected command evidence", input.Pin.Name, input.Pin.SHA256, input.Destination)
		}
	}
	for _, endpoint := range []string{"cohort.h5ad_concat", "multiqc"} {
		operation := operations[endpoint]
		if len(operation.StagedInputs) != len(staged) {
			t.Errorf("%s exact staged origins = %d, want all %d", endpoint, len(operation.StagedInputs), len(staged))
		}
	}

	// These deterministic files prove engine scheduling and publication only.
	// Selected-command proof above comes from validated argv and separately
	// re-opened official bytes, never from placeholder contents.
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

func TestOfficialSCRNACommandEvidenceRejectsChangedStagedByte(t *testing.T) {
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
	officialInputs := make([]scrnaseqscenario.OfficialInput, 0, len(staged))
	for _, input := range staged {
		officialInputs = append(officialInputs, scrnaseqscenario.OfficialInput{Name: input.Pin.Name, Path: input.Destination, SHA256: input.Pin.SHA256})
	}
	productRuntime := scrnaseqscenario.NewRuntimeWithOfficialInputs(t, scrnaseq.DefaultConfig(), workspace, officialInputs)
	changed := filepath.Join(workspace, filepath.FromSlash("in/reads/Sample_Y_S1_L002_R1_001.fastq.gz"))
	fileHandle, err := os.OpenFile(changed, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fileHandle.Write([]byte("changed")); err != nil {
		_ = fileHandle.Close()
		t.Fatal(err)
	}
	if err := fileHandle.Close(); err != nil {
		t.Fatal(err)
	}
	if err := productRuntime.Run(t.Context()); err == nil {
		t.Fatal("Run changed official scRNA fixture = nil, want exact-byte rejection")
	}
	errorsView := productRuntime.InspectRecords(gobble.ViewErrors)
	if rendered := fmt.Sprint(errorsView); !strings.Contains(rendered, "sha256") || !strings.Contains(rendered, "Sample_Y_S1_L002_R1_001.fastq.gz") {
		t.Fatalf("changed-byte errors = %#v, want path-specific sha256 rejection", errorsView)
	}
}
