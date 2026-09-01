package failure_test

import (
	"errors"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/rnaseqscenario"
)

func TestRNAInvalidRouteFlagFailsBeforeRun(t *testing.T) {
	samples, _ := rnaseqscenario.Samples(t)
	config := rnaseq.DefaultConfig()
	config.STAR.ExtraArgs = []string{"--outSAMtype", "SAM"}
	graph, err := gobble.Compose(rnaseq.Build(samples, config))
	var structured *gobble.Error
	if graph != nil || !errors.As(err, &structured) || structured == nil || !rnaseq.Lifecycle().Failure {
		t.Fatalf("invalid RNA route Compose() = (%v, %v), want structured pre-run failure", graph, err)
	}
}

func TestRNACommandFailureIsContainedInspectableAndRecoverable(t *testing.T) {
	runtime := rnaseqscenario.NewRuntime(t, rnaseq.DefaultConfig())
	const failed = "WT_REP1.trim_galore"
	runtime.Fail(failed)
	err := runtime.Run(t.Context())
	var structured *gobble.Error
	if !errors.As(err, &structured) || structured == nil {
		t.Fatalf("Run failure = %v, want structured RNA failure", err)
	}
	errorsView := runtime.InspectRecords(gobble.ViewErrors)
	logsView := runtime.InspectRecords(gobble.ViewLogs)
	remaining := runtime.InspectRecords(gobble.ViewRemaining)
	if !recordsContain(errorsView, failed) || !recordsContain(logsView, failed) {
		t.Fatalf("errors/logs omit failed RNA unit %s: errors=%#v logs=%#v", failed, errorsView, logsView)
	}
	if len(remaining) == 0 {
		t.Fatal("remaining is empty after contained RNA command failure")
	}
	if err := runtime.Release(); err != nil {
		t.Fatalf("Release after failure: %v", err)
	}
	runtime.Succeed(failed)
	if err := runtime.Resume(t.Context()); err != nil {
		t.Fatalf("Resume after corrected command failure: %v", err)
	}
	if remaining = runtime.InspectRecords(gobble.ViewRemaining); len(remaining) != 0 {
		t.Fatalf("remaining after recovery = %#v, want empty", remaining)
	}
}

func recordsContain(records []map[string]any, value string) bool {
	for _, record := range records {
		if containsValue(record, value) {
			return true
		}
	}
	return false
}

func containsValue(subject any, value string) bool {
	switch typed := subject.(type) {
	case string:
		return typed == value
	case []any:
		for _, item := range typed {
			if containsValue(item, value) {
				return true
			}
		}
	case map[string]any:
		for _, item := range typed {
			if containsValue(item, value) {
				return true
			}
		}
	}
	return false
}
