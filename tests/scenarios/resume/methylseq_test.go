package resume_test

import (
	"slices"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/methylseqscenario"
)

func TestMethylExtractorChangePreservesUpstreamIdentity(t *testing.T) {
	plain := methylseqscenario.Plan(t, methylseq.DefaultConfig())
	changedConfig := methylseq.DefaultConfig()
	changedConfig.Extractor.CoverageCutoff = 3
	changed := methylseqscenario.Plan(t, changedConfig)
	for _, id := range []string{"SRR389222_sub1.bismark_align", "SRR389222_sub1.bismark_deduplicate"} {
		if !slices.Equal(pc.TaskByID(t, plain, id).Command, pc.TaskByID(t, changed, id).Command) {
			t.Fatalf("extractor-only change altered upstream task %s", id)
		}
	}
	if slices.Equal(pc.TaskByID(t, plain, "SRR389222_sub1.bismark_methylation_extractor").Command, pc.TaskByID(t, changed, "SRR389222_sub1.bismark_methylation_extractor").Command) || !methylseq.Lifecycle.Resume || methylseq.Lifecycle.PreLiftResumable {
		t.Fatal("Methyl selective resume or graph-generation boundary is incorrect")
	}
}

func TestMethylInspectReleaseResumeReusesCompletedGraph(t *testing.T) {
	runtime := methylseqscenario.NewRuntime(t, methylseq.DefaultConfig())
	if err := runtime.Run(t.Context()); err != nil {
		t.Fatalf("Run(Methyl graph): %v", err)
	}
	if err := runtime.Release(); err != nil {
		t.Fatalf("Release(Methyl graph): %v", err)
	}
	if err := runtime.Resume(t.Context()); err != nil {
		t.Fatalf("Resume(Methyl graph): %v", err)
	}
	reuse := runtime.InspectRecords(gobble.ViewReuse)
	if len(reuse) == 0 {
		t.Fatal("Methyl Resume reuse is empty")
	}
	for _, record := range reuse {
		if record["decision"] != "reused" || record["reason"] != "reused-identity-matched" {
			t.Fatalf("reuse = %#v, want reused-identity-matched", record)
		}
	}
}
