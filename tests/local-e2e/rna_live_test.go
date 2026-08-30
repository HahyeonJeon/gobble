//go:build live

package local_e2e_test

import (
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
		dumpTaskLogs(t, dir, "salmon_quant", "tximport", "deseq2_qc", "multiqc")
		fatalAPIError(t, "Run(rnaseq.Pipeline())", err)
	}
	assertOccupied(t, dir)
	assertRNAProductOutputs(t, dir)
	assertSTARMappedAndSplices(t, dir, []string{"WT_REP1", "WT_REP2", "RAP1_UNINDUCED_REP1", "RAP1_UNINDUCED_REP2", "RAP1_IAA_30M_REP1"})
	recoverAfterSuccessAPI(t, g, dir, 2)
}
