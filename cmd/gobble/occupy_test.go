package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/cmd/gobble/testdata/hostpipe"
	"github.com/HahyeonJeon/gobble/internal/testutil"
)

var testIdentityOnce sync.Once
var testIdentity gobble.Identity
var testIdentityErr error

func testOccupyOption(t *testing.T) gobble.OccupyOption {
	t.Helper()
	testIdentityOnce.Do(func() {
		testIdentity, testIdentityErr = gobble.IdentityFromBuildInfo("github.com/HahyeonJeon/gobble/cmd/gobble")
	})
	if testIdentityErr != nil {
		t.Fatalf("IdentityFromBuildInfo() error = %v", testIdentityErr)
	}
	return gobble.WithIdentity(testIdentity)
}

func TestRunSuccessOmitsCap(t *testing.T) {
	watchDriverTemps(t)
	dir := t.TempDir()
	res := runCLI("run", "./testdata/hostpipe", "--workspace", dir)
	requireOpSuccess(t, res, "run")
	requireOccupied(t, dir)
	g, err := gobble.Compose(hostpipe.Pipeline())
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	libErr := gobble.Run(t.Context(), g, dir, 0, testOccupyOption(t))
	second := runCLI("run", "./testdata/hostpipe", "--workspace", dir)
	requireDomainError(t, second, libErr)
}

func TestRunCapZero(t *testing.T) {
	watchDriverTemps(t)
	dir := t.TempDir()
	res := runCLI("run", "./testdata/hostpipe", "--workspace", dir, "--cap", "0")
	requireOpSuccess(t, res, "run")
	requireOccupied(t, dir)
}

func TestRunCapRejected(t *testing.T) {
	watchDriverTemps(t)
	dir := t.TempDir()
	g, err := gobble.Compose(hostpipe.Pipeline())
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	libErr := gobble.Run(t.Context(), g, dir, -1, testOccupyOption(t))
	res := runCLI("run", "./testdata/hostpipe", "--workspace", dir, "--cap=-1")
	requireDomainError(t, res, libErr)
}

func TestRunWorkspaceRefuse(t *testing.T) {
	watchDriverTemps(t)
	g, err := gobble.Compose(hostpipe.Pipeline())
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	t.Run("missing", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "absent")
		libErr := gobble.Run(t.Context(), g, missing, 0, testOccupyOption(t))
		res := runCLI("run", "./testdata/hostpipe", "--workspace", missing)
		requireDomainError(t, res, libErr)
		if _, statErr := os.Stat(missing); !os.IsNotExist(statErr) {
			t.Fatalf("run created missing workspace")
		}
	})
	t.Run("not a directory", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		libErr := gobble.Run(t.Context(), g, file, 0, testOccupyOption(t))
		res := runCLI("run", "./testdata/hostpipe", "--workspace", file)
		requireDomainError(t, res, libErr)
	})
}

func TestRunInspectReleaseResume(t *testing.T) {
	watchDriverTemps(t)
	dir := t.TempDir()
	res := runCLI("run", "./testdata/hostpipe", "--workspace", dir)
	requireOpSuccess(t, res, "run")
	requireOccupied(t, dir)

	inspect := runCLI("inspect", "run", "--workspace", dir)
	if inspect.code != 0 {
		t.Fatalf("inspect exit = %d\nstderr: %s", inspect.code, inspect.stderr)
	}
	if len(inspect.stdout) == 0 {
		t.Fatal("inspect stdout empty")
	}

	rel := runCLI("release", "--workspace", dir)
	requireOpSuccess(t, rel, "release")

	resume := runCLI("resume", "./testdata/hostpipe", "--workspace", dir)
	requireOpSuccess(t, resume, "resume")
	requireOccupied(t, dir)

	g, err := gobble.Compose(hostpipe.Pipeline())
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	libErr := gobble.Resume(t.Context(), g, dir, 0, testOccupyOption(t))
	occupied := runCLI("resume", "./testdata/hostpipe", "--workspace", dir)
	requireDomainError(t, occupied, libErr)
}

func TestResumeWorkspaceRefuse(t *testing.T) {
	watchDriverTemps(t)
	g, err := gobble.Compose(hostpipe.Pipeline())
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	t.Run("missing", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "absent")
		libErr := gobble.Resume(t.Context(), g, missing, 0, testOccupyOption(t))
		res := runCLI("resume", "./testdata/hostpipe", "--workspace", missing)
		requireDomainError(t, res, libErr)
		if _, statErr := os.Stat(missing); !os.IsNotExist(statErr) {
			t.Fatalf("resume created missing workspace")
		}
	})
	t.Run("not a directory", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		libErr := gobble.Resume(t.Context(), g, file, 0, testOccupyOption(t))
		res := runCLI("resume", "./testdata/hostpipe", "--workspace", file)
		requireDomainError(t, res, libErr)
	})
}

func TestSignalCancel(t *testing.T) {
	watchDriverTemps(t)
	bin := buildGobble(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, sig := range []os.Signal{syscall.SIGINT, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			dir := t.TempDir()
			cmd := exec.Command(bin, "run", "./testdata/sleeppipe", "--workspace", dir)
			cmd.Dir = cwd
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			waited := make(chan error, 1)
			go func() { waited <- cmd.Wait() }()
			t.Cleanup(func() {
				if cmd.ProcessState == nil && cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
			})
			waitOccupancy(t, dir)
			if err := cmd.Process.Signal(sig); err != nil {
				t.Fatal(err)
			}
			var waitErr error
			select {
			case waitErr = <-waited:
			case <-time.After(15 * time.Second):
				_ = cmd.Process.Kill()
				<-waited
				t.Fatalf("timeout waiting for canceled run\nstdout: %s\nstderr: %s", stdout.Bytes(), stderr.Bytes())
			}
			res := cliResult{stdout: stdout.Bytes(), stderr: stderr.Bytes(), code: exitCode(waitErr)}
			if len(res.stdout) != 0 {
				t.Fatalf("stdout = %q, want empty", res.stdout)
			}
			if res.code != 1 {
				t.Fatalf("exit = %d, want 1\nstderr: %s", res.code, res.stderr)
			}
			var ge gobble.Error
			if err := json.Unmarshal(res.stderr, &ge); err != nil {
				t.Fatalf("stderr JSON: %v\n%s", err, res.stderr)
			}
			if ge.Op != "run" {
				t.Fatalf("op = %q, want run", ge.Op)
			}
			if len(ge.Defects) == 0 || ge.Defects[0].Code != gobble.DefectCanceled {
				t.Fatalf("defects = %#v, want canceled", ge.Defects)
			}
			requireOccupied(t, dir)

			g, err := gobble.Compose(hostpipe.Pipeline())
			if err != nil {
				t.Fatalf("Compose() error = %v", err)
			}
			libErr := gobble.Run(t.Context(), g, dir, 0, testOccupyOption(t))
			second := runCLI("run", "./testdata/hostpipe", "--workspace", dir)
			requireDomainError(t, second, libErr)

			var releaseStdout, releaseStderr bytes.Buffer
			releaseCmd := exec.Command(bin, "release", "--workspace", dir)
			releaseCmd.Dir = cwd
			releaseCmd.Stdout = &releaseStdout
			releaseCmd.Stderr = &releaseStderr
			releaseErr := releaseCmd.Run()
			rel := cliResult{stdout: releaseStdout.Bytes(), stderr: releaseStderr.Bytes(), code: exitCode(releaseErr)}
			requireOpSuccess(t, rel, "release")
		})
	}
}

func requireOpSuccess(t *testing.T, res cliResult, op string) {
	t.Helper()
	if res.code != 0 {
		t.Fatalf("exit = %d\nstderr: %s", res.code, res.stderr)
	}
	if len(res.stderr) != 0 {
		t.Fatalf("stderr = %q, want empty", res.stderr)
	}
	want := "{\"op\":\"" + op + "\"}\n"
	if string(res.stdout) != want {
		t.Fatalf("stdout = %q, want %q", res.stdout, want)
	}
}

func requireOccupied(t *testing.T, workspace string) {
	t.Helper()
	if !occupancyActive(t, workspace) {
		t.Fatalf("occupancy inactive in %s", workspace)
	}
}

func occupancyActive(t *testing.T, workspace string) bool {
	t.Helper()
	data, err := os.ReadFile(testutil.ControlPath(t, filepath.Join(workspace, ".gobble", "run.json")))
	if err != nil {
		return false
	}
	var run struct {
		Occupancy *struct {
			Active bool `json:"active"`
		} `json:"occupancy"`
	}
	if json.Unmarshal(data, &run) != nil || run.Occupancy == nil {
		return false
	}
	return run.Occupancy.Active
}

func waitOccupancy(t *testing.T, workspace string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if occupancyActive(t, workspace) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timeout waiting for occupancy")
}

func buildGobble(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gobble")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build gobble: %v\n%s", err, out)
	}
	return bin
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return -1
}
