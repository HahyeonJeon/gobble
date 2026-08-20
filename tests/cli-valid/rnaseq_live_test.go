//go:build live

package cli_valid_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/assets"
)

const rnaSeqPkg = "./tests/cli-valid/rnaseq"

func TestRNASeqCLIRun(t *testing.T) {
	requireDocker(t)
	bin := buildGobble(t)
	dir := t.TempDir()
	stageRNASeqPins(t, dir)

	compose := runGobble(t, bin, "compose", rnaSeqPkg)
	requireSuccess(t, compose)
	if string(compose.stdout) != "{\"op\":\"compose\",\"pipeline\":\"rnaseq\"}\n" {
		t.Fatalf("compose stdout = %q, want {\"op\":\"compose\",\"pipeline\":\"rnaseq\"}\\n", compose.stdout)
	}

	validate := runGobble(t, bin, "validate", rnaSeqPkg)
	requireSuccess(t, validate)
	if string(validate.stdout) != "{\"op\":\"validate\"}\n" {
		t.Fatalf("validate stdout = %q, want {\"op\":\"validate\"}\\n", validate.stdout)
	}

	plan := runGobble(t, bin, "plan", rnaSeqPkg)
	requireSuccess(t, plan)

	run := runGobble(t, bin, "run", rnaSeqPkg, "--workspace", dir, "--cap", "2")
	requireSuccess(t, run)
	if string(run.stdout) != "{\"op\":\"run\"}\n" {
		t.Fatalf("run stdout = %q, want {\"op\":\"run\"}\\n", run.stdout)
	}

	logPath := filepath.Join(dir, filepath.FromSlash("work/star-align/Log.final.out"))
	assertUniquelyMappedAbove(t, logPath, 10)
	assertSplicesRecorded(t, logPath)
	for _, rel := range []string{
		"work/star-align/Aligned.out.bam",
		"work/multiqc/multiqc_report.html",
	} {
		requireRegularFile(t, filepath.Join(dir, filepath.FromSlash(rel)))
	}

	requireRemainingEmpty(t, bin, dir)
}

func stageRNASeqPins(t *testing.T, dir string) {
	t.Helper()
	pins := []struct {
		pin assets.Pin
		rel string
	}{
		{assets.PinRNAGenomeFASTA, "in/genome.fasta"},
		{assets.PinRNAGTF, "in/genes.gtf"},
		{assets.PinRNATest1FASTQ, "in/SRR6357072_1.fastq.gz"},
		{assets.PinRNATest2FASTQ, "in/SRR6357072_2.fastq.gz"},
	}
	for _, p := range pins {
		src, err := assets.FetchPin(p.pin)
		if err != nil {
			t.Fatalf("download %s: %v", p.pin.URL, err)
		}
		stageFile(t, dir, p.rel, src)
	}
}

func assertUniquelyMappedAbove(t *testing.T, path string, floor int) {
	t.Helper()
	n := uniquelyMappedReads(t, path)
	if n < floor {
		t.Fatalf("uniquely mapped reads = %d, want >= %d in %s", n, floor, path)
	}
}

func uniquelyMappedReads(t *testing.T, path string) int {
	t.Helper()
	return starLogInt(t, path, "Uniquely mapped reads number")
}

const starSplicesTotalField = "Number of splices: Total"

func assertSplicesRecorded(t *testing.T, path string) {
	t.Helper()
	n := starLogInt(t, path, starSplicesTotalField)
	t.Logf("%s = %d", starSplicesTotalField, n)
	if n < 1 {
		t.Fatalf("%s = %d, want >= 1 in %s", starSplicesTotalField, n, path)
	}
}

func starLogInt(t *testing.T, path, field string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, field) {
			continue
		}
		i := strings.LastIndex(line, "|")
		if i < 0 {
			t.Fatalf("%s line %q: missing |", field, line)
		}
		n, err := strconv.Atoi(strings.TrimSpace(line[i+1:]))
		if err != nil {
			t.Fatalf("%s line %q: %v", field, line, err)
		}
		return n
	}
	t.Fatalf("%s: missing %s", path, field)
	return 0
}
