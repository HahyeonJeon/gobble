package stop_test

import (
	"errors"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/scrnaseq"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/scrnaseqscenario"
)

func TestSCRNAStopIsInspectableAndRecoversByReleaseResume(t *testing.T) {
	runtime := scrnaseqscenario.NewRuntime(t, scrnaseq.DefaultConfig())
	const blocked = "Sample_Y.simpleaf_quant"
	runtime.Block(blocked)
	err := cancelStartedRun(t, runtime.Run, runtime.Started())
	var structured *gobble.Error
	if !errors.As(err, &structured) || !hasSCRNACanceledDefect(structured) || !scrnaseq.Lifecycle().Stop {
		t.Fatalf("Run cancellation error = %v, want canceled scRNA defect", err)
	}
	run := runtime.InspectObject(gobble.ViewRun)
	occupancy, _ := run["occupancy"].(map[string]any)
	remaining := runtime.InspectRecords(gobble.ViewRemaining)
	if occupancy["active"] != true || !recordsContainSCRNAIdentity(remaining, blocked) || !recordsContainSCRNAIdentity(remaining, "cohort.h5ad_concat") {
		t.Fatalf("stopped scRNA state = occupancy %#v remaining %#v", occupancy, remaining)
	}
	if err := runtime.Release(); err != nil {
		t.Fatalf("Release after scRNA stop: %v", err)
	}
	runtime.Succeed(blocked)
	if err := runtime.Resume(t.Context()); err != nil {
		t.Fatalf("Resume after scRNA stop: %v", err)
	}
	if remaining := runtime.InspectRecords(gobble.ViewRemaining); len(remaining) != 0 {
		t.Fatalf("remaining after scRNA stop recovery = %#v", remaining)
	}
}

func hasSCRNACanceledDefect(err *gobble.Error) bool {
	for _, defect := range err.Defects {
		if defect.Code == gobble.DefectCanceled {
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
