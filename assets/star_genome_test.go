package assets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestSTARGenomeGenerateStandaloneComposeBuildPlan(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "genome", Ext: ".fasta"}
	opts := STARGenomeGenerateOptions{
		ExtraArgs: []string{"--genomeSAindexNbases", "7"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := STARGenomeGeneratePipeline(fasta, opts)
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "star_genome_generate")
	if task.Name != "star_genome_generate" {
		t.Fatalf("task name = %q, want star_genome_generate", task.Name)
	}
	if task.Image != starImage {
		t.Fatalf("image = %q, want %q", task.Image, starImage)
	}
	if !containsAll(task.Command,
		"STAR", "--runMode", "genomeGenerate",
		"--genomeDir", "work/star-genome",
		"--genomeFastaFiles", "in/genome.fasta",
		"--runThreadN", "2",
		"--genomeSAindexNbases", "7",
	) {
		t.Fatalf("command = %#v, want named flags then extra-args", task.Command)
	}
	n := len(task.Command)
	if n < 2 || task.Command[n-2] != "--genomeSAindexNbases" || task.Command[n-1] != "7" {
		t.Fatalf("command tail = %#v, want [--genomeSAindexNbases 7]", task.Command)
	}
	assertUniqueParamNames(t, task.Params)
	assertGroupMembers(t, task.Outputs, "index", wantSTARGenomeMembers("work/star-genome"))
}

func TestSTARGenomeGenerateNestedModule(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "genome", Ext: ".fasta"}
	p := gobble.NewPipeline("assay")
	h := p.AddInput("fasta", fasta)
	mod := AddModule(p, "ref")
	ports := AddSTARGenomeGenerate(mod, h, STARGenomeGenerateOptions{ExtraArgs: []string{"--genomeSAindexNbases", "7"}})
	if ports.Index.IsZero() {
		t.Fatalf("ports.Index IsZero = true, want false")
	}
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "ref.star_genome_generate")
	if task.Name != "star_genome_generate" {
		t.Fatalf("nested name = %q, want star_genome_generate", task.Name)
	}
	if task.Module != "ref" {
		t.Fatalf("nested module = %q, want ref", task.Module)
	}
	if !containsAll(task.Command, "--genomeSAindexNbases", "7") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
	assertGroupMembers(t, task.Outputs, "index", wantSTARGenomeMembers("work/star-genome"))
}

func TestSTARGenomeGenerateStandaloneRun(t *testing.T) {
	requireDocker(t)
	src := cachePin(t, PinWGSGenomeFASTA)
	dir := t.TempDir()
	stageFile(t, dir, "in/genome.fasta", src)
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "genome", Ext: ".fasta"}
	p := STARGenomeGeneratePipeline(fasta, STARGenomeGenerateOptions{
		ExtraArgs: []string{"--genomeSAindexNbases", "7"},
		Resources: gobble.Resources{CPU: 1},
	})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, rel := range starGenomePublishedPaths("work/star-genome") {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("published %s: %v", rel, err)
		}
	}
}

func wantSTARGenomeMembers(dir string) []groupMemberWant {
	return []groupMemberWant{
		{Name: "Genome", Path: dir + "/Genome"},
		{Name: "SA", Path: dir + "/SA"},
		{Name: "SAindex", Path: dir + "/SAindex"},
		{Name: "chrLength", Path: dir + "/chrLength.txt"},
		{Name: "chrName", Path: dir + "/chrName.txt"},
		{Name: "chrNameLength", Path: dir + "/chrNameLength.txt"},
		{Name: "chrStart", Path: dir + "/chrStart.txt"},
		{Name: "genomeParameters", Path: dir + "/genomeParameters.txt"},
	}
}

func starGenomePublishedPaths(dir string) []string {
	ms := wantSTARGenomeMembers(dir)
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.Path
	}
	return out
}
