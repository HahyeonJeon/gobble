package rseqcinferexperiment_test

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	rseqcinferexperiment "github.com/HahyeonJeon/gobble/assets/modules/rseqc-inferexperiment"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContract(t *testing.T) {
	bam, bai, bed := gobble.Literal("in/sample.bam"), gobble.Literal("in/sample.bam.bai"), gobble.Literal("in/genes.bed")
	task := cc.Task(t, rseqcinferexperiment.Pipeline(bam, bai, bed, rseqcinferexperiment.Options{}), "rseqc_inferexperiment")
	if !strings.Contains(task.Script, "'infer_experiment.py' '-i' 'in/sample.bam' '-r' 'in/genes.bed'") {
		t.Fatalf("script = %q, want RSeQC argv", task.Script)
	}
	pc.AssertIOPath(t, task.Inputs, "bai", "in/sample.bam.bai")
	pc.AssertIOPath(t, task.Outputs, "report", "work/rseqc/sample.infer_experiment.txt")
	cc.Invalid(t, rseqcinferexperiment.Pipeline(bam, bai, bed, rseqcinferexperiment.Options{Options: modules.Options{Image: "alpine:latest"}}))
}
