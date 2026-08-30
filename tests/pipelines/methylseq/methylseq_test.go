package methylseqevidence_test

import (
	"bytes"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/internal/sourcecheck"
)

const methylFixtureSheet = "testdata/methylseq-samplesheet.csv"

func TestParseOfficialRepeatedSingleAndPairedRuns(t *testing.T) {
	samples := loadSamples(t)
	if got, want := len(samples), 3; got != want {
		t.Fatalf("sample count = %d, want %d", got, want)
	}
	if samples[0].Name != "SRR389222_sub1" || len(samples[0].Runs) != 1 || samples[0].Runs[0].Fastq2 != "" {
		t.Fatalf("first sample = %+v, want official single-end row", samples[0])
	}
	if samples[1].Name != "SRR389222_sub2" || len(samples[1].Runs) != 2 || samples[1].Runs[0].ID != "run_1" || samples[1].Runs[1].ID != "run_2" {
		t.Fatalf("repeated sample = %+v, want two ordered runs", samples[1])
	}
	if samples[2].Name != "Ecoli_10K_methylated" || samples[2].Runs[0].Fastq2 == "" {
		t.Fatalf("paired sample = %+v, want official E. coli pair", samples[2])
	}
	samples[1].Runs[0].Fastq1 = "changed.fastq.gz"
	again := loadSamples(t)
	if again[1].Runs[0].Fastq1 == "changed.fastq.gz" {
		t.Fatal("Load retained a caller-owned run slice")
	}
}

func TestParseRejectsInvalidMethylData(t *testing.T) {
	tests := []struct {
		name string
		csv  string
		want string
	}{
		{name: "unknown column", csv: "sample,fastq_1,fastq_2,genome\na,in/a.fq.gz,,x\n", want: "unknown Methyl samplesheet column"},
		{name: "missing mate header", csv: "sample,fastq_1\na,in/a.fq.gz\n", want: "missing required Methyl samplesheet column"},
		{name: "URL", csv: "sample,fastq_1,fastq_2\na,https://example.invalid/a.fq.gz,\n", want: "workspace-relative"},
		{name: "mixed read mode", csv: "sample,fastq_1,fastq_2\na,in/a.fq.gz,in/a2.fq.gz\na,in/b.fq.gz,\n", want: "mixes single-end and paired-end"},
		{name: "duplicate run", csv: "sample,fastq_1,fastq_2\na,in/a.fq.gz,\na,in/a.fq.gz,\n", want: "duplicate sequencing run"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := methylseq.Parse(strings.NewReader(test.csv))
			if err == nil || !strings.Contains(errorDetails(err), test.want) || !gobble.IsSampleSheetError(err) {
				t.Fatalf("Parse error = %v (%s), want structured detail %q", err, errorDetails(err), test.want)
			}
		})
	}
}

func TestDirectionalBismarkPlanDeclaresSelectedProduct(t *testing.T) {
	raw := pc.MustPlanJSON(t, methylseq.Build(loadSamples(t), methylseq.DefaultConfig()))
	tasks := pc.AllTasks(t, raw)
	for _, id := range []string{
		"reference.bismark_genome_preparation",
		"SRR389222_sub1.run_1_raw_r1.fastqc",
		"SRR389222_sub1.trim_galore",
		"SRR389222_sub1.post_trim_r1.fastqc",
		"SRR389222_sub1.bismark_align",
		"SRR389222_sub1.bismark_deduplicate",
		"SRR389222_sub1.bismark_methylation_extractor",
		"SRR389222_sub1.bismark_report",
		"SRR389222_sub2.consolidate_r1.cat_fastq",
		"Ecoli_10K_methylated.run_1_raw_r2.fastqc",
		"Ecoli_10K_methylated.post_trim_r2.fastqc",
		"Ecoli_10K_methylated.bismark_align",
		"bismark_summary",
		"multiqc",
	} {
		pc.MustHaveTaskID(t, tasks, id)
	}
	for name, want := range map[string]int{
		"bismark_genome_preparation":    1,
		"trim_galore":                   3,
		"bismark_align":                 3,
		"bismark_deduplicate":           3,
		"bismark_methylation_extractor": 3,
		"bismark_report":                3,
		"bismark_summary":               1,
		"multiqc":                       1,
	} {
		if got := pc.CountTasksNamed(tasks, name); got != want {
			t.Errorf("%s task count = %d, want %d", name, got, want)
		}
	}
	pc.AssertNoTaskName(t, tasks, "fastp", "bwa_mem", "bwameth_align", "dss", "metilene", "methylkit", "dmr", "coverage2cytosine")

	genome := pc.TaskByID(t, raw, "reference.bismark_genome_preparation")
	pc.AssertTreeIO(t, genome.Outputs, "index", "work/reference/bismark-index")
	pc.AssertIOPath(t, genome.Inputs, "fasta", "work/reference/bismark-index/genome.fa")
	pc.AssertIOSource(t, genome.Inputs, "fasta", "in/reference/genome.fa")
	for _, id := range []string{"SRR389222_sub1.bismark_align", "SRR389222_sub2.bismark_align", "Ecoli_10K_methylated.bismark_align"} {
		align := pc.TaskByID(t, raw, id)
		pc.AssertTreeIO(t, align.Inputs, "index", "work/reference/bismark-index")
		if !pc.ContainsAll(align.Command, "--bowtie2", "--genome", "work/reference/bismark-index") || pc.ContainsAll(align.Command, "--non_directional") {
			t.Fatalf("task %s command = %#v, want directional Bowtie2", id, align.Command)
		}
	}
	if paired := pc.TaskByID(t, raw, "Ecoli_10K_methylated.bismark_align"); !pc.ContainsAll(paired.Command, "-1", "work/Ecoli_10K_methylated/trim-galore/Ecoli_10K_methylated_val_1.fq.gz", "-2", "work/Ecoli_10K_methylated/trim-galore/Ecoli_10K_methylated_val_2.fq.gz") {
		t.Fatalf("paired alignment command = %#v", paired.Command)
	}
	if single := pc.TaskByID(t, raw, "SRR389222_sub1.bismark_align"); pc.ContainsAll(single.Command, "-1") || pc.ContainsAll(single.Command, "-2") {
		t.Fatalf("single alignment command = %#v, want positional read", single.Command)
	}

	dedup := pc.TaskByID(t, raw, "Ecoli_10K_methylated.bismark_deduplicate")
	pc.AssertIOPath(t, dedup.Outputs, "deduplicated_bam", "results/methylseq/bismark/Ecoli_10K_methylated/Ecoli_10K_methylated_pe.deduplicated.bam")
	extractor := pc.TaskByID(t, raw, "Ecoli_10K_methylated.bismark_methylation_extractor")
	for name, path := range map[string]string{
		"cpg":      "results/methylseq/methylation-calls/Ecoli_10K_methylated/CpG_context_Ecoli_10K_methylated_pe.deduplicated.txt.gz",
		"chg":      "results/methylseq/methylation-calls/Ecoli_10K_methylated/CHG_context_Ecoli_10K_methylated_pe.deduplicated.txt.gz",
		"chh":      "results/methylseq/methylation-calls/Ecoli_10K_methylated/CHH_context_Ecoli_10K_methylated_pe.deduplicated.txt.gz",
		"coverage": "results/methylseq/methylation-calls/Ecoli_10K_methylated/Ecoli_10K_methylated_pe.deduplicated.bismark.cov.gz",
		"bedgraph": "results/methylseq/methylation-calls/Ecoli_10K_methylated/Ecoli_10K_methylated_pe.deduplicated.bedGraph.gz",
	} {
		pc.AssertIOPath(t, extractor.Outputs, name, path)
	}
	if !pc.ContainsAll(extractor.Command, "--comprehensive", "--paired-end", "--no_overlap", "--ignore_r2", "2") {
		t.Fatalf("extractor command = %#v, want selected defaults", extractor.Command)
	}
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "Ecoli_10K_methylated.bismark_report").Outputs, "html", "results/methylseq/reports/Ecoli_10K_methylated/Ecoli_10K_methylated.bismark_report.html")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "bismark_summary").Outputs, "html", "results/methylseq/summary/bismark_summary_report.html")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "multiqc").Outputs, "html", "results/methylseq/multiqc/multiqc_report.html")
	pc.AssertTreeIO(t, pc.TaskByID(t, raw, "multiqc").Outputs, "data", "results/methylseq/multiqc/multiqc_data")

	for _, task := range tasks {
		if task.Image != "" && !strings.Contains(task.Image, "@sha256:") {
			t.Errorf("task %s image = %q, want immutable digest", task.ID, task.Image)
		}
		if task.Name != "cat_fastq" && task.Script != "" {
			t.Errorf("task %s uses a script, want one direct executable", task.ID)
		}
	}
}

func TestReadyBismarkTreeSkipsGenomePreparation(t *testing.T) {
	config := methylseq.DefaultConfig()
	config.Reference.FASTA = gobble.PathSpec{}
	config.Reference.BismarkIndex = gobble.DeclareTree(gobble.Dir("in/reference/BismarkIndex"))
	raw := pc.MustPlanJSON(t, methylseq.Build(loadSamples(t), config))
	if got := pc.CountTasksNamed(pc.AllTasks(t, raw), "bismark_genome_preparation"); got != 0 {
		t.Fatalf("genome-preparation count = %d, want 0 for ready Tree", got)
	}
	align := pc.TaskByID(t, raw, "Ecoli_10K_methylated.bismark_align")
	pc.AssertTreeIO(t, align.Inputs, "index", "in/reference/BismarkIndex")
	if !pc.ContainsAll(align.Command, "--genome", "in/reference/BismarkIndex") {
		t.Fatalf("alignment command = %#v, want ready Tree path", align.Command)
	}
}

func TestPipelineAdapterMatchesTypedBuild(t *testing.T) {
	previous := gobble.SampleSheetPath()
	gobble.SetSampleSheetPath(methylFixtureSheet)
	t.Cleanup(func() { gobble.SetSampleSheetPath(previous) })
	got := pc.MustPlanJSON(t, methylseq.Pipeline())
	want := pc.MustPlanJSON(t, methylseq.Build(loadSamples(t), methylseq.DefaultConfig()))
	if !bytes.Equal(got, want) {
		t.Fatal("Pipeline plan differs from Load plus DefaultConfig plus Build")
	}
}

func TestTypedCustomizationIsVisibleAndDefaultsAreFresh(t *testing.T) {
	first := methylseq.DefaultConfig()
	first.BismarkAlign.Local = true
	first.Extractor.CoverageCutoff = 3
	first.Summary.ExtraArgs = []string{"--title", "Gobble Methyl"}
	custom := pc.MustPlanJSON(t, methylseq.Build(loadSamples(t), first))
	if !pc.ContainsAll(pc.TaskByID(t, custom, "SRR389222_sub1.bismark_align").Command, "--local") || !pc.ContainsAll(pc.TaskByID(t, custom, "SRR389222_sub1.bismark_methylation_extractor").Command, "--cutoff", "3") || !pc.ContainsAll(pc.TaskByID(t, custom, "bismark_summary").Command, "--title", "Gobble Methyl") {
		t.Fatal("typed Methyl customization is absent from selected commands")
	}
	plain := pc.MustPlanJSON(t, methylseq.Build(loadSamples(t), methylseq.DefaultConfig()))
	if pc.ContainsAll(pc.TaskByID(t, plain, "SRR389222_sub1.bismark_align").Command, "--local") || pc.ContainsAll(pc.TaskByID(t, plain, "bismark_summary").Command, "Gobble Methyl") {
		t.Fatal("DefaultConfig retained caller mutation")
	}
	if !slices.Equal(pc.TaskByID(t, custom, "SRR389222_sub1.trim_galore").Command, pc.TaskByID(t, plain, "SRR389222_sub1.trim_galore").Command) {
		t.Fatal("extractor/alignment customization changed upstream trimming identity")
	}
}

func TestBuildRejectsInvalidInputConfigAndRouteExtras(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*methylseq.Config)
		code   gobble.DefectCode
	}{
		{name: "reference escape", mutate: func(config *methylseq.Config) { config.Reference.FASTA = gobble.Literal("../genome.fa") }, code: gobble.DefectInvalidPath},
		{name: "ready Tree missing directory", mutate: func(config *methylseq.Config) { config.Reference.BismarkIndex = gobble.DeclareTree(gobble.Directory{}) }, code: gobble.DefectInvalidPath},
		{name: "ready Tree escape", mutate: func(config *methylseq.Config) {
			config.Reference.BismarkIndex = gobble.DeclareTree(gobble.Dir("../BismarkIndex"))
		}, code: gobble.DefectInvalidPath},
		{name: "special library", mutate: func(config *methylseq.Config) { config.BismarkAlign.ExtraArgs = []string{"--pbat"} }, code: gobble.DefectInvalidValue},
		{name: "minimap2 aligner", mutate: func(config *methylseq.Config) { config.BismarkAlign.ExtraArgs = []string{"--mm2"} }, code: gobble.DefectInvalidValue},
		{name: "alternate genome", mutate: func(config *methylseq.Config) {
			config.BismarkAlign.ExtraArgs = []string{"--genome_folder", "in/other-index"}
		}, code: gobble.DefectInvalidValue},
		{name: "alternate output directory", mutate: func(config *methylseq.Config) { config.BismarkAlign.ExtraArgs = []string{"-owork/other"} }, code: gobble.DefectInvalidValue},
		{name: "alternate basename", mutate: func(config *methylseq.Config) { config.BismarkAlign.ExtraArgs = []string{"-Bother"} }, code: gobble.DefectInvalidValue},
		{name: "alternate prefix", mutate: func(config *methylseq.Config) { config.BismarkAlign.ExtraArgs = []string{"--prefix=other"} }, code: gobble.DefectInvalidValue},
		{name: "deduplication output escape", mutate: func(config *methylseq.Config) { config.Deduplicate.ExtraArgs = []string{"-oother"} }, code: gobble.DefectInvalidValue},
		{name: "extractor output escape", mutate: func(config *methylseq.Config) { config.Extractor.ExtraArgs = []string{"-oother"} }, code: gobble.DefectInvalidValue},
		{name: "sample report output escape", mutate: func(config *methylseq.Config) { config.Report.ExtraArgs = []string{"-oother"} }, code: gobble.DefectInvalidValue},
		{name: "summary output escape", mutate: func(config *methylseq.Config) { config.Summary.ExtraArgs = []string{"-oother"} }, code: gobble.DefectInvalidValue},
		{name: "typed special library", mutate: func(config *methylseq.Config) { config.LibraryMode = methylseq.LibraryMode("pbat") }, code: gobble.DefectInvalidValue},
		{name: "alternate aligner", mutate: func(config *methylseq.Config) { config.BismarkGenome.ExtraArgs = []string{"--hisat2"} }, code: gobble.DefectInvalidValue},
		{name: "undeclared extractor outputs", mutate: func(config *methylseq.Config) { config.Extractor.ExtraArgs = []string{"--cytosine_report"} }, code: gobble.DefectInvalidValue},
		{name: "undeclared FastQC output", mutate: func(config *methylseq.Config) { config.FastQC.ExtraArgs = []string{"--extract"} }, code: gobble.DefectInvalidValue},
		{name: "changed MultiQC output", mutate: func(config *methylseq.Config) { config.MultiQC.ExtraArgs = []string{"--filename", "other.html"} }, code: gobble.DefectInvalidValue},
		{name: "disabled required result", mutate: func(config *methylseq.Config) { config.Publication.Reports = false }, code: gobble.DefectInvalidValue},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := methylseq.DefaultConfig()
			test.mutate(&config)
			graph, err := gobble.Compose(methylseq.Build(loadSamples(t), config))
			if graph != nil || !hasDefect(err, test.code) {
				t.Fatalf("Compose() = (%v, %v), want nil graph and %s", graph, err, test.code)
			}
		})
	}
	graph, err := gobble.Compose(methylseq.Build(nil, methylseq.DefaultConfig()))
	if graph != nil || !hasDefect(err, gobble.DefectInvalidSampleSheet) {
		t.Fatalf("empty Build Compose() = (%v, %v), want invalid-samplesheet", graph, err)
	}
}

func TestSingleSampleIsSupported(t *testing.T) {
	samples := []methylseq.Sample{{Name: "single", Runs: []methylseq.Run{{ID: "run_1", Fastq1: "in/single.fastq.gz"}}}}
	raw := pc.MustPlanJSON(t, methylseq.Build(samples, methylseq.DefaultConfig()))
	if pc.CountTasksNamed(pc.AllTasks(t, raw), "bismark_align") != 1 || pc.CountTasksNamed(pc.AllTasks(t, raw), "bismark_summary") != 1 {
		t.Fatal("one valid sample did not produce the complete selected path")
	}
}

func TestBuildHasNoAmbientOrNetworkInput(t *testing.T) {
	for _, source := range []string{"../../../assets/pipelines/methylseq/build.go", "../../../assets/pipelines/methylseq/config.go"} {
		data, err := os.ReadFile(source)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", source, err)
		}
		if strings.Contains(string(data), "http://") || strings.Contains(string(data), "https://") {
			t.Errorf("product source %s contains a network location", source)
		}
	}
	sourcecheck.AssertNoCall(t, "../../../assets/pipelines/methylseq/build.go", "SampleSheetPath", "Load", "Getenv", "Open", "ReadFile", "Get", "AddTask")
}

func TestLoadMissingSheetIsStructured(t *testing.T) {
	_, err := methylseq.Load("testdata/does-not-exist.csv")
	if !hasDefect(err, gobble.DefectNotFound) || !gobble.IsSampleSheetError(err) {
		t.Fatalf("Load missing error = %v, want structured samplesheet not-found", err)
	}
}

func loadSamples(t *testing.T) []methylseq.Sample {
	t.Helper()
	samples, err := methylseq.Load(methylFixtureSheet)
	if err != nil {
		t.Fatalf("Load(%s): %v", methylFixtureSheet, err)
	}
	return samples
}

func errorDetails(err error) string {
	var structured *gobble.Error
	if !errors.As(err, &structured) || structured == nil {
		return ""
	}
	var details strings.Builder
	for _, defect := range structured.Defects {
		details.WriteString(defect.Message)
		details.WriteByte('\n')
	}
	return details.String()
}

func hasDefect(err error, code gobble.DefectCode) bool {
	var structured *gobble.Error
	if !errors.As(err, &structured) || structured == nil {
		return false
	}
	for _, defect := range structured.Defects {
		if defect.Code == code {
			return true
		}
	}
	return false
}
