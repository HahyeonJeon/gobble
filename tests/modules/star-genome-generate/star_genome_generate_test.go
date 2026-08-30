package stargenomegenerate_test

import (
	"os"
	"testing"

	"github.com/HahyeonJeon/gobble"
	. "github.com/HahyeonJeon/gobble/assets/modules/star-genome-generate"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestSTARGenomeGenerateStandaloneComposeBuildPlan(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	opts := STARGenomeGenerateOptions{
		ExtraArgs: []string{"--genomeSAindexNbases", "7"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := STARGenomeGeneratePipeline(fasta, opts)
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "star_genome_generate")
	if task.Name != "star_genome_generate" {
		t.Fatalf("task name = %q, want star_genome_generate", task.Name)
	}
	if task.Image != "community.wave.seqera.io/library/htslib_samtools_star_gawk:ae438e9a604351a4" {
		t.Fatalf("image = %q, want locked STAR pin", task.Image)
	}
	if pc.ContainsAll(task.Command, "--sjdbGTFfile") {
		t.Fatalf("command = %#v, no-GTF compose must omit --sjdbGTFfile", task.Command)
	}
	if !pc.ContainsAll(task.Command,
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
	pc.AssertUniqueParamNames(t, task.Params)
	pc.AssertIOPath(t, task.Inputs, "fasta", "in/genome.fasta")
	pc.AssertTreeIO(t, task.Outputs, "index", "work/star-genome")
}

func TestSTARGenomeGenerateStandaloneComposeBuildPlanGTF(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	gtf := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genes", Ext: ".gtf"}
	opts := STARGenomeGenerateOptions{
		GTF:       gtf,
		ExtraArgs: []string{"--genomeSAindexNbases", "7", "--sjdbOverhang", "100"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := STARGenomeGeneratePipeline(fasta, opts)
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "star_genome_generate")
	if task.Image != "community.wave.seqera.io/library/htslib_samtools_star_gawk:ae438e9a604351a4" {
		t.Fatalf("image = %q, want locked STAR pin", task.Image)
	}
	if !pc.ContainsAll(task.Command,
		"STAR", "--runMode", "genomeGenerate",
		"--genomeDir", "work/star-genome",
		"--genomeFastaFiles", "in/genome.fasta",
		"--sjdbGTFfile", "in/genes.gtf",
		"--runThreadN", "2",
		"--genomeSAindexNbases", "7",
		"--sjdbOverhang", "100",
	) {
		t.Fatalf("command = %#v, want GTF flags then extra-args", task.Command)
	}
	pc.AssertIOPath(t, task.Inputs, "fasta", "in/genome.fasta")
	pc.AssertIOPath(t, task.Inputs, "gtf", "in/genes.gtf")
	pc.AssertTreeIO(t, task.Outputs, "index", "work/star-genome")
}

func TestSTARGenomeGenerateNestedModule(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	p := gobble.NewPipeline("assay")
	h := p.AddInput("fasta", fasta)
	mod := p.AddModule("ref")
	ports := AddSTARGenomeGenerate(mod, h, gobble.Handle{}, STARGenomeGenerateOptions{ExtraArgs: []string{"--genomeSAindexNbases", "7"}})
	if ports.Index.IsZero() {
		t.Fatalf("ports.Index IsZero = true, want false")
	}
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "ref.star_genome_generate")
	if task.Name != "star_genome_generate" {
		t.Fatalf("nested name = %q, want star_genome_generate", task.Name)
	}
	if task.Module != "ref" {
		t.Fatalf("nested module = %q, want ref", task.Module)
	}
	if !pc.ContainsAll(task.Command, "--genomeSAindexNbases", "7") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
	if pc.ContainsAll(task.Command, "--sjdbGTFfile") {
		t.Fatalf("command = %#v, zero gtf must omit --sjdbGTFfile", task.Command)
	}
	pc.AssertTreeIO(t, task.Outputs, "index", "work/star-genome")
}

func starGenomePublishedPaths(dir string, sjdb bool) []string {
	files := starGenomeLiveFiles(sjdb)
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = dir + "/" + f
	}
	return out
}

func starGenomeLiveFiles(sjdb bool) []string {
	base := []string{
		"Genome", "SA", "SAindex",
		"chrLength.txt", "chrName.txt", "chrNameLength.txt", "chrStart.txt",
		"genomeParameters.txt",
	}
	if !sjdb {
		return base
	}
	return append(base,
		"Log.out",
		"exonGeTrInfo.tab",
		"exonInfo.tab",
		"geneInfo.tab",
		"sjdbInfo.txt",
		"sjdbList.fromGTF.out.tab",
		"sjdbList.out.tab",
		"transcriptInfo.tab",
	)
}

func listRegularRel(t *testing.T, abs, prefix string) []string {
	t.Helper()
	entries, err := os.ReadDir(abs)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", abs, err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		out = append(out, prefix+"/"+e.Name())
	}
	return out
}

func sameStringSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := make(map[string]int, len(want))
	for _, s := range want {
		seen[s]++
	}
	for _, s := range got {
		if seen[s] == 0 {
			return false
		}
		seen[s]--
	}
	return true
}
