package bismarkmethylationextractor_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	bismarkextractor "github.com/HahyeonJeon/gobble/assets/modules/bismark-methylation-extractor"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestLiftedExtractorDeclaresEveryComprehensiveOutput(t *testing.T) {
	options := bismarkextractor.Options{OutDir: gobble.Dir("results/calls/sample"), ExcludeOverlap: true, IgnoreR2: 2, CoverageCutoff: 1}
	task := cc.Task(t, bismarkextractor.Pipeline(gobble.Literal("results/bismark/sample/sample_pe.deduplicated.bam"), true, options), "bismark_methylation_extractor")
	if !pc.ContainsAll(task.Command, "--comprehensive", "--paired-end", "--no_overlap", "--ignore_r2", "2", "--cutoff", "1") {
		t.Fatalf("command = %#v", task.Command)
	}
	for name, path := range map[string]string{
		"cpg":      "results/calls/sample/CpG_context_sample_pe.deduplicated.txt.gz",
		"chg":      "results/calls/sample/CHG_context_sample_pe.deduplicated.txt.gz",
		"chh":      "results/calls/sample/CHH_context_sample_pe.deduplicated.txt.gz",
		"bedgraph": "results/calls/sample/sample_pe.deduplicated.bedGraph.gz",
		"coverage": "results/calls/sample/sample_pe.deduplicated.bismark.cov.gz",
		"report":   "results/calls/sample/sample_pe.deduplicated_splitting_report.txt",
		"mbias":    "results/calls/sample/sample_pe.deduplicated.M-bias.txt",
	} {
		pc.AssertIOPath(t, task.Outputs, name, path)
	}
	for _, extra := range [][]string{{"--CX"}, {"--cyt"}, {"--mult", "2"}, {"--outp", "work/other"}} {
		t.Run(extra[0], func(t *testing.T) {
			cc.Invalid(t, bismarkextractor.Pipeline(gobble.Literal("sample.bam"), false, bismarkextractor.Options{Options: modules.Options{ExtraArgs: extra}}))
		})
	}
}
