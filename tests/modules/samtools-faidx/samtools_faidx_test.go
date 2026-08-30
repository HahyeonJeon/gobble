package samtoolsfaidx_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	samtoolsfaidx "github.com/HahyeonJeon/gobble/assets/modules/samtools-faidx"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContract(t *testing.T) {
	fasta := gobble.Literal("in/genome.fa")
	task := cc.Task(t, samtoolsfaidx.Pipeline(fasta, samtoolsfaidx.Options{}), "samtools_faidx")
	if !pc.ContainsAll(task.Command, "samtools", "faidx", "in/genome.fa") {
		t.Fatalf("command = %#v, want samtools faidx argv", task.Command)
	}
	pc.AssertIOPath(t, task.Outputs, "fai", "in/genome.fa.fai")
	cc.Invalid(t, samtoolsfaidx.Pipeline(fasta, samtoolsfaidx.Options{Options: modules.Options{Image: "alpine:latest"}}))
}
