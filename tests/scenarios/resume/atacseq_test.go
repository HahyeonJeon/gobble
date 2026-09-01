package resume_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/atacseq"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/atacseqscenario"
)

func TestATACResumeReusesAlignmentAndRerunsPeakDescendants(t *testing.T) {
	runtime := atacseqscenario.NewRuntime(t, atacseq.DefaultConfig())
	if err := runtime.Run(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Release(); err != nil {
		t.Fatal(err)
	}
	samples, _ := atacseqscenario.Samples(t)
	config := atacseq.DefaultConfig()
	config.PeakMode = atacseq.PeakNarrow
	if err := runtime.ResumeWith(t.Context(), samples, config); err != nil {
		t.Fatal(err)
	}
	reuse := runtime.InspectRecords(gobble.ViewReuse)
	for identity, want := range map[string]string{
		"reference.bwa_index":                                   "reused",
		"OSMOTIC_STRESS_T0_PE.replicate_1.run_001.bwa_mem":      "reused",
		"OSMOTIC_STRESS_T0_PE.replicate_1.peaks.macs2_callpeak": "rerun",
		"peak_qc.replicates.plot_macs2_qc":                      "rerun",
		"peak_qc.replicates.plot_homer_annotatepeaks":           "rerun",
		"consensus.replicates.atac_consensus_peaks":             "rerun",
		"consensus.replicates.featurecounts_merge_matrices":     "rerun",
		"igv.igv_session":                                       "rerun",
	} {
		requireDecision(t, reuse, identity, want)
	}
	lifecycle := atacseq.Lifecycle()
	if !lifecycle.Resume || lifecycle.PreLiftResumable {
		t.Fatal("ATAC resume participation or first-generation boundary is wrong")
	}
}

func requireDecision(t *testing.T, records []map[string]any, identity, want string) {
	t.Helper()
	for _, record := range records {
		if record["identity"] == identity {
			if got := record["decision"]; got != want {
				t.Fatalf("reuse decision for %s = %q, want %q", identity, got, want)
			}
			return
		}
	}
	t.Fatalf("reuse records omit %s", identity)
}
