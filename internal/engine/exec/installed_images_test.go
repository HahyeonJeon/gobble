package exec

import (
	"context"
	"testing"
)

func TestPreparedDockerRejectsMissingOrChangedImageWithoutPull(t *testing.T) {
	for _, images := range []map[string]string{{}, {pinnedAlpine: "sha256:wrong"}} {
		fake, job := useSubmissionDaemon(t)
		job.Record = func(context.Context, Handle, Report) error { return nil }
		calls := []string{}
		fake.Before = func(args []string) { calls = append(calls, args[0]) }
		ctx := WithInstalledImages(t.Context(), images)
		// Changing the caller's map cannot relax this execution's accepted set.
		images[pinnedAlpine] = "changed-after-binding"
		if _, _, err := NewDocker().Submit(ctx, job); err == nil {
			t.Fatal("unavailable accepted image submitted")
		}
		for _, call := range calls {
			if call == "pull" || call == "create" || call == "start" {
				t.Fatalf("unexpected effect %s", call)
			}
		}
	}
}

func TestPreparedDockerCancellationDuringToolCheckIsNotUnknown(t *testing.T) {
	fake, job := useSubmissionDaemon(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	job.Record = func(context.Context, Handle, Report) error { return nil }
	fake.Before = func(args []string) {
		if args[0] == "image" {
			cancel()
		}
	}
	h, _, err := NewDocker().Submit(WithInstalledImages(ctx, map[string]string{pinnedAlpine: "sha256:expected"}), job)
	if err != context.Canceled || h.Submission != nil || h.RuntimeID != "" {
		t.Fatalf("cancellation manufactured an unknown execution: %+v %v", h, err)
	}
}
