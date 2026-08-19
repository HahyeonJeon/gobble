package assets

import (
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestBismarkExtractorStem(t *testing.T) {
	got := bismarkExtractorStem(gobble.PathSpec{Dir: gobble.Dir("work/bismark-align"), Name: "aligned_pe", Ext: ".bam"})
	if got != "aligned_pe" {
		t.Fatalf("stem(bam) = %q, want aligned_pe", got)
	}
	got = bismarkExtractorStem(gobble.PathSpec{Dir: gobble.Dir("work"), Name: "reads", Ext: ".sam"})
	if got != "reads" {
		t.Fatalf("stem(sam) = %q, want reads", got)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatalf("stem(invalid) panic = nil, want error")
			}
		}()
		bismarkExtractorStem(gobble.PathSpec{Name: "."})
	}()
}

func TestBismarkMethylationExtractorStandaloneComposeBuildPlan(t *testing.T) {
	bam := gobble.PathSpec{Dir: gobble.Dir("work/bismark-align"), Name: "aligned_pe", Ext: ".bam"}
	opts := BismarkMethylationExtractorOptions{
		ExtraArgs: []string{"--no_overlap"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := BismarkMethylationExtractorPipeline(bam, opts)
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "bismark_methylation_extractor")
	if task.Name != "bismark_methylation_extractor" {
		t.Fatalf("task name = %q, want bismark_methylation_extractor", task.Name)
	}
	if task.Image != wantBismarkImage {
		t.Fatalf("image = %q, want %q", task.Image, wantBismarkImage)
	}
	if containsAll(task.Command, "--multicore") || containsAll(task.Command, "--parallel") {
		t.Fatalf("command = %#v, extractor must not copy Resources.CPU", task.Command)
	}
	if !containsAll(task.Command,
		"bismark_methylation_extractor",
		"--bedGraph", "--counts", "--gzip", "--report", "--comprehensive", "-p",
		"--output_dir", "work/bismark-extractor",
		"--no_overlap",
		"work/bismark-align/aligned_pe.bam",
	) {
		t.Fatalf("command = %#v, want named flags then extra-args then BAM", task.Command)
	}
	n := len(task.Command)
	if n < 2 || task.Command[n-2] != "--no_overlap" || task.Command[n-1] != "work/bismark-align/aligned_pe.bam" {
		t.Fatalf("command tail = %#v, want [--no_overlap work/bismark-align/aligned_pe.bam]", task.Command)
	}
	assertUniqueParamNames(t, task.Params)
	assertIOPath(t, task.Outputs, "bedgraph", "work/bismark-extractor/aligned_pe.bedGraph.gz")
	assertIOPath(t, task.Outputs, "coverage", "work/bismark-extractor/aligned_pe.bismark.cov.gz")
	assertIOPath(t, task.Outputs, "report", "work/bismark-extractor/aligned_pe_splitting_report.txt")
	assertIOPath(t, task.Outputs, "mbias", "work/bismark-extractor/aligned_pe.M-bias.txt")
	assertIOPath(t, task.Outputs, "cpg", "work/bismark-extractor/CpG_context_aligned_pe.txt.gz")
}

func TestBismarkMethylationExtractorNestedModule(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "genome", Ext: ".fasta"}
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Name: "test_2", Ext: ".fastq.gz"}
	p := gobble.NewPipeline("assay")
	hf := p.AddInput("fasta", fasta)
	h1 := p.AddInput("r1", r1)
	h2 := p.AddInput("r2", r2)
	mod := AddModule(p, "methyl")
	idx := AddBismarkGenome(mod, hf, BismarkGenomeOptions{})
	aln := AddBismarkAlign(mod, hf, idx.Index, h1, h2, BismarkAlignOptions{})
	ports := AddBismarkMethylationExtractor(mod, aln.BAM, BismarkMethylationExtractorOptions{ExtraArgs: []string{"--no_overlap"}})
	if ports.BedGraph.IsZero() || ports.Coverage.IsZero() || ports.Report.IsZero() || ports.Mbias.IsZero() || ports.CpG.IsZero() {
		t.Fatalf("ports IsZero, want false")
	}
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "methyl.bismark_methylation_extractor")
	if task.Name != "bismark_methylation_extractor" || task.Module != "methyl" {
		t.Fatalf("nested task = %+v, want name bismark_methylation_extractor module methyl", task)
	}
	if !containsAll(task.Command, "--no_overlap") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
	assertIOPath(t, task.Outputs, "bedgraph", "work/bismark-extractor/aligned_pe.bedGraph.gz")
	assertIOPath(t, task.Outputs, "cpg", "work/bismark-extractor/CpG_context_aligned_pe.txt.gz")
}

func TestBismarkMethylationExtractorNestedRun(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	stageMethylPins(t, dir)
	p := gobble.NewPipeline("methyl")
	hf := p.AddInput("fasta", pinnedMethylFASTA())
	h1 := p.AddInput("r1", pinnedMethylFASTQ1())
	h2 := p.AddInput("r2", pinnedMethylFASTQ2())
	idx := AddBismarkGenome(p, hf, BismarkGenomeOptions{Resources: gobble.Resources{CPU: 1}})
	aln := AddBismarkAlign(p, hf, idx.Index, h1, h2, BismarkAlignOptions{Resources: gobble.Resources{CPU: 1}})
	AddBismarkMethylationExtractor(p, aln.BAM, BismarkMethylationExtractorOptions{})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, rel := range []string{
		"work/bismark-extractor/aligned_pe.bedGraph.gz",
		"work/bismark-extractor/aligned_pe.bismark.cov.gz",
		"work/bismark-extractor/aligned_pe_splitting_report.txt",
		"work/bismark-extractor/aligned_pe.M-bias.txt",
		"work/bismark-extractor/CpG_context_aligned_pe.txt.gz",
	} {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("published %s: %v", rel, err)
		}
	}
	unique := uniquePEAlignments(t, filepath.Join(dir, filepath.FromSlash("work/bismark-align/aligned_PE_report.txt")))
	t.Logf("unique paired-end alignments = %d", unique)
	assertUniqueAlignmentFloor(t, unique)
	assertMethylationCallRows(t, unique,
		filepath.Join(dir, filepath.FromSlash("work/bismark-extractor/CpG_context_aligned_pe.txt.gz")),
		filepath.Join(dir, filepath.FromSlash("work/bismark-extractor/aligned_pe.bismark.cov.gz")),
	)
}

func assertMethylationCallRows(t *testing.T, unique int, paths ...string) {
	t.Helper()
	rows := 0
	for _, path := range paths {
		n := methylationCallRows(t, path)
		t.Logf("methylation call rows in %s = %d", filepath.Base(path), n)
		rows += n
	}
	if unique > 0 && rows == 0 {
		t.Fatalf("no methylation call row in %v", paths)
	}
}

func methylationCallRows(t *testing.T, path string) int {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open(%s): %v", path, err)
	}
	defer f.Close()
	var r io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			t.Fatalf("gzip %s: %v", path, err)
		}
		defer gz.Close()
		r = gz
	}
	sc := bufio.NewScanner(r)
	n := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "Bismark") || strings.HasPrefix(line, "#") {
			continue
		}
		if len(strings.Fields(line)) < 4 {
			continue
		}
		n++
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan %s: %v", path, err)
	}
	return n
}
