package ucscbedgraphtobigwig_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	ucscbedgraphtobigwig "github.com/HahyeonJeon/gobble/assets/modules/ucsc-bedgraphtobigwig"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContract(t *testing.T) {
	bedgraph, sizes := gobble.Literal("in/coverage.bedGraph"), gobble.Literal("in/chrom.sizes")
	task := cc.Task(t, ucscbedgraphtobigwig.Pipeline(bedgraph, sizes, ucscbedgraphtobigwig.Options{}), "ucsc_bedgraphtobigwig")
	if !pc.ContainsAll(task.Command, "bedGraphToBigWig", "in/coverage.bedGraph", "in/chrom.sizes", "results/coverage/coverage.bigWig") {
		t.Fatalf("command = %#v, want bedGraphToBigWig argv", task.Command)
	}
	pc.AssertIOPath(t, task.Outputs, "bigwig", "results/coverage/coverage.bigWig")
	cc.Invalid(t, ucscbedgraphtobigwig.Pipeline(bedgraph, sizes, ucscbedgraphtobigwig.Options{Options: modules.Options{Image: "alpine:latest"}}))
}
