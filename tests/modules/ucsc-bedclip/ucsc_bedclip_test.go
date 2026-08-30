package ucscbedclip_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	ucscbedclip "github.com/HahyeonJeon/gobble/assets/modules/ucsc-bedclip"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContract(t *testing.T) {
	bedgraph, sizes := gobble.Literal("in/coverage.bedGraph"), gobble.Literal("in/chrom.sizes")
	task := cc.Task(t, ucscbedclip.Pipeline(bedgraph, sizes, ucscbedclip.Options{}), "ucsc_bedclip")
	if !pc.ContainsAll(task.Command, "bedClip", "in/coverage.bedGraph", "in/chrom.sizes", "work/coverage/coverage.clipped.bedGraph") {
		t.Fatalf("command = %#v, want bedClip argv", task.Command)
	}
	pc.AssertIOPath(t, task.Outputs, "clipped", "work/coverage/coverage.clipped.bedGraph")
	cc.Invalid(t, ucscbedclip.Pipeline(bedgraph, sizes, ucscbedclip.Options{Options: modules.Options{Image: "alpine:latest"}}))
}
