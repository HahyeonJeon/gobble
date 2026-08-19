package assets

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestSTARAlignStandaloneComposeBuildPlan(t *testing.T) {
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"}
	opts := STARAlignOptions{
		ExtraArgs: []string{"--outFilterMultimapNmax", "1"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := STARAlignPipeline(r1, r2, opts)
	raw := mustPlanJSON(t, p)
	tasks := planAllTasks(t, raw)
	assertNoTaskName(t, tasks, "index_files")
	task := planTask(t, raw, "star_align")
	if task.Name != "star_align" {
		t.Fatalf("task name = %q, want star_align", task.Name)
	}
	if task.Image != starImage {
		t.Fatalf("image = %q, want %q", task.Image, starImage)
	}
	if commandHasSamtools(task.Command) {
		t.Fatalf("command = %#v, must not contain samtools", task.Command)
	}
	if !containsAll(task.Command,
		"STAR",
		"--genomeDir", "work/star-genome",
		"--readFilesIn", "in/test_1.fastq.gz", "in/test_2.fastq.gz",
		"--readFilesCommand", "zcat",
		"--runThreadN", "2",
		"--outFileNamePrefix", "work/star-align/",
		"--outSAMtype", "BAM", "Unsorted",
		"--outFilterMultimapNmax", "1",
	) {
		t.Fatalf("command = %#v, want named flags then extra-args", task.Command)
	}
	n := len(task.Command)
	if n < 2 || task.Command[n-2] != "--outFilterMultimapNmax" || task.Command[n-1] != "1" {
		t.Fatalf("command tail = %#v, want [--outFilterMultimapNmax 1]", task.Command)
	}
	assertUniqueParamNames(t, task.Params)
	assertIOPath(t, task.Outputs, "bam", "work/star-align/Aligned.out.bam")
	assertIOPath(t, task.Outputs, "log_final", "work/star-align/Log.final.out")
	assertGroupMembers(t, task.Inputs, "index", wantSTARGenomeMembers("work/star-genome"))
}

func TestSTARAlignNestedModule(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"}
	p := gobble.NewPipeline("assay")
	hf := p.AddInput("fasta", fasta)
	h1 := p.AddInput("r1", r1)
	h2 := p.AddInput("r2", r2)
	mod := AddModule(p, "align")
	idx := AddSTARGenomeGenerate(mod, hf, gobble.Handle{}, STARGenomeGenerateOptions{ExtraArgs: []string{"--genomeSAindexNbases", "7"}})
	ports := AddSTARAlign(mod, idx.Index, h1, h2, STARAlignOptions{ExtraArgs: []string{"--outFilterMultimapNmax", "1"}})
	if ports.BAM.IsZero() {
		t.Fatalf("ports.BAM IsZero = true, want false")
	}
	if ports.LogFinalOut.IsZero() {
		t.Fatalf("ports.LogFinalOut IsZero = true, want false")
	}
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "align.star_align")
	if task.Name != "star_align" || task.Module != "align" {
		t.Fatalf("nested task = %+v, want name star_align module align", task)
	}
	if commandHasSamtools(task.Command) {
		t.Fatalf("command = %#v, must not contain samtools", task.Command)
	}
	if !containsAll(task.Command, "--outFilterMultimapNmax", "1") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
	assertIOPath(t, task.Outputs, "bam", "work/star-align/Aligned.out.bam")
	assertIOPath(t, task.Outputs, "log_final", "work/star-align/Log.final.out")
	assertGroupMembers(t, task.Inputs, "index", wantSTARGenomeMembers("work/star-genome"))
}

func TestSTARAlignNestedRun(t *testing.T) {
	requireDocker(t)
	srcFASTA := cachePin(t, PinRNAGenomeFASTA)
	srcGTF := cachePin(t, PinRNAGTF)
	srcR1 := cachePin(t, PinRNATest1FASTQ)
	srcR2 := cachePin(t, PinRNATest2FASTQ)
	dir := t.TempDir()
	stageFile(t, dir, "in/genome.fasta", srcFASTA)
	stageFile(t, dir, "in/genes.gtf", srcGTF)
	stageFile(t, dir, "in/SRR6357072_1.fastq.gz", srcR1)
	stageFile(t, dir, "in/SRR6357072_2.fastq.gz", srcR2)
	p := gobble.NewPipeline("rna")
	hf := p.AddInput("fasta", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"})
	hg := p.AddInput("gtf", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genes", Ext: ".gtf"})
	h1 := p.AddInput("r1", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "SRR6357072_1", Ext: ".fastq.gz"})
	h2 := p.AddInput("r2", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "SRR6357072_2", Ext: ".fastq.gz"})
	idx := AddSTARGenomeGenerate(p, hf, hg, STARGenomeGenerateOptions{
		ExtraArgs: []string{"--genomeSAindexNbases", "7", "--sjdbOverhang", "100"},
		Resources: gobble.Resources{CPU: 1},
	})
	ports := AddSTARAlign(p, idx.Index, h1, h2, STARAlignOptions{
		SJDB:      true,
		Resources: gobble.Resources{CPU: 1},
	})
	if ports.LogFinalOut.IsZero() {
		t.Fatalf("ports.LogFinalOut IsZero = true, want false")
	}
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(t.Context(), g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	bam := filepath.Join(dir, filepath.FromSlash("work/star-align/Aligned.out.bam"))
	info, err := os.Stat(bam)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("published BAM: %v", err)
	}
	logPath := filepath.Join(dir, filepath.FromSlash("work/star-align/Log.final.out"))
	assertUniquelyMappedAbove(t, logPath, 10)
	assertSplicesRecorded(t, logPath)
}

func assertUniquelyMappedAbove(t *testing.T, path string, floor int) {
	t.Helper()
	n := uniquelyMappedReads(t, path)
	if n < floor {
		t.Fatalf("uniquely mapped reads = %d, want >= %d in %s", n, floor, path)
	}
}

func uniquelyMappedReads(t *testing.T, path string) int {
	t.Helper()
	return starLogInt(t, path, "Uniquely mapped reads number")
}

const starSplicesTotalField = "Number of splices: Total"

func assertSplicesRecorded(t *testing.T, path string) {
	t.Helper()
	n := starLogInt(t, path, starSplicesTotalField)
	t.Logf("%s = %d", starSplicesTotalField, n)
	if n < 1 {
		t.Fatalf("%s = %d, want >= 1 in %s", starSplicesTotalField, n, path)
	}
}

func starLogInt(t *testing.T, path, field string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, field) {
			continue
		}
		i := strings.LastIndex(line, "|")
		if i < 0 {
			t.Fatalf("%s line %q: missing |", field, line)
		}
		n, err := strconv.Atoi(strings.TrimSpace(line[i+1:]))
		if err != nil {
			t.Fatalf("%s line %q: %v", field, line, err)
		}
		return n
	}
	t.Fatalf("%s: missing %s", path, field)
	return 0
}
