package cutchromsizes_test

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	cutchromsizes "github.com/HahyeonJeon/gobble/assets/modules/cut-chrom-sizes"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContract(t *testing.T) {
	fai := gobble.Literal("in/genome.fa.fai")
	task := cc.Task(t, cutchromsizes.Pipeline(fai, cutchromsizes.Options{}), "cut_chrom_sizes")
	if !strings.Contains(task.Script, "'cut' '-f1,2' 'in/genome.fa.fai' > 'work/reference/chrom.sizes'") {
		t.Fatalf("script = %q, want cut projection", task.Script)
	}
	pc.AssertIOPath(t, task.Outputs, "sizes", "work/reference/chrom.sizes")
	cc.Invalid(t, cutchromsizes.Pipeline(fai, cutchromsizes.Options{Options: modules.Options{Image: "alpine:latest"}}))
}
