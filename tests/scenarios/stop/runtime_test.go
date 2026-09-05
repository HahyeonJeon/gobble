package stop_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// Wait for either task admission or an early Run error. A test context alone
// does not expire before the package timeout, so bound both lifecycle waits.
func cancelStartedRun(t *testing.T, run func(context.Context) error, started <-chan struct{}) error {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- run(ctx) }()
	timer := time.NewTimer(time.Minute)
	defer timer.Stop()
	select {
	case <-started:
		cancel()
	case err := <-done:
		detail, _ := json.Marshal(err)
		t.Fatalf("Run returned before the blocked task started: %v (%s)", err, detail)
	case <-timer.C:
		t.Fatal("blocked task did not start within one minute")
	case <-ctx.Done():
		t.Fatal("test canceled before the blocked task started")
	}
	timer.Reset(time.Minute)
	select {
	case err := <-done:
		return err
	case <-timer.C:
		t.Fatal("Run did not settle within one minute of cancellation")
	}
	return nil
}
