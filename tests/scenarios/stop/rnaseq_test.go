package stop_test

import (
	"context"
	"errors"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/rnaseqscenario"
)

func TestRNAUsesOrdinaryCancellableTasks(t *testing.T) {
	tasks := pc.AllTasks(t, rnaseqscenario.Plan(t, rnaseq.DefaultConfig()))
	for _, task := range tasks {
		for _, token := range task.Command {
			if token == "retry" || token == "cancel" {
				t.Fatalf("RNA task %s adds assay-specific control token %q", task.ID, token)
			}
		}
	}
	if !rnaseq.Lifecycle.Stop {
		t.Fatal("RNA stop participation is false")
	}
}

func TestRNAStopIsInspectableAndRecoversByReleaseResume(t *testing.T) {
	runtime := rnaseqscenario.NewRuntime(t, rnaseq.DefaultConfig())
	const blocked = "WT_REP1.trim_galore"
	runtime.Block(blocked)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- runtime.Run(ctx) }()
	select {
	case <-runtime.Started():
		cancel()
	case <-t.Context().Done():
		t.Fatal("RNA blocked task did not start")
	}
	err := <-done
	var structured *gobble.Error
	if !errors.As(err, &structured) || !hasCanceledDefect(structured) {
		t.Fatalf("Run cancellation error = %v, want canceled defect", err)
	}
	run := runtime.InspectObject(gobble.ViewRun)
	occupancy, _ := run["occupancy"].(map[string]any)
	if occupancy["active"] != true {
		t.Fatalf("occupancy = %#v, want active after stop", occupancy)
	}
	if remaining := runtime.InspectRecords(gobble.ViewRemaining); len(remaining) == 0 {
		t.Fatal("remaining is empty after stopped RNA run")
	}
	if err := runtime.Release(); err != nil {
		t.Fatalf("Release after stop: %v", err)
	}
	runtime.Succeed(blocked)
	if err := runtime.Resume(t.Context()); err != nil {
		t.Fatalf("Resume after stop: %v", err)
	}
	if remaining := runtime.InspectRecords(gobble.ViewRemaining); len(remaining) != 0 {
		t.Fatalf("remaining after recovery = %#v, want empty", remaining)
	}
}

func hasCanceledDefect(err *gobble.Error) bool {
	for _, defect := range err.Defects {
		if defect.Code == gobble.DefectCanceled {
			return true
		}
	}
	return false
}
