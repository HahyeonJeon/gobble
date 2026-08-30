package picardmarkduplicates_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	picardmarkduplicates "github.com/HahyeonJeon/gobble/assets/modules/picard-markduplicates"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContract(t *testing.T) {
	bam := gobble.Literal("in/sample.bam")
	task := cc.Task(t, picardmarkduplicates.Pipeline(bam, picardmarkduplicates.Options{}), "picard_markduplicates")
	if !pc.ContainsAll(task.Command, "picard", "MarkDuplicates", "--INPUT", "in/sample.bam", "--OUTPUT", "work/picard-markduplicates/marked.bam", "--METRICS_FILE", "work/picard-markduplicates/marked.metrics.txt") {
		t.Fatalf("command = %#v, want Picard argv", task.Command)
	}
	pc.AssertIOPath(t, task.Outputs, "marked_bam", "work/picard-markduplicates/marked.bam")
	cc.Invalid(t, picardmarkduplicates.Pipeline(bam, picardmarkduplicates.Options{Options: modules.Options{Image: "alpine:latest"}}))
}
