package bwamem_test

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	bwaindex "github.com/HahyeonJeon/gobble/assets/modules/bwa-index"
	bwamem "github.com/HahyeonJeon/gobble/assets/modules/bwa-mem"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestSelectedBWAMemStandalone(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	prefix := gobble.PathSpec{Dir: gobble.Dir("in/index"), Base: "genome"}
	read1 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	read2 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"}
	p := bwamem.ProductPipeline(fasta, bwaGroup(prefix), read1, read2, bwamem.Options{
		Options:     modules.Options{Resources: gobble.Resources{CPU: 2}, ExtraArgs: []string{"-M"}},
		IndexPrefix: prefix,
		ReadGroup:   "@RG\\tID:test\\tSM:test",
	})
	task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "bwa_mem")
	if task.Image != string(bwamem.DefaultImage) || !strings.Contains(task.Script, "'bwa' 'mem'") || !strings.Contains(task.Script, "in/index/genome") || !strings.Contains(task.Script, "-M") {
		t.Fatalf("selected BWA MEM task = %+v", task)
	}
	pc.AssertIOPath(t, task.Outputs, "sam", "work/bwa-mem/aligned.sam")
}

func TestSelectedBWAMemConsumesSelectedBWAIndex(t *testing.T) {
	p := gobble.NewPipeline("assay")
	hf := p.AddInput("fasta", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"})
	h1 := p.AddInput("read1", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"})
	h2 := p.AddInput("read2", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"})
	parent := p.AddModule("alignment")
	index, err := bwaindex.Add(parent, hf, bwaindex.Options{})
	if err != nil {
		t.Fatal(err)
	}
	ports, err := bwamem.Add(parent, hf, index.Index, h1, h2, bwamem.Options{IndexPrefix: index.Prefix, ReadGroup: "@RG\\tID:test\\tSM:test"})
	if err != nil || ports.SAM.IsZero() {
		t.Fatalf("Add selected BWA MEM = (%+v, %v)", ports, err)
	}
	task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "alignment.bwa_mem")
	if task.Image != string(bwamem.DefaultImage) || task.Module != "alignment" {
		t.Fatalf("nested selected task = %+v", task)
	}
}

func bwaGroup(prefix gobble.PathSpec) gobble.Group {
	group := make(gobble.Group, 0, 5)
	for _, name := range []string{"amb", "ann", "bwt", "pac", "sa"} {
		group = append(group, gobble.Member{Name: name, Spec: prefix.AppendExt("." + name)})
	}
	return group
}
