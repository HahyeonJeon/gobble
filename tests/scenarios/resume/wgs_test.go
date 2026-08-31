package resume_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/HahyeonJeon/gobble"
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

func TestWGSChangedSexRerunsAffectedSampleAndCohortWork(t *testing.T) {
	runtime := wgsscenario.NewRuntime(t, wgs.DefaultConfig())
	if err := runtime.Run(t.Context()); err != nil {
		var runErr *gobble.Error
		if errors.As(err, &runErr) {
			t.Fatalf("Run(WGS graph) defects: %#v", runErr.Defects)
		}
		t.Fatalf("Run(WGS graph): %v", err)
	}
	if err := runtime.Release(); err != nil {
		t.Fatalf("Release(WGS graph): %v", err)
	}
	samples, _ := wgsscenario.Samples(t)
	samples[0].Sex = "XY"
	if err := runtime.ResumeWith(t.Context(), samples, wgs.DefaultConfig()); err != nil {
		var resumeErr *gobble.Error
		if errors.As(err, &resumeErr) {
			t.Fatalf("ResumeWith(changed WGS sample identity) defects: %#v", resumeErr.Defects)
		}
		t.Fatalf("ResumeWith(changed WGS sample identity): %v", err)
	}
	reuse := runtime.InspectRecords(gobble.ViewReuse)
	if len(reuse) == 0 {
		t.Fatal("changed WGS Resume reuse is empty")
	}
	for identity, want := range map[string]string{
		"reference.bwa_index":                                            "reused",
		"patient2.testT.L001.fastp":                                      "reused",
		"patient1.testN.L001.fastp":                                      "rerun",
		"bqsr_intervals.patient2.testT.gatk4_baserecalibrator":           "reused",
		"joint_intervals.database.gatk4_genomicsdbimport/interval_001/0": "rerun",
		"joint_gather.joint.gatk4_mergevcfs":                             "rerun",
	} {
		requireReuseDecision(t, reuse, identity, want)
	}
}

func requireReuseDecision(t *testing.T, records []map[string]any, identity, want string) {
	t.Helper()
	for _, record := range records {
		if record["identity"] != identity {
			continue
		}
		if got := record["decision"]; got != want {
			t.Fatalf("reuse decision for %s = %q (%q), want %q", identity, got, record["reason"], want)
		}
		reason, _ := record["reason"].(string)
		if want == "reused" && reason != "reused-identity-matched" {
			t.Fatalf("reuse reason for %s = %q, want reused-identity-matched", identity, reason)
		}
		if want == "rerun" && (reason == "" || reason == "reused-identity-matched") {
			t.Fatalf("rerun reason for %s = %q, want changed-graph reason", identity, reason)
		}
		return
	}
	t.Fatalf("reuse records omit %s: %#v", identity, records)
}
