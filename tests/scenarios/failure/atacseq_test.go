package failure_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/atacseq"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/atacseqscenario"
)

func TestATACInputFailureIsContainedInspectableAndRecoverable(t *testing.T) {
	runtime := atacseqscenario.NewRuntime(t, atacseq.DefaultConfig())
	const (
		failed     = "OSMOTIC_STRESS_T0_PE.replicate_1.peaks.macs2_callpeak"
		upstream   = "OSMOTIC_STRESS_T0_PE.replicate_1.run_001.bwa_mem"
		downstream = "consensus.replicates.featurecounts_merge_matrices"
	)
	runtime.FailInput(failed)
	err := runtime.Run(t.Context())
	var structured *gobble.Error
	if !errors.As(err, &structured) || !hasATACDefectUnit(structured, failed) || !atacseq.Lifecycle.Failure {
		t.Fatalf("Run input failure = %v, want contained ATAC task failure", err)
	}
	errorsView := runtime.InspectRecords(gobble.ViewErrors)
	logsView := runtime.InspectRecords(gobble.ViewLogs)
	remaining := runtime.InspectRecords(gobble.ViewRemaining)
	if !recordsContainATACValue(errorsView, failed) || !recordsContainATACValue(logsView, failed) || !recordsContainATACValue(logsView, "simulated ATAC input rejection for "+failed+"\n") {
		t.Fatalf("ATAC errors/logs omit failed input unit: errors=%#v logs=%#v", errorsView, logsView)
	}
	if !recordsContainATACIdentity(remaining, failed) || !recordsContainATACIdentity(remaining, downstream) || !recordsContainATACIdentity(remaining, "multiqc") {
		t.Fatalf("ATAC failure remaining work does not block failed descendants: %#v", remaining)
	}
	reusable := filepath.Join(runtime.Workspace(), "results", "atacseq", "samples", "OSMOTIC_STRESS_T0_PE", "replicate_1", "alignment", "OSMOTIC_STRESS_T0_PE_R1.filtered.bam")
	if info, statErr := os.Stat(reusable); statErr != nil || !info.Mode().IsRegular() {
		t.Fatalf("reusable ATAC upstream output %s: %v", reusable, statErr)
	}
	run := runtime.InspectObject(gobble.ViewRun)
	occupancy, _ := run["occupancy"].(map[string]any)
	if occupancy["active"] != true {
		t.Fatalf("failed ATAC occupancy = %#v, want active recovery state", occupancy)
	}
	if err := runtime.Release(); err != nil {
		t.Fatalf("Release after ATAC input failure: %v", err)
	}
	runtime.Succeed(failed)
	if err := runtime.Resume(t.Context()); err != nil {
		t.Fatalf("Resume after corrected ATAC input failure: %v", err)
	}
	reuse := runtime.InspectRecords(gobble.ViewReuse)
	requireATACDecision(t, reuse, upstream, "reused")
	requireATACDecision(t, reuse, failed, "rerun")
	requireATACDecision(t, reuse, downstream, "rerun")
	if remaining := runtime.InspectRecords(gobble.ViewRemaining); len(remaining) != 0 {
		t.Fatalf("remaining after ATAC failure recovery = %#v, want empty", remaining)
	}
}

func hasATACDefectUnit(err *gobble.Error, unit string) bool {
	if err != nil {
		for _, defect := range err.Defects {
			if defect.Unit == unit {
				return true
			}
		}
	}
	return false
}

func recordsContainATACIdentity(records []map[string]any, identity string) bool {
	for _, record := range records {
		if record["identity"] == identity {
			return true
		}
	}
	return false
}

func recordsContainATACValue(records []map[string]any, value string) bool {
	for _, record := range records {
		if containsATACValue(record, value) {
			return true
		}
	}
	return false
}

func containsATACValue(subject any, value string) bool {
	switch typed := subject.(type) {
	case string:
		return typed == value
	case []any:
		for _, item := range typed {
			if containsATACValue(item, value) {
				return true
			}
		}
	case map[string]any:
		for _, item := range typed {
			if containsATACValue(item, value) {
				return true
			}
		}
	}
	return false
}

func requireATACDecision(t *testing.T, records []map[string]any, identity, want string) {
	t.Helper()
	for _, record := range records {
		if record["identity"] == identity {
			if got := record["decision"]; got != want {
				t.Fatalf("ATAC reuse decision for %s = %q, want %q", identity, got, want)
			}
			return
		}
	}
	t.Fatalf("ATAC reuse records omit %s: %#v", identity, records)
}
