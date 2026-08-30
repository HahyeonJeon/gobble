package sampleretentiontrimmed_test

import (
	"compress/gzip"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	sampleretentiontrimmed "github.com/HahyeonJeon/gobble/assets/modules/sample-retention-trimmed"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContractAndBoundary(t *testing.T) {
	read := gobble.Literal("in/read.fastq.gz")
	task := cc.Task(t, sampleretentiontrimmed.Pipeline(read, 2, sampleretentiontrimmed.Options{}), "sample_retention_trimmed")
	if !strings.Contains(task.Script, "minimum=2") || !strings.Contains(task.Script, "NR % 4") {
		t.Fatalf("script = %q, want complete-FASTQ boundary at two records", task.Script)
	}
	pc.AssertIOPath(t, task.Outputs, "accepted", "work/sample-retention-trimmed/trimmed_reads.accepted.txt")

	dir := t.TempDir()
	writeGzip(t, filepath.Join(dir, "in/read.fastq.gz"), "@a\nA\n+\n!\n@b\nA\n+\n!\n")
	if err := os.MkdirAll(filepath.Join(dir, "work/sample-retention-trimmed"), 0o755); err != nil {
		t.Fatal(err)
	}
	runScript(t, dir, task.Script)
	got, err := os.ReadFile(filepath.Join(dir, "work/sample-retention-trimmed/trimmed_reads.accepted.txt"))
	if err != nil || string(got) != "2\n" {
		t.Fatalf("accepted count = %q, %v, want 2", got, err)
	}
	for _, test := range []struct {
		name string
		body string
	}{
		{name: "below threshold", body: "@a\nA\n+\n!\n"},
		{name: "malformed FASTQ", body: "@a\nA\n+\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			writeGzip(t, filepath.Join(dir, "in/read.fastq.gz"), test.body)
			if err := os.MkdirAll(filepath.Join(dir, "work/sample-retention-trimmed"), 0o755); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("sh", "-c", "set -eu\n"+task.Script)
			command.Dir = dir
			if output, err := command.CombinedOutput(); err == nil {
				t.Fatalf("gate accepted invalid input; output=%s", output)
			}
		})
	}

	cc.Invalid(t, sampleretentiontrimmed.Pipeline(read, -1, sampleretentiontrimmed.Options{}))
	cc.Invalid(t, sampleretentiontrimmed.Pipeline(read, 2, sampleretentiontrimmed.Options{Options: modules.Options{ExtraArgs: []string{"--bypass"}}}))
}

func writeGzip(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := gzip.NewWriter(file)
	if _, err := zw.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func runScript(t *testing.T, dir, script string) {
	t.Helper()
	command := exec.Command("sh", "-c", "set -eu\n"+script)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("script failed: %v\n%s\n%s", err, output, script)
	}
}
