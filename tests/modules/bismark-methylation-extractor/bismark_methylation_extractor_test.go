package bismarkmethylationextractor_test

import (
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	bismarkalign "github.com/HahyeonJeon/gobble/assets/modules/bismark-align"
	bismarkgenome "github.com/HahyeonJeon/gobble/assets/modules/bismark-genome"
	. "github.com/HahyeonJeon/gobble/assets/modules/bismark-methylation-extractor"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	evidence "github.com/HahyeonJeon/gobble/tests/modules/bismark-genome"
)

func TestBismarkExtractorStem(t *testing.T) {
	for _, test := range []struct {
		name string
		bam  gobble.PathSpec
		want string
	}{
		{name: "bam", bam: gobble.PathSpec{Dir: gobble.Dir("work/bismark-align"), Base: "aligned_pe", Ext: ".bam"}, want: "work/bismark-extractor/aligned_pe.bedGraph.gz"},
		{name: "sam", bam: gobble.PathSpec{Dir: gobble.Dir("work"), Base: "reads", Ext: ".sam"}, want: "work/bismark-extractor/reads.bedGraph.gz"},
	} {
		t.Run(test.name, func(t *testing.T) {
			task := pc.TaskByID(t, pc.MustPlanJSON(t, BismarkMethylationExtractorPipeline(test.bam, BismarkMethylationExtractorOptions{})), "bismark_methylation_extractor")
			pc.AssertIOPath(t, task.Outputs, "bedgraph", test.want)
		})
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatalf("stem(invalid) panic = nil, want error")
			}
		}()
		BismarkMethylationExtractorPipeline(gobble.PathSpec{Base: "."}, BismarkMethylationExtractorOptions{})
	}()
}

func TestBismarkMethylationExtractorStandaloneComposeBuildPlan(t *testing.T) {
	bam := gobble.PathSpec{Dir: gobble.Dir("work/bismark-align"), Base: "aligned_pe", Ext: ".bam"}
	opts := BismarkMethylationExtractorOptions{
		ExtraArgs: []string{"--no_overlap"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := BismarkMethylationExtractorPipeline(bam, opts)
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "bismark_methylation_extractor")
	if task.Name != "bismark_methylation_extractor" {
		t.Fatalf("task name = %q, want bismark_methylation_extractor", task.Name)
	}
	if task.Image != evidence.Image {
		t.Fatalf("image = %q, want %q", task.Image, evidence.Image)
	}
	if pc.ContainsAll(task.Command, "--multicore") || pc.ContainsAll(task.Command, "--parallel") {
		t.Fatalf("command = %#v, extractor must not copy Resources.CPU", task.Command)
	}
	if !pc.ContainsAll(task.Command,
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
	pc.AssertUniqueParamNames(t, task.Params)
	pc.AssertIOPath(t, task.Outputs, "bedgraph", "work/bismark-extractor/aligned_pe.bedGraph.gz")
	pc.AssertIOPath(t, task.Outputs, "coverage", "work/bismark-extractor/aligned_pe.bismark.cov.gz")
	pc.AssertIOPath(t, task.Outputs, "report", "work/bismark-extractor/aligned_pe_splitting_report.txt")
	pc.AssertIOPath(t, task.Outputs, "mbias", "work/bismark-extractor/aligned_pe.M-bias.txt")
	pc.AssertIOPath(t, task.Outputs, "cpg", "work/bismark-extractor/CpG_context_aligned_pe.txt.gz")
}

func TestBismarkMethylationExtractorNestedModule(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"}
	p := gobble.NewPipeline("assay")
	hf := p.AddInput("fasta", fasta)
	h1 := p.AddInput("r1", r1)
	h2 := p.AddInput("r2", r2)
	mod := p.AddModule("methyl")
	idx := bismarkgenome.AddBismarkGenome(mod, hf, bismarkgenome.BismarkGenomeOptions{})
	aln := bismarkalign.AddBismarkAlign(mod, hf, idx.Index, h1, h2, bismarkalign.BismarkAlignOptions{})
	ports := AddBismarkMethylationExtractor(mod, aln.BAM, BismarkMethylationExtractorOptions{ExtraArgs: []string{"--no_overlap"}})
	if ports.BedGraph.IsZero() || ports.Coverage.IsZero() || ports.Report.IsZero() || ports.Mbias.IsZero() || ports.CpG.IsZero() {
		t.Fatalf("ports IsZero, want false")
	}
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "methyl.bismark_methylation_extractor")
	if task.Name != "bismark_methylation_extractor" || task.Module != "methyl" {
		t.Fatalf("nested task = %+v, want name bismark_methylation_extractor module methyl", task)
	}
	if !pc.ContainsAll(task.Command, "--no_overlap") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
	pc.AssertIOPath(t, task.Outputs, "bedgraph", "work/bismark-extractor/aligned_pe.bedGraph.gz")
	pc.AssertIOPath(t, task.Outputs, "cpg", "work/bismark-extractor/CpG_context_aligned_pe.txt.gz")
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
