//go:build live

package sampleretentionmapped_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	sampleretentionmapped "github.com/HahyeonJeon/gobble/assets/modules/sample-retention-mapped"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestPinnedCommandRetainsBoundary(t *testing.T) {
	task := cc.Task(t, sampleretentionmapped.Pipeline(gobble.Literal("in/Log.final.out"), 5, sampleretentionmapped.Options{}), "sample_retention_mapped")
	for _, test := range []struct {
		name       string
		log        string
		wantOutput string
		wantError  bool
	}{
		{name: "boundary", log: "Uniquely mapped reads % | 5%\n", wantOutput: "5\n"},
		{name: "below threshold", log: "Uniquely mapped reads % | 4.99%\n", wantError: true},
		{name: "missing metric", log: "Number of input reads | 100\n", wantError: true},
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
				t.Fatalf("mapped-retention image command failed: %v\n%s", err, output)
			}
			got, err := os.ReadFile(filepath.Join(dir, "work/sample-retention-mapped/mapped_reads.accepted.txt"))
			if err != nil || string(got) != test.wantOutput {
				t.Fatalf("accepted percent = %q, %v, want %q", got, err, test.wantOutput)
			}
		})
	}
}
