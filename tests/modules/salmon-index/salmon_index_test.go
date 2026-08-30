package salmonindex_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	salmonindex "github.com/HahyeonJeon/gobble/assets/modules/salmon-index"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContract(t *testing.T) {
	transcriptome := gobble.Literal("in/transcriptome.fa")
	task := cc.Task(t, salmonindex.Pipeline(transcriptome, salmonindex.Options{}), "salmon_index")
	if !pc.ContainsAll(task.Command, "salmon", "index", "-t", "in/transcriptome.fa", "-i", "work/reference/salmon-index", "-p", "2") {
		t.Fatalf("command = %#v, want Salmon index argv", task.Command)
	}
	pc.AssertTreeIO(t, task.Outputs, "index", "work/reference/salmon-index")
	cc.Invalid(t, salmonindex.Pipeline(transcriptome, salmonindex.Options{Options: modules.Options{Image: "alpine:latest"}}))
}
