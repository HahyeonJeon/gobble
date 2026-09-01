package failure_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/wgsscenario"
)

func TestWGSCommandFailureIsContainedInspectableAndRecoverable(t *testing.T) {
	runtime := wgsscenario.NewRuntime(t, wgs.DefaultConfig())
	const (
		failed     = "patient1.testN.gvcf_gather.gatk4_mergevcfs"
		upstream   = "patient1.testN.bqsr_gather.samtools_merge"
		downstream = "joint_intervals.database.gatk4_genomicsdbimport"
	)
	runtime.FailCommand(failed)
	err := runtime.Run(t.Context())
	var structured *gobble.Error
	if !errors.As(err, &structured) || !hasWGSDefectUnit(structured, failed) || !wgs.Lifecycle().Failure {
		t.Fatalf("Run command failure = %v, want contained WGS task failure", err)
	}
	errorsView := runtime.InspectRecords(gobble.ViewErrors)
	logsView := runtime.InspectRecords(gobble.ViewLogs)
	remaining := runtime.InspectRecords(gobble.ViewRemaining)
	if !recordsContainWGSValue(errorsView, failed) || !recordsContainWGSValue(logsView, failed) || !recordsContainWGSValue(logsView, "simulated WGS command failure for "+failed+"\n") {
		t.Fatalf("WGS errors/logs omit failed unit: errors=%#v logs=%#v", errorsView, logsView)
	}
	if !recordsContainWGSIdentity(remaining, failed) || !recordsContainWGSIdentity(remaining, downstream) || !recordsContainWGSIdentity(remaining, "joint_gather.joint.gatk4_mergevcfs") {
		t.Fatalf("WGS failure remaining work does not block failed descendants: %#v", remaining)
	}
	reusable := filepath.Join(runtime.Workspace(), "results", "wgs", "samples", "patient1", "testN", "alignment", "testN.recalibrated.bam")
	if info, statErr := os.Stat(reusable); statErr != nil || !info.Mode().IsRegular() {
		t.Fatalf("reusable WGS upstream output %s: %v", reusable, statErr)
	}
	run := runtime.InspectObject(gobble.ViewRun)
	occupancy, _ := run["occupancy"].(map[string]any)
	if occupancy["active"] != true {
		t.Fatalf("failed WGS occupancy = %#v, want active recovery state", occupancy)
	}
	if err := runtime.Release(); err != nil {
		t.Fatalf("Release after WGS command failure: %v", err)
	}
	runtime.Succeed(failed)
	if err := runtime.Resume(t.Context()); err != nil {
		t.Fatalf("Resume after corrected WGS command failure: %v", err)
	}
	reuse := runtime.InspectRecords(gobble.ViewReuse)
	requireWGSDecision(t, reuse, upstream, "reused")
	requireWGSDecision(t, reuse, failed, "rerun")
	requireWGSDecision(t, reuse, downstream+"/interval_001/0", "rerun")
	for _, record := range runtime.InspectRecords(gobble.ViewRemaining) {
		if record["remaining"] == true {
			t.Fatalf("remaining after WGS failure recovery = %#v, want completed", record)
		}
	}
}

func hasWGSDefectUnit(err *gobble.Error, unit string) bool {
	for _, defect := range err.Defects {
		if defect.Unit == unit {
			return true
		}
	}
	return false
}

func recordsContainWGSIdentity(records []map[string]any, identity string) bool {
	for _, record := range records {
		if record["identity"] == identity {
			return true
		}
	}
	return false
}

func recordsContainWGSValue(records []map[string]any, value string) bool {
	for _, record := range records {
		if containsWGSValue(record, value) {
			return true
		}
	}
	return false
}

func containsWGSValue(subject any, value string) bool {
	switch typed := subject.(type) {
	case string:
		return typed == value
	case []any:
		for _, item := range typed {
			if containsWGSValue(item, value) {
				return true
			}
		}
	case map[string]any:
		for _, item := range typed {
			if containsWGSValue(item, value) {
				return true
			}
		}
	}
	return false
}

func requireWGSDecision(t *testing.T, records []map[string]any, identity, want string) {
	t.Helper()
	for _, record := range records {
		if record["identity"] == identity {
			if got := record["decision"]; got != want {
				t.Fatalf("WGS reuse decision for %s = %q, want %q", identity, got, want)
			}
			return
		}
	}
	t.Fatalf("WGS reuse records omit %s: %#v", identity, records)
}
