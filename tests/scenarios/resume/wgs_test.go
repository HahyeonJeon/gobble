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
	reuse := runChangedWGSRecovery(t, func(samples *[]wgs.Sample, _ *wgs.Config) {
		(*samples)[0].Sex = "XY"
	})
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

func TestWGSChangedPatientIdentityRerunsAffectedSampleAndCohortWork(t *testing.T) {
	reuse := runChangedWGSRecovery(t, func(samples *[]wgs.Sample, _ *wgs.Config) {
		(*samples)[0].Patient = "changed_patient"
	})
	for identity, want := range map[string]string{
		"reference.bwa_index":                                                            "reused",
		"patient2.testT.L001.fastp":                                                      "reused",
		"haplotype_intervals.patient2.testT.gatk4_haplotypecaller":                       "reused",
		"changed_patient.testN.L001.fastp":                                               "rerun",
		"bqsr_intervals.changed_patient.testN.gatk4_baserecalibrator/interval_001/0":     "rerun",
		"haplotype_intervals.changed_patient.testN.gatk4_haplotypecaller/interval_001/0": "rerun",
		"changed_patient.testN.gvcf_gather.gatk4_mergevcfs":                              "rerun",
		"joint_intervals.database.gatk4_genomicsdbimport/interval_001/0":                 "rerun",
		"joint_gather.joint.gatk4_mergevcfs":                                             "rerun",
	} {
		requireReuseDecision(t, reuse, identity, want)
	}
}

func TestWGSChangedSampleNameRerunsAffectedSampleAndCohortWork(t *testing.T) {
	reuse := runChangedWGSRecovery(t, func(samples *[]wgs.Sample, _ *wgs.Config) {
		(*samples)[0].Name = "changed_sample"
	})
	for identity, want := range map[string]string{
		"reference.bwa_index":                                                              "reused",
		"patient2.testT.L001.fastp":                                                        "reused",
		"haplotype_intervals.patient2.testT.gatk4_haplotypecaller":                         "reused",
		"patient1.changed_sample.L001.fastp":                                               "rerun",
		"bqsr_intervals.patient1.changed_sample.gatk4_baserecalibrator/interval_001/0":     "rerun",
		"haplotype_intervals.patient1.changed_sample.gatk4_haplotypecaller/interval_001/0": "rerun",
		"patient1.changed_sample.gvcf_gather.gatk4_mergevcfs":                              "rerun",
		"joint_intervals.database.gatk4_genomicsdbimport/interval_001/0":                   "rerun",
		"joint_gather.joint.gatk4_mergevcfs":                                               "rerun",
	} {
		requireReuseDecision(t, reuse, identity, want)
	}
}

func TestWGSChangedIntervalMembershipRerunsAffectedSampleAndCohortWork(t *testing.T) {
	reuse := runChangedWGSRecovery(t, func(_ *[]wgs.Sample, config *wgs.Config) {
		config.Reference.Intervals = append(config.Reference.Intervals, gobble.Member{
			Name: "interval_003",
			Spec: gobble.PathSpec{Dir: gobble.Dir("in/reference/intervals"), Base: "interval_003", Ext: ".bed"},
		})
	})
	for identity, want := range map[string]string{
		"reference.bwa_index":       "reused",
		"patient1.testN.L001.fastp": "reused",
		"haplotype_intervals.patient1.testN.gatk4_haplotypecaller/interval_001/0": "rerun",
		"bqsr_intervals.patient1.testN.gatk4_baserecalibrator/interval_003/0":     "rerun",
		"haplotype_intervals.patient2.testT.gatk4_haplotypecaller/interval_003/0": "rerun",
		"patient1.testN.bqsr_gather.samtools_merge":                               "rerun",
		"patient2.testT.bqsr_gather.samtools_merge":                               "rerun",
		"patient1.testN.gvcf_gather.gatk4_mergevcfs":                              "rerun",
		"patient2.testT.gvcf_gather.gatk4_mergevcfs":                              "rerun",
		"joint_intervals.database.gatk4_genomicsdbimport/interval_003/0":          "rerun",
		"joint_intervals.database.gatk4_genomicsdbimport/interval_001/0":          "rerun",
		"joint_gather.joint.gatk4_mergevcfs":                                      "rerun",
	} {
		requireReuseDecision(t, reuse, identity, want)
	}
}

func TestWGSChangedCohortMembershipRerunsNewSampleAndCohortWork(t *testing.T) {
	reuse := runChangedWGSRecovery(t, func(samples *[]wgs.Sample, _ *wgs.Config) {
		*samples = append(*samples, wgs.Sample{
			Patient: "patient3",
			Name:    "testU",
			Sex:     "XX",
			Lanes: []wgs.Lane{{
				ID: "L001", Fastq1: "in/reads/test_1.fastq.gz", Fastq2: "in/reads/test_2.fastq.gz",
			}},
		})
	})
	for identity, want := range map[string]string{
		"reference.bwa_index":                                                     "reused",
		"patient1.testN.L001.fastp":                                               "reused",
		"patient2.testT.gvcf_gather.gatk4_mergevcfs":                              "reused",
		"patient3.testU.L001.fastp":                                               "rerun",
		"bqsr_intervals.patient3.testU.gatk4_baserecalibrator/interval_001/0":     "rerun",
		"haplotype_intervals.patient3.testU.gatk4_haplotypecaller/interval_001/0": "rerun",
		"patient3.testU.gvcf_gather.gatk4_mergevcfs":                              "rerun",
		"joint_intervals.database.gatk4_genomicsdbimport/interval_001/0":          "rerun",
		"joint_gather.joint.gatk4_mergevcfs":                                      "rerun",
	} {
		requireReuseDecision(t, reuse, identity, want)
	}
}

func runChangedWGSRecovery(t *testing.T, mutate func(*[]wgs.Sample, *wgs.Config)) []map[string]any {
	t.Helper()
	runtime := wgsscenario.NewRuntime(t, wgs.DefaultConfig())
	requireWGSOperation(t, "Run(WGS graph)", runtime.Run(t.Context()))
	requireWGSOperation(t, "Release(WGS graph)", runtime.Release())
	samples, _ := wgsscenario.Samples(t)
	config := wgs.DefaultConfig()
	mutate(&samples, &config)
	requireWGSOperation(t, "ResumeWith(changed WGS graph)", runtime.ResumeWith(t.Context(), samples, config))
	reuse := runtime.InspectRecords(gobble.ViewReuse)
	if len(reuse) == 0 {
		t.Fatal("changed WGS Resume reuse is empty")
	}
	return reuse
}

func requireWGSOperation(t *testing.T, operation string, err error) {
	t.Helper()
	if err == nil {
		return
	}
	var gobbleErr *gobble.Error
	if errors.As(err, &gobbleErr) {
		t.Fatalf("%s defects: %#v", operation, gobbleErr.Defects)
	}
	t.Fatalf("%s: %v", operation, err)
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
