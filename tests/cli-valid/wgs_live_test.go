//go:build live

package cli_valid_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble/assets"
)

const wgsPkg = "./tests/cli-valid/wgs"

func TestWGSCLIRecover(t *testing.T) {
	requireDocker(t)
	bin := buildGobble(t)
	dir := t.TempDir()
	stageWGSPins(t, dir)

	compose := runGobble(t, bin, "compose", wgsPkg)
	requireSuccess(t, compose)
	if string(compose.stdout) != "{\"op\":\"compose\",\"pipeline\":\"wgs\"}\n" {
		t.Fatalf("compose stdout = %q, want {\"op\":\"compose\",\"pipeline\":\"wgs\"}\\n", compose.stdout)
	}

	validate := runGobble(t, bin, "validate", wgsPkg)
	requireSuccess(t, validate)
	if string(validate.stdout) != "{\"op\":\"validate\"}\n" {
		t.Fatalf("validate stdout = %q, want {\"op\":\"validate\"}\\n", validate.stdout)
	}

	plan := runGobble(t, bin, "plan", wgsPkg)
	requireSuccess(t, plan)

	run := runGobble(t, bin, "run", wgsPkg, "--workspace", dir, "--cap", "2")
	requireSuccess(t, run)
	if string(run.stdout) != "{\"op\":\"run\"}\n" {
		t.Fatalf("run stdout = %q, want {\"op\":\"run\"}\\n", run.stdout)
	}

	for _, rel := range []string{
		"work/multiqc/multiqc_report.html",
		"work/sample1/samtools-sort/aligned.bam",
		"work/sample2/samtools-sort/aligned.bam",
	} {
		requireRegularFile(t, filepath.Join(dir, filepath.FromSlash(rel)))
	}

	requireRemainingEmpty(t, bin, dir)

	occupied := runGobble(t, bin, "run", wgsPkg, "--workspace", dir, "--cap", "2")
	requireOccupiedWorkspace(t, occupied)

	inspectRun := runGobble(t, bin, "inspect", "run", "--workspace", dir)
	requireSuccess(t, inspectRun)
	if !occupancyActive(inspectRun.stdout) {
		t.Fatalf("inspect run occupancy inactive: %s", inspectRun.stdout)
	}

	releaseWorkspace(t, bin, dir)

	resume := runGobble(t, bin, "resume", wgsPkg, "--workspace", dir, "--cap", "2")
	requireSuccess(t, resume)
	if string(resume.stdout) != "{\"op\":\"resume\"}\n" {
		t.Fatalf("resume stdout = %q, want {\"op\":\"resume\"}\\n", resume.stdout)
	}

	requireRemainingEmpty(t, bin, dir)
	requireReuseIdentityMatched(t, bin, dir)
}

func stageWGSPins(t *testing.T, dir string) {
	t.Helper()
	pins := []struct {
		pin assets.Pin
		rel string
	}{
		{assets.PinWGSTest1FASTQ, "in/test_1.fastq.gz"},
		{assets.PinWGSTest2FASTQ, "in/test_2.fastq.gz"},
		{assets.PinWGSGenomeFASTA, "in/genome.fasta"},
		{assets.PinWGSGenomeFAI, "in/genome.fasta.fai"},
	}
	for _, p := range pins {
		src, err := assets.FetchPin(p.pin)
		if err != nil {
			t.Fatalf("download %s: %v", p.pin.URL, err)
		}
		stageFile(t, dir, p.rel, src)
	}
}

func stageFile(t *testing.T, workspace, rel, src string) {
	t.Helper()
	dst := filepath.Join(workspace, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", dst, err)
	}
	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("Open(%s) error = %v", src, err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatalf("Create(%s) error = %v", dst, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		t.Fatalf("copy %s: %v", dst, err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("Close(%s) error = %v", dst, err)
	}
}

func requireRegularFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("published %s: %v", path, err)
	}
}
