package bismarkgenomeevidence_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	. "github.com/HahyeonJeon/gobble/assets/modules/bismark-genome"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	evidence "github.com/HahyeonJeon/gobble/tests/modules/bismark-genome"
)

func TestBismarkGenomeStandaloneComposeBuildPlan(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	opts := BismarkGenomeOptions{
		ExtraArgs: []string{"--verbose"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := BismarkGenomePipeline(fasta, opts)
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "bismark_genome")
	if task.Name != "bismark_genome" {
		t.Fatalf("task name = %q, want bismark_genome", task.Name)
	}
	if task.Image != evidence.Image {
		t.Fatalf("image = %q, want %q", task.Image, evidence.Image)
	}
	if !pc.ContainsAll(task.Command,
		"bismark_genome_preparation", "--bowtie2",
		"--parallel", "2",
		"--verbose",
		"work/bismark-genome",
	) {
		t.Fatalf("command = %#v, want named flags then extra-args then folder", task.Command)
	}
	n := len(task.Command)
	if n < 2 || task.Command[n-2] != "--verbose" || task.Command[n-1] != "work/bismark-genome" {
		t.Fatalf("command tail = %#v, want [--verbose work/bismark-genome]", task.Command)
	}
	pc.AssertUniqueParamNames(t, task.Params)
	pc.AssertIOPath(t, task.Inputs, "fasta", "work/bismark-genome/genome.fasta")
	pc.AssertIOSource(t, task.Inputs, "fasta", "in/genome.fasta")
	pc.AssertGroupMembers(t, task.Outputs, "index", evidence.Members("work/bismark-genome"))
}

func TestBismarkGenomeNestedModule(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	p := gobble.NewPipeline("assay")
	h := p.AddInput("fasta", fasta)
	mod := p.AddModule("ref")
	ports := AddBismarkGenome(mod, h, BismarkGenomeOptions{ExtraArgs: []string{"--verbose"}})
	if ports.Index.IsZero() {
		t.Fatalf("ports.Index IsZero = true, want false")
	}
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "ref.bismark_genome")
	if task.Name != "bismark_genome" {
		t.Fatalf("nested name = %q, want bismark_genome", task.Name)
	}
	if task.Module != "ref" {
		t.Fatalf("nested module = %q, want ref", task.Module)
	}
	if !pc.ContainsAll(task.Command, "--verbose") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
	pc.AssertGroupMembers(t, task.Outputs, "index", evidence.Members("work/bismark-genome"))
}
