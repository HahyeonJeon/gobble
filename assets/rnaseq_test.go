package assets

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

const rnaFixtureSheet = "testdata/rnaseq-samplesheet.csv"

func TestRNASeqComposeBuildPlan(t *testing.T) {
	withSampleSheet(t, rnaFixtureSheet)
	raw := mustPlanJSON(t, RNASeq())
	tasks := planAllTasks(t, raw)

	for _, id := range []string{
		"ctrl1.raw.fastqc", "ctrl2.raw.fastqc", "treat1.raw.fastqc", "treat2.raw.fastqc",
		"ctrl1.clean.fastqc", "ctrl1.fastp", "ctrl1.star_align", "ctrl1.samtools_sort",
		"ctrl1.samtools_index", "ctrl1.featurecounts",
		"ctrl2.star_align", "treat1.star_align", "treat2.star_align",
		"star_genome_generate", "merge_counts", "deseq2", "multiqc",
	} {
		mustHaveTaskID(t, tasks, id)
	}

	if got := countTasksNamed(tasks, "star_genome_generate"); got != 1 {
		t.Fatalf("star_genome_generate count = %d, want 1", got)
	}
	if got := countTasksNamed(tasks, "star_align"); got != 4 {
		t.Fatalf("star_align count = %d, want 4", got)
	}
	if got := countTasksNamed(tasks, "merge_counts"); got != 1 {
		t.Fatalf("merge_counts count = %d, want 1", got)
	}
	if got := countTasksNamed(tasks, "deseq2"); got != 1 {
		t.Fatalf("deseq2 count = %d, want 1", got)
	}
	if got := countTasksNamed(tasks, "multiqc"); got != 1 {
		t.Fatalf("multiqc count = %d, want 1", got)
	}
	if got := countTasksNamed(tasks, "featurecounts"); got != 4 {
		t.Fatalf("featurecounts count = %d, want 4", got)
	}

	rawQC := planTask(t, raw, "ctrl1.raw.fastqc")
	if rawQC.Module != "raw" {
		t.Fatalf("ctrl1.raw.fastqc module = %q, want raw", rawQC.Module)
	}
	cleanQC := planTask(t, raw, "ctrl1.clean.fastqc")
	if cleanQC.Module != "clean" {
		t.Fatalf("ctrl1.clean.fastqc module = %q, want clean", cleanQC.Module)
	}

	gg := planTask(t, raw, "star_genome_generate")
	if !containsAll(gg.Command,
		"--sjdbGTFfile", "in/genes.gtf",
		"--runThreadN", "2",
		"--genomeSAindexNbases", "7",
		"--sjdbOverhang", "100",
	) {
		t.Fatalf("genomeGenerate command = %#v, want GTF, threads, and extra-args", gg.Command)
	}
	assertIOPath(t, gg.Inputs, "fasta", "in/genome.fasta")
	assertIOPath(t, gg.Inputs, "gtf", "in/genes.gtf")
	assertTreeIO(t, gg.Outputs, "index", "work/star-genome")

	align := planTask(t, raw, "ctrl1.star_align")
	if !containsAll(align.Command, "--runThreadN", "2") {
		t.Fatalf("align command = %#v, want --runThreadN 2", align.Command)
	}
	assertIOPath(t, align.Outputs, "bam", "work/ctrl1/star-align/Aligned.out.bam")
	assertIOPath(t, align.Outputs, "log_final", "work/ctrl1/star-align/Log.final.out")
	assertTreeIO(t, align.Inputs, "index", "work/star-genome")

	fc := planTask(t, raw, "ctrl1.featurecounts")
	if !containsAll(fc.Command, "-s", "2", "-p", "work/ctrl1/samtools-sort/Aligned.bam") {
		t.Fatalf("featurecounts command = %#v, want reverse strand and sorted BAM", fc.Command)
	}
	assertIOPath(t, fc.Inputs, "bam", "work/ctrl1/samtools-sort/Aligned.bam")
	assertIOPath(t, fc.Outputs, "counts", "work/ctrl1/featurecounts/counts.txt")

	if !containsAll(planTask(t, raw, "deseq2").Command, "ctrl", "treat", "work/deseq2/results.csv") {
		t.Fatalf("deseq2 command = %#v, want ctrl/treat groups and results dest", planTask(t, raw, "deseq2").Command)
	}
	assertIOPath(t, planTask(t, raw, "merge_counts").Outputs, "counts", "work/deseq2/counts.csv")
	assertIOPath(t, planTask(t, raw, "deseq2").Outputs, "results", "work/deseq2/results.csv")

	assertIOPath(t, planTask(t, raw, "ctrl1.fastp").Inputs, "r1", "in/SRR6357072_1.fastq.gz")
	assertIOPath(t, planTask(t, raw, "ctrl1.fastp").Inputs, "r2", "in/SRR6357072_2.fastq.gz")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_0", "work/ctrl1/raw/fastqc/SRR6357072_1_fastqc.html")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_4", "work/ctrl1/fastp/SRR6357072_1.fastp.json")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_6", "work/ctrl1/star-align/Log.final.out")
	for _, in := range planTask(t, raw, "multiqc").Inputs {
		if strings.HasSuffix(in.Path, ".bam") || strings.Contains(in.Path, "counts") {
			t.Fatalf("multiqc input %q path = %q, MultiQC must not consume BAM or count tables", in.Name, in.Path)
		}
	}

	assertNoTaskName(t, tasks, "bwa_index", "bwa_mem", "bismark_align", "bismark_genome", "bismark_methylation_extractor")
}

func TestRNASeqEmptyStrandednessBindsReverse(t *testing.T) {
	csv := strings.Join([]string{
		"sample,read1,read2,group,strandedness",
		"ctrl1,in/SRR6357072_1.fastq.gz,in/SRR6357072_2.fastq.gz,ctrl,",
		"treat1,in/SRR6357072_1.fastq.gz,in/SRR6357072_2.fastq.gz,treat,",
	}, "\n") + "\n"
	withSampleSheet(t, writeTempSheet(t, csv))
	raw := mustPlanJSON(t, RNASeq())
	fc := planTask(t, raw, "ctrl1.featurecounts")
	if !containsAll(fc.Command, "-s", "2") {
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
			p := RNASeq()
			ge := mustComposeSheetError(t, p)
			if !hasSheetMessage(ge, tt.msg) {
				t.Fatalf("defects = %+v, want message %q", ge.Defects, tt.msg)
			}
		})
	}
}

func TestRNASeqBadSheetIsSampleSheetError(t *testing.T) {
	withSampleSheet(t, filepath.Join(t.TempDir(), "missing.csv"))
	p := RNASeq()
	if p == nil {
		t.Fatal("RNASeq() = nil, want pipeline")
	}
	mustComposeSheetError(t, p)

	withSampleSheet(t, writeTempSheet(t, "not a samplesheet\n"))
	mustComposeSheetError(t, RNASeq())
}

func TestRNASeqOmitsRawAddTask(t *testing.T) {
	assertNoCall(t, "rnaseq.go", "AddTask")
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
