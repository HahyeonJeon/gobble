package run_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/wgsscenario"
)

func TestWGSRunCandidateDeclaresRequiredFinalArtifacts(t *testing.T) {
	raw := wgsscenario.Plan(t, wgs.DefaultConfig())
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "patient1.testN.bqsr_gather.samtools_merge").Outputs, "bam", "results/wgs/samples/patient1/testN/alignment/testN.recalibrated.bam")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "patient1.testN.gvcf_gather.gatk4_mergevcfs").Outputs, "vcf", "results/wgs/samples/patient1/testN/gvcf/testN.g.vcf.gz")
	pc.AssertTreeIO(t, pc.TaskByID(t, raw, "joint_intervals.database.gatk4_genomicsdbimport").Outputs, "database", "work/joint/cohort-604439d34707f4df78926ac0c7b9dbd1159db10516bd9425c1616da5448a347e/intervals-c78d0c9ad1526d46003e719d6ad3e19575f5715709ac8bf394df7d320359d77c/genomicsdb")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "joint_gather.joint.gatk4_mergevcfs").Outputs, "vcf", "results/wgs/joint/joint_germline.vcf.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Outputs, "html", "results/wgs/multiqc/multiqc_report.html")
	if !wgs.Lifecycle().Run {
		t.Fatal("WGS run participation is false")
	}
}

func TestWGSRunExecutesOwnedGraphAndPublishesFinals(t *testing.T) {
	runtime := wgsscenario.NewRuntime(t, wgs.DefaultConfig())
	if err := runtime.Run(t.Context()); err != nil {
		t.Fatalf("Run(WGS graph): %v\nerrors: %#v", err, runtime.InspectRecords(gobble.ViewErrors))
	}
	for _, record := range runtime.InspectRecords(gobble.ViewRemaining) {
		if record["remaining"] == true {
			t.Fatalf("remaining record = %#v, want no remaining work", record)
		}
	}
	for _, rel := range []string{
		"results/wgs/samples/patient1/testN/alignment/testN.recalibrated.bam",
		"results/wgs/samples/patient2/testT/gvcf/testT.g.vcf.gz",
		"results/wgs/joint/joint_germline.vcf.gz",
		"results/wgs/multiqc/multiqc_report.html",
	} {
		if info, err := os.Stat(filepath.Join(runtime.Workspace(), filepath.FromSlash(rel))); err != nil || !info.Mode().IsRegular() {
			t.Errorf("required artifact %s: %v", rel, err)
		}
	}
}
