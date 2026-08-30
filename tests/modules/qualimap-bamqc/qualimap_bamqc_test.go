package qualimapbamqc_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	qualimapbamqc "github.com/HahyeonJeon/gobble/assets/modules/qualimap-bamqc"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContract(t *testing.T) {
	bam, gtf := gobble.Literal("in/sample.bam"), gobble.Literal("in/genes.gtf")
	task := cc.Task(t, qualimapbamqc.Pipeline(bam, gtf, qualimapbamqc.Options{Strandedness: gobble.StrandednessForward, Paired: true}), "qualimap_bamqc")
	if !pc.ContainsAll(task.Command, "qualimap", "rnaseq", "-bam", "in/sample.bam", "-gtf", "in/genes.gtf", "-p", "strand-specific-forward", "-pe") {
		t.Fatalf("command = %#v, want forward paired Qualimap argv", task.Command)
	}
	pc.AssertIOPath(t, task.Outputs, "report", "work/qualimap/sample/rnaseq_qc_results.txt")
	cc.Invalid(t, qualimapbamqc.Pipeline(bam, gtf, qualimapbamqc.Options{Options: modules.Options{Image: "alpine:latest"}}))
}
