//go:build live

package local_e2e_test

import (
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
)

func TestRNASeqRecover(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	stageRNASeqPins(t, dir)
	withSampleSheet(t, packSheet(t, rnaSheetRel))
	g, err := gobble.Compose(rnaseq.Pipeline())
	if err != nil {
		t.Fatalf("Compose(rnaseq.Pipeline()) error = %v", err)
	}
	assertMultiQCOmitsBAM(t, g)
	if err := gobble.Run(t.Context(), g, dir, 2, testOccupyOption(t)); err != nil {
		dumpTaskLogs(t, dir, "deseq2", "merge_counts")
		fatalAPIError(t, "Run(rnaseq.Pipeline())", err)
	}
	assertOccupied(t, dir)
	assertDESeq2ResultsShape(t, dir)
	assertSTARMappedAndSplices(t, dir, []string{"ctrl1", "ctrl2", "treat1", "treat2"})
	requireRegularFile(t, filepath.Join(dir, filepath.FromSlash("work/multiqc/multiqc_report.html")))
	recoverAfterSuccessAPI(t, g, dir, 2)
}
