package methylseqevidence_test

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	bismarkalign "github.com/HahyeonJeon/gobble/assets/modules/bismark-align"
	bismarkgenome "github.com/HahyeonJeon/gobble/assets/modules/bismark-genome"
	. "github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/internal/sourcecheck"
	bismarkevidence "github.com/HahyeonJeon/gobble/tests/modules/bismark-genome"
)

const methylFixtureSheet = "testdata/methylseq-samplesheet.csv"

const (
	methylTwoRowRuleMessage = "Methyl samplesheet requires at least two samples"
	methylMateRuleMessage   = "Methyl samplesheet requires read2 on every row"
)

func TestMethylSeqComposeBuildPlan(t *testing.T) {
	withSampleSheet(t, methylFixtureSheet)
	raw := pc.MustPlanJSON(t, Pipeline())
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != "d6cd91ea0e4962f3cea1eec9633912a1f6f9b01261445b28f356c9420b23e517" {
		t.Fatalf("graph snapshot = %s, want pre-move Methyl-seq identity", got)
	}
	tasks := pc.AllTasks(t, raw)

	pc.MustHaveTaskID(t, tasks, "sample1.raw.fastqc")
	pc.MustHaveTaskID(t, tasks, "sample1.clean.fastqc")
	pc.MustHaveTaskID(t, tasks, "sample1.fastp")
	pc.MustHaveTaskID(t, tasks, "sample1.bismark_align")
	pc.MustHaveTaskID(t, tasks, "sample1.bismark_methylation_extractor")
	pc.MustHaveTaskID(t, tasks, "sample2.raw.fastqc")
	pc.MustHaveTaskID(t, tasks, "sample2.fastp")
	pc.MustHaveTaskID(t, tasks, "sample2.bismark_align")
	pc.MustHaveTaskID(t, tasks, "sample2.bismark_methylation_extractor")
	pc.MustHaveTaskID(t, tasks, "bismark_genome")
	pc.MustHaveTaskID(t, tasks, "multiqc")

	if got := pc.CountTasksNamed(tasks, "bismark_genome"); got != 1 {
		t.Fatalf("bismark_genome count = %d, want 1", got)
	}
	if got := pc.CountTasksNamed(tasks, "bismark_align"); got != 2 {
		t.Fatalf("bismark_align count = %d, want 2", got)
	}
	if got := pc.CountTasksNamed(tasks, "bismark_methylation_extractor"); got != 2 {
		t.Fatalf("bismark_methylation_extractor count = %d, want 2", got)
	}
	if got := pc.CountTasksNamed(tasks, "multiqc"); got != 1 {
		t.Fatalf("multiqc count = %d, want 1", got)
	}

	rawQC := pc.TaskByID(t, raw, "sample1.raw.fastqc")
	if rawQC.Module != "raw" {
		t.Fatalf("sample1.raw.fastqc module = %q, want raw", rawQC.Module)
	}
	cleanQC := pc.TaskByID(t, raw, "sample1.clean.fastqc")
	if cleanQC.Module != "clean" {
		t.Fatalf("sample1.clean.fastqc module = %q, want clean", cleanQC.Module)
	}

	genome := pc.TaskByID(t, raw, "bismark_genome")
	if genome.Image != bismarkevidence.Image {
		t.Fatalf("bismark_genome image = %q, want %q", genome.Image, bismarkevidence.Image)
	}
	if !pc.ContainsAll(genome.Command, "work/bismark-genome") {
		t.Fatalf("bismark_genome command = %#v, want restaged dest", genome.Command)
	}
	pc.AssertIOPath(t, genome.Inputs, "fasta", "work/bismark-genome/genome.fa")
	pc.AssertIOSource(t, genome.Inputs, "fasta", "in/genome.fa")
	pc.AssertGroupMembers(t, genome.Outputs, "index", bismarkevidence.Members("work/bismark-genome"))

	align := pc.TaskByID(t, raw, "sample1.bismark_align")
	if align.Image != bismarkevidence.Image {
		t.Fatalf("bismark_align image = %q, want %q", align.Image, bismarkevidence.Image)
	}
	if !pc.ContainsAll(align.Command, "--genome", "work/bismark-genome", "--output_dir", "work/sample1/bismark-align") {
		t.Fatalf("bismark_align command = %#v, want restaged genome and --output_dir", align.Command)
	}
	pc.AssertIOPath(t, align.Outputs, "report", "work/sample1/bismark-align/aligned_PE_report.txt")
	pc.AssertGroupMembers(t, align.Inputs, "index", bismarkevidence.Members("work/bismark-genome"))

	extractor := pc.TaskByID(t, raw, "sample1.bismark_methylation_extractor")
	if extractor.Image != bismarkevidence.Image {
		t.Fatalf("extractor image = %q, want %q", extractor.Image, bismarkevidence.Image)
	}
	if !pc.ContainsAll(extractor.Command, "--output_dir", "work/sample1/bismark-extract", "-p") {
		t.Fatalf("extractor command = %#v, want --output_dir and paired-end -p", extractor.Command)
	}

	pc.AssertIOPath(t, pc.TaskByID(t, raw, "sample1.fastp").Inputs, "r1", "in/Ecoli_10K_methylated_R1.fastq.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "sample1.fastp").Inputs, "r2", "in/Ecoli_10K_methylated_R2.fastq.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Inputs, "report_0", "work/sample1/raw/fastqc/Ecoli_10K_methylated_R1_fastqc.html")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Inputs, "report_4", "work/sample1/fastp/Ecoli_10K_methylated_R1.fastp.json")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Inputs, "report_6", "work/sample1/bismark-align/aligned_PE_report.txt")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Inputs, "report_7", "work/sample1/bismark-extract/aligned_pe_splitting_report.txt")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Inputs, "report_8", "work/sample1/bismark-extract/aligned_pe.M-bias.txt")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Inputs, "report_9", "work/sample1/bismark-extract/aligned_pe.bismark.cov.gz")
	for _, in := range pc.TaskByID(t, raw, "multiqc").Inputs {
		if strings.HasSuffix(in.Path, ".bam") {
			t.Fatalf("multiqc input %q path = %q, MultiQC must not consume BAM", in.Name, in.Path)
		}
	}

	pc.AssertNoTaskName(t, tasks, "bwa_index", "bwa_mem", "star_align", "star_genome_generate", "index_files")
	pc.AssertNoTaskName(t, tasks, "dss", "metilene", "methylkit", "dmr")
}

func TestMethylSeqCPUFlagsCompose(t *testing.T) {
	p := gobble.NewPipeline("methyl-cpu")
	hf := p.AddInput("fasta", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fa"})
	h1 := p.AddInput("r1", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "Ecoli_10K_methylated_R1", Ext: ".fastq.gz"})
	h2 := p.AddInput("r2", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "Ecoli_10K_methylated_R2", Ext: ".fastq.gz"})
	idx := bismarkgenome.AddBismarkGenome(p, hf, bismarkgenome.BismarkGenomeOptions{Resources: gobble.Resources{CPU: 2}})
	bismarkalign.AddBismarkAlign(p, hf, idx.Index, h1, h2, bismarkalign.BismarkAlignOptions{Resources: gobble.Resources{CPU: 2}})
	raw := pc.MustPlanJSON(t, p)
	genome := pc.TaskByID(t, raw, "bismark_genome")
	if !pc.ContainsAll(genome.Command, "--parallel", "2") {
		t.Fatalf("bismark_genome command = %#v, want --parallel 2", genome.Command)
	}
	align := pc.TaskByID(t, raw, "bismark_align")
	if !pc.ContainsAll(align.Command, "-p", "2") {
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
			ge := mustComposeSheetError(t, Pipeline())
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
	p := Pipeline()
	ge := mustComposeSheetError(t, p)
	if !hasSheetMessage(ge, methylTwoRowRuleMessage) {
		t.Fatalf("defects = %+v, want message %q", ge.Defects, methylTwoRowRuleMessage)
	}
}

func TestMethylSeqBadSheetIsSampleSheetError(t *testing.T) {
	withSampleSheet(t, filepath.Join(t.TempDir(), "missing.csv"))
	p := Pipeline()
	if p == nil {
		t.Fatal("MethylSeq() = nil, want pipeline")
	}
	mustComposeSheetError(t, p)

	withSampleSheet(t, writeTempSheet(t, "not a samplesheet\n"))
	mustComposeSheetError(t, Pipeline())
}

func TestMethylSeqOmitsRawAddTask(t *testing.T) {
	sourcecheck.AssertNoCall(t, "../../../assets/pipelines/methylseq/methylseq.go", "AddTask")
}

func withSampleSheet(t *testing.T, path string) {
	t.Helper()
	previous := gobble.SampleSheetPath()
	gobble.SetSampleSheetPath(path)
	t.Cleanup(func() { gobble.SetSampleSheetPath(previous) })
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
	graph, err := gobble.Compose(p)
	if graph != nil {
		t.Fatal("Compose() graph != nil, want no tasks")
	}
	if !gobble.IsSampleSheetError(err) {
		t.Fatalf("IsSampleSheetError() = false, error = %v", err)
	}
	var gobbleError *gobble.Error
	if !errors.As(err, &gobbleError) {
		t.Fatalf("error = %v, want *Error", err)
	}
	return gobbleError
}

func hasSheetMessage(gobbleError *gobble.Error, message string) bool {
	if gobbleError == nil {
		return false
	}
	for _, defect := range gobbleError.Defects {
		if defect.Message == message {
			return true
		}
	}
	return false
}
