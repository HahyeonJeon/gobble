package samtoolsstats_test

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	samtoolsstats "github.com/HahyeonJeon/gobble/assets/modules/samtools-stats"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContract(t *testing.T) {
	bam := gobble.Literal("in/sample.bam")
	task := cc.Task(t, samtoolsstats.Pipeline(bam, samtoolsstats.Options{}), "samtools_stats")
	if !strings.Contains(task.Script, "'samtools' 'stats' 'in/sample.bam' > 'work/samtools-stats/alignment.stats.txt'") {
		t.Fatalf("script = %q, want samtools stats redirect", task.Script)
	}
	pc.AssertIOPath(t, task.Outputs, "stats", "work/samtools-stats/alignment.stats.txt")
	cc.Invalid(t, samtoolsstats.Pipeline(bam, samtoolsstats.Options{Options: modules.Options{Image: "alpine:latest"}}))
}
