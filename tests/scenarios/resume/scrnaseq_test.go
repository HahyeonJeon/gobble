package resume_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/scrnaseq"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/scrnaseqscenario"
)

func TestSCRNAChangedGraphReusesRawMatricesAndRerunsQCatchDescendants(t *testing.T) {
	runtime := scrnaseqscenario.NewRuntime(t, scrnaseq.DefaultConfig())
	if err := runtime.Run(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Release(); err != nil {
		t.Fatal(err)
	}
	samples, _ := scrnaseqscenario.Samples(t)
	config := scrnaseq.DefaultConfig()
	config.QCatch.RemoveDoublets = true
	if err := runtime.ResumeWith(t.Context(), samples, config); err != nil {
		t.Fatal(err)
	}
	reuse := runtime.InspectRecords(gobble.ViewReuse)
	for identity, want := range map[string]string{
		"reference.simpleaf_index":  "reused",
		"Sample_X.simpleaf_quant":   "reused",
		"Sample_X.matrix_to_h5ad":   "reused",
		"Sample_X.anndatar_convert": "reused",
		"cohort.h5ad_concat":        "reused",
		"Sample_X.qcatch":           "rerun",
		"multiqc":                   "rerun",
	} {
		requireSCRNADecision(t, reuse, identity, want)
	}
	if !scrnaseq.Lifecycle.Resume || scrnaseq.Lifecycle.PreLiftResumable {
		t.Fatal("scRNA resume participation or first-generation boundary is wrong")
	}
}

func requireSCRNADecision(t *testing.T, records []map[string]any, identity, want string) {
	t.Helper()
	for _, record := range records {
		if record["identity"] == identity {
			if got := record["decision"]; got != want {
				t.Fatalf("scRNA reuse decision for %s = %q, want %q", identity, got, want)
			}
			return
		}
	}
	t.Fatalf("scRNA reuse records omit %s", identity)
}
