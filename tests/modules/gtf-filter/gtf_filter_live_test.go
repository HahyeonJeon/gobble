//go:build live

package gtffilter_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	gtffilter "github.com/HahyeonJeon/gobble/assets/modules/gtf-filter"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestPinnedCommandFiltersSequenceAndEmptyTranscriptID(t *testing.T) {
	task := cc.Task(t, gtffilter.Pipeline(gobble.Literal("in/genes.gtf"), gobble.Literal("in/genome.fasta"), gtffilter.Options{}), "gtf_filter")
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "in"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "work/reference"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "in/genome.fasta"), []byte(">chr1\nACGT\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gtf := "chr1\ttest\texon\t1\t2\t.\t+\t.\tgene_id \"g1\"; transcript_id \"t1\";\n" +
		"chr1\ttest\texon\t3\t4\t.\t+\t.\tgene_id \"g2\"; transcript_id \"\";\n" +
		"chr2\ttest\texon\t1\t2\t.\t+\t.\tgene_id \"g3\"; transcript_id \"t3\";\n"
	if err := os.WriteFile(filepath.Join(dir, "in/genes.gtf"), []byte(gtf), 0o644); err != nil {
		t.Fatal(err)
	}
	args := []string{"run", "--rm", "--network", "none", "--volume", dir + ":/work", "--workdir", "/work", task.Image}
	args = append(args, task.Command...)
	if output, err := exec.Command("docker", args...).CombinedOutput(); err != nil {
		t.Fatalf("GTF filter image command failed: %v\n%s", err, output)
	}
	got, err := os.ReadFile(filepath.Join(dir, "work/reference/genes.filtered.gtf"))
	if err != nil {
		t.Fatal(err)
	}
	want := "chr1\ttest\texon\t1\t2\t.\t+\t.\tgene_id \"g1\"; transcript_id \"t1\";\n"
	if string(got) != want {
		t.Fatalf("filtered GTF = %q, want %q", got, want)
	}
}
