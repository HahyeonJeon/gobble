package bedtoolsgenomecov_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	bedtoolsgenomecov "github.com/HahyeonJeon/gobble/assets/modules/bedtools-genomecov"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContract(t *testing.T) {
	bam := gobble.Literal("in/sample.bam")
	task := cc.Task(t, bedtoolsgenomecov.Pipeline(bam, bedtoolsgenomecov.Options{Strand: "+"}), "bedtools_genomecov")
	if !pc.ContainsAll(task.Command, "sh", "-c") || !pc.ContainsSubstring([]string{task.Script}, "'bedtools' 'genomecov' '-bg' '-split' '-ibam' 'in/sample.bam' '-strand' '+'") {
		t.Fatalf("task = %+v, want strand-aware redirected bedtools command", task)
	}
	pc.AssertIOPath(t, task.Outputs, "bedgraph", "work/coverage/coverage.bedGraph")
	cc.Invalid(t, bedtoolsgenomecov.Pipeline(bam, bedtoolsgenomecov.Options{Options: modules.Options{Image: "alpine:latest"}}))
}
