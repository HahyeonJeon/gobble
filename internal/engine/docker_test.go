package engine

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func requireDocker(t *testing.T) {
	t.Helper()
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skipf("docker info: %v", err)
	}
}

const pinnedAlpine = "alpine:3.21"

func TestDockerRunArgs(t *testing.T) {
	task := TaskPlan{
		Image:   pinnedAlpine,
		Command: []string{"cp", "in/sample.txt", "out/docker/sample.txt"},
	}
	args := dockerRunArgs("/iso", task, task.Command)
	joined := strings.Join(args, " ")
	for _, banned := range []string{"--cpus", "--memory"} {
		if strings.Contains(joined, banned) {
			t.Fatalf("docker argv %v contains %s", args, banned)
		}
	}
	want := []string{
		"run", "--rm",
		"--user", strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid()),
		"--network=none",
		"--entrypoint", "cp",
		"-v", "/iso:" + containerWorkDir,
		"-w", containerWorkDir,
		pinnedAlpine,
		"in/sample.txt", "out/docker/sample.txt",
	}
	if len(args) != len(want) {
		t.Fatalf("docker argv got %#v, want %#v", args, want)
	}
	for i, arg := range want {
		if args[i] != arg {
			t.Fatalf("docker argv got %#v, want %#v", args, want)
		}
	}
}

func TestEmptyImageNeverReachesDocker(t *testing.T) {
	orig := dockerCLI
	t.Cleanup(func() { dockerCLI = orig })
	dockerCLI = func(args []string, stdout, stderr io.Writer) (int, error) {
		t.Fatalf("docker invoked for empty Image: %v", args)
		return -1, errors.New("docker invoked")
	}
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	defects := Run(Request{
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
	defects := Run(Request{Workspace: dir, Document: doc})
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
}

func TestRunDockerBadImageContained(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc("gobble-missing-image:not-a-tag", "local", "in/sample.txt", "out/sample.txt")
	defects := Run(Request{Workspace: dir, Document: doc})
	if !hasDefect(defects, DefectFailed, "copy") {
		t.Fatalf("bad image Run() defects %v, want failed copy", defects)
	}
	if _, err := os.Stat(filepath.Join(dir, "out", "sample.txt")); !os.IsNotExist(err) {
		t.Fatalf("failed docker output was published")
	}
	if _, err := os.Stat(filepath.Join(dir, ControlDir, "tasks", "copy", "work")); err != nil {
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

func TestRunDockerUnparseableMemory(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	doc := sampleDoc(pinnedAlpine, "local", "in/sample.txt", "out/sample.txt")
	doc.Tasks[0].Resources.Memory = "not-a-size"
	defects := Run(Request{Workspace: dir, Document: doc})
	if !hasDefect(defects, DefectInvalidMemory, "copy") {
		t.Fatalf("unparseable memory docker Run() defects %v, want invalid-memory copy", defects)
	}
	if _, err := os.Stat(filepath.Join(dir, ControlDir)); !os.IsNotExist(err) {
		t.Fatalf("unparseable memory docker Run occupied workspace")
	}
}

func TestDockerRunArgsNonZeroResources(t *testing.T) {
	task := TaskPlan{
		Image:     pinnedAlpine,
		Command:   []string{"true"},
		Resources: ResourcePlan{CPU: 1.5, Memory: "512m"},
		Env:       map[string]string{"HOME": "/tmp", "FOO": "bar"},
	}
	args := dockerRunArgs("/iso", task, task.Command)
	if !hasArgPair(args, "--cpus", "1.5") {
		t.Fatalf("non-zero docker argv %v, want --cpus 1.5", args)
	}
	if !hasArgPair(args, "--memory", "512m") {
		t.Fatalf("non-zero docker argv %v, want --memory 512m", args)
	}
	if !hasArgPair(args, "-e", "HOME=/tmp") || !hasArgPair(args, "-e", "FOO=bar") {
		t.Fatalf("non-zero docker argv %v, want -e HOME=/tmp and -e FOO=bar", args)
	}
	for i, arg := range args {
		if arg == "-e" && i+1 < len(args) && !strings.Contains(args[i+1], "=") {
			t.Fatalf("value-less -e in %v", args)
		}
	}
}

func TestDockerRunArgsZeroResourcesOmitFlags(t *testing.T) {
	task := TaskPlan{
		Image:     pinnedAlpine,
		Command:   []string{"true"},
		Resources: ResourcePlan{CPU: 0, Memory: ""},
	}
	args := dockerRunArgs("/iso", task, task.Command)
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "--cpus") || strings.Contains(joined, "--memory") {
		t.Fatalf("zero docker argv %v contains resource flags", args)
	}
	task.Resources.Memory = "0m"
	args = dockerRunArgs("/iso", task, task.Command)
	if strings.Contains(strings.Join(args, " "), "--memory") {
		t.Fatalf("zero-memory docker argv %v contains --memory", args)
	}
}

func hasArgPair(args []string, flag, value string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}

func TestDockerDaemonFailureNamed(t *testing.T) {
	orig := dockerCLI
	t.Cleanup(func() { dockerCLI = orig })
	dockerCLI = func(args []string, stdout, stderr io.Writer) (int, error) {
		return -1, errors.New("Cannot connect to the Docker daemon")
	}
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	defects := Run(Request{
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
