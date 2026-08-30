package rnaseqevidence_test

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	. "github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/internal/sourcecheck"
)

const rnaFixtureSheet = "testdata/rnaseq-samplesheet.csv"

const (
	rnaGroupRuleMessage        = "RNA samplesheet requires group on every row and exactly two groups"
	rnaStrandednessRuleMessage = "RNA samplesheet strandedness must be unstranded, forward, or reverse"
	rnaMateRuleMessage         = "RNA samplesheet requires read2 on every row"
)

func TestRNASeqComposeBuildPlan(t *testing.T) {
	withSampleSheet(t, rnaFixtureSheet)
	raw := pc.MustPlanJSON(t, Pipeline())
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != "827931c2a6addaf716b8a9ee62057177b3a1838d135d967dc575906fbd948667" {
		t.Fatalf("graph snapshot = %s, want pre-move RNA-seq identity", got)
	}
	tasks := pc.AllTasks(t, raw)

	for _, id := range []string{
		"ctrl1.raw.fastqc", "ctrl2.raw.fastqc", "treat1.raw.fastqc", "treat2.raw.fastqc",
		"ctrl1.clean.fastqc", "ctrl1.fastp", "ctrl1.star_align", "ctrl1.samtools_sort",
		"ctrl1.samtools_index", "ctrl1.featurecounts",
		"ctrl2.star_align", "treat1.star_align", "treat2.star_align",
		"star_genome_generate", "merge_counts", "deseq2", "multiqc",
	} {
		pc.MustHaveTaskID(t, tasks, id)
	}

	if got := pc.CountTasksNamed(tasks, "star_genome_generate"); got != 1 {
		t.Fatalf("star_genome_generate count = %d, want 1", got)
	}
	if got := pc.CountTasksNamed(tasks, "star_align"); got != 4 {
		t.Fatalf("star_align count = %d, want 4", got)
	}
	if got := pc.CountTasksNamed(tasks, "merge_counts"); got != 1 {
		t.Fatalf("merge_counts count = %d, want 1", got)
	}
	if got := pc.CountTasksNamed(tasks, "deseq2"); got != 1 {
		t.Fatalf("deseq2 count = %d, want 1", got)
	}
	if got := pc.CountTasksNamed(tasks, "multiqc"); got != 1 {
		t.Fatalf("multiqc count = %d, want 1", got)
	}
	if got := pc.CountTasksNamed(tasks, "featurecounts"); got != 4 {
		t.Fatalf("featurecounts count = %d, want 4", got)
	}

	rawQC := pc.TaskByID(t, raw, "ctrl1.raw.fastqc")
	if rawQC.Module != "raw" {
		t.Fatalf("ctrl1.raw.fastqc module = %q, want raw", rawQC.Module)
	}
	cleanQC := pc.TaskByID(t, raw, "ctrl1.clean.fastqc")
	if cleanQC.Module != "clean" {
		t.Fatalf("ctrl1.clean.fastqc module = %q, want clean", cleanQC.Module)
	}

	gg := pc.TaskByID(t, raw, "star_genome_generate")
	if !pc.ContainsAll(gg.Command,
		"--sjdbGTFfile", "in/genes.gtf",
		"--runThreadN", "2",
		"--genomeSAindexNbases", "7",
		"--sjdbOverhang", "100",
	) {
		t.Fatalf("genomeGenerate command = %#v, want GTF, threads, and extra-args", gg.Command)
	}
	pc.AssertIOPath(t, gg.Inputs, "fasta", "in/genome.fasta")
	pc.AssertIOPath(t, gg.Inputs, "gtf", "in/genes.gtf")
	pc.AssertTreeIO(t, gg.Outputs, "index", "work/star-genome")

	align := pc.TaskByID(t, raw, "ctrl1.star_align")
	if !pc.ContainsAll(align.Command, "--runThreadN", "2") {
		t.Fatalf("align command = %#v, want --runThreadN 2", align.Command)
	}
	pc.AssertIOPath(t, align.Outputs, "bam", "work/ctrl1/star-align/Aligned.out.bam")
	pc.AssertIOPath(t, align.Outputs, "log_final", "work/ctrl1/star-align/Log.final.out")
	pc.AssertTreeIO(t, align.Inputs, "index", "work/star-genome")

	fc := pc.TaskByID(t, raw, "ctrl1.featurecounts")
	if !pc.ContainsAll(fc.Command, "-s", "2", "-p", "work/ctrl1/samtools-sort/Aligned.bam") {
		t.Fatalf("featurecounts command = %#v, want reverse strand and sorted BAM", fc.Command)
	}
	pc.AssertIOPath(t, fc.Inputs, "bam", "work/ctrl1/samtools-sort/Aligned.bam")
	pc.AssertIOPath(t, fc.Outputs, "counts", "work/ctrl1/featurecounts/counts.txt")

	if !pc.ContainsAll(pc.TaskByID(t, raw, "deseq2").Command, "ctrl", "treat", "work/deseq2/results.csv") {
		t.Fatalf("deseq2 command = %#v, want ctrl/treat groups and results dest", pc.TaskByID(t, raw, "deseq2").Command)
	}
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "merge_counts").Outputs, "counts", "work/deseq2/counts.csv")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "deseq2").Outputs, "results", "work/deseq2/results.csv")

	pc.AssertIOPath(t, pc.TaskByID(t, raw, "ctrl1.fastp").Inputs, "r1", "in/SRR6357072_1.fastq.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "ctrl1.fastp").Inputs, "r2", "in/SRR6357072_2.fastq.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Inputs, "report_0", "work/ctrl1/raw/fastqc/SRR6357072_1_fastqc.html")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Inputs, "report_4", "work/ctrl1/fastp/SRR6357072_1.fastp.json")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Inputs, "report_6", "work/ctrl1/star-align/Log.final.out")
	for _, in := range pc.TaskByID(t, raw, "multiqc").Inputs {
		if strings.HasSuffix(in.Path, ".bam") || strings.Contains(in.Path, "counts") {
			t.Fatalf("multiqc input %q path = %q, MultiQC must not consume BAM or count tables", in.Name, in.Path)
		}
	}

	pc.AssertNoTaskName(t, tasks, "bwa_index", "bwa_mem", "bismark_align", "bismark_genome", "bismark_methylation_extractor")
}

func TestRNASeqEmptyStrandednessBindsReverse(t *testing.T) {
	csv := strings.Join([]string{
		"sample,read1,read2,group,strandedness",
		"ctrl1,in/SRR6357072_1.fastq.gz,in/SRR6357072_2.fastq.gz,ctrl,",
		"treat1,in/SRR6357072_1.fastq.gz,in/SRR6357072_2.fastq.gz,treat,",
	}, "\n") + "\n"
	withSampleSheet(t, writeTempSheet(t, csv))
	raw := pc.MustPlanJSON(t, Pipeline())
	fc := pc.TaskByID(t, raw, "ctrl1.featurecounts")
	if !pc.ContainsAll(fc.Command, "-s", "2") {
		t.Fatalf("featurecounts command = %#v, want default reverse -s 2", fc.Command)
	}
}

func TestRNASeqTwoGroupRule(t *testing.T) {
	tests := []struct {
		name string
		csv  string
		msg  string
	}{
		{
			name: "one group",
			csv: "sample,read1,read2,group\n" +
				"ctrl1,in/SRR6357072_1.fastq.gz,in/SRR6357072_2.fastq.gz,ctrl\n" +
				"ctrl2,in/SRR6357072_1.fastq.gz,in/SRR6357072_2.fastq.gz,ctrl\n",
			msg: rnaGroupRuleMessage,
		},
		{
			name: "three groups",
			csv: "sample,read1,read2,group\n" +
				"a,in/SRR6357072_1.fastq.gz,in/SRR6357072_2.fastq.gz,g1\n" +
				"b,in/SRR6357072_1.fastq.gz,in/SRR6357072_2.fastq.gz,g2\n" +
				"c,in/SRR6357072_1.fastq.gz,in/SRR6357072_2.fastq.gz,g3\n",
			msg: rnaGroupRuleMessage,
		},
		{
			name: "missing group column",
			csv: "sample,read1,read2\n" +
				"ctrl1,in/SRR6357072_1.fastq.gz,in/SRR6357072_2.fastq.gz\n" +
				"treat1,in/SRR6357072_1.fastq.gz,in/SRR6357072_2.fastq.gz\n",
			msg: rnaGroupRuleMessage,
		},
		{
			name: "empty group cell",
			csv: "sample,read1,read2,group\n" +
				"ctrl1,in/SRR6357072_1.fastq.gz,in/SRR6357072_2.fastq.gz,ctrl\n" +
				"treat1,in/SRR6357072_1.fastq.gz,in/SRR6357072_2.fastq.gz,\n",
			msg: rnaGroupRuleMessage,
		},
		{
			name: "invalid strandedness",
			csv: "sample,read1,read2,group,strandedness\n" +
				"ctrl1,in/SRR6357072_1.fastq.gz,in/SRR6357072_2.fastq.gz,ctrl,bogus\n" +
				"treat1,in/SRR6357072_1.fastq.gz,in/SRR6357072_2.fastq.gz,treat,reverse\n",
			msg: rnaStrandednessRuleMessage,
		},
		{
			name: "empty read2",
			csv: "sample,read1,read2,group\n" +
				"ctrl1,in/SRR6357072_1.fastq.gz,,ctrl\n" +
				"treat1,in/SRR6357072_1.fastq.gz,in/SRR6357072_2.fastq.gz,treat\n",
			msg: rnaMateRuleMessage,
		},
		{
			name: "omitted read2 header",
			csv: "sample,read1,group\n" +
				"ctrl1,in/SRR6357072_1.fastq.gz,ctrl\n" +
				"treat1,in/SRR6357072_1.fastq.gz,treat\n",
			msg: rnaMateRuleMessage,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withSampleSheet(t, writeTempSheet(t, tt.csv))
			p := Pipeline()
			ge := mustComposeSheetError(t, p)
			if !hasSheetMessage(ge, tt.msg) {
				t.Fatalf("defects = %+v, want message %q", ge.Defects, tt.msg)
			}
		})
	}
}

func TestRNASeqBadSheetIsSampleSheetError(t *testing.T) {
	withSampleSheet(t, filepath.Join(t.TempDir(), "missing.csv"))
	p := Pipeline()
	if p == nil {
		t.Fatal("RNASeq() = nil, want pipeline")
	}
	mustComposeSheetError(t, p)

	withSampleSheet(t, writeTempSheet(t, "not a samplesheet\n"))
	mustComposeSheetError(t, Pipeline())
}

func TestRNASeqOmitsRawAddTask(t *testing.T) {
	sourcecheck.AssertNoCall(t, "../../../assets/pipelines/rnaseq/rnaseq.go", "AddTask")
}

func withSampleSheet(t *testing.T, path string) {
	t.Helper()
	prev := gobble.SampleSheetPath()
	gobble.SetSampleSheetPath(path)
	t.Cleanup(func() { gobble.SetSampleSheetPath(prev) })
}

func writeTempSheet(t *testing.T, csv string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "samplesheet.csv")
	if err := os.WriteFile(path, []byte(csv), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
	return path
}

func mustComposeSheetError(t *testing.T, p *gobble.Pipeline) *gobble.Error {
	t.Helper()
	if p == nil {
		t.Fatal("constructor returned nil pipeline")
	}
	g, err := gobble.Compose(p)
	if g != nil {
		t.Fatal("Compose() graph != nil, want no tasks")
	}
	if !gobble.IsSampleSheetError(err) {
		t.Fatalf("IsSampleSheetError() = false, error = %v", err)
	}
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		t.Fatalf("error = %v, want *Error", err)
	}
	return ge
}

func hasSheetMessage(ge *gobble.Error, message string) bool {
	if ge == nil {
		return false
	}
	for _, d := range ge.Defects {
		if d.Message == message {
			return true
		}
	}
	return false
}
