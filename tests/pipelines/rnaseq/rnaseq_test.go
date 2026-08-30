package rnaseqevidence_test

import (
	"bytes"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/internal/sourcecheck"
)

const rnaFixtureSheet = "testdata/rnaseq-samplesheet.csv"
const rnaLiveFixtureSheet = "testdata/rnaseq-live-samplesheet.csv"

func TestParseOfficialMixedRunAndReadModes(t *testing.T) {
	samples, err := rnaseq.Load(rnaFixtureSheet)
	if err != nil {
		t.Fatalf("Load(%s) error = %v", rnaFixtureSheet, err)
	}
	if got, want := len(samples), 5; got != want {
		t.Fatalf("sample count = %d, want %d", got, want)
	}
	if got, want := len(samples[0].Runs), 2; got != want {
		t.Fatalf("WT_REP1 run count = %d, want %d", got, want)
	}
	if samples[0].Runs[0].ID != "run_1" || samples[0].Runs[1].ID != "run_2" {
		t.Fatalf("WT_REP1 run IDs = %+v, want run_1 and run_2", samples[0].Runs)
	}
	if samples[0].Strandedness != rnaseq.StrandednessAuto || samples[0].Runs[0].Fastq2 == "" {
		t.Fatalf("WT_REP1 = %+v, want paired auto sample", samples[0])
	}
	if samples[2].Runs[0].Fastq2 != "" {
		t.Fatalf("RAP1_UNINDUCED_REP1 = %+v, want single-end sample", samples[2])
	}
	if got, want := len(samples[3].Runs), 2; got != want {
		t.Fatalf("RAP1_UNINDUCED_REP2 run count = %d, want %d", got, want)
	}

	// Returned run slices are caller-owned.
	samples[0].Runs[0].Fastq1 = "changed.fastq.gz"
	again, err := rnaseq.Load(rnaFixtureSheet)
	if err != nil {
		t.Fatalf("second Load error = %v", err)
	}
	if again[0].Runs[0].Fastq1 == "changed.fastq.gz" {
		t.Fatal("Load retained a caller-owned run slice")
	}
	metadata, err := rnaseq.Parse(strings.NewReader("sample,fastq_1,fastq_2,strandedness,seq_platform,seq_center\na,in/a.fq.gz,,forward,ILLUMINA,center-a\n"))
	if err != nil || metadata[0].SeqPlatform != "ILLUMINA" || metadata[0].SeqCenter != "center-a" {
		t.Fatalf("optional sequencing metadata = %+v, %v", metadata, err)
	}
}

func TestLiveSheetPreservesOfficialRowSemantics(t *testing.T) {
	fixture, err := os.ReadFile(rnaFixtureSheet)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", rnaFixtureSheet, err)
	}
	live, err := os.ReadFile(rnaLiveFixtureSheet)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", rnaLiveFixtureSheet, err)
	}
	if !bytes.Equal(fixture, live) {
		t.Fatalf("live sheet changes official staged row semantics\nfixture:\n%s\nlive:\n%s", fixture, live)
	}
}

func TestParseRejectsInvalidRNAData(t *testing.T) {
	tests := []struct {
		name string
		csv  string
		want string
	}{
		{name: "unknown column", csv: "sample,fastq_1,fastq_2,strandedness,group\na,in/a.fq.gz,,reverse,x\n", want: "unknown RNA samplesheet column"},
		{name: "non-exact header", csv: "sample,fastq_1,fastq_2,strandedness \na,in/a.fq.gz,,reverse\n", want: "unknown RNA samplesheet column"},
		{name: "missing mate header", csv: "sample,fastq_1,strandedness\na,in/a.fq.gz,reverse\n", want: "missing required RNA samplesheet column"},
		{name: "URL", csv: "sample,fastq_1,fastq_2,strandedness\na,https://example.invalid/a.fq.gz,,reverse\n", want: "workspace-relative"},
		{name: "bad strand", csv: "sample,fastq_1,fastq_2,strandedness\na,in/a.fq.gz,,unknown\n", want: "strandedness must be"},
		{name: "mixed read mode", csv: "sample,fastq_1,fastq_2,strandedness\na,in/a.fq.gz,in/a2.fq.gz,reverse\na,in/b.fq.gz,,reverse\n", want: "mixes single-end and paired-end"},
		{name: "metadata conflict", csv: "sample,fastq_1,fastq_2,strandedness,seq_center\na,in/a.fq.gz,,reverse,x\na,in/b.fq.gz,,reverse,y\n", want: "metadata disagrees"},
		{name: "duplicate run", csv: "sample,fastq_1,fastq_2,strandedness\na,in/a.fq.gz,,reverse\na,in/a.fq.gz,,reverse\n", want: "duplicate sequencing run"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := rnaseq.Parse(strings.NewReader(test.csv))
			if err == nil || !strings.Contains(errDetails(err), test.want) {
				t.Fatalf("Parse error = %v (%s), want detail %q", err, errDetails(err), test.want)
			}
			if !gobble.IsSampleSheetError(err) {
				t.Fatalf("IsSampleSheetError(%v) = false, want true", err)
			}
		})
	}
}

func TestSTARSalmonPlanDeclaresSelectedProduct(t *testing.T) {
	samples := loadSamples(t)
	pipeline := rnaseq.Build(samples, rnaseq.DefaultConfig())
	if _, err := gobble.Compose(pipeline); err != nil {
		var structured *gobble.Error
		errors.As(err, &structured)
		t.Fatalf("Compose defects = %+v", structured.Defects)
	}
	raw := pc.MustPlanJSON(t, pipeline)
	tasks := pc.AllTasks(t, raw)

	for _, id := range []string{
		"reference.gtf_filter",
		"reference.transcriptome.gffread_transcriptome",
		"reference.gunzip",
		"reference.gene_intervals.gffread_bed",
		"reference.samtools_faidx",
		"reference.cut_chrom_sizes",
		"reference.star_genome_generate",
		"reference.salmon_index",
		"WT_REP1.consolidate_r1.cat_fastq",
		"WT_REP1.trim_galore",
		"WT_REP1.strandedness.salmon_strandedness",
		"WT_REP1.star_align",
		"WT_REP1.salmon_quant",
		"WT_REP1.picard_markduplicates",
		"WT_REP1.stringtie",
		"WT_REP1.coverage_combined.ucsc_bedgraphtobigwig",
		"WT_REP1.rseqc_inferexperiment",
		"WT_REP1.qualimap_bamqc",
		"WT_REP1.dupradar",
		"WT_REP1.featurecounts_biotype_qc",
		"cohort.tximport",
		"cohort_qc.deseq2_qc",
		"multiqc",
	} {
		pc.MustHaveTaskID(t, tasks, id)
	}

	if got, want := pc.CountTasksNamed(tasks, "star_align"), len(samples); got != want {
		t.Fatalf("star_align task count = %d, want %d", got, want)
	}
	if got, want := pc.CountTasksNamed(tasks, "salmon_quant"), len(samples); got != want {
		t.Fatalf("salmon_quant task count = %d, want %d", got, want)
	}
	if got := pc.CountTasksNamed(tasks, "salmon_strandedness"); got != 1 {
		t.Fatalf("salmon_strandedness task count = %d, want 1 auto sample", got)
	}
	pc.AssertNoTaskName(t, tasks, "merge_counts", "featurecounts", "deseq2", "fastp", "bwa_mem", "hisat2_align", "rsem")

	for _, task := range tasks {
		if task.Image != "" && !strings.Contains(task.Image, "@sha256:") {
			t.Errorf("task %s image = %q, want immutable tag and digest", task.ID, task.Image)
		}
		if pc.ContainsAll(task.Command, "--contrast") || pc.ContainsAll(task.Command, "--design") {
			t.Errorf("task %s command exposes study design/contrast: %#v", task.ID, task.Command)
		}
	}
	for id, wants := range map[string][]string{
		"WT_REP1.salmon_quant":             {"'--libType' 'IU'", "'--libType' 'ISF'", "'--libType' 'ISR'"},
		"WT_REP1.stringtie":                {"'--fr'", "'--rf'"},
		"WT_REP1.qualimap_bamqc":           {"'non-strand-specific'", "'strand-specific-forward'", "'strand-specific-reverse'"},
		"WT_REP1.dupradar":                 {"'0'", "'1'", "'2'"},
		"WT_REP1.featurecounts_biotype_qc": {"'-s' '0'", "'-s' '1'", "'-s' '2'"},
	} {
		script := pc.TaskByID(t, raw, id).Script
		for _, want := range wants {
			if !strings.Contains(script, want) {
				t.Fatalf("task %s script omits %q: %s", id, want, script)
			}
		}
	}

	star := pc.TaskByID(t, raw, "WT_REP1.star_align")
	if !pc.ContainsAll(star.Command, "--quantMode", "TranscriptomeSAM", "GeneCounts", "--outSAMtype", "BAM", "Unsorted") {
		t.Fatalf("STAR command = %#v, want genome/transcriptome selected outputs", star.Command)
	}
	if !pc.ContainsAll(pc.TaskByID(t, raw, "reference.star_genome_generate").Command, "--genomeSAindexNbases", "7") {
		t.Fatal("STAR genomeGenerate omits the official small-reference index setting")
	}
	pc.AssertIOPath(t, star.Outputs, "transcript_bam", "work/WT_REP1/star/Aligned.toTranscriptome.out.bam")
	for _, id := range []string{
		"reference.transcriptome.gffread_transcriptome",
		"reference.gene_intervals.gffread_bed",
		"reference.star_genome_generate",
		"WT_REP1.star_align",
		"WT_REP1.salmon_quant",
		"WT_REP1.stringtie",
		"WT_REP1.qualimap_bamqc",
		"WT_REP1.dupradar",
		"WT_REP1.featurecounts_biotype_qc",
		"cohort.tximport",
	} {
		pc.AssertIOPath(t, pc.TaskByID(t, raw, id).Inputs, "gtf", "work/reference/genes.filtered.gtf")
	}
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "WT_REP1.picard_markduplicates").Outputs, "marked_bam", "results/rnaseq/bam/WT_REP1/WT_REP1.marked.bam")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "cohort.tximport").Outputs, "gene_counts", "results/rnaseq/matrices/gene_counts.tsv")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "cohort.tximport").Outputs, "transcript_tpm", "results/rnaseq/matrices/transcript_tpm.tsv")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "cohort.tximport").Outputs, "gene_scaled", "results/rnaseq/matrices/gene_counts_scaled.tsv")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "cohort_qc.deseq2_qc").Outputs, "pca", "results/rnaseq/deseq2-qc/pca.pdf")
	pc.AssertTreeIO(t, pc.TaskByID(t, raw, "multiqc").Outputs, "data", "results/rnaseq/multiqc/multiqc_data")

	single := pc.TaskByID(t, raw, "RAP1_UNINDUCED_REP1.trim_galore")
	if pc.ContainsAll(single.Command, "--paired") {
		t.Fatalf("single-end Trim Galore command = %#v, must not contain --paired", single.Command)
	}
	paired := pc.TaskByID(t, raw, "WT_REP1.trim_galore")
	if !pc.ContainsAll(paired.Command, "--paired") {
		t.Fatalf("paired Trim Galore command = %#v, want --paired", paired.Command)
	}
	catTask := pc.TaskByID(t, raw, "WT_REP1.consolidate_r1.cat_fastq")
	if !pc.ContainsAll(catTask.Command, "sh", "-c") || !strings.Contains(catTask.Script, "'cat'") || strings.Contains(catTask.Script, "--output") {
		t.Fatalf("cat FASTQ task = command %#v script %q, want one quoted stdout command", catTask.Command, catTask.Script)
	}
}

func TestAutoStrandednessPrecedesAndControlsEveryDependentStage(t *testing.T) {
	raw := pc.MustPlanJSON(t, rnaseq.Build(loadSamples(t), rnaseq.DefaultConfig()))
	inferredPath := "work/WT_REP1/strandedness/WT_REP1/strandedness.txt"
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "WT_REP1.strandedness.salmon_strandedness").Outputs, "strandedness", inferredPath)
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "WT_REP1.star_align").Inputs, "prerequisite_2", inferredPath)

	for _, id := range []string{
		"WT_REP1.salmon_quant",
		"WT_REP1.stringtie",
		"WT_REP1.qualimap_bamqc",
		"WT_REP1.dupradar",
		"WT_REP1.featurecounts_biotype_qc",
	} {
		task := pc.TaskByID(t, raw, id)
		pc.AssertIOPath(t, task.Inputs, "strandedness", inferredPath)
		for _, want := range []string{"unstranded)", "forward)", "reverse)"} {
			if !strings.Contains(task.Script, want) {
				t.Fatalf("task %s script omits inferred case %q: %s", id, want, task.Script)
			}
		}
	}
	for _, id := range []string{
		"WT_REP1.coverage_forward.bedtools_genomecov_inferred",
		"WT_REP1.coverage_reverse.bedtools_genomecov_inferred",
	} {
		task := pc.TaskByID(t, raw, id)
		pc.AssertIOPath(t, task.Inputs, "strandedness", inferredPath)
		if !strings.Contains(task.Script, "unstranded) ;;") || !strings.Contains(task.Script, "-strand") {
			t.Fatalf("coverage task %s does not condition directional output on inference: %s", id, task.Script)
		}
	}
}

func TestExplicitReverseCoverageMatchesInferredReverseDirection(t *testing.T) {
	raw := pc.MustPlanJSON(t, rnaseq.Build(loadSamples(t), rnaseq.DefaultConfig()))
	forward := pc.TaskByID(t, raw, "WT_REP2.coverage_forward.bedtools_genomecov")
	reverse := pc.TaskByID(t, raw, "WT_REP2.coverage_reverse.bedtools_genomecov")
	if !strings.Contains(forward.Script, "'-strand' '-'") || !strings.Contains(reverse.Script, "'-strand' '+'") {
		t.Fatalf("explicit reverse coverage strands = forward %#v reverse %#v, want - and +", forward.Command, reverse.Command)
	}

	autoForward := pc.TaskByID(t, raw, "WT_REP1.coverage_forward.bedtools_genomecov_inferred").Script
	autoReverse := pc.TaskByID(t, raw, "WT_REP1.coverage_reverse.bedtools_genomecov_inferred").Script
	if !strings.Contains(autoForward, "reverse) 'bedtools' 'genomecov' '-bg' '-split' '-ibam' 'results/rnaseq/bam/WT_REP1/WT_REP1.marked.bam' '-strand' '-'") ||
		!strings.Contains(autoReverse, "reverse) 'bedtools' 'genomecov' '-bg' '-split' '-ibam' 'results/rnaseq/bam/WT_REP1/WT_REP1.marked.bam' '-strand' '+'") {
		t.Fatalf("inferred reverse coverage does not match explicit reverse: forward=%q reverse=%q", autoForward, autoReverse)
	}
}

func TestPipelineAdapterMatchesBuild(t *testing.T) {
	previous := gobble.SampleSheetPath()
	gobble.SetSampleSheetPath(rnaFixtureSheet)
	t.Cleanup(func() { gobble.SetSampleSheetPath(previous) })
	got := pc.MustPlanJSON(t, rnaseq.Pipeline())
	want := pc.MustPlanJSON(t, rnaseq.Build(loadSamples(t), rnaseq.DefaultConfig()))
	if !bytes.Equal(got, want) {
		t.Fatal("Pipeline plan differs from Load plus DefaultConfig plus Build")
	}
}

func TestBuildRejectsInvalidConfigAndOptionCollisions(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*rnaseq.Config)
		want   gobble.DefectCode
	}{
		{name: "reference escape", mutate: func(config *rnaseq.Config) { config.Reference.FASTA = gobble.Literal("../genome.fa") }, want: gobble.DefectInvalidPath},
		{name: "STAR owned flag", mutate: func(config *rnaseq.Config) { config.STAR.ExtraArgs = []string{"--outSAMtype", "SAM"} }, want: gobble.DefectInvalidValue},
		{name: "Salmon route flag", mutate: func(config *rnaseq.Config) { config.Salmon.ExtraArgs = []string{"-a", "other.bam"} }, want: gobble.DefectInvalidValue},
		{name: "DESeq2 contrast", mutate: func(config *rnaseq.Config) { config.DESeq2QC.ExtraArgs = []string{"--contrast", "a,b"} }, want: gobble.DefectInvalidValue},
		{name: "DESeq2 unsupported extra", mutate: func(config *rnaseq.Config) { config.DESeq2QC.ExtraArgs = []string{"--alpha", "0.05"} }, want: gobble.DefectInvalidValue},
		{name: "dupRadar unsupported extra", mutate: func(config *rnaseq.Config) { config.DupRadar.ExtraArgs = []string{"--arbitrary"} }, want: gobble.DefectInvalidValue},
		{name: "tximport extra operands", mutate: func(config *rnaseq.Config) { config.TxImport.ExtraArgs = []string{"fake-sample", "fake-quant.sf"} }, want: gobble.DefectInvalidValue},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := rnaseq.DefaultConfig()
			test.mutate(&config)
			graph, err := gobble.Compose(rnaseq.Build(loadSamples(t), config))
			if graph != nil || !hasDefect(err, test.want) {
				t.Fatalf("Compose() = (%v, %v), want nil graph and %s", graph, err, test.want)
			}
		})
	}

	graph, err := gobble.Compose(rnaseq.Build(nil, rnaseq.DefaultConfig()))
	if graph != nil || !hasDefect(err, gobble.DefectInvalidSampleSheet) {
		t.Fatalf("empty Build Compose() = (%v, %v), want invalid-samplesheet", graph, err)
	}
}

func TestConfigCustomizationIsVisibleAndDefaultsAreFresh(t *testing.T) {
	samples := loadSamples(t)
	first := rnaseq.DefaultConfig()
	first.Salmon.ExtraArgs = append(first.Salmon.ExtraArgs, "--validateMappings")
	custom := pc.MustPlanJSON(t, rnaseq.Build(samples, first))
	if !strings.Contains(pc.TaskByID(t, custom, "WT_REP1.salmon_quant").Script, "'--validateMappings'") {
		t.Fatal("custom Salmon option is absent from plan")
	}

	second := rnaseq.DefaultConfig()
	plain := pc.MustPlanJSON(t, rnaseq.Build(samples, second))
	if strings.Contains(pc.TaskByID(t, plain, "WT_REP1.salmon_quant").Script, "'--validateMappings'") {
		t.Fatal("DefaultConfig retained a prior caller mutation")
	}
	if !slices.Equal(pc.TaskByID(t, custom, "WT_REP1.star_align").Command, pc.TaskByID(t, plain, "WT_REP1.star_align").Command) {
		t.Fatal("Salmon-only customization changed STAR command identity")
	}
}

func TestTypedPoliciesAreValidatedAndPlanVisible(t *testing.T) {
	config := rnaseq.DefaultConfig()
	if config.SampleRemoval.MinTrimmedReads != 10000 || config.SampleRemoval.MinMappedPercent != 5 || config.StrandednessInference.StrandedFraction != 0.8 || config.StrandednessInference.UnstrandedDifference != 0.1 {
		t.Fatalf("default typed policies = %+v %+v, want nf-core 3.26.0 boundaries", config.SampleRemoval, config.StrandednessInference)
	}
	raw := pc.MustPlanJSON(t, rnaseq.Build(loadSamples(t), config))
	trimGate := pc.TaskByID(t, raw, "WT_REP1.sample_retention_trimmed")
	mappedGate := pc.TaskByID(t, raw, "WT_REP1.sample_retention_mapped")
	inference := pc.TaskByID(t, raw, "WT_REP1.strandedness.salmon_strandedness")
	if !slices.Contains(trimGate.Command, "10000") || !slices.Contains(mappedGate.Command, "5") || !strings.Contains(inference.Script, "limit=0.8") || !strings.Contains(inference.Script, "limit=0.1") {
		t.Fatalf("policy tasks omit typed defaults: trim=%#v mapped=%#v inference=%q", trimGate.Command, mappedGate.Command, inference.Script)
	}

	invalid := rnaseq.DefaultConfig()
	invalid.StrandednessInference.StrandedFraction = 0.4
	if graph, err := gobble.Compose(rnaseq.Build(loadSamples(t), invalid)); graph != nil || !hasDefect(err, gobble.DefectInvalidValue) {
		t.Fatalf("invalid inference policy Compose() = (%v, %v), want invalid-value", graph, err)
	}
	invalid = rnaseq.DefaultConfig()
	invalid.Publication.Matrices = false
	if graph, err := gobble.Compose(rnaseq.Build(loadSamples(t), invalid)); graph != nil || !hasDefect(err, gobble.DefectInvalidValue) {
		t.Fatalf("disabled required publication Compose() = (%v, %v), want invalid-value", graph, err)
	}

	published := rnaseq.DefaultConfig()
	published.Publication.TrimmedReads = true
	published.Publication.STARAlignments = true
	published.Publication.GeneratedReference = true
	publishedPlan := pc.MustPlanJSON(t, rnaseq.Build(loadSamples(t), published))
	pc.AssertIOPath(t, pc.TaskByID(t, publishedPlan, "WT_REP1.trim_galore").Outputs, "trimmed_read1", "results/rnaseq/intermediates/trimmed/WT_REP1/WT_REP1_val_1.fq.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, publishedPlan, "WT_REP1.star_align").Outputs, "genome_bam", "results/rnaseq/intermediates/star/WT_REP1/Aligned.out.bam")
	pc.AssertTreeIO(t, pc.TaskByID(t, publishedPlan, "reference.star_genome_generate").Outputs, "index", "results/rnaseq/reference/star-index")
}

func TestUncompressedGTFPolicyRemovesOnlyGunzipStage(t *testing.T) {
	config := rnaseq.DefaultConfig()
	config.Reference.GTF = gobble.PathSpec{Dir: gobble.Dir("in/reference"), Base: "genes", Ext: ".gtf"}
	config.Reference.GTFCompressed = false
	tasks := pc.AllTasks(t, pc.MustPlanJSON(t, rnaseq.Build(loadSamples(t), config)))
	if pc.CountTasksNamed(tasks, "gunzip") != 0 || pc.CountTasksNamed(tasks, "star_align") == 0 || pc.CountTasksNamed(tasks, "salmon_quant") == 0 {
		t.Fatalf("uncompressed GTF tasks omit required route or retain gunzip: %+v", tasks)
	}
}

func loadSamples(t *testing.T) []rnaseq.Sample {
	t.Helper()
	samples, err := rnaseq.Load(rnaFixtureSheet)
	if err != nil {
		t.Fatalf("Load(%s) error = %v", rnaFixtureSheet, err)
	}
	return samples
}

func errDetails(err error) string {
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

func TestLoadMissingSheetIsStructured(t *testing.T) {
	_, err := rnaseq.Load("testdata/does-not-exist.csv")
	if !hasDefect(err, gobble.DefectNotFound) || !gobble.IsSampleSheetError(err) {
		t.Fatalf("Load missing error = %v, want structured samplesheet not-found", err)
	}
}

func TestBuildHasNoAmbientOrNetworkInput(t *testing.T) {
	sourcecheck.AssertNoCall(t, "../../../assets/pipelines/rnaseq/build.go", "SampleSheetPath", "Load", "Getenv", "Open", "ReadFile", "Get")
	sourcecheck.AssertNoCall(t, "../../../assets/pipelines/rnaseq/build.go", "AddTask")
}

func TestFixtureIsCommittedText(t *testing.T) {
	data, err := os.ReadFile(rnaFixtureSheet)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", rnaFixtureSheet, err)
	}
	if bytes.Contains(data, []byte("https://")) {
		t.Fatal("staged product samplesheet contains a remote URL")
	}
}
