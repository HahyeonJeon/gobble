package dupradar_test

import (
	"slices"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	"github.com/HahyeonJeon/gobble/assets/modules/dupradar"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContract(t *testing.T) {
	bam, gtf := gobble.Literal("in/sample.bam"), gobble.Literal("in/genes.gtf")
	task := cc.Task(t, dupradar.Pipeline(bam, gtf, dupradar.Options{Strandedness: gobble.StrandednessReverse, Paired: true}), "dupradar")
	if !pc.ContainsAll(task.Command, "Rscript", "-e", "in/sample.bam", "in/genes.gtf", "2", "true") || slices.Contains(task.Command, "--") {
		t.Fatalf("command = %#v, want reverse paired dupRadar argv", task.Command)
	}
	pc.AssertIOPath(t, task.Outputs, "multiqc", "work/dupradar/sample_dup_intercept_mqc.txt")
	cc.Invalid(t, dupradar.Pipeline(bam, gtf, dupradar.Options{Options: modules.Options{Image: "alpine:latest"}}))
}
