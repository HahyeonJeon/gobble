package run_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/wgsscenario"
)

func TestWGSRunCandidateDeclaresRequiredFinalArtifacts(t *testing.T) {
	raw := wgsscenario.Plan(t, wgs.DefaultConfig())
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "patient1.testN.bqsr_gather.samtools_merge").Outputs, "bam", "results/wgs/samples/patient1/testN/alignment/testN.recalibrated.bam")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "patient1.testN.gvcf_gather.gatk4_mergevcfs").Outputs, "vcf", "results/wgs/samples/patient1/testN/gvcf/testN.g.vcf.gz")
	pc.AssertTreeIO(t, pc.TaskByID(t, raw, "joint_intervals.database.gatk4_genomicsdbimport").Outputs, "database", "work/joint/cohort-604439d34707f4df78926ac0c7b9dbd1159db10516bd9425c1616da5448a347e/genomicsdb")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "joint_gather.joint.gatk4_mergevcfs").Outputs, "vcf", "results/wgs/joint/joint_germline.vcf.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Outputs, "html", "results/wgs/multiqc/multiqc_report.html")
	if !wgs.Lifecycle.Run {
		t.Fatal("WGS run participation is false")
	}
}
