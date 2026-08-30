package fqlint_test

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	fqlint "github.com/HahyeonJeon/gobble/assets/modules/fq-lint"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContract(t *testing.T) {
	fastq := gobble.Literal("in/reads.fastq.gz")
	task := cc.Task(t, fqlint.Pipeline(fastq, fqlint.Options{}), "fq_lint")
	if !strings.Contains(task.Script, "'fq' 'lint' 'in/reads.fastq.gz'") {
		t.Fatalf("script = %q, want fq lint argv", task.Script)
	}
	pc.AssertIOPath(t, task.Outputs, "report", "work/fq-lint/reads.fq_lint.txt")
	cc.Invalid(t, fqlint.Pipeline(fastq, fqlint.Options{Options: modules.Options{Image: "alpine:latest"}}))
}
