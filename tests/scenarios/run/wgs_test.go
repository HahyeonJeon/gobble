package run_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/wgsscenario"
)

func TestWGSRunCandidateDeclaresRequiredFinalArtifacts(t *testing.T) {
	raw := wgsscenario.Plan(t, wgs.DefaultConfig())
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "testN_bqsr_gather.testN.gatk4_gatherbamfiles").Outputs, "bam", "results/wgs/samples/testN/alignment/testN.recalibrated.bam")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "testN_gvcf_gather.testN.gatk4_mergevcfs").Outputs, "vcf", "results/wgs/samples/testN/gvcf/testN.g.vcf.gz")
	pc.AssertTreeIO(t, pc.TaskByID(t, raw, "joint_database_interval_001.gatk4_genomicsdbimport").Outputs, "database", "work/joint/genomicsdb/interval_001")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "joint_gather.joint.gatk4_mergevcfs").Outputs, "vcf", "results/wgs/joint/joint_germline.vcf.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Outputs, "html", "results/wgs/multiqc/multiqc_report.html")
	if !wgs.Lifecycle.Run {
		t.Fatal("WGS run participation is false")
	}
}
