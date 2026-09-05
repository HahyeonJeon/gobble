package exec

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/internal/testutil"
)

func useSubmissionDaemon(t *testing.T) (*testutil.Docker, Job) {
	t.Helper()
	t.Setenv("DOCKER_HOST", "")
	t.Setenv("DOCKER_CONTEXT", "")
	fake := &testutil.Docker{Root: t.TempDir()}
	orig := DockerCLI
	DockerCLI = fake.CLI
	t.Cleanup(func() { DockerCLI = orig })
	work := filepath.Join(t.TempDir(), "work")
	if err := os.Mkdir(work, 0o755); err != nil {
		t.Fatal(err)
	}
	return fake, Job{Identity: "copy", Image: pinnedAlpine, Isolate: work,
		Argv: []string{"true"}, Submission: &Submission{Token: strings.Repeat("b", 32)}}
}

func TestDockerStartsOnlyAfterDurableAcknowledgment(t *testing.T) {
	for _, failure := range []string{"none", "intent", "runtime-id", "cancel", "daemon-changed"} {
		t.Run(failure, func(t *testing.T) {
			fake, job := useSubmissionDaemon(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			intent, created := false, false
			job.Record = func(_ context.Context, h Handle, _ Report) error {
				if !h.Submission.Created {
					if failure == "intent" {
						return errors.New("cannot persist intent")
					}
					intent = h.Submission.DaemonID == "test-daemon" && h.Submission.Endpoint == "unix:///gobble-test.sock"
					return nil
				}
				if !intent || h.RuntimeID == "" {
					t.Fatal("created ID arrived without an owning daemon")
				}
				if failure == "runtime-id" {
					return errors.New("disk full")
				}
				created = true
				if failure == "cancel" {
					cancel()
				}
				if failure == "daemon-changed" {
					st, err := fake.State()
					if err != nil {
						return err
					}
					st.DaemonID = "replacement-daemon"
					return fake.Save(st)
				}
				return nil
			}
			fake.Before = func(args []string) {
				if args[0] == "create" && !intent {
					t.Error("create preceded durable intent")
				}
				if args[0] == "start" && !created {
					t.Error("start preceded durable runtime ID")
				}
			}
			_, _, err := NewDocker().Submit(ctx, job)
			if (err != nil) != (failure != "none") {
				t.Fatalf("Submit error = %v for %s", err, failure)
			}
			st, err := fake.State()
			if err != nil {
				t.Fatal(err)
			}
			wantStarts := 0
			if failure == "none" {
				wantStarts = 1
			}
			if st.Starts != wantStarts {
				t.Fatalf("starts = %d, want %d", st.Starts, wantStarts)
			}
		})
	}
}

func TestDockerRecoveryFencesUncertainStart(t *testing.T) {
	fake, job := useSubmissionDaemon(t)
	var recorded Handle
	job.Record = func(_ context.Context, h Handle, _ Report) error {
		if h.Submission.Created {
			recorded = h
			return errors.New("controller lost after recording ID")
		}
		return nil
	}
	if _, _, err := NewDocker().Submit(t.Context(), job); err == nil {
		t.Fatal("missing injected interruption")
	}
	d := NewDocker()
	r, err := d.Reconcile(t.Context(), recorded)
	if err != nil || r.Running || !r.NeedsRemoval {
		t.Fatalf("created container must be fenced: %+v, %v", r, err)
	}
	if err := d.Cancel(t.Context(), recorded); err != nil {
		t.Fatal(err)
	}
	if _, err := fake.CLI(t.Context(), []string{"start", recorded.RuntimeID}, nil, io.Discard, io.Discard); err == nil {
		t.Fatal("a delayed start could launch a removed container")
	}
	r, err = d.Poll(t.Context(), recorded)
	if err != nil || r.Running || r.NeedsRemoval || r.RuntimeID != "" {
		t.Fatalf("removed container is not settled: %+v, %v", r, err)
	}
}

func TestDockerRecoveryRefusesUnprovedOwnership(t *testing.T) {
	for _, failure := range []string{"daemon", "listing", "label", "remove"} {
		t.Run(failure, func(t *testing.T) {
			fake, job := useSubmissionDaemon(t)
			job.Record = func(context.Context, Handle, Report) error { return nil }
			h, _, err := NewDocker().Submit(t.Context(), job)
			if err != nil {
				t.Fatal(err)
			}
			st, err := fake.State()
			if err != nil {
				t.Fatal(err)
			}
			switch failure {
			case "daemon":
				st.DaemonID = "replacement"
			case "listing":
				fake.Fail = "container"
			case "label":
				st.Token = strings.Repeat("c", 32)
			case "remove":
				fake.Fail = "rm"
			}
			if err := fake.Save(st); err != nil {
				t.Fatal(err)
			}
			if err := NewDocker().Cancel(t.Context(), h); err == nil {
				t.Fatal("unknown or unremoved container was accepted as canceled")
			}
			st, err = fake.State()
			if err != nil || !st.Exists {
				t.Fatalf("unproved container was removed: %+v, %v", st, err)
			}
		})
	}
}

func TestDockerSubmissionPinsSelectedEndpoint(t *testing.T) {
	fake, job := useSubmissionDaemon(t)
	t.Setenv("DOCKER_CONTEXT", "user-context")
	t.Setenv("DOCKER_HOST", "unix:///overridden-by-context.sock")
	original := DockerCLI
	DockerCLI = func(ctx context.Context, args, env []string, stdout, stderr io.Writer) (int, error) {
		if args[0] != "context" {
			if !contains(env, "DOCKER_HOST=unix:///gobble-test.sock") || contains(env, "DOCKER_CONTEXT=user-context") {
				t.Errorf("execution did not stay on the selected socket: %v", args)
			}
		}
		return original(ctx, args, env, stdout, stderr)
	}
	job.Record = func(context.Context, Handle, Report) error {
		t.Setenv("DOCKER_HOST", "unix:///changed-during-submission.sock")
		return nil
	}
	if _, _, err := NewDocker().Submit(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	if st, err := fake.State(); err != nil || st.Starts != 1 {
		t.Fatalf("submission did not complete: %+v, %v", st, err)
	}
}
