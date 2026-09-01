package bismarkgenomeevidence_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	bismarkgenome "github.com/HahyeonJeon/gobble/assets/modules/bismark-genome"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestSelectedBismarkGenomeStandalone(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fa"}
	p := bismarkgenome.Pipeline(fasta, bismarkgenome.Options{Options: modules.Options{Resources: gobble.Resources{CPU: 4}}})
	task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "bismark_genome_preparation")
	if task.Image != string(bismarkgenome.DefaultImage) {
		t.Fatalf("image = %q, want selected immutable image %q", task.Image, bismarkgenome.DefaultImage)
	}
	if !pc.ContainsAll(task.Command, "bismark_genome_preparation", "--bowtie2", "--parallel", "4", "work/bismark-index") {
		t.Fatalf("command = %#v, want selected Bismark genome argv", task.Command)
	}
	pc.AssertTreeIO(t, task.Outputs, "index", "work/bismark-index")
}

func TestSelectedBismarkGenomeNestedModule(t *testing.T) {
	p := gobble.NewPipeline("assay")
	h := p.AddInput("fasta", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fa"})
	ports, err := bismarkgenome.Add(p.AddModule("reference"), h, bismarkgenome.Options{})
	if err != nil || ports.Index.IsZero() {
		t.Fatalf("Add selected Bismark genome = (%+v, %v)", ports, err)
	}
	task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "reference.bismark_genome_preparation")
	if task.Module != "reference" || task.Image != string(bismarkgenome.DefaultImage) {
		t.Fatalf("nested selected task = %+v", task)
	}
}
