package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestInitCreatesBuildableProjectWithoutReplacingExistingFiles(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOBBLE_SOURCE", root)
	project := filepath.Join(t.TempDir(), "My pipeline 한글")
	var out, stderr bytes.Buffer
	if code := runInit(&request{pkg: project}, &out, &stderr); code != 0 {
		t.Fatalf("init: %s", stderr.String())
	}
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = project
	if data, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated pipeline: %v\n%s", err, data)
	}
	input, err := os.ReadFile(filepath.Join(project, "runs", "hello", "inputs", "sequences.fasta"))
	if err != nil || bytes.Count(input, []byte(">")) != 2 {
		t.Fatalf("teaching input: %s %v", input, err)
	}
	before, err := os.ReadFile(filepath.Join(project, "pipeline.go"))
	if err != nil {
		t.Fatal(err)
	}
	if code := runInit(&request{pkg: project}, &out, &stderr); code == 0 {
		t.Fatal("existing project replaced")
	}
	after, err := os.ReadFile(filepath.Join(project, "pipeline.go"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("existing pipeline changed")
	}
}
