package failure_test

import (
	"errors"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/methylseqscenario"
)

func TestMethylInvalidRouteFlagFailsBeforeRun(t *testing.T) {
	samples, _ := methylseqscenario.Samples(t)
	config := methylseq.DefaultConfig()
	config.BismarkAlign.ExtraArgs = []string{"--pbat"}
	graph, err := gobble.Compose(methylseq.Build(samples, config))
	var structured *gobble.Error
	if graph != nil || !errors.As(err, &structured) || structured == nil || !methylseq.Lifecycle().Failure {
		t.Fatalf("invalid Methyl route Compose() = (%v, %v), want structured pre-run failure", graph, err)
	}
}

func TestMethylCommandFailureIsContainedInspectableAndRecoverable(t *testing.T) {
	runtime := methylseqscenario.NewRuntime(t, methylseq.DefaultConfig())
	const failed = "SRR389222_sub1.trim_galore"
	runtime.Fail(failed)
	err := runtime.Run(t.Context())
	var structured *gobble.Error
	if !errors.As(err, &structured) || structured == nil {
		t.Fatalf("Run failure = %v, want structured Methyl failure", err)
	}
	if !recordsContainMethyl(runtime.InspectRecords(gobble.ViewErrors), failed) || !recordsContainMethyl(runtime.InspectRecords(gobble.ViewLogs), failed) || len(runtime.InspectRecords(gobble.ViewRemaining)) == 0 {
		t.Fatal("contained Methyl failure is not fully inspectable")
	}
	if err := runtime.Release(); err != nil {
		t.Fatalf("Release after failure: %v", err)
	}
	runtime.Succeed(failed)
	if err := runtime.Resume(t.Context()); err != nil {
		t.Fatalf("Resume after corrected failure: %v", err)
	}
	if remaining := runtime.InspectRecords(gobble.ViewRemaining); len(remaining) != 0 {
		t.Fatalf("remaining after recovery = %#v", remaining)
	}
}

func recordsContainMethyl(records []map[string]any, value string) bool {
	for _, record := range records {
		if containsMethylValue(record, value) {
			return true
		}
	}
	return false
}

func containsMethylValue(subject any, value string) bool {
	switch typed := subject.(type) {
	case string:
		return typed == value
	case []any:
		for _, item := range typed {
			if containsMethylValue(item, value) {
				return true
			}
		}
	case map[string]any:
		for _, item := range typed {
			if containsMethylValue(item, value) {
				return true
			}
		}
	}
	return false
}
