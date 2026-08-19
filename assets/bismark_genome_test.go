package assets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestBismarkGenomeStandaloneComposeBuildPlan(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "genome", Ext: ".fasta"}
	opts := BismarkGenomeOptions{
		ExtraArgs: []string{"--verbose"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := BismarkGenomePipeline(fasta, opts)
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "bismark_genome")
	if task.Name != "bismark_genome" {
		t.Fatalf("task name = %q, want bismark_genome", task.Name)
	}
	if task.Image != bismarkImage {
		t.Fatalf("image = %q, want %q", task.Image, bismarkImage)
	}
	if !containsAll(task.Command,
		"bismark_genome_preparation", "--bowtie2",
		"--parallel", "2",
		"--verbose",
		"in",
	) {
		t.Fatalf("command = %#v, want named flags then extra-args then folder", task.Command)
	}
	n := len(task.Command)
	if n < 2 || task.Command[n-2] != "--verbose" || task.Command[n-1] != "in" {
		t.Fatalf("command tail = %#v, want [--verbose in]", task.Command)
	}
	assertUniqueParamNames(t, task.Params)
	assertIOPath(t, task.Inputs, "fasta", "in/genome.fasta")
	assertGroupMembers(t, task.Outputs, "index", wantBismarkGenomeMembers("in"))
}

func TestBismarkGenomeNestedModule(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "genome", Ext: ".fasta"}
	p := gobble.NewPipeline("assay")
	h := p.AddInput("fasta", fasta)
	mod := AddModule(p, "ref")
	ports := AddBismarkGenome(mod, h, BismarkGenomeOptions{ExtraArgs: []string{"--verbose"}})
	if ports.Index.IsZero() {
		t.Fatalf("ports.Index IsZero = true, want false")
	}
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "ref.bismark_genome")
	if task.Name != "bismark_genome" {
		t.Fatalf("nested name = %q, want bismark_genome", task.Name)
	}
	if task.Module != "ref" {
		t.Fatalf("nested module = %q, want ref", task.Module)
	}
	if !containsAll(task.Command, "--verbose") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
	assertGroupMembers(t, task.Outputs, "index", wantBismarkGenomeMembers("in"))
}

func TestBismarkGenomeStandaloneRun(t *testing.T) {
	requireDocker(t)
	src := cachePin(t, PinWGSGenomeFASTA)
	dir := t.TempDir()
	stageFile(t, dir, "in/genome.fasta", src)
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "genome", Ext: ".fasta"}
	p := BismarkGenomePipeline(fasta, BismarkGenomeOptions{Resources: gobble.Resources{CPU: 1}})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, rel := range bismarkGenomePublishedPaths("in") {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("published %s: %v", rel, err)
		}
	}
}

func wantBismarkGenomeMembers(dir string) []groupMemberWant {
	ct := dir + "/Bisulfite_Genome/CT_conversion"
	ga := dir + "/Bisulfite_Genome/GA_conversion"
	return []groupMemberWant{
		{Name: "CT_1", Path: ct + "/BS_CT.1.bt2"},
		{Name: "CT_2", Path: ct + "/BS_CT.2.bt2"},
		{Name: "CT_3", Path: ct + "/BS_CT.3.bt2"},
		{Name: "CT_4", Path: ct + "/BS_CT.4.bt2"},
		{Name: "CT_rev1", Path: ct + "/BS_CT.rev.1.bt2"},
		{Name: "CT_rev2", Path: ct + "/BS_CT.rev.2.bt2"},
		{Name: "CT_mfa", Path: ct + "/genome_mfa.CT_conversion.fa"},
		{Name: "GA_1", Path: ga + "/BS_GA.1.bt2"},
		{Name: "GA_2", Path: ga + "/BS_GA.2.bt2"},
		{Name: "GA_3", Path: ga + "/BS_GA.3.bt2"},
		{Name: "GA_4", Path: ga + "/BS_GA.4.bt2"},
		{Name: "GA_rev1", Path: ga + "/BS_GA.rev.1.bt2"},
		{Name: "GA_rev2", Path: ga + "/BS_GA.rev.2.bt2"},
		{Name: "GA_mfa", Path: ga + "/genome_mfa.GA_conversion.fa"},
	}
}

func bismarkGenomePublishedPaths(dir string) []string {
	ms := wantBismarkGenomeMembers(dir)
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.Path
	}
	return out
}
