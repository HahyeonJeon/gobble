//go:build live

package local_e2e_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunLocalCLIRecover(t *testing.T) {
	requireDocker(t)
	bin := buildGobble(t)
	dir := t.TempDir()
	copyRunLocalInput(t, dir)

	compose := runGobble(t, bin, "compose", runLocalPkg)
	requireCLIOp(t, compose, "{\"op\":\"compose\",\"pipeline\":\"run-local\"}\n")

	validate := runGobble(t, bin, "validate", runLocalPkg)
	requireCLIOp(t, validate, "{\"op\":\"validate\"}\n")

	plan := runGobble(t, bin, "plan", runLocalPkg)
	requireSuccess(t, plan)
	wantPlan := readModuleFile(t, "testdata/run-local/plan.json")
	if !bytes.Equal(plan.stdout, wantPlan) {
		t.Fatalf("plan stdout != testdata/run-local/plan.json\ngot:\n%s\nwant:\n%s", plan.stdout, wantPlan)
	}

	run := runGobble(t, bin, "run", runLocalPkg, "--workspace", dir, "--cap", "2")
	requireCLIOp(t, run, "{\"op\":\"run\"}\n")

	requireFixtureText(t, filepath.Join(dir, "out", "docker", "sample.txt"))
	requireFixtureText(t, filepath.Join(dir, "out", "process", "sample.txt"))
	pwd, err := os.ReadFile(filepath.Join(dir, "out", "docker", "pwd.txt"))
	if err != nil {
		t.Fatalf("published container cwd: %v", err)
	}
	if strings.TrimSpace(string(pwd)) != "/work" {
		t.Fatalf("container cwd got %q, want /work", pwd)
	}

	recoverAfterSuccessCLI(t, bin, runLocalPkg, dir, "--cap", "2")
}

func TestWGSCLIRecover(t *testing.T) {
	requireDocker(t)
	bin := buildGobble(t)
	dir := t.TempDir()
	stageWGSPins(t, dir)

	compose := runGobble(t, bin, "compose", wgsPkg)
	requireCLIOp(t, compose, "{\"op\":\"compose\",\"pipeline\":\"wgs\"}\n")

	validate := runGobble(t, bin, "validate", wgsPkg)
	requireCLIOp(t, validate, "{\"op\":\"validate\"}\n")

	plan := runGobble(t, bin, "plan", wgsPkg)
	requireSuccess(t, plan)

	run := runGobble(t, bin, "run", wgsPkg, "--workspace", dir, "--cap", "2")
	requireCLIOp(t, run, "{\"op\":\"run\"}\n")

	for _, rel := range []string{
		"work/multiqc/multiqc_report.html",
		"work/sample1/samtools-sort/aligned.bam",
		"work/sample2/samtools-sort/aligned.bam",
	} {
		requireRegularFile(t, filepath.Join(dir, filepath.FromSlash(rel)))
	}

	recoverAfterSuccessCLI(t, bin, wgsPkg, dir, "--cap", "2")
}

func TestRNASeqCLIRecover(t *testing.T) {
	requireDocker(t)
	bin := buildGobble(t)
	dir := t.TempDir()
	stageRNASeqPins(t, dir)
	sheet := packSheet(t, rnaSheetRel)

	compose := runGobble(t, bin, "compose", rnaSeqPkg, "--sample", sheet)
	requireCLIOp(t, compose, "{\"op\":\"compose\",\"pipeline\":\"rnaseq\"}\n")

	validate := runGobble(t, bin, "validate", rnaSeqPkg, "--sample", sheet)
	requireCLIOp(t, validate, "{\"op\":\"validate\"}\n")

	plan := runGobble(t, bin, "plan", rnaSeqPkg, "--sample", sheet)
	requireSuccess(t, plan)

	run := runGobble(t, bin, "run", rnaSeqPkg, "--workspace", dir, "--cap", "2", "--sample", sheet)
	if run.code != 0 {
		dumpTaskLogs(t, dir, "salmon_quant", "tximport", "deseq2_qc", "multiqc")
	}
	requireCLIOp(t, run, "{\"op\":\"run\"}\n")

	assertRNAProductOutputs(t, dir)
	assertSTARMappedAndSplices(t, dir, []string{"WT_REP1", "WT_REP2", "RAP1_UNINDUCED_REP1", "RAP1_UNINDUCED_REP2", "RAP1_IAA_30M_REP1"})

	recoverAfterSuccessCLI(t, bin, rnaSeqPkg, dir, "--cap", "2", "--sample", sheet)
}

func TestMethylSeqCLIRecover(t *testing.T) {
	requireDocker(t)
	bin := buildGobble(t)
	dir := t.TempDir()
	stageMethylPins(t, dir)
	sheet := packSheet(t, methylSheetRel)

	compose := runGobble(t, bin, "compose", methylSeqPkg, "--sample", sheet)
	requireCLIOp(t, compose, "{\"op\":\"compose\",\"pipeline\":\"methylseq\"}\n")

	validate := runGobble(t, bin, "validate", methylSeqPkg, "--sample", sheet)
	requireCLIOp(t, validate, "{\"op\":\"validate\"}\n")

	plan := runGobble(t, bin, "plan", methylSeqPkg, "--sample", sheet)
	requireSuccess(t, plan)

	run := runGobble(t, bin, "run", methylSeqPkg, "--workspace", dir, "--cap", "1", "--sample", sheet)
	requireCLIOp(t, run, "{\"op\":\"run\"}\n")

	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash("in/reference/Bisulfite_Genome"))); !os.IsNotExist(err) {
		t.Fatalf("Bisulfite_Genome written into caller input: %v", err)
	}
	assertMethylExtractorOutputs(t, dir, []string{"SRR389222_sub1", "SRR389222_sub2", "Ecoli_10K_methylated"})
	requireRegularFile(t, filepath.Join(dir, filepath.FromSlash("results/methylseq/summary/bismark_summary_report.html")))
	requireRegularFile(t, filepath.Join(dir, filepath.FromSlash("results/methylseq/multiqc/multiqc_report.html")))

	recoverAfterSuccessCLI(t, bin, methylSeqPkg, dir, "--cap", "1", "--sample", sheet)
}
