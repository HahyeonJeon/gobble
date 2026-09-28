package gobble_test

import (
	"context"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestPreparedContinuationBindsActualRuntime(t *testing.T) {
	original := gobble.LaunchIntent{EngineImage: "sha256:" + strings.Repeat("a", 64), DaemonID: "accepted-daemon"}
	for _, mismatch := range []string{"engine", "daemon"} {
		t.Run(mismatch, func(t *testing.T) {
			t.Setenv("GOBBLE_RUNTIME_IMAGE_ID", original.EngineImage)
			t.Setenv("GOBBLE_DAEMON_ID", original.DaemonID)
			if mismatch == "engine" {
				t.Setenv("GOBBLE_RUNTIME_IMAGE_ID", "different")
			} else {
				t.Setenv("GOBBLE_DAEMON_ID", "different")
			}
			ws := t.TempDir()
			if _, err := gobble.ReviewPreparedContinuation(context.Background(), ws, nil, original); err == nil || !strings.Contains(err.Error(), "runtime differs") {
				t.Fatal(err)
			}
			if _, err := gobble.ContinuePreparedPipeline(context.Background(), ws, nil, original, gobble.ContinuationIntent{}); err == nil || !strings.Contains(err.Error(), "runtime differs") {
				t.Fatal(err)
			}
			if _, _, err := gobble.ReadPreparedContinuation(ws, original, gobble.ContinuationIntent{}); err == nil || !strings.Contains(err.Error(), "runtime differs") {
				t.Fatal(err)
			}
		})
	}
}
