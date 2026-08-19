package assets

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestBismarkAlignStandaloneComposeBuildPlan(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"}
	opts := BismarkAlignOptions{
		ExtraArgs: []string{"--quiet"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := BismarkAlignPipeline(fasta, r1, r2, opts)
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "bismark_align")
	if task.Name != "bismark_align" {
		t.Fatalf("task name = %q, want bismark_align", task.Name)
	}
	if task.Image != wantBismarkImage {
		t.Fatalf("image = %q, want %q", task.Image, wantBismarkImage)
	}
	if commandHasSamtools(task.Command) {
		t.Fatalf("command = %#v, must not contain samtools", task.Command)
	}
	if !containsAll(task.Command,
		"bismark",
		"--genome", "work/bismark-genome",
		"--bam",
		"--output_dir", "work/bismark-align",
		"--basename", "aligned",
		"-p", "2",
		"-1", "in/test_1.fastq.gz",
		"-2", "in/test_2.fastq.gz",
		"--quiet",
	) {
		t.Fatalf("command = %#v, want named flags then extra-args", task.Command)
	}
	n := len(task.Command)
	if n < 1 || task.Command[n-1] != "--quiet" {
		t.Fatalf("command tail = %#v, want [--quiet]", task.Command)
	}
	assertUniqueParamNames(t, task.Params)
	assertIOPath(t, task.Inputs, "fasta", "work/bismark-genome/genome.fasta")
	assertIOSource(t, task.Inputs, "fasta", "in/genome.fasta")
	assertIOPath(t, task.Outputs, "bam", "work/bismark-align/aligned_pe.bam")
	assertIOPath(t, task.Outputs, "report", "work/bismark-align/aligned_PE_report.txt")
	assertGroupMembers(t, task.Inputs, "index", wantBismarkGenomeMembers("work/bismark-genome"))
	assertNoTaskName(t, planAllTasks(t, raw), "index_files")
	for _, rec := range planAllTasks(t, raw) {
		if len(rec.Command) == 1 && rec.Command[0] == "true" {
			t.Fatalf("fixture task %s still present", rec.ID)
		}
	}
}

func TestBismarkAlignNestedModule(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"}
	p := gobble.NewPipeline("assay")
	hf := p.AddInput("fasta", fasta)
	h1 := p.AddInput("r1", r1)
	h2 := p.AddInput("r2", r2)
	mod := AddModule(p, "align")
	idx := AddBismarkGenome(mod, hf, BismarkGenomeOptions{ExtraArgs: []string{"--verbose"}})
	ports := AddBismarkAlign(mod, hf, idx.Index, h1, h2, BismarkAlignOptions{ExtraArgs: []string{"--quiet"}})
	if ports.BAM.IsZero() || ports.Report.IsZero() {
		t.Fatalf("ports BAM/Report IsZero = %v/%v, want false", ports.BAM.IsZero(), ports.Report.IsZero())
	}
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "align.bismark_align")
	if task.Name != "bismark_align" || task.Module != "align" {
		t.Fatalf("nested task = %+v, want name bismark_align module align", task)
	}
	if commandHasSamtools(task.Command) {
		t.Fatalf("command = %#v, must not contain samtools", task.Command)
	}
	if !containsAll(task.Command, "--quiet") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
	assertIOPath(t, task.Outputs, "bam", "work/bismark-align/aligned_pe.bam")
	assertIOPath(t, task.Outputs, "report", "work/bismark-align/aligned_PE_report.txt")
	assertGroupMembers(t, task.Inputs, "index", wantBismarkGenomeMembers("work/bismark-genome"))
}

func TestBismarkAlignNestedRun(t *testing.T) {
	requireDocker(t)
	dir := t.TempDir()
	stageMethylPins(t, dir)
	p := gobble.NewPipeline("methyl")
	hf := p.AddInput("fasta", pinnedMethylFASTA())
	h1 := p.AddInput("r1", pinnedMethylFASTQ1())
	h2 := p.AddInput("r2", pinnedMethylFASTQ2())
	idx := AddBismarkGenome(p, hf, BismarkGenomeOptions{Resources: gobble.Resources{CPU: 1}})
	AddBismarkAlign(p, hf, idx.Index, h1, h2, BismarkAlignOptions{Resources: gobble.Resources{CPU: 1}})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, filepath.FromSlash("work/bismark-align/aligned_pe.bam")))
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("published BAM: %v", err)
	}
	unique := uniquePEAlignments(t, filepath.Join(dir, filepath.FromSlash("work/bismark-align/aligned_PE_report.txt")))
	t.Logf("unique paired-end alignments = %d", unique)
	assertUniqueAlignmentFloor(t, unique)
}

const uniquePEAlignmentField = "Number of paired-end alignments with a unique best hit:"

func uniquePEAlignments(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, uniquePEAlignmentField) {
			continue
		}
		_, value, ok := strings.Cut(line, ":")
		if !ok {
			t.Fatalf("unique-alignment line %q missing value", line)
		}
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			t.Fatalf("unique-alignment count in %q: %v", line, err)
		}
		return n
	}
	t.Fatalf("%s missing unique-alignment field %q", path, uniquePEAlignmentField)
	return 0
}

func assertUniqueAlignmentFloor(t *testing.T, unique int) {
	t.Helper()
	if unique < 1 {
		t.Fatalf("unique paired-end alignments = %d, want floor > 0", unique)
	}
}
