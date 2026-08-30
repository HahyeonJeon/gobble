package bismarkgenomeevidence_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	bismarkgenome "github.com/HahyeonJeon/gobble/assets/modules/bismark-genome"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestLiftedBismarkGenomeProducesCompleteTree(t *testing.T) {
	options := bismarkgenome.Options{OutDir: gobble.Dir("work/reference/bismark-index")}
	task := cc.Task(t, bismarkgenome.Pipeline(gobble.Literal("in/reference/genome.fa"), options), "bismark_genome_preparation")
	if !pc.ContainsAll(task.Command, "bismark_genome_preparation", "--bowtie2", "work/reference/bismark-index") {
		t.Fatalf("command = %#v, want directional Bowtie2 genome preparation", task.Command)
	}
	pc.AssertIOPath(t, task.Inputs, "fasta", "work/reference/bismark-index/genome.fa")
	pc.AssertIOSource(t, task.Inputs, "fasta", "in/reference/genome.fa")
	pc.AssertTreeIO(t, task.Outputs, "index", "work/reference/bismark-index")
	if task.Image != string(bismarkgenome.DefaultImage) {
		t.Fatalf("image = %q, want %q", task.Image, bismarkgenome.DefaultImage)
	}
	cc.Invalid(t, bismarkgenome.Pipeline(gobble.Literal("in/reference/genome.fa"), bismarkgenome.Options{Options: modules.Options{ExtraArgs: []string{"--hisat2"}}}))
}
