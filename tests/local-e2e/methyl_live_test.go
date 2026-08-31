//go:build live

package local_e2e_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
)

func TestMethylSeqRecover(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	stageMethylPins(t, dir)
	withSampleSheet(t, packSheet(t, methylSheetRel))
	g, err := gobble.Compose(methylseq.Pipeline())
	if err != nil {
		t.Fatalf("Compose(methylseq.Pipeline()) error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1, testOccupyOption(t)); err != nil {
		fatalAPIError(t, "Run(methylseq.Pipeline())", err)
	}
	assertOccupied(t, dir)
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash("in/reference/Bisulfite_Genome"))); !os.IsNotExist(err) {
		t.Fatalf("Bisulfite_Genome written into caller input: %v", err)
	}
	assertMethylExtractorOutputs(t, dir, []string{"SRR389222_sub1", "SRR389222_sub2", "Ecoli_10K_methylated"})
	requireRegularFile(t, filepath.Join(dir, filepath.FromSlash("results/methylseq/summary/bismark_summary_report.html")))
	requireRegularFile(t, filepath.Join(dir, filepath.FromSlash("results/methylseq/multiqc/multiqc_report.html")))
	unique := uniquePEAlignments(t, filepath.Join(dir, filepath.FromSlash("work/Ecoli_10K_methylated/bismark-align/Ecoli_10K_methylated_PE_report.txt")))
	t.Logf("Ecoli_10K_methylated unique paired-end alignments = %d", unique)
	assertUniqueAlignmentFloor(t, unique)
	assertMethylationCallRows(t, unique,
		filepath.Join(dir, filepath.FromSlash("results/methylseq/methylation-calls/Ecoli_10K_methylated/CpG_context_Ecoli_10K_methylated_pe.deduplicated.txt.gz")),
		filepath.Join(dir, filepath.FromSlash("results/methylseq/methylation-calls/Ecoli_10K_methylated/Ecoli_10K_methylated_pe.deduplicated.bismark.cov.gz")),
	)
	recoverAfterSuccessAPI(t, g, dir, 1)
}

func TestMethylSeqReadyIndexRecover(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	stageMethylPins(t, dir)
	stageMethylReadyIndex(t, dir)
	samples, err := methylseq.Load(packSheet(t, methylSheetRel))
	if err != nil {
		t.Fatalf("Load Methyl fixture: %v", err)
	}
	config := methylseq.DefaultConfig()
	config.Reference.FASTA = gobble.PathSpec{}
	config.Reference.BismarkIndex = gobble.DeclareTree(gobble.Dir("in/reference/BismarkIndex"))
	t.Chdir(dir)
	g, err := gobble.Compose(methylseq.Build(samples, config))
	if err != nil {
		t.Fatalf("Compose ready-index Methyl graph: %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1, testOccupyOption(t)); err != nil {
		fatalAPIError(t, "Run ready-index Methyl graph", err)
	}
	assertMethylExtractorOutputs(t, dir, []string{"SRR389222_sub1", "SRR389222_sub2", "Ecoli_10K_methylated"})
	requireRegularFile(t, filepath.Join(dir, filepath.FromSlash("results/methylseq/summary/bismark_summary_report.html")))
	requireRegularFile(t, filepath.Join(dir, filepath.FromSlash("results/methylseq/multiqc/multiqc_report.html")))
	recoverAfterSuccessAPI(t, g, dir, 1)
}
