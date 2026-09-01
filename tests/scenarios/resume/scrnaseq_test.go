package resume_test

import (
	"strings"
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
	lifecycle := scrnaseq.Lifecycle()
	if !lifecycle.Resume || lifecycle.PreLiftResumable {
		t.Fatal("scRNA resume participation or first-generation boundary is wrong")
	}
}

func TestSCRNAChangedSampleRunRerunsSampleAndCohortWhileReusingUnrelatedWork(t *testing.T) {
	runtime := scrnaseqscenario.NewRuntime(t, scrnaseq.DefaultConfig())
	if err := runtime.Run(t.Context()); err != nil {
		t.Fatalf("Run(scRNA graph): %v", err)
	}
	if err := runtime.Release(); err != nil {
		t.Fatalf("Release(scRNA graph): %v", err)
	}
	samples, _ := scrnaseqscenario.Samples(t)
	samples[1].Runs[1].Fastq1 = "in/reads/Sample_Y_S1_L002_R1_changed.fastq.gz"
	if err := runtime.ResumeWith(t.Context(), samples, scrnaseq.DefaultConfig()); err != nil {
		t.Fatalf("ResumeWith(changed Sample_Y run): %v", err)
	}

	rerun := map[string]bool{
		"Sample_Y.run_002.raw_fastqc_r1.fastqc": true,
		"Sample_Y.consolidate_r1.cat_fastq":     true,
		"Sample_Y.simpleaf_quant":               true,
		"Sample_Y.qcatch":                       true,
		"Sample_Y.matrix_to_h5ad":               true,
		"Sample_Y.anndatar_convert":             true,
		"cohort.h5ad_concat":                    true,
		"multiqc":                               true,
	}
	requiredReuse := map[string]bool{
		"reference.simpleaf_index":              false,
		"Sample_X.run_001.raw_fastqc_r1.fastqc": false,
		"Sample_X.simpleaf_quant":               false,
		"Sample_X.qcatch":                       false,
		"Sample_X.matrix_to_h5ad":               false,
		"Sample_X.anndatar_convert":             false,
		"Sample_Y.run_002.raw_fastqc_r2.fastqc": false,
		"Sample_Y.consolidate_r2.cat_fastq":     false,
	}
	reuse := runtime.InspectRecords(gobble.ViewReuse)
	if len(reuse) == 0 {
		t.Fatal("changed-sample scRNA Resume reuse is empty")
	}
	seenRerun := make(map[string]bool, len(rerun))
	for _, record := range reuse {
		identity, ok := record["identity"].(string)
		if !ok || identity == "" {
			t.Fatalf("scRNA reuse record has no identity: %#v", record)
		}
		want := "reused"
		if rerun[identity] {
			want = "rerun"
			seenRerun[identity] = true
		}
		if got := record["decision"]; got != want {
			t.Errorf("changed-sample reuse decision for %s = %q (%q), want %q", identity, got, record["reason"], want)
		}
		reason, _ := record["reason"].(string)
		if want == "reused" && reason != "reused-identity-matched" {
			t.Errorf("reuse reason for %s = %q, want reused-identity-matched", identity, reason)
		}
		if want == "rerun" && (reason == "" || strings.HasPrefix(reason, "reused-")) {
			t.Errorf("rerun reason for %s = %q, want changed-graph reason", identity, reason)
		}
		if _, required := requiredReuse[identity]; required {
			requiredReuse[identity] = true
		}
	}
	for identity := range rerun {
		if !seenRerun[identity] {
			t.Errorf("changed-sample reuse records omit required rerun %s", identity)
		}
	}
	for identity, seen := range requiredReuse {
		if !seen {
			t.Errorf("changed-sample reuse records omit required unrelated reuse %s", identity)
		}
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
