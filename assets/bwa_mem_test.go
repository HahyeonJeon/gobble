package assets

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestBWAMemStandaloneComposeBuildPlan(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "genome", Ext: ".fasta"}
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_2", Ext: ".fastq.gz"}
	opts := BWAMemOptions{
		ExtraArgs: []string{"-M"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := BWAMemPipeline(fasta, r1, r2, opts)
	raw := mustPlanJSON(t, p)
	if strings.Contains(string(raw), "index_files") {
		t.Fatalf("plan still includes index_files fixture: %s", raw)
	}
	task := planTask(t, raw, "bwa_mem")
	if task.Name != "bwa_mem" {
		t.Fatalf("task name = %q, want bwa_mem", task.Name)
	}
	if task.Image != bwaImage {
		t.Fatalf("image = %q, want %q", task.Image, bwaImage)
	}
	if commandHasSamtools(task.Command) {
		t.Fatalf("command = %#v, must not contain samtools", task.Command)
	}
	if !containsAll(task.Command,
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
	assertUniqueParamNames(t, task.Params)
	assertIOPath(t, task.Outputs, "sam", "work/bwa-mem/aligned.sam")
	assertGroupMembers(t, task.Inputs, "index", []groupMemberWant{
		{Name: "amb", Path: "in/genome.fasta.amb"},
		{Name: "ann", Path: "in/genome.fasta.ann"},
		{Name: "bwt", Path: "in/genome.fasta.bwt"},
		{Name: "pac", Path: "in/genome.fasta.pac"},
		{Name: "sa", Path: "in/genome.fasta.sa"},
	})
}

func TestBWAMemNestedModule(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "genome", Ext: ".fasta"}
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_2", Ext: ".fastq.gz"}
	p := gobble.NewPipeline("assay")
	hf := p.AddInput("fasta", fasta)
	h1 := p.AddInput("r1", r1)
	h2 := p.AddInput("r2", r2)
	mod := AddModule(p, "align")
	idx := AddBWAIndex(mod, hf, BWAIndexOptions{})
	ports := AddBWAMem(mod, hf, idx.Index, h1, h2, BWAMemOptions{ExtraArgs: []string{"-M"}})
	if ports.SAM.IsZero() {
		t.Fatalf("ports.SAM IsZero = true, want false")
	}
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "align.bwa_mem")
	if task.Name != "bwa_mem" || task.Module != "align" {
		t.Fatalf("nested task = %+v, want name bwa_mem module align", task)
	}
	if commandHasSamtools(task.Command) {
		t.Fatalf("command = %#v, must not contain samtools", task.Command)
	}
	if !containsAll(task.Command, "-M") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
	assertGroupMembers(t, task.Inputs, "index", []groupMemberWant{
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
