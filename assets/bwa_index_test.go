package assets

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestBWAIndexStandaloneComposeBuildPlan(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	opts := BWAIndexOptions{
		ExtraArgs: []string{"-a", "is"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := BWAIndexPipeline(fasta, opts)
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "bwa_index")
	if task.Name != "bwa_index" {
		t.Fatalf("task name = %q, want bwa_index", task.Name)
	}
	if task.Image != bwaImage {
		t.Fatalf("image = %q, want %q", task.Image, bwaImage)
	}
	if !containsAll(task.Command, "bwa", "index", "-a", "is", "in/genome.fasta") {
		t.Fatalf("command = %#v, want bwa index extra-args then FASTA", task.Command)
	}
	if containsAll(task.Command, "-t") || containsAll(task.Command, "--threads") {
		t.Fatalf("command = %#v, bwa index must not copy Resources.CPU", task.Command)
	}
	n := len(task.Command)
	if n < 3 || task.Command[n-3] != "-a" || task.Command[n-2] != "is" || task.Command[n-1] != "in/genome.fasta" {
		t.Fatalf("command tail = %#v, want [-a is in/genome.fasta]", task.Command)
	}
	assertUniqueParamNames(t, task.Params)
	assertGroupMembers(t, task.Outputs, "index", []groupMemberWant{
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
	mod := AddModule(p, "ref")
	ports := AddBWAIndex(mod, h, BWAIndexOptions{ExtraArgs: []string{"-a", "is"}})
	if ports.Index.IsZero() {
		t.Fatalf("ports.Index IsZero = true, want false")
	}
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "ref.bwa_index")
	if task.Name != "bwa_index" {
		t.Fatalf("nested name = %q, want bwa_index", task.Name)
	}
	if task.Module != "ref" {
		t.Fatalf("nested module = %q, want ref", task.Module)
	}
	if !containsAll(task.Command, "-a", "is") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
	assertGroupMembers(t, task.Outputs, "index", []groupMemberWant{
		{Name: "amb", Path: "in/genome.fasta.amb"},
		{Name: "ann", Path: "in/genome.fasta.ann"},
		{Name: "bwt", Path: "in/genome.fasta.bwt"},
		{Name: "pac", Path: "in/genome.fasta.pac"},
		{Name: "sa", Path: "in/genome.fasta.sa"},
	})
}

type groupMemberWant struct {
	Name string
	Path string
}

func assertGroupMembers(t *testing.T, ios []planIORec, name string, want []groupMemberWant) {
	t.Helper()
	for _, io := range ios {
		if io.Name != name {
			continue
		}
		if io.Path != "" {
			t.Fatalf("%s path = %q, want empty group path", name, io.Path)
		}
		if len(io.Members) != len(want) {
			t.Fatalf("%s members = %#v, want %#v", name, io.Members, want)
		}
		for i, m := range io.Members {
			if m.Name != want[i].Name || m.Path != want[i].Path {
				t.Fatalf("%s member[%d] = {%s %s}, want {%s %s}", name, i, m.Name, m.Path, want[i].Name, want[i].Path)
			}
		}
		return
	}
	t.Fatalf("missing group IO %q in %#v", name, ios)
}
