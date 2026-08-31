package resume_test

import (
	"slices"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/wgsscenario"
)

func TestWGSResumeIdentityTracksSampleAndCohortChanges(t *testing.T) {
	plain := wgsscenario.Plan(t, wgs.DefaultConfig())
	changedConfig := wgs.DefaultConfig()
	changedConfig.HaplotypeCaller.ExtraArgs = []string{"--min-pruning", "1"}
	changed := wgsscenario.Plan(t, changedConfig)
	plainCaller := pc.TaskByID(t, plain, "haplotype_intervals.patient1.testN.gatk4_haplotypecaller")
	changedCaller := pc.TaskByID(t, changed, "haplotype_intervals.patient1.testN.gatk4_haplotypecaller")
	if slices.Equal([]string{plainCaller.Script}, []string{changedCaller.Script}) || !wgs.Lifecycle.Resume || wgs.Lifecycle.PreLiftResumable {
		t.Fatal("WGS caller change is not visible to resume identity or lift boundary")
	}
	if !slices.Equal(pc.TaskByID(t, plain, "reference.bwa_index").Command, pc.TaskByID(t, changed, "reference.bwa_index").Command) {
		t.Fatal("caller-only change unexpectedly changed reference preparation")
	}
}
