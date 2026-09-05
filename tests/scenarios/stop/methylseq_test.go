package stop_test

import (
	"errors"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/methylseqscenario"
)

func TestMethylStopIsInspectableAndRecoversByReleaseResume(t *testing.T) {
	runtime := methylseqscenario.NewRuntime(t, methylseq.DefaultConfig())
	const blocked = "SRR389222_sub1.trim_galore"
	runtime.Block(blocked)
	err := cancelStartedRun(t, runtime.Run, runtime.Started())
	var structured *gobble.Error
	if !errors.As(err, &structured) || !containsCanceled(structured) || !methylseq.Lifecycle().Stop {
		t.Fatalf("Run cancellation error = %v, want canceled Methyl defect", err)
	}
	run := runtime.InspectObject(gobble.ViewRun)
	occupancy, _ := run["occupancy"].(map[string]any)
	if occupancy["active"] != true || len(runtime.InspectRecords(gobble.ViewRemaining)) == 0 {
		t.Fatalf("stopped state = occupancy %#v remaining %#v", occupancy, runtime.InspectRecords(gobble.ViewRemaining))
	}
	if err := runtime.Release(); err != nil {
		t.Fatalf("Release after stop: %v", err)
	}
	runtime.Succeed(blocked)
	if err := runtime.Resume(t.Context()); err != nil {
		t.Fatalf("Resume after stop: %v", err)
	}
	if remaining := runtime.InspectRecords(gobble.ViewRemaining); len(remaining) != 0 {
		t.Fatalf("remaining after recovery = %#v", remaining)
	}
}

func containsCanceled(err *gobble.Error) bool {
	for _, defect := range err.Defects {
		if defect.Code == gobble.DefectCanceled {
			return true
		}
	}
	return false
}
