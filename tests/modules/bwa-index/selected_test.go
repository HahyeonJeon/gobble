package bwaindex_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	bwaindex "github.com/HahyeonJeon/gobble/assets/modules/bwa-index"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestSelectedBWAIndexStandalone(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	p := bwaindex.ProductPipeline(fasta, bwaindex.Options{
		Options: modules.Options{},
		OutDir:  gobble.Dir("work/reference/bwa"),
		Prefix:  "genome",
	})
	task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "bwa_index")
	if task.Image != string(bwaindex.DefaultImage) {
		t.Fatalf("image = %q, want selected immutable image %q", task.Image, bwaindex.DefaultImage)
	}
	if !pc.ContainsAll(task.Command, "bwa", "index", "-p", "work/reference/bwa/genome", "in/genome.fasta") {
		t.Fatalf("command = %#v, want selected bwa index argv", task.Command)
	}
	pc.AssertGroupMembers(t, task.Outputs, "index", []pc.Member{
		{Name: "amb", Path: "work/reference/bwa/genome.amb"},
		{Name: "ann", Path: "work/reference/bwa/genome.ann"},
		{Name: "bwt", Path: "work/reference/bwa/genome.bwt"},
		{Name: "pac", Path: "work/reference/bwa/genome.pac"},
		{Name: "sa", Path: "work/reference/bwa/genome.sa"},
	})
}

func TestSelectedBWAIndexNestedModule(t *testing.T) {
	p := gobble.NewPipeline("assay")
	h := p.AddInput("fasta", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"})
	ports, err := bwaindex.Add(p.AddModule("reference"), h, bwaindex.Options{})
	if err != nil || ports.Index.IsZero() {
		t.Fatalf("Add selected BWA index = (%+v, %v)", ports, err)
	}
	task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "reference.bwa_index")
	if task.Module != "reference" || task.Image != string(bwaindex.DefaultImage) {
		t.Fatalf("nested selected task = %+v", task)
	}
}
