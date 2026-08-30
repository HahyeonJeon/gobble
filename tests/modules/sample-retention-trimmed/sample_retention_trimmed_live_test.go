//go:build live

package sampleretentiontrimmed_test

import (
	"compress/gzip"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	sampleretentiontrimmed "github.com/HahyeonJeon/gobble/assets/modules/sample-retention-trimmed"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestPinnedCommandRetainsBoundary(t *testing.T) {
	task := cc.Task(t, sampleretentiontrimmed.Pipeline(gobble.Literal("in/read.fastq.gz"), 2, sampleretentiontrimmed.Options{}), "sample_retention_trimmed")
	for _, test := range []struct {
		name       string
		body       string
		wantOutput string
		wantError  bool
	}{
		{name: "boundary", body: "@a\nA\n+\n!\n@b\nA\n+\n!\n", wantOutput: "2\n"},
		{name: "below threshold", body: "@a\nA\n+\n!\n", wantError: true},
		{name: "malformed FASTQ", body: "@a\nA\n+\n", wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			writeGzip(t, filepath.Join(dir, "in/read.fastq.gz"), test.body)
			if err := os.MkdirAll(filepath.Join(dir, "work/sample-retention-trimmed"), 0o755); err != nil {
				t.Fatal(err)
			}
			args := []string{"run", "--rm", "--network", "none", "--volume", dir + ":/work", "--workdir", "/work", task.Image}
			args = append(args, task.Command...)
			output, err := exec.Command("docker", args...).CombinedOutput()
			if test.wantError {
				if err == nil {
					t.Fatalf("image command accepted invalid input; output=%s", output)
				}
				return
			}
			if err != nil {
				t.Fatalf("trimmed-retention image command failed: %v\n%s", err, output)
			}
			got, err := os.ReadFile(filepath.Join(dir, "work/sample-retention-trimmed/trimmed_reads.accepted.txt"))
			if err != nil || string(got) != test.wantOutput {
				t.Fatalf("accepted count = %q, %v, want %q", got, err, test.wantOutput)
			}
		})
	}
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
	t.Cleanup(func() { _ = file.Close() })
	zw := gzip.NewWriter(file)
	t.Cleanup(func() { _ = zw.Close() })
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
