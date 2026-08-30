package catfastq_test

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	catfastq "github.com/HahyeonJeon/gobble/assets/modules/cat-fastq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContract(t *testing.T) {
	inputs := []gobble.PathSpec{gobble.Literal("in/run1.fastq.gz"), gobble.Literal("in/run2.fastq.gz")}
	task := cc.Task(t, catfastq.Pipeline(inputs, catfastq.Options{}), "cat_fastq")
	if !strings.Contains(task.Script, "'cat' 'in/run1.fastq.gz' 'in/run2.fastq.gz' > 'work/cat-fastq/reads.merged.fastq.gz'") {
		t.Fatalf("script = %q, want ordered argv and one redirect", task.Script)
	}
	pc.AssertIOPath(t, task.Outputs, "fastq", "work/cat-fastq/reads.merged.fastq.gz")
	cc.Invalid(t, catfastq.Pipeline(inputs, catfastq.Options{Options: modules.Options{Image: "alpine:latest"}}))
	cc.Invalid(t, catfastq.Pipeline(inputs[:1], catfastq.Options{}))
}
