package stop_test

import (
	"context"
	"errors"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/wgsscenario"
)

func TestWGSStopIsInspectableAndRecoversByReleaseResume(t *testing.T) {
	runtime := wgsscenario.NewRuntime(t, wgs.DefaultConfig())
	const blocked = "patient1.testN.bqsr_gather.samtools_merge"
	runtime.Block(blocked)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- runtime.Run(ctx) }()
	select {
	case <-runtime.Started():
		cancel()
	case <-t.Context().Done():
		t.Fatal("WGS blocked interval-gather task did not start")
	}
	err := <-done
	var structured *gobble.Error
	if !errors.As(err, &structured) || !hasWGSCanceledDefect(structured) || !wgs.Lifecycle().Stop {
		t.Fatalf("Run cancellation error = %v, want canceled WGS defect", err)
	}
	run := runtime.InspectObject(gobble.ViewRun)
	occupancy, _ := run["occupancy"].(map[string]any)
	remaining := runtime.InspectRecords(gobble.ViewRemaining)
	if occupancy["active"] != true || !recordsContainWGSIdentity(remaining, blocked) || !recordsContainWGSIdentity(remaining, "haplotype_intervals.patient1.testN.gatk4_haplotypecaller") {
		t.Fatalf("stopped WGS state = occupancy %#v remaining %#v", occupancy, remaining)
	}
	if err := runtime.Release(); err != nil {
		t.Fatalf("Release after WGS stop: %v", err)
	}
	released := runtime.InspectObject(gobble.ViewRun)
	releasedOccupancy, _ := released["occupancy"].(map[string]any)
	if releasedOccupancy["active"] != false {
		t.Fatalf("released WGS occupancy = %#v, want inactive", releasedOccupancy)
	}
	runtime.Succeed(blocked)
	if err := runtime.Resume(t.Context()); err != nil {
		t.Fatalf("Resume after WGS stop: %v", err)
	}
	for _, record := range runtime.InspectRecords(gobble.ViewRemaining) {
		if record["remaining"] == true {
			t.Fatalf("remaining after WGS stop recovery = %#v, want completed", record)
		}
	}
}

func hasWGSCanceledDefect(err *gobble.Error) bool {
	for _, defect := range err.Defects {
		if defect.Code == gobble.DefectCanceled {
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
