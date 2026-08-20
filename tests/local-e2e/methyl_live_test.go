//go:build live

package local_e2e_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets"
)

func TestMethylSeqRecover(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	stageMethylPins(t, dir)
	withSampleSheet(t, packSheet(t, methylSheetRel))
	g, err := gobble.Compose(assets.MethylSeq())
	if err != nil {
		t.Fatalf("Compose(assets.MethylSeq()) error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1); err != nil {
		t.Fatalf("Run(assets.MethylSeq()) error = %v", err)
	}
	assertOccupied(t, dir)
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash("in/Bisulfite_Genome"))); !os.IsNotExist(err) {
		t.Fatalf("Bisulfite_Genome written into in/: %v", err)
	}
	assertMethylExtractorOutputs(t, dir, []string{"sample1", "sample2"})
	requireRegularFile(t, filepath.Join(dir, filepath.FromSlash("work/multiqc/multiqc_report.html")))
	for _, sample := range []string{"sample1", "sample2"} {
		unique := uniquePEAlignments(t, filepath.Join(dir, filepath.FromSlash("work/"+sample+"/bismark-align/aligned_PE_report.txt")))
		t.Logf("%s unique paired-end alignments = %d", sample, unique)
		assertUniqueAlignmentFloor(t, unique)
		assertMethylationCallRows(t, unique,
			filepath.Join(dir, filepath.FromSlash("work/"+sample+"/bismark-extract/CpG_context_aligned_pe.txt.gz")),
			filepath.Join(dir, filepath.FromSlash("work/"+sample+"/bismark-extract/aligned_pe.bismark.cov.gz")),
		)
	}
	recoverAfterSuccessAPI(t, g, dir, 1)
}
