//go:build live

package cli_valid_test

import (
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/assets"
)

const methylSeqPkg = "./tests/cli-valid/methylseq"

func TestMethylSeqCLIRun(t *testing.T) {
	requireDocker(t)
	bin := buildGobble(t)
	dir := t.TempDir()
	stageMethylPins(t, dir)

	compose := runGobble(t, bin, "compose", methylSeqPkg)
	requireSuccess(t, compose)
	if string(compose.stdout) != "{\"op\":\"compose\",\"pipeline\":\"methylseq\"}\n" {
		t.Fatalf("compose stdout = %q, want {\"op\":\"compose\",\"pipeline\":\"methylseq\"}\\n", compose.stdout)
	}

	validate := runGobble(t, bin, "validate", methylSeqPkg)
	requireSuccess(t, validate)
	if string(validate.stdout) != "{\"op\":\"validate\"}\n" {
		t.Fatalf("validate stdout = %q, want {\"op\":\"validate\"}\\n", validate.stdout)
	}

	plan := runGobble(t, bin, "plan", methylSeqPkg)
	requireSuccess(t, plan)

	run := runGobble(t, bin, "run", methylSeqPkg, "--workspace", dir, "--cap", "1")
	requireSuccess(t, run)
	if string(run.stdout) != "{\"op\":\"run\"}\n" {
		t.Fatalf("run stdout = %q, want {\"op\":\"run\"}\\n", run.stdout)
	}

	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash("in/Bisulfite_Genome"))); !os.IsNotExist(err) {
		t.Fatalf("Bisulfite_Genome written into in/: %v", err)
	}
	unique := uniquePEAlignments(t, filepath.Join(dir, filepath.FromSlash("work/bismark-align/aligned_PE_report.txt")))
	t.Logf("unique paired-end alignments = %d", unique)
	assertUniqueAlignmentFloor(t, unique)
	assertMethylationCallRows(t, unique,
		filepath.Join(dir, filepath.FromSlash("work/bismark-extractor/CpG_context_aligned_pe.txt.gz")),
		filepath.Join(dir, filepath.FromSlash("work/bismark-extractor/aligned_pe.bismark.cov.gz")),
	)

	requireRemainingEmpty(t, bin, dir)
}

func stageMethylPins(t *testing.T, dir string) {
	t.Helper()
	pins := []struct {
		pin assets.Pin
		rel string
	}{
		{assets.PinMethylGenomeFASTA, "in/genome.fa"},
		{assets.PinMethylTest1FASTQ, "in/Ecoli_10K_methylated_R1.fastq.gz"},
		{assets.PinMethylTest2FASTQ, "in/Ecoli_10K_methylated_R2.fastq.gz"},
	}
	for _, p := range pins {
		src, err := assets.FetchPin(p.pin)
		if err != nil {
			t.Fatalf("download %s: %v", p.pin.URL, err)
		}
		stageFile(t, dir, p.rel, src)
	}
}

const uniquePEAlignmentField = "Number of paired-end alignments with a unique best hit:"

func uniquePEAlignments(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, uniquePEAlignmentField) {
			continue
		}
		_, value, ok := strings.Cut(line, ":")
		if !ok {
			t.Fatalf("unique-alignment line %q missing value", line)
		}
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			t.Fatalf("unique-alignment count in %q: %v", line, err)
		}
		return n
	}
	t.Fatalf("%s missing unique-alignment field %q", path, uniquePEAlignmentField)
	return 0
}

func assertUniqueAlignmentFloor(t *testing.T, unique int) {
	t.Helper()
	if unique < 1 {
		t.Fatalf("unique paired-end alignments = %d, want floor > 0", unique)
	}
}

func assertMethylationCallRows(t *testing.T, unique int, paths ...string) {
	t.Helper()
	rows := 0
	for _, path := range paths {
		n := methylationCallRows(t, path)
		t.Logf("methylation call rows in %s = %d", filepath.Base(path), n)
		rows += n
	}
	if unique > 0 && rows == 0 {
		t.Fatalf("no methylation call row in %v", paths)
	}
}

func methylationCallRows(t *testing.T, path string) int {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open(%s): %v", path, err)
	}
	defer f.Close()
	var r io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			t.Fatalf("gzip %s: %v", path, err)
		}
		defer gz.Close()
		r = gz
	}
	sc := bufio.NewScanner(r)
	n := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "Bismark") || strings.HasPrefix(line, "#") {
			continue
		}
		if len(strings.Fields(line)) < 4 {
			continue
		}
		n++
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan %s: %v", path, err)
	}
	return n
}
