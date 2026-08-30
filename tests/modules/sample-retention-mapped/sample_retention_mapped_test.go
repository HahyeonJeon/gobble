package sampleretentionmapped_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	sampleretentionmapped "github.com/HahyeonJeon/gobble/assets/modules/sample-retention-mapped"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContractAndBoundary(t *testing.T) {
	starLog := gobble.Literal("in/Log.final.out")
	task := cc.Task(t, sampleretentionmapped.Pipeline(starLog, 5, sampleretentionmapped.Options{}), "sample_retention_mapped")
	if !strings.Contains(task.Script, "Uniquely mapped reads %") || !strings.Contains(task.Script, "minimum=5") {
		t.Fatalf("script = %q, want STAR uniquely-mapped boundary at five percent", task.Script)
	}
	pc.AssertIOPath(t, task.Outputs, "accepted", "work/sample-retention-mapped/mapped_reads.accepted.txt")

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "in"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "in/Log.final.out"), []byte("Uniquely mapped reads % | 5%\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "work/sample-retention-mapped"), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("sh", "-c", "set -eu\n"+task.Script)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("script failed: %v\n%s\n%s", err, output, task.Script)
	}
	got, err := os.ReadFile(filepath.Join(dir, "work/sample-retention-mapped/mapped_reads.accepted.txt"))
	if err != nil || string(got) != "5\n" {
		t.Fatalf("accepted percent = %q, %v, want 5", got, err)
	}
	for _, test := range []struct {
		name string
		log  string
	}{
		{name: "below threshold", log: "Uniquely mapped reads % | 4.99%\n"},
		{name: "missing metric", log: "Number of input reads | 100\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "in"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "in/Log.final.out"), []byte(test.log), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(dir, "work/sample-retention-mapped"), 0o755); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("sh", "-c", "set -eu\n"+task.Script)
			command.Dir = dir
			if output, err := command.CombinedOutput(); err == nil {
				t.Fatalf("gate accepted invalid input; output=%s", output)
			}
		})
	}

	cc.Invalid(t, sampleretentionmapped.Pipeline(starLog, 101, sampleretentionmapped.Options{}))
	cc.Invalid(t, sampleretentionmapped.Pipeline(starLog, 5, sampleretentionmapped.Options{Options: modules.Options{ExtraArgs: []string{"--bypass"}}}))
}
