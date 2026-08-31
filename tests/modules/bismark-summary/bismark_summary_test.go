package bismarksummary_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	bismarksummary "github.com/HahyeonJeon/gobble/assets/modules/bismark-summary"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestBismarkSummaryRestagesRelatedReportsTogether(t *testing.T) {
	p := gobble.NewPipeline("summary")
	input := func(name, path string) gobble.Handle { return p.AddInput(name, gobble.Literal(path)) }
	sample := bismarksummary.SampleReports{
		BAM: input("bam", "work/align/sample_pe.bam"), AlignmentReport: input("alignment", "work/align/sample_PE_report.txt"),
		DeduplicationReport: input("dedup", "work/dedup/sample_pe.deduplication_report.txt"), SplittingReport: input("splitting", "work/calls/sample_pe.deduplicated_splitting_report.txt"), MBiasReport: input("mbias", "work/calls/sample_pe.deduplicated.M-bias.txt"),
	}
	if _, err := bismarksummary.Add(p, []bismarksummary.SampleReports{sample}, bismarksummary.Options{}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "bismark_summary")
	if !pc.ContainsAll(task.Command, "bismark2summary", "--basename", "results/methylseq/summary/bismark_summary_report", "work/bismark-summary-inputs/sample_pe.bam") {
		t.Fatalf("command = %#v", task.Command)
	}
	pc.AssertIOPath(t, task.Inputs, "alignment_report_1", "work/bismark-summary-inputs/sample_PE_report.txt")
	pc.AssertIOPath(t, task.Inputs, "deduplication_report_1", "work/bismark-summary-inputs/sample_pe.deduplication_report.txt")
	pc.AssertIOPath(t, task.Outputs, "html", "results/methylseq/summary/bismark_summary_report.html")
	pc.AssertIOPath(t, task.Outputs, "text", "results/methylseq/summary/bismark_summary_report.txt")
	cc.Invalid(t, bismarksummary.Pipeline([]bismarksummary.SampleInputs{{
		BAM:                 gobble.Literal("work/align/sample_pe.bam"),
		AlignmentReport:     gobble.Literal("work/align/sample_PE_report.txt"),
		DeduplicationReport: gobble.Literal("work/dedup/sample_pe.deduplication_report.txt"),
		SplittingReport:     gobble.Literal("work/calls/sample_pe.deduplicated_splitting_report.txt"),
		MBiasReport:         gobble.Literal("work/calls/sample_pe.deduplicated.M-bias.txt"),
	}}, bismarksummary.Options{Options: modules.Options{ExtraArgs: []string{"--base=other"}}}))
}
