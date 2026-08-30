package stringtie_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	"github.com/HahyeonJeon/gobble/assets/modules/stringtie"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContract(t *testing.T) {
	bam, gtf := gobble.Literal("in/sample.bam"), gobble.Literal("in/genes.gtf")
	task := cc.Task(t, stringtie.Pipeline(bam, gtf, stringtie.Options{Strandedness: gobble.StrandednessReverse}), "stringtie")
	if !pc.ContainsAll(task.Command, "stringtie", "in/sample.bam", "-G", "in/genes.gtf", "--rf") {
		t.Fatalf("command = %#v, want reverse StringTie argv", task.Command)
	}
	pc.AssertIOPath(t, task.Outputs, "abundance", "work/stringtie/sample.gene.abundance.txt")
	cc.Invalid(t, stringtie.Pipeline(bam, gtf, stringtie.Options{Options: modules.Options{Image: "alpine:latest"}}))
}
