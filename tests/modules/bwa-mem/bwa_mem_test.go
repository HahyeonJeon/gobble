package bwamem_test

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	bwaindex "github.com/HahyeonJeon/gobble/assets/modules/bwa-index"
	. "github.com/HahyeonJeon/gobble/assets/modules/bwa-mem"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestBWAMemStandaloneComposeBuildPlan(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"}
	opts := BWAMemOptions{
		ExtraArgs: []string{"-M"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := BWAMemPipeline(fasta, r1, r2, opts)
	raw := pc.MustPlanJSON(t, p)
	if strings.Contains(string(raw), "index_files") {
		t.Fatalf("plan still includes index_files fixture: %s", raw)
	}
	task := pc.TaskByID(t, raw, "bwa_mem")
	if task.Name != "bwa_mem" {
		t.Fatalf("task name = %q, want bwa_mem", task.Name)
	}
	if task.Image != "quay.io/biocontainers/bwa:0.7.18--h577a1d6_2" {
		t.Fatalf("image = %q, want locked BWA pin", task.Image)
	}
	if commandHasSamtools(task.Command) {
		t.Fatalf("command = %#v, must not contain samtools", task.Command)
	}
	if !pc.ContainsAll(task.Command,
		"bwa", "mem",
		"-t", "2",
		"-o", "work/bwa-mem/aligned.sam",
		"-M",
		"in/genome.fasta",
		"in/test_1.fastq.gz",
		"in/test_2.fastq.gz",
	) {
		t.Fatalf("command = %#v, want named flags, extra-args, then positionals", task.Command)
	}
	pc.AssertUniqueParamNames(t, task.Params)
	pc.AssertIOPath(t, task.Outputs, "sam", "work/bwa-mem/aligned.sam")
	pc.AssertGroupMembers(t, task.Inputs, "index", []pc.Member{
		{Name: "amb", Path: "in/genome.fasta.amb"},
		{Name: "ann", Path: "in/genome.fasta.ann"},
		{Name: "bwt", Path: "in/genome.fasta.bwt"},
		{Name: "pac", Path: "in/genome.fasta.pac"},
		{Name: "sa", Path: "in/genome.fasta.sa"},
	})
}

func TestBWAMemNestedModule(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"}
	p := gobble.NewPipeline("assay")
	hf := p.AddInput("fasta", fasta)
	h1 := p.AddInput("r1", r1)
	h2 := p.AddInput("r2", r2)
	mod := p.AddModule("align")
	idx := bwaindex.AddBWAIndex(mod, hf, bwaindex.BWAIndexOptions{})
	ports := AddBWAMem(mod, hf, idx.Index, h1, h2, BWAMemOptions{ExtraArgs: []string{"-M"}})
	if ports.SAM.IsZero() {
		t.Fatalf("ports.SAM IsZero = true, want false")
	}
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "align.bwa_mem")
	if task.Name != "bwa_mem" || task.Module != "align" {
		t.Fatalf("nested task = %+v, want name bwa_mem module align", task)
	}
	if commandHasSamtools(task.Command) {
		t.Fatalf("command = %#v, must not contain samtools", task.Command)
	}
	if !pc.ContainsAll(task.Command, "-M") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
	pc.AssertGroupMembers(t, task.Inputs, "index", []pc.Member{
		{Name: "amb", Path: "in/genome.fasta.amb"},
		{Name: "ann", Path: "in/genome.fasta.ann"},
		{Name: "bwt", Path: "in/genome.fasta.bwt"},
		{Name: "pac", Path: "in/genome.fasta.pac"},
		{Name: "sa", Path: "in/genome.fasta.sa"},
	})
}

func commandHasSamtools(cmd []string) bool {
	for _, s := range cmd {
		if s == "samtools" || strings.Contains(s, "samtools") {
			return true
		}
	}
	return false
}
