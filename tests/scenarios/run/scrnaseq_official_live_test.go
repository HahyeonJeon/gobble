//go:build live

package run_test

import (
	"bytes"
	"encoding/json"
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

func TestOfficialSCRNAFixtureProvesEverySelectedBindAndArgvWithoutImages(t *testing.T) {
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
	samples, _ := scrnaseqscenario.Samples(t)
	raw := pc.MustPlanJSON(t, scrnaseq.Build(samples, scrnaseq.DefaultConfig()))
	commands, err := scrnaseqscenario.ProveOfficialBindings(workspace, raw, officialInputs)
	if err != nil {
		t.Fatalf("prove exact official scRNA binds and argv: %v", err)
	}
	changedScript := bytes.Replace(raw, []byte("import scanpy as sc"), []byte("import scanpy as sx"), 1)
	if bytes.Equal(changedScript, raw) {
		t.Fatal("official scRNA plan omits expected frozen Scanpy script token")
	}
	if _, err := scrnaseqscenario.ProveOfficialBindings(workspace, changedScript, officialInputs); err == nil {
		t.Fatal("changed embedded scRNA script proof = nil, want frozen-oracle rejection")
	} else if !strings.Contains(err.Error(), "frozen oracle") {
		t.Fatalf("changed embedded script error = %q, want frozen-oracle rejection", err)
	}
	aliasedMate := bytes.ReplaceAll(
		raw,
		[]byte("in/reads/Sample_X_S1_L001_R1_001.fastq.gz"),
		[]byte("in/reads/Sample_Y_S1_L001_R1_001.fastq.gz"),
	)
	if bytes.Equal(aliasedMate, raw) {
		t.Fatal("official scRNA plan omits expected Sample_X mate path")
	}
	if _, err := scrnaseqscenario.ProveOfficialBindings(workspace, aliasedMate, officialInputs); err == nil {
		t.Fatal("same-SHA wrong-sample bind proof = nil, want independent identity-oracle rejection")
	} else if !strings.Contains(err.Error(), "independent exact SHA-256 identities") {
		t.Fatalf("same-SHA wrong-sample bind error = %q, want independent identity-oracle rejection", err)
	}
	rewiredPort := rewireOfficialBind(
		t,
		raw,
		"Sample_X.qcatch.quant",
		"Sample_X.simpleaf_quant.map",
	)
	if _, err := scrnaseqscenario.ProveOfficialBindings(workspace, rewiredPort, officialInputs); err == nil {
		t.Fatal("wrong producer-port bind proof = nil, want independent bind-oracle rejection")
	} else if !strings.Contains(err.Error(), "independent producer-port set") {
		t.Fatalf("wrong producer-port bind error = %q, want independent bind-oracle rejection", err)
	}
	selectedCommands := map[string]bool{
		"cat_fastq": true, "fastqc": true, "gtf_gene_filter": true,
		"gffread_transcriptome": true, "gtf_to_t2g": true,
		"simpleaf_index": true, "simpleaf_quant": true, "qcatch": true,
		"matrix_to_h5ad": true, "anndatar_convert": true,
		"h5ad_concat": true, "multiqc": true,
	}
	seenCommands := make(map[string]bool, len(selectedCommands))
	wantByPath := make(map[string]string, len(staged))
	boundOfficial := make(map[string]bool, len(staged))
	for _, input := range staged {
		wantByPath[input.Destination] = input.Pin.SHA256
	}
	for _, task := range pc.AllTasks(t, raw) {
		command, ok := commands[task.ID]
		if !ok {
			t.Errorf("selected task %s (%s) has no bind and argv evidence", task.ID, task.Name)
			continue
		}
		if command.TaskName != task.Name || len(command.Argv) == 0 || len(command.BoundInputs) != len(task.Inputs) {
			t.Errorf("selected task %s evidence = %+v, want complete argv and all %d input binds", task.ID, command, len(task.Inputs))
		}
		if !selectedCommands[task.Name] {
			t.Errorf("selected command %s uses unrecognized contract %q", task.ID, task.Name)
		}
		seenCommands[task.Name] = true
		for i, bound := range command.BoundInputs {
			if i >= len(task.Inputs) || bound.Name != task.Inputs[i].Name || len(bound.Paths) == 0 || len(bound.OfficialInputs) == 0 {
				t.Errorf("selected task %s bound input %d = %+v, want named path and official origins", task.ID, i, bound)
			}
			for _, input := range bound.OfficialInputs {
				if want := wantByPath[input.Path]; want == "" || input.SHA256 != want {
					t.Errorf("selected task %s bind %s official identity = %+v, want exact manifest SHA-256", task.ID, bound.Name, input)
				}
				boundOfficial[input.Path] = true
			}
		}
	}
	for command := range selectedCommands {
		if !seenCommands[command] {
			t.Errorf("official evidence omitted selected command contract %s", command)
		}
	}
	for _, input := range staged {
		if !boundOfficial[input.Destination] {
			t.Errorf("official identity %s %s at %s is absent from selected task binds", input.Pin.Name, input.Pin.SHA256, input.Destination)
		}
	}
	for _, endpoint := range []string{"cohort.h5ad_concat", "multiqc"} {
		origins := make(map[string]bool, len(staged))
		for _, bound := range commands[endpoint].BoundInputs {
			for _, input := range bound.OfficialInputs {
				origins[input.Path] = true
			}
		}
		if len(origins) != len(staged) {
			t.Errorf("%s exact staged bind origins = %d, want all %d", endpoint, len(origins), len(staged))
		}
	}

	// Run uses a hermetic Docker double after the independent static bind and
	// argv proof. Its outputs establish only occupancy and publication behavior.
	productRuntime := scrnaseqscenario.NewRuntimeWithWorkspaceInputs(t, scrnaseq.DefaultConfig(), workspace)
	if err := productRuntime.Run(t.Context()); err != nil {
		t.Fatalf("Run exact official scRNA occupancy evidence: %v\nerrors: %#v\nlogs: %#v", err, productRuntime.InspectRecords(gobble.ViewErrors), productRuntime.InspectRecords(gobble.ViewLogs))
	}

	// These deterministic files prove engine scheduling and publication only.
	// Selected-task proof above comes from frozen argv oracles and declared bind
	// lineage rooted at staged official hashes, never from placeholder contents.
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

func rewireOfficialBind(t *testing.T, raw []byte, to, from string) []byte {
	t.Helper()
	var plan struct {
		Tasks []pc.Task `json:"tasks"`
		DAG   struct {
			Edges []struct {
				From string `json:"from"`
				To   string `json:"to"`
			} `json:"edges"`
		} `json:"dag"`
	}
	if err := json.Unmarshal(raw, &plan); err != nil {
		t.Fatalf("decode official plan for adversarial bind: %v", err)
	}
	changed := false
	for i := range plan.DAG.Edges {
		if plan.DAG.Edges[i].To == to {
			plan.DAG.Edges[i].From = from
			changed = true
		}
	}
	if !changed {
		t.Fatalf("official plan has no bind endpoint %q", to)
	}
	encoded, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("encode adversarial official bind: %v", err)
	}
	return encoded
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
	samples, _ := scrnaseqscenario.Samples(t)
	raw := pc.MustPlanJSON(t, scrnaseq.Build(samples, scrnaseq.DefaultConfig()))
	if _, err := scrnaseqscenario.ProveOfficialBindings(workspace, raw, officialInputs); err == nil {
		t.Fatal("changed official scRNA bind proof = nil, want exact-byte rejection")
	} else if message := err.Error(); !strings.Contains(message, "sha256") || !strings.Contains(message, "Sample_Y_S1_L002_R1_001.fastq.gz") {
		t.Fatalf("changed-byte error = %q, want path-specific sha256 rejection", message)
	}
}
