package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gobexec "github.com/HahyeonJeon/gobble/internal/engine/exec"
	"github.com/HahyeonJeon/gobble/internal/testutil"
)

func submissionRequest(workspace string) Request {
	return Request{Identity: testInstallIdentity(), Workspace: workspace,
		Document: sampleDoc(pinnedAlpine, "local", "in/sample.txt", "out/sample.txt")}
}

func useSubmissionModel(t *testing.T, workspace string) *testutil.Docker {
	t.Helper()
	t.Setenv("DOCKER_CONTEXT", "")
	t.Setenv("DOCKER_HOST", "")
	fake := &testutil.Docker{Root: filepath.Join(workspace, "daemon-model")}
	orig := gobexec.DockerCLI
	gobexec.DockerCLI = fake.CLI
	t.Cleanup(func() {
		gobexec.DockerCLI = orig
		DropHeldLease(workspace)
	})
	return fake
}

func TestDockerStartSeesCommittedRuntimeID(t *testing.T) {
	workspace := t.TempDir()
	writeCheckFile(t, filepath.Join(workspace, "in", "sample.txt"), "reads")
	fake := useSubmissionModel(t, workspace)
	gobexec.DockerCLI = func(ctx context.Context, args, env []string, stdout, stderr io.Writer) (int, error) {
		if args[0] == "create" || args[0] == "start" {
			_, _, _, file, _, d := readCoherentControl(workspace)
			if len(d) > 0 || len(file.Tasks) != 1 {
				return -1, fmt.Errorf("backend started without a readable checkpoint: %v", d)
			}
			st := file.Tasks[0]
			if st.Submission == nil || st.Submission.DaemonID != "test-daemon" {
				return -1, errors.New("backend created without durable daemon identity")
			}
			if args[0] == "start" && (st.RuntimeID != args[1] || !st.Submission.Created) {
				return -1, errors.New("backend started without durable runtime ID")
			}
		}
		return fake.CLI(ctx, args, env, stdout, stderr)
	}
	if d := Run(t.Context(), submissionRequest(workspace)); len(d) > 0 {
		t.Fatal(d)
	}
	if raw, err := os.ReadFile(filepath.Join(workspace, "out", "sample.txt")); err != nil || string(raw) != "reads" {
		t.Fatalf("publication: %q, %v", raw, err)
	}
}

func TestDockerSubmissionSurvivesControllerDeath(t *testing.T) {
	for _, boundary := range []string{"before-create", "after-create", "before-start", "after-start", "after-rm"} {
		t.Run(boundary, func(t *testing.T) {
			workspace := t.TempDir()
			writeCheckFile(t, filepath.Join(workspace, "in", "sample.txt"), "reads")
			fake := useSubmissionModel(t, workspace)
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			cmd := osexec.CommandContext(ctx, os.Args[0], "-test.run=^TestDockerSubmissionCrashWriter$")
			cmd.Env = append(os.Environ(), "GOBBLE_TEST_SUBMISSION_WORKSPACE="+workspace, "GOBBLE_TEST_SUBMISSION_PHASE="+boundary)
			out, err := cmd.CombinedOutput()
			var exited *osexec.ExitError
			if !errors.As(err, &exited) || exited.ExitCode() != 73 {
				t.Fatalf("crash writer: %v\n%s", err, out)
			}
			req := submissionRequest(workspace)
			if _, d := Inspect(workspace, viewRun, "", req.Identity); len(d) > 0 {
				t.Fatalf("Inspect after death: %v", d)
			}
			if d := Resume(t.Context(), req); len(d) > 0 {
				t.Fatalf("Resume after death: %v", d)
			}
			st, err := fake.State()
			wantStarts := 1
			if boundary == "after-start" || boundary == "after-rm" {
				wantStarts++
			}
			if err != nil || st.Starts != wantStarts || st.Exists || st.Running {
				t.Fatalf("unexpected work after recovery: %+v, %v", st, err)
			}
			if task := taskStates(t, workspace)["copy"]; task.Attempt != 2 || task.Status != StatusSucceeded {
				t.Fatalf("recovered attempt: %+v", task)
			}
			if raw, err := os.ReadFile(filepath.Join(workspace, "out", "sample.txt")); err != nil || string(raw) != "reads" {
				t.Fatalf("recovered output: %q, %v", raw, err)
			}
		})
	}
}

func TestDockerSubmissionCrashWriter(t *testing.T) {
	workspace := os.Getenv("GOBBLE_TEST_SUBMISSION_WORKSPACE")
	if workspace == "" {
		return
	}
	boundary := os.Getenv("GOBBLE_TEST_SUBMISSION_PHASE")
	fake := useSubmissionModel(t, workspace)
	fake.RunContinuously = boundary != "after-rm"
	fake.Before = func(args []string) {
		if boundary == "before-"+args[0] {
			os.Exit(73)
		}
	}
	fake.After = func(args []string) {
		if boundary == "after-"+args[0] {
			os.Exit(73)
		}
	}
	d := Run(t.Context(), submissionRequest(workspace))
	t.Fatalf("did not reach submission boundary %s: %v", boundary, d)
}

func TestDockerRecoveryRestoresObservationBeforeResume(t *testing.T) {
	workspace := t.TempDir()
	writeCheckFile(t, filepath.Join(workspace, "in", "sample.txt"), "reads")
	fake := useSubmissionModel(t, workspace)
	fake.Fail = "start"
	req := submissionRequest(workspace)
	if d := Run(t.Context(), req); !hasDefect(d, DefectUnknownBackend, "copy") {
		t.Fatalf("uncertain start: %v", d)
	}
	fake.Fail = "container"
	if d := Release(workspace, req.Identity); !hasDefect(d, DefectUnknownBackend, "copy") {
		t.Fatalf("unobservable recovery was accepted: %v", d)
	}
	if d := Resume(t.Context(), req); len(d) == 0 {
		t.Fatal("Resume launched work before reconciliation")
	}
	fake.Fail = ""
	if d := Release(workspace, req.Identity); len(d) > 0 {
		t.Fatal(d)
	}
	if d := Resume(t.Context(), req); len(d) > 0 {
		t.Fatal(d)
	}
	st, err := fake.State()
	if err != nil || st.Starts != 1 {
		t.Fatalf("duplicate or missing execution: %+v, %v", st, err)
	}
}

func TestAdmissionWriteFailureNeverSubmits(t *testing.T) {
	workspace := t.TempDir()
	writeCheckFile(t, filepath.Join(workspace, "in", "sample.txt"), "reads")
	fake := useSubmissionModel(t, workspace)
	s, d := occupy(submissionRequest(workspace))
	if len(d) > 0 {
		t.Fatal(d)
	}
	defer s.lease.mutator.Unlock()
	path := filepath.Join(workspace, ControlDir, checkpointPointerFile)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if s.launch(t.Context(), "copy", make(chan startEvent, 1), make(chan report, 1)) || s.persist == nil {
		t.Fatal("admission continued after a failed checkpoint")
	}
	if st, err := fake.State(); err != nil || st.Exists || st.Starts != 0 {
		t.Fatalf("backend reached after failed admission: %+v, %v", st, err)
	}
	if !strings.Contains(s.persist.Error(), "directory") {
		t.Fatalf("unexpected admission failure: %v", s.persist)
	}
}

func TestDockerReleaseRetriesTerminalContainerRemoval(t *testing.T) {
	workspace := t.TempDir()
	writeCheckFile(t, filepath.Join(workspace, "in", "sample.txt"), "reads")
	fake := useSubmissionModel(t, workspace)
	fake.Fail = "rm"
	req := submissionRequest(workspace)
	if d := Run(t.Context(), req); len(d) > 0 {
		t.Fatal(d)
	}
	if task := taskStates(t, workspace)["copy"]; task.Status != StatusSucceeded || task.RuntimeID == "" {
		t.Fatalf("terminal task lost its retained container: %+v", task)
	}
	fake.Fail = ""
	if d := Release(workspace, req.Identity); len(d) > 0 {
		t.Fatalf("Release did not retry terminal cleanup: %v", d)
	}
	if task := taskStates(t, workspace)["copy"]; task.Status != StatusSucceeded || task.RuntimeID != "" {
		t.Fatalf("cleanup changed task outcome or retained its container: %+v", task)
	}
}
