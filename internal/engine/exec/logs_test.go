package exec

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestDockerLogsVisibleBeforeTaskStops(t *testing.T) {
	dir := t.TempDir()
	isolate := filepath.Join(dir, "work")
	if err := os.Mkdir(isolate, 0o755); err != nil {
		t.Fatal(err)
	}
	ready := make(chan struct{})
	var calls atomic.Int32
	original := DockerCLI
	DockerCLI = func(ctx context.Context, args, env []string, stdout, stderr io.Writer) (int, error) {
		calls.Add(1)
		if _, err := io.WriteString(stdout, "alignment started\n"); err != nil {
			return -1, err
		}
		if _, err := io.WriteString(stderr, "tool progress\n"); err != nil {
			return -1, err
		}
		close(ready)
		<-ctx.Done()
		return -1, ctx.Err()
	}
	defer func() { DockerCLI = original }()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	d := NewDocker()
	h := Handle{RuntimeID: "container", Isolate: isolate, Submission: &Submission{Token: "token"}}
	d.followLogs(ctx, h)
	d.followLogs(ctx, h)
	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatal("collector did not start")
	}
	for name, want := range map[string]string{"stdout": "alignment started\n", "stderr": "tool progress\n"} {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || string(got) != want {
			t.Fatalf("live %s=%q %v", name, got, err)
		}
	}
	if err := d.stopLogs(ctx, h.RuntimeID); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("started %d collectors", calls.Load())
	}
}
