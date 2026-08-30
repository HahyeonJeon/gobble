package resume_test

import (
	"slices"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/rnaseqscenario"
)

func TestRNASalmonChangePreservesUpstreamSTARIdentity(t *testing.T) {
	plain := rnaseqscenario.Plan(t, rnaseq.DefaultConfig())
	changedConfig := rnaseq.DefaultConfig()
	changedConfig.Salmon.ExtraArgs = []string{"--validateMappings"}
	changed := rnaseqscenario.Plan(t, changedConfig)
	if !slices.Equal(pc.TaskByID(t, plain, "WT_REP1.star_align").Command, pc.TaskByID(t, changed, "WT_REP1.star_align").Command) {
		t.Fatal("Salmon-only change altered upstream STAR identity")
	}
	if slices.Equal(pc.TaskByID(t, plain, "WT_REP1.salmon_quant").Command, pc.TaskByID(t, changed, "WT_REP1.salmon_quant").Command) || !rnaseq.Lifecycle.Resume || rnaseq.Lifecycle.PreLiftResumable {
		t.Fatal("RNA selective resume or graph-generation boundary is incorrect")
	}
}

func TestRNAInspectReleaseResumeReusesCompletedGraph(t *testing.T) {
	runtime := rnaseqscenario.NewRuntime(t, rnaseq.DefaultConfig())
	if err := runtime.Run(t.Context()); err != nil {
		t.Fatalf("Run(RNA graph): %v", err)
	}
	if err := runtime.Release(); err != nil {
		t.Fatalf("Release(RNA graph): %v", err)
	}
	if err := runtime.Resume(t.Context()); err != nil {
		t.Fatalf("Resume(RNA graph): %v", err)
	}
	reuse := runtime.InspectRecords(gobble.ViewReuse)
	if len(reuse) == 0 {
		t.Fatal("RNA Resume reuse is empty")
	}
	for _, record := range reuse {
		if record["decision"] != "reused" || record["reason"] != "reused-identity-matched" {
			t.Fatalf("reuse = %#v, want reused-identity-matched", record)
		}
	}
}
