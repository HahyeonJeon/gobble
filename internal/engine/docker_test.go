package engine

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gobexec "github.com/HahyeonJeon/gobble/internal/engine/exec"
)

const pinnedAlpine = "alpine:3.21"

const containerWorkDir = "/work"

func TestEmptyImageNeverReachesDocker(t *testing.T) {
	orig := gobexec.DockerCLI
	t.Cleanup(func() { gobexec.DockerCLI = orig })
	gobexec.DockerCLI = func(ctx context.Context, args []string, stdout, stderr io.Writer) (int, error) {
		t.Fatalf("docker invoked for empty Image: %v", args)
		return -1, errors.New("docker invoked")
	}
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	defects := Run(t.Context(), Request{
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	})
	if len(defects) != 0 {
		t.Fatalf("empty Image Run() defects %v, want none", defects)
	}
	got, err := os.ReadFile(filepath.Join(dir, "out", "sample.txt"))
	if err != nil {
		t.Fatalf("published output: %v", err)
	}
	if string(got) != "reads" {
		t.Fatalf("published output got %q, want reads", got)
	}
}

func TestRunDockerUnparseableMemory(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc(pinnedAlpine, "local", "in/sample.txt", "out/sample.txt")
	doc.Tasks[0].Resources.Memory = "not-a-size"
	defects := Run(t.Context(), Request{Workspace: dir, Document: doc})
	if !hasDefect(defects, DefectInvalidMemory, "copy") {
		t.Fatalf("unparseable memory docker Run() defects %v, want invalid-memory copy", defects)
	}
	if _, err := os.Stat(filepath.Join(dir, ControlDir)); !os.IsNotExist(err) {
		t.Fatalf("unparseable memory docker Run occupied workspace")
	}
}

func TestDockerDaemonFailureNamed(t *testing.T) {
	orig := gobexec.DockerCLI
	t.Cleanup(func() { gobexec.DockerCLI = orig })
	gobexec.DockerCLI = func(ctx context.Context, args []string, stdout, stderr io.Writer) (int, error) {
		return -1, errors.New("Cannot connect to the Docker daemon")
	}
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	defects := Run(t.Context(), Request{
		Workspace: dir,
		Document:  sampleDoc(pinnedAlpine, "local", "in/sample.txt", "out/sample.txt"),
	})
	if !hasDefect(defects, DefectFailed, "copy") {
		t.Fatalf("daemon-down Run() defects %v, want failed copy", defects)
	}
	if defects[0].Message == "" || strings.Contains(strings.ToLower(defects[0].Message), "skip") {
		t.Fatalf("daemon-down message got %q, want named failure", defects[0].Message)
	}
	if _, err := os.Stat(filepath.Join(dir, "out", "sample.txt")); !os.IsNotExist(err) {
		t.Fatalf("daemon-down published output")
	}
}
