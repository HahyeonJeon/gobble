package gunzip_test

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	"github.com/HahyeonJeon/gobble/assets/modules/gunzip"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContract(t *testing.T) {
	input := gobble.Literal("in/genes.gtf.gz")
	task := cc.Task(t, gunzip.Pipeline(input, gunzip.Options{}), "gunzip")
	if !strings.Contains(task.Script, "'gzip' '-cd' 'in/genes.gtf.gz' > 'work/gunzip/decompressed.gtf'") {
		t.Fatalf("script = %q, want gzip stdout contract", task.Script)
	}
	pc.AssertIOPath(t, task.Outputs, "file", "work/gunzip/decompressed.gtf")
	cc.Invalid(t, gunzip.Pipeline(input, gunzip.Options{Options: modules.Options{Image: "alpine:latest"}}))
}
