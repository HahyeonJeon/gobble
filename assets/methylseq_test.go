package assets

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

const methylFixtureSheet = "testdata/methylseq-samplesheet.csv"

func TestMethylSeqComposeBuildPlan(t *testing.T) {
	withSampleSheet(t, methylFixtureSheet)
	raw := mustPlanJSON(t, MethylSeq())
	tasks := planAllTasks(t, raw)

	mustHaveTaskID(t, tasks, "sample1.raw.fastqc")
	mustHaveTaskID(t, tasks, "sample1.clean.fastqc")
	mustHaveTaskID(t, tasks, "sample1.fastp")
	mustHaveTaskID(t, tasks, "sample1.bismark_align")
	mustHaveTaskID(t, tasks, "sample1.bismark_methylation_extractor")
	mustHaveTaskID(t, tasks, "sample2.raw.fastqc")
	mustHaveTaskID(t, tasks, "sample2.fastp")
	mustHaveTaskID(t, tasks, "sample2.bismark_align")
	mustHaveTaskID(t, tasks, "sample2.bismark_methylation_extractor")
	mustHaveTaskID(t, tasks, "bismark_genome")
	mustHaveTaskID(t, tasks, "multiqc")

	if got := countTasksNamed(tasks, "bismark_genome"); got != 1 {
		t.Fatalf("bismark_genome count = %d, want 1", got)
	}
	if got := countTasksNamed(tasks, "bismark_align"); got != 2 {
		t.Fatalf("bismark_align count = %d, want 2", got)
	}
	if got := countTasksNamed(tasks, "bismark_methylation_extractor"); got != 2 {
		t.Fatalf("bismark_methylation_extractor count = %d, want 2", got)
	}
	if got := countTasksNamed(tasks, "multiqc"); got != 1 {
		t.Fatalf("multiqc count = %d, want 1", got)
	}

	rawQC := planTask(t, raw, "sample1.raw.fastqc")
	if rawQC.Module != "raw" {
		t.Fatalf("sample1.raw.fastqc module = %q, want raw", rawQC.Module)
	}
	cleanQC := planTask(t, raw, "sample1.clean.fastqc")
	if cleanQC.Module != "clean" {
		t.Fatalf("sample1.clean.fastqc module = %q, want clean", cleanQC.Module)
	}

	genome := planTask(t, raw, "bismark_genome")
	if genome.Image != wantBismarkImage {
		t.Fatalf("bismark_genome image = %q, want %q", genome.Image, wantBismarkImage)
	}
	if !containsAll(genome.Command, "work/bismark-genome") {
		t.Fatalf("bismark_genome command = %#v, want restaged dest", genome.Command)
	}
	assertIOPath(t, genome.Inputs, "fasta", "work/bismark-genome/genome.fa")
	assertIOSource(t, genome.Inputs, "fasta", "in/genome.fa")
	assertGroupMembers(t, genome.Outputs, "index", wantBismarkGenomeMembers("work/bismark-genome"))

	align := planTask(t, raw, "sample1.bismark_align")
	if align.Image != wantBismarkImage {
		t.Fatalf("bismark_align image = %q, want %q", align.Image, wantBismarkImage)
	}
	if !containsAll(align.Command, "--genome", "work/bismark-genome", "--output_dir", "work/sample1/bismark-align") {
		t.Fatalf("bismark_align command = %#v, want restaged genome and --output_dir", align.Command)
	}
	assertIOPath(t, align.Outputs, "report", "work/sample1/bismark-align/aligned_PE_report.txt")
	assertGroupMembers(t, align.Inputs, "index", wantBismarkGenomeMembers("work/bismark-genome"))

	extractor := planTask(t, raw, "sample1.bismark_methylation_extractor")
	if extractor.Image != wantBismarkImage {
		t.Fatalf("extractor image = %q, want %q", extractor.Image, wantBismarkImage)
	}
	if !containsAll(extractor.Command, "--output_dir", "work/sample1/bismark-extract", "-p") {
		t.Fatalf("extractor command = %#v, want --output_dir and paired-end -p", extractor.Command)
	}

	assertIOPath(t, planTask(t, raw, "sample1.fastp").Inputs, "r1", "in/Ecoli_10K_methylated_R1.fastq.gz")
	assertIOPath(t, planTask(t, raw, "sample1.fastp").Inputs, "r2", "in/Ecoli_10K_methylated_R2.fastq.gz")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_0", "work/sample1/raw/fastqc/Ecoli_10K_methylated_R1_fastqc.html")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_4", "work/sample1/fastp/Ecoli_10K_methylated_R1.fastp.json")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_6", "work/sample1/bismark-align/aligned_PE_report.txt")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_7", "work/sample1/bismark-extract/aligned_pe_splitting_report.txt")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_8", "work/sample1/bismark-extract/aligned_pe.M-bias.txt")
	assertIOPath(t, planTask(t, raw, "multiqc").Inputs, "report_9", "work/sample1/bismark-extract/aligned_pe.bismark.cov.gz")
	for _, in := range planTask(t, raw, "multiqc").Inputs {
		if strings.HasSuffix(in.Path, ".bam") {
			t.Fatalf("multiqc input %q path = %q, MultiQC must not consume BAM", in.Name, in.Path)
		}
	}

	assertNoTaskName(t, tasks, "bwa_index", "bwa_mem", "star_align", "star_genome_generate", "index_files")
	assertNoTaskName(t, tasks, "dss", "metilene", "methylkit", "dmr")
}

func TestMethylSeqCPUFlagsCompose(t *testing.T) {
	p := gobble.NewPipeline("methyl-cpu")
	hf := p.AddInput("fasta", pinnedMethylFASTA())
	h1 := p.AddInput("r1", pinnedMethylFASTQ1())
	h2 := p.AddInput("r2", pinnedMethylFASTQ2())
	idx := AddBismarkGenome(p, hf, BismarkGenomeOptions{Resources: gobble.Resources{CPU: 2}})
	AddBismarkAlign(p, hf, idx.Index, h1, h2, BismarkAlignOptions{Resources: gobble.Resources{CPU: 2}})
	raw := mustPlanJSON(t, p)
	genome := planTask(t, raw, "bismark_genome")
	if !containsAll(genome.Command, "--parallel", "2") {
		t.Fatalf("bismark_genome command = %#v, want --parallel 2", genome.Command)
	}
	align := planTask(t, raw, "bismark_align")
	if !containsAll(align.Command, "-p", "2") {
		t.Fatalf("bismark_align command = %#v, want -p 2", align.Command)
	}
}

func TestMethylSeqEmptyRead2IsSampleSheetError(t *testing.T) {
	tests := []struct {
		name string
		csv  string
	}{
		{
			name: "empty read2",
			csv: "sample,read1,read2\n" +
				"sample1,in/Ecoli_10K_methylated_R1.fastq.gz,\n" +
				"sample2,in/Ecoli_10K_methylated_R1.fastq.gz,in/Ecoli_10K_methylated_R2.fastq.gz\n",
		},
		{
			name: "omitted read2 header",
			csv: "sample,read1\n" +
				"sample1,in/Ecoli_10K_methylated_R1.fastq.gz\n" +
				"sample2,in/Ecoli_10K_methylated_R2.fastq.gz\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withSampleSheet(t, writeTempSheet(t, tt.csv))
			ge := mustComposeSheetError(t, MethylSeq())
			if !hasSheetMessage(ge, methylMateRuleMessage) {
				t.Fatalf("defects = %+v, want message %q", ge.Defects, methylMateRuleMessage)
			}
		})
	}
}

func TestMethylSeqTwoRowRule(t *testing.T) {
	csv := "sample,read1,read2\n" +
		"sample1,in/Ecoli_10K_methylated_R1.fastq.gz,in/Ecoli_10K_methylated_R2.fastq.gz\n"
	withSampleSheet(t, writeTempSheet(t, csv))
	p := MethylSeq()
	ge := mustComposeSheetError(t, p)
	if !hasSheetMessage(ge, methylTwoRowRuleMessage) {
		t.Fatalf("defects = %+v, want message %q", ge.Defects, methylTwoRowRuleMessage)
	}
}

func TestMethylSeqBadSheetIsSampleSheetError(t *testing.T) {
	withSampleSheet(t, filepath.Join(t.TempDir(), "missing.csv"))
	p := MethylSeq()
	if p == nil {
		t.Fatal("MethylSeq() = nil, want pipeline")
	}
	mustComposeSheetError(t, p)

	withSampleSheet(t, writeTempSheet(t, "not a samplesheet\n"))
	mustComposeSheetError(t, MethylSeq())
}

func TestMethylSeqOmitsRawAddTask(t *testing.T) {
	assertNoCall(t, "methylseq.go", "AddTask")
}
