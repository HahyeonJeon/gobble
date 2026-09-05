//go:build live

package engine

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func requireDocker(t *testing.T) {
	t.Helper()
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Fatalf("docker info: %v", err)
	}
}

func TestRunDockerPublishes(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc(pinnedAlpine, "local", "in/sample.txt", "out/docker/sample.txt")
	doc.Tasks[0].Command = []string{"sh", "-c", "pwd > out/docker/pwd.txt && cp in/sample.txt out/docker/sample.txt"}
	doc.Tasks[0].Outputs = []IO{
		{Name: "out", Path: "out/docker/sample.txt"},
		{Name: "pwd", Path: "out/docker/pwd.txt"},
	}
	defects := Run(t.Context(), Request{
		Identity:  testInstallIdentity(),
		Workspace: dir,
		Document:  doc,
	})
	if len(defects) != 0 {
		t.Fatalf("docker Run() defects %v, want none", defects)
	}
	got, err := os.ReadFile(filepath.Join(dir, "out", "docker", "sample.txt"))
	if err != nil {
		t.Fatalf("published docker output: %v", err)
	}
	if string(got) != "reads" {
		t.Fatalf("published docker output got %q, want reads", got)
	}
	pwd, err := os.ReadFile(filepath.Join(dir, "out", "docker", "pwd.txt"))
	if err != nil {
		t.Fatalf("published pwd: %v", err)
	}
	if strings.TrimSpace(string(pwd)) != containerWorkDir {
		t.Fatalf("container cwd got %q, want %s", pwd, containerWorkDir)
	}
	raw := mustJSONFile(t, filepath.Join(dir, ControlDir, TasksFile))
	var file jsonTasksFile
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("tasks.json: %v", err)
	}
	if len(file.Tasks) != 1 {
		t.Fatalf("tasks.json tasks got %d, want 1", len(file.Tasks))
	}
	st := file.Tasks[0]
	if st.Status != StatusSucceeded || st.Executor != executorDocker {
		t.Fatalf("task state got status %q executor %q", st.Status, st.Executor)
	}
	if st.Image != pinnedAlpine {
		t.Fatalf("recorded image got %q, want %s", st.Image, pinnedAlpine)
	}
	if st.Resources.CPU != 1 || st.Resources.Memory != "512m" {
		t.Fatalf("recorded resources got cpu %v memory %q", st.Resources.CPU, st.Resources.Memory)
	}
	if st.RuntimeID != "" {
		t.Fatalf("successful Docker cleanup left runtime_id %q", st.RuntimeID)
	}
	if st.Reason == "log-copy-failed" {
		t.Fatal("successful Docker task did not preserve its logs")
	}
	for _, name := range []string{"stdout", "stderr"} {
		path := filepath.Join(dir, ControlDir, "tasks", "copy", "_", "0", "1", name)
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("Docker %s log is not a regular file: %v", name, err)
		}
	}
	if st.ImageDigest == "" {
		t.Fatalf("docker image_digest empty")
	}
}

func TestRunDockerBadImageContained(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc("gobble-missing-image:not-a-tag", "local", "in/sample.txt", "out/sample.txt")
	defects := Run(t.Context(), Request{
		Identity:  testInstallIdentity(),
		Workspace: dir,
		Document:  doc,
	})
	if !hasDefect(defects, DefectFailed, "copy") {
		t.Fatalf("bad image Run() defects %v, want failed copy", defects)
	}
	if _, err := os.Stat(filepath.Join(dir, "out", "sample.txt")); !os.IsNotExist(err) {
		t.Fatalf("failed docker output was published")
	}
	if _, err := os.Stat(filepath.Join(dir, ControlDir, "tasks", "copy", "_", "0", "1", "work")); err != nil {
		t.Fatalf("work directory after docker failure: %v", err)
	}
	raw := mustJSONFile(t, filepath.Join(dir, ControlDir, TasksFile))
	var file jsonTasksFile
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("tasks.json: %v", err)
	}
	if file.Tasks[0].Status != StatusFailed || file.Tasks[0].Error == nil || file.Tasks[0].Error.Message == "" {
		t.Fatalf("failed docker state got %#v", file.Tasks[0])
	}
	if !strings.Contains(file.Tasks[0].Error.Message, "docker") {
		t.Fatalf("failed docker message got %q, want docker named failure", file.Tasks[0].Error.Message)
	}
}
