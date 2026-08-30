package staralign_test

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	. "github.com/HahyeonJeon/gobble/assets/modules/star-align"
	stargenomegenerate "github.com/HahyeonJeon/gobble/assets/modules/star-genome-generate"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestSTARAlignStandaloneComposeBuildPlan(t *testing.T) {
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"}
	opts := STARAlignOptions{
		ExtraArgs: []string{"--outFilterMultimapNmax", "1"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := STARAlignPipeline(r1, r2, opts)
	raw := pc.MustPlanJSON(t, p)
	tasks := pc.AllTasks(t, raw)
	pc.AssertNoTaskName(t, tasks, "index_files")
	task := pc.TaskByID(t, raw, "star_align")
	if task.Name != "star_align" {
		t.Fatalf("task name = %q, want star_align", task.Name)
	}
	if task.Image != "community.wave.seqera.io/library/htslib_samtools_star_gawk:ae438e9a604351a4" {
		t.Fatalf("image = %q, want locked STAR pin", task.Image)
	}
	if pc.ContainsSubstring(task.Command, "samtools") {
		t.Fatalf("command = %#v, must not contain samtools", task.Command)
	}
	if !pc.ContainsAll(task.Command,
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
	pc.AssertUniqueParamNames(t, task.Params)
	pc.AssertIOPath(t, task.Outputs, "bam", "work/star-align/Aligned.out.bam")
	pc.AssertIOPath(t, task.Outputs, "log_final", "work/star-align/Log.final.out")
	pc.AssertTreeIO(t, task.Inputs, "index", "work/star-genome")
}

func TestSTARAlignNestedModule(t *testing.T) {
	fasta := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
	r1 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	r2 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"}
	p := gobble.NewPipeline("assay")
	hf := p.AddInput("fasta", fasta)
	h1 := p.AddInput("r1", r1)
	h2 := p.AddInput("r2", r2)
	mod := p.AddModule("align")
	idx := stargenomegenerate.AddSTARGenomeGenerate(mod, hf, gobble.Handle{}, stargenomegenerate.STARGenomeGenerateOptions{ExtraArgs: []string{"--genomeSAindexNbases", "7"}})
	ports := AddSTARAlign(mod, idx.Index, h1, h2, STARAlignOptions{ExtraArgs: []string{"--outFilterMultimapNmax", "1"}})
	if ports.BAM.IsZero() {
		t.Fatalf("ports.BAM IsZero = true, want false")
	}
	if ports.LogFinalOut.IsZero() {
		t.Fatalf("ports.LogFinalOut IsZero = true, want false")
	}
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "align.star_align")
	if task.Name != "star_align" || task.Module != "align" {
		t.Fatalf("nested task = %+v, want name star_align module align", task)
	}
	if pc.ContainsSubstring(task.Command, "samtools") {
		t.Fatalf("command = %#v, must not contain samtools", task.Command)
	}
	if !pc.ContainsAll(task.Command, "--outFilterMultimapNmax", "1") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
	pc.AssertIOPath(t, task.Outputs, "bam", "work/star-align/Aligned.out.bam")
	pc.AssertIOPath(t, task.Outputs, "log_final", "work/star-align/Log.final.out")
	pc.AssertTreeIO(t, task.Inputs, "index", "work/star-genome")
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
