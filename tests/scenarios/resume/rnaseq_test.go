package resume_test

import (
	"slices"
	"testing"

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
