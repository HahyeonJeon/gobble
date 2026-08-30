package bismarkdeduplicate_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	bismarkdeduplicate "github.com/HahyeonJeon/gobble/assets/modules/bismark-deduplicate"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestBismarkDeduplicateDeclaresActualOutputs(t *testing.T) {
	options := bismarkdeduplicate.Options{OutDir: gobble.Dir("results/bismark/sample"), Prefix: "sample_pe"}
	task := cc.Task(t, bismarkdeduplicate.Pipeline(gobble.Literal("work/align/sample_pe.bam"), true, options), "bismark_deduplicate")
	if !pc.ContainsAll(task.Command, "deduplicate_bismark", "--paired", "--bam", "--output_dir", "results/bismark/sample", "--outfile", "sample_pe", "work/align/sample_pe.bam") {
		t.Fatalf("command = %#v", task.Command)
	}
	pc.AssertIOPath(t, task.Outputs, "deduplicated_bam", "results/bismark/sample/sample_pe.deduplicated.bam")
	pc.AssertIOPath(t, task.Outputs, "report", "results/bismark/sample/sample_pe.deduplication_report.txt")
	cc.Invalid(t, bismarkdeduplicate.Pipeline(gobble.Literal("work/align/sample.bam"), false, bismarkdeduplicate.Options{Options: modules.Options{ExtraArgs: []string{"--sam"}}}))
}
