package exec

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
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

func TestStoppedDockerTaskUsesItsCompletedLiveLogs(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{false: "complete", true: "collector failure"}[failed], func(t *testing.T) {
			attempt := t.TempDir()
			work := filepath.Join(attempt, "work")
			if err := os.Mkdir(work, 0o755); err != nil {
				t.Fatal(err)
			}
			started, finish := make(chan struct{}), make(chan struct{})
			original := DockerCLI
			t.Cleanup(func() { DockerCLI = original })
			DockerCLI = func(ctx context.Context, args, env []string, stdout, stderr io.Writer) (int, error) {
				switch strings.Join(args, " ") {
				case "logs --follow container":
					io.WriteString(stdout, "started\n")
					close(started)
					select {
					case <-finish:
					case <-ctx.Done():
						return -1, ctx.Err()
					}
					io.WriteString(stdout, "finished\n")
					if failed {
						return 23, nil
					}
					return 0, nil
				case "rm -f container":
					return 0, nil
				default:
					t.Errorf("unexpected Docker call %v", args)
					return 1, nil
				}
			}
			ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
			defer cancel()
			d := NewDocker()
			h := Handle{Identity: "task", RuntimeID: "container", Isolate: work, Submission: &Submission{Token: "token"}}
			d.followLogs(ctx, h)
			select {
			case <-started:
			case <-ctx.Done():
				t.Fatal("collector did not start")
			}
			close(finish)
			r, err := d.finishStopped(ctx, h, 0)
			if err != nil {
				t.Fatal(err)
			}
			wantReason := ""
			if failed {
				wantReason = "log-copy-failed"
			}
			if r.Reason != wantReason || r.Exit != 0 || r.RuntimeID != "" {
				t.Fatalf("final report = %#v, want exit 0 with reason %q", r, wantReason)
			}
			got, err := os.ReadFile(filepath.Join(attempt, "stdout"))
			if err != nil || string(got) != "started\nfinished\n" {
				t.Fatalf("final logs = %q, %v", got, err)
			}
		})
	}
}
