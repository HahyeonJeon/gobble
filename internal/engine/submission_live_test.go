//go:build live

package engine

import (
	"context"
	"errors"
	"io"
	"os"
	osexec "os/exec"
	"path/filepath"
	"testing"
	"time"

	gobexec "github.com/HahyeonJeon/gobble/internal/engine/exec"
)

func liveSubmissionRequest(workspace string) Request {
	req := submissionRequest(workspace)
	req.Document.Tasks[0].Command = []string{"sh", "-c", "sleep 60; cp in/sample.txt out/sample.txt"}
	return req
}

// This is an actual daemon gate, separate from the file-backed daemon model.
func TestDockerLiveControllerDeathRecovery(t *testing.T) {
	requireDocker(t)
	for _, boundary := range []string{"after-create", "before-start", "after-start"} {
		t.Run(boundary, func(t *testing.T) {
			workspace := t.TempDir()
			writeCheckFile(t, filepath.Join(workspace, "in", "sample.txt"), "reads")
			req := liveSubmissionRequest(workspace)
			t.Cleanup(func() {
				_ = Release(workspace, req.Identity)
				DropHeldLease(workspace)
			})
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
			defer cancel()
			cmd := osexec.CommandContext(ctx, os.Args[0], "-test.run=^TestDockerLiveCrashWriter$")
			cmd.Env = append(os.Environ(), "GOBBLE_TEST_LIVE_WORKSPACE="+workspace, "GOBBLE_TEST_LIVE_PHASE="+boundary)
			out, err := cmd.CombinedOutput()
			var exited *osexec.ExitError
			if !errors.As(err, &exited) || exited.ExitCode() != 73 {
				t.Fatalf("live crash writer: %v\n%s", err, out)
			}
			if d := Release(workspace, req.Identity); len(d) > 0 {
				t.Fatalf("live recovery: %v", d)
			}
			_, _, _, file, _, d := readCoherentControl(workspace)
			if len(d) > 0 || len(file.Tasks) != 1 {
				t.Fatalf("live recovered state: %v", d)
			}
			st := file.Tasks[0]
			h, ok := backendHandle(workspace, "copy", &st)
			if !ok {
				t.Fatal("lost durable submission identity")
			}
			r, err := gobexec.NewDocker().Reconcile(ctx, h)
			if err != nil || r.Running || r.NeedsRemoval || r.RuntimeID != "" {
				t.Fatalf("old live container remains: %+v, %v", r, err)
			}
			// Resume unfinished work with a short command, avoiding a minute-long
			// workload in every smoke case. Public selective rerun checks apply.
			req.Document.Tasks[0].Command = []string{"cp", "in/sample.txt", "out/sample.txt"}
			if d := Resume(ctx, req); len(d) > 0 {
				t.Fatalf("live Resume: %v", d)
			}
			if raw, err := os.ReadFile(filepath.Join(workspace, "out", "sample.txt")); err != nil || string(raw) != "reads" {
				t.Fatalf("live resumed output: %q, %v", raw, err)
			}
		})
	}
}

func TestDockerLiveCrashWriter(t *testing.T) {
	workspace := os.Getenv("GOBBLE_TEST_LIVE_WORKSPACE")
	if workspace == "" {
		return
	}
	boundary := os.Getenv("GOBBLE_TEST_LIVE_PHASE")
	original := gobexec.DockerCLI
	gobexec.DockerCLI = func(ctx context.Context, args, env []string, stdout, stderr io.Writer) (int, error) {
		if boundary == "before-"+args[0] {
			os.Exit(73)
		}
		code, err := original(ctx, args, env, stdout, stderr)
		if err == nil && code == 0 && boundary == "after-"+args[0] {
			os.Exit(73)
		}
		return code, err
	}
	d := Run(t.Context(), liveSubmissionRequest(workspace))
	t.Fatalf("did not reach live boundary %s: %v", boundary, d)
}
