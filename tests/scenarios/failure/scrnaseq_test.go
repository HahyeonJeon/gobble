package failure_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/scrnaseq"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/scrnaseqscenario"
)

func TestSCRNAQCatchFailureIsContainedInspectableAndRecoverable(t *testing.T) {
	runtime := scrnaseqscenario.NewRuntime(t, scrnaseq.DefaultConfig())
	const (
		failed   = "Sample_X.qcatch"
		upstream = "Sample_X.simpleaf_quant"
	)
	runtime.FailInput(failed)
	err := runtime.Run(t.Context())
	var structured *gobble.Error
	if !errors.As(err, &structured) || !hasSCRNADefectUnit(structured, failed) || !scrnaseq.Lifecycle().Failure {
		t.Fatalf("Run input failure = %v, want contained scRNA task failure", err)
	}
	errorsView := runtime.InspectRecords(gobble.ViewErrors)
	logsView := runtime.InspectRecords(gobble.ViewLogs)
	remaining := runtime.InspectRecords(gobble.ViewRemaining)
	if !recordsContainSCRNAValue(errorsView, failed) || !recordsContainSCRNAValue(logsView, failed) || !recordsContainSCRNAValue(logsView, "simulated scRNA input rejection for "+failed+"\n") {
		t.Fatalf("scRNA errors/logs omit failed QCatch unit: errors=%#v logs=%#v", errorsView, logsView)
	}
	if !recordsContainSCRNAIdentity(remaining, failed) || !recordsContainSCRNAIdentity(remaining, "multiqc") {
		t.Fatalf("scRNA failure remaining work does not block failed descendants: %#v", remaining)
	}
	reusable := filepath.Join(runtime.Workspace(), "results", "scrnaseq", "samples", "Sample_X", "simpleaf", "af_quant", ".gobble-tree.json")
	if info, statErr := os.Stat(reusable); statErr != nil || !info.Mode().IsRegular() {
		t.Fatalf("reusable scRNA quantification Tree %s: %v", reusable, statErr)
	}
	if err := runtime.Release(); err != nil {
		t.Fatalf("Release after scRNA QCatch failure: %v", err)
	}
	runtime.Succeed(failed)
	if err := runtime.Resume(t.Context()); err != nil {
		t.Fatalf("Resume after corrected scRNA failure: %v", err)
	}
	reuse := runtime.InspectRecords(gobble.ViewReuse)
	requireSCRNADecision(t, reuse, upstream, "reused")
	requireSCRNADecision(t, reuse, failed, "rerun")
	requireSCRNADecision(t, reuse, "multiqc", "rerun")
	if remaining := runtime.InspectRecords(gobble.ViewRemaining); len(remaining) != 0 {
		t.Fatalf("remaining after scRNA failure recovery = %#v", remaining)
	}
}

func hasSCRNADefectUnit(err *gobble.Error, unit string) bool {
	for _, defect := range err.Defects {
		if defect.Unit == unit {
			return true
		}
	}
	return false
}

func recordsContainSCRNAIdentity(records []map[string]any, identity string) bool {
	for _, record := range records {
		if record["identity"] == identity {
			return true
		}
	}
	return false
}

func recordsContainSCRNAValue(records []map[string]any, value string) bool {
	for _, record := range records {
		if containsSCRNAValue(record, value) {
			return true
		}
	}
	return false
}

func containsSCRNAValue(subject any, value string) bool {
	switch typed := subject.(type) {
	case string:
		return typed == value
	case []any:
		for _, item := range typed {
			if containsSCRNAValue(item, value) {
				return true
			}
		}
	case map[string]any:
		for _, item := range typed {
			if containsSCRNAValue(item, value) {
				return true
			}
		}
	}
	return false
}

func requireSCRNADecision(t *testing.T, records []map[string]any, identity, want string) {
	t.Helper()
	for _, record := range records {
		if record["identity"] == identity {
			if got := record["decision"]; got != want {
				t.Fatalf("scRNA reuse decision for %s = %q, want %q", identity, got, want)
			}
			return
		}
	}
	t.Fatalf("scRNA reuse records omit %s", identity)
}
