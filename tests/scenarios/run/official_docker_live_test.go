//go:build live

package run_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
	methylseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/methylseq"
	rnaseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/rnaseq"
	wgsevidence "github.com/HahyeonJeon/gobble/tests/pipelines/wgs"
)

func TestOfficialWGSDockerRunPublishesFinals(t *testing.T) {
	requireOfficialDocker(t)
	root := officialModuleRoot(t)
	workspace := t.TempDir()
	if _, err := wgsevidence.StageOfficial(filepath.Join(root, filepath.FromSlash(wgsevidence.CacheDir)), workspace); err != nil {
		t.Fatalf("StageOfficial WGS: %v", err)
	}
	samples, err := wgs.Load(filepath.Join(root, filepath.FromSlash(wgsevidence.FixtureSheet)))
	if err != nil {
		t.Fatal(err)
	}
	runOfficialGraph(t, wgs.Build(samples, wgs.DefaultConfig()), workspace, 2)
	requireOfficialFiles(t, workspace,
		"results/wgs/samples/patient1/testN/alignment/testN.recalibrated.bam",
		"results/wgs/samples/patient2/testT/alignment/testT.recalibrated.bam",
		"results/wgs/joint/joint_germline.vcf.gz",
		"results/wgs/multiqc/multiqc_report.html",
	)
}

func TestOfficialRNASeqDockerRunPublishesFinals(t *testing.T) {
	requireOfficialDocker(t)
	root := officialModuleRoot(t)
	workspace := t.TempDir()
	if _, err := rnaseqevidence.StageOfficial(filepath.Join(root, filepath.FromSlash(rnaseqevidence.CacheDir)), workspace); err != nil {
		t.Fatalf("StageOfficial RNA-seq: %v", err)
	}
	samples, err := rnaseq.Load(filepath.Join(root, filepath.FromSlash(rnaseqevidence.LiveFixtureSheet)))
	if err != nil {
		t.Fatal(err)
	}
	runOfficialGraph(t, rnaseq.Build(samples, rnaseq.DefaultConfig()), workspace, 2)
	requireOfficialFiles(t, workspace,
		"results/rnaseq/bam/WT_REP1/WT_REP1.marked.bam",
		"results/rnaseq/matrices/gene_counts.tsv",
		"results/rnaseq/matrices/transcript_tpm.tsv",
		"results/rnaseq/multiqc/multiqc_report.html",
	)
}

func TestOfficialMethylSeqDockerRunPublishesFinals(t *testing.T) {
	requireOfficialDocker(t)
	root := officialModuleRoot(t)
	workspace := t.TempDir()
	if _, err := methylseqevidence.StageOfficial(filepath.Join(root, filepath.FromSlash(methylseqevidence.CacheDir)), workspace); err != nil {
		t.Fatalf("StageOfficial Methyl-seq: %v", err)
	}
	samples, err := methylseq.Load(filepath.Join(root, filepath.FromSlash(methylseqevidence.FixtureSheet)))
	if err != nil {
		t.Fatal(err)
	}
	runOfficialGraph(t, methylseq.Build(samples, methylseq.DefaultConfig()), workspace, 1)
	requireOfficialFiles(t, workspace,
		"results/methylseq/bismark/Ecoli_10K_methylated/Ecoli_10K_methylated_pe.deduplicated.bam",
		"results/methylseq/methylation-calls/Ecoli_10K_methylated/CpG_context_Ecoli_10K_methylated_pe.deduplicated.txt.gz",
		"results/methylseq/summary/bismark_summary_report.html",
		"results/methylseq/multiqc/multiqc_report.html",
	)
}

func runOfficialGraph(t *testing.T, pipeline *gobble.Pipeline, workspace string, cap int) {
	t.Helper()
	graph, err := gobble.Compose(pipeline)
	if err != nil {
		t.Fatalf("Compose official graph: %v", err)
	}
	identity, err := gobble.IdentityFromBuildInfo("github.com/HahyeonJeon/gobble/tests/scenarios/run")
	if err != nil {
		t.Fatalf("IdentityFromBuildInfo: %v", err)
	}
	if err := gobble.Run(t.Context(), graph, workspace, cap, gobble.WithIdentity(identity)); err != nil {
		t.Fatalf("Run official graph: %v", err)
	}
}

func requireOfficialFiles(t *testing.T, workspace string, paths ...string) {
	t.Helper()
	for _, path := range paths {
		info, err := os.Stat(filepath.Join(workspace, filepath.FromSlash(path)))
		if err != nil || !info.Mode().IsRegular() {
			t.Errorf("official final %s: %v", path, err)
		}
	}
}

func requireOfficialDocker(t *testing.T) {
	t.Helper()
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Fatalf("docker info: %v", err)
	}
}

func officialModuleRoot(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			t.Fatalf("go.mod not found from %s", directory)
		}
		directory = parent
	}
}
