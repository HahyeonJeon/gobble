package bwaindex_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	. "github.com/HahyeonJeon/gobble/assets/modules/bwa-index"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestBWAIndexStandaloneComposeBuildPlan(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	opts := BWAIndexOptions{
		ExtraArgs: []string{"-a", "is"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := BWAIndexPipeline(fasta, opts)
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "bwa_index")
	if task.Name != "bwa_index" {
		t.Fatalf("task name = %q, want bwa_index", task.Name)
	}
	if task.Image != "quay.io/biocontainers/bwa:0.7.18--h577a1d6_2" {
		t.Fatalf("image = %q, want locked BWA pin", task.Image)
	}
	if !pc.ContainsAll(task.Command, "bwa", "index", "-a", "is", "in/genome.fasta") {
		t.Fatalf("command = %#v, want bwa index extra-args then FASTA", task.Command)
	}
	if pc.ContainsAll(task.Command, "-t") || pc.ContainsAll(task.Command, "--threads") {
		t.Fatalf("command = %#v, bwa index must not copy Resources.CPU", task.Command)
	}
	n := len(task.Command)
	if n < 3 || task.Command[n-3] != "-a" || task.Command[n-2] != "is" || task.Command[n-1] != "in/genome.fasta" {
		t.Fatalf("command tail = %#v, want [-a is in/genome.fasta]", task.Command)
	}
	pc.AssertUniqueParamNames(t, task.Params)
	pc.AssertGroupMembers(t, task.Outputs, "index", []pc.Member{
		{Name: "amb", Path: "in/genome.fasta.amb"},
		{Name: "ann", Path: "in/genome.fasta.ann"},
		{Name: "bwt", Path: "in/genome.fasta.bwt"},
		{Name: "pac", Path: "in/genome.fasta.pac"},
		{Name: "sa", Path: "in/genome.fasta.sa"},
	})
}

func TestBWAIndexNestedModule(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	p := gobble.NewPipeline("assay")
	h := p.AddInput("fasta", fasta)
	mod := p.AddModule("ref")
	ports := AddBWAIndex(mod, h, BWAIndexOptions{ExtraArgs: []string{"-a", "is"}})
	if ports.Index.IsZero() {
		t.Fatalf("ports.Index IsZero = true, want false")
	}
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "ref.bwa_index")
	if task.Name != "bwa_index" {
		t.Fatalf("nested name = %q, want bwa_index", task.Name)
	}
	if task.Module != "ref" {
		t.Fatalf("nested module = %q, want ref", task.Module)
	}
	if !pc.ContainsAll(task.Command, "-a", "is") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
	pc.AssertGroupMembers(t, task.Outputs, "index", []pc.Member{
		{Name: "amb", Path: "in/genome.fasta.amb"},
		{Name: "ann", Path: "in/genome.fasta.ann"},
		{Name: "bwt", Path: "in/genome.fasta.bwt"},
		{Name: "pac", Path: "in/genome.fasta.pac"},
		{Name: "sa", Path: "in/genome.fasta.sa"},
	})
}
