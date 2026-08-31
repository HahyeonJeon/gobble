package stop_test

import (
	"context"
	"errors"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/atacseq"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/atacseqscenario"
)

func TestATACStopIsInspectableAndRecoversByReleaseResume(t *testing.T) {
	runtime := atacseqscenario.NewRuntime(t, atacseq.DefaultConfig())
	const blocked = "OSMOTIC_STRESS_T15_PE.replicate_1.run_002.trim_galore"
	runtime.Block(blocked)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- runtime.Run(ctx) }()
	select {
	case <-runtime.Started():
		cancel()
	case <-t.Context().Done():
		t.Fatal("ATAC blocked task did not start")
	}
	err := <-done
	var structured *gobble.Error
	if !errors.As(err, &structured) || !hasATACCanceledDefect(structured) || !atacseq.Lifecycle.Stop {
		t.Fatalf("Run cancellation error = %v, want canceled ATAC defect", err)
	}
	run := runtime.InspectObject(gobble.ViewRun)
	occupancy, _ := run["occupancy"].(map[string]any)
	remaining := runtime.InspectRecords(gobble.ViewRemaining)
	if occupancy["active"] != true || !recordsContainATACIdentity(remaining, blocked) || !recordsContainATACIdentity(remaining, "consensus.replicates.featurecounts_merge_matrices") {
		t.Fatalf("stopped ATAC state = occupancy %#v remaining %#v", occupancy, remaining)
	}
	if err := runtime.Release(); err != nil {
		t.Fatalf("Release after ATAC stop: %v", err)
	}
	released := runtime.InspectObject(gobble.ViewRun)
	releasedOccupancy, _ := released["occupancy"].(map[string]any)
	if releasedOccupancy["active"] != false {
		t.Fatalf("released ATAC occupancy = %#v, want inactive", releasedOccupancy)
	}
	runtime.Succeed(blocked)
	if err := runtime.Resume(t.Context()); err != nil {
		t.Fatalf("Resume after ATAC stop: %v", err)
	}
	if remaining := runtime.InspectRecords(gobble.ViewRemaining); len(remaining) != 0 {
		t.Fatalf("remaining after ATAC stop recovery = %#v, want empty", remaining)
	}
}

func hasATACCanceledDefect(err *gobble.Error) bool {
	for _, defect := range err.Defects {
		if defect.Code == gobble.DefectCanceled {
			return true
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
