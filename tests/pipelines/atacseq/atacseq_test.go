package atacseqevidence_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/atacseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/internal/sourcecheck"
)

const fixtureSheet = "testdata/atacseq-samplesheet.csv"

func TestParseTypedRunsReplicatesAndModes(t *testing.T) {
	samples := loadSamples(t)
	if len(samples) != 4 || len(samples[0].Replicates) != 2 || len(samples[1].Replicates) != 1 || len(samples[1].Replicates[0].Runs) != 2 || len(samples[2].Replicates) != 2 || len(samples[3].Replicates[0].Runs) != 2 {
		t.Fatalf("Load() = %#v, want official PE/SE replicate and technical-run structure", samples)
	}
	if samples[1].Replicates[0].Runs[0].ID != "run_001" || samples[1].Replicates[0].Runs[1].ID != "run_002" || samples[2].Replicates[0].Runs[0].Fastq2 != "" {
		t.Fatalf("typed technical runs = %#v, want stable IDs and single-end identity", samples)
	}
}

func TestParserResolvesControlsAndRejectsStructuralDefects(t *testing.T) {
	controlled := "sample,fastq_1,fastq_2,replicate,control,control_replicate\n" +
		"INPUT,in/i.fastq.gz,,1,,\n" +
		"TREATMENT,in/t.fastq.gz,,1,INPUT,1\n"
	samples, err := atacseq.Parse(strings.NewReader(controlled))
	if err != nil || samples[1].Replicates[0].Control == nil || samples[1].Replicates[0].Control.Sample != "INPUT" {
		t.Fatalf("Parse(controlled) = (%#v, %v), want explicit resolved link", samples, err)
	}
	tests := []struct {
		name  string
		sheet string
		unit  string
	}{
		{name: "replicate gap", sheet: "sample,fastq_1,fastq_2,replicate\nA,in/a.fastq.gz,,1\nA,in/b.fastq.gz,,3\n", unit: "A"},
		{name: "missing control", sheet: "sample,fastq_1,fastq_2,replicate,control,control_replicate\nA,in/a.fastq.gz,,1,MISSING,1\nB,in/b.fastq.gz,,1,,\n", unit: "A"},
		{name: "conflicting run mode", sheet: "sample,fastq_1,fastq_2,replicate\nA,in/a.fastq.gz,,1\nA,in/b.fastq.gz,in/c.fastq.gz,1\nB,in/d.fastq.gz,,1\n", unit: "A"},
		{name: "unknown column", sheet: "sample,fastq_1,fastq_2,replicate,contrast\nA,in/a.fastq.gz,,1,x\nB,in/b.fastq.gz,,1,x\n", unit: "samplesheet"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := atacseq.Parse(strings.NewReader(test.sheet))
			requireDefect(t, err, gobble.DefectInvalidSampleSheet, test.unit)
		})
	}
}

func TestBuildSelectedVerticalUsesComposeTimeStrictFanIn(t *testing.T) {
	pipeline := atacseq.Build(loadSamples(t), atacseq.DefaultConfig())
	if _, err := gobble.Compose(pipeline); err != nil {
		var composeErr *gobble.Error
		if errors.As(err, &composeErr) {
			t.Fatalf("Compose defects = %#v", composeErr.Defects)
		}
		t.Fatal(err)
	}
	raw := pc.MustPlanJSON(t, pipeline)
	tasks := pc.AllTasks(t, raw)
	for name, want := range map[string]int{
		"samtools_faidx": 1, "atac_reference_intervals": 1, "bwa_index": 1,
		"bwa_mem": 8, "picard_merge_sam_files": 8, "macs2_callpeak": 8,
		"ataqv": 8, "atac_consensus_peaks": 2, "featurecounts_atac": 4,
		"featurecounts_merge_matrices": 2, "deseq2_qc": 2, "ataqv_mkarv": 1, "igv_session": 1, "multiqc": 1,
	} {
		if got := pc.CountTasksNamed(tasks, name); got != want {
			t.Errorf("%s count = %d, want %d", name, got, want)
		}
	}
	for _, id := range []string{
		"reference.samtools_faidx", "reference.bwa_index",
		"OSMOTIC_STRESS_T15_PE.replicate_1.run_002.bwa_mem",
		"OSMOTIC_STRESS_T0_PE.replicate_1.technical_run_merge.picard_merge_sam_files",
		"OSMOTIC_STRESS_T0_PE.aggregate.replicate_merge.picard_merge_sam_files",
		"OSMOTIC_STRESS_T0_PE.replicate_1.peaks.macs2_callpeak",
		"coverage_qc.deeptools_plot_fingerprint",
		"consensus.replicates.atac_consensus_peaks",
		"consensus.aggregates.featurecounts_merge_matrices",
		"ataqv.ataqv_mkarv", "igv.igv_session", "multiqc",
	} {
		pc.MustHaveTaskID(t, tasks, id)
	}
	replicateConsensus := pc.TaskByID(t, raw, "consensus.replicates.atac_consensus_peaks")
	if got := countInputs(replicateConsensus.Inputs, "peaks_"); got != 6 {
		t.Fatalf("replicate consensus peak fan-in = %d, want all 6 members", got)
	}
	pairedCounts := pc.TaskByID(t, raw, "consensus.replicates.paired_end.featurecounts_atac")
	singleCounts := pc.TaskByID(t, raw, "consensus.replicates.single_end.featurecounts_atac")
	if got := countInputs(pairedCounts.Inputs, "bam_") + countInputs(singleCounts.Inputs, "bam_"); got != 6 {
		t.Fatalf("replicate matrix BAM fan-in = %d, want all 6 members", got)
	}
	if got := countInputs(pc.TaskByID(t, raw, "consensus.replicates.featurecounts_merge_matrices").Inputs, "matrix_"); got != 2 {
		t.Fatalf("mixed-mode matrix fan-in = %d, want both mode matrices", got)
	}
	multiQC := pc.TaskByID(t, raw, "multiqc")
	if got := countTreeInputPathsContaining(t, multiQC.Inputs, "/qc/alignment/picard"); got != 8 {
		t.Fatalf("MultiQC Picard CollectMultipleMetrics fan-in = %d, want all 8 replicate and aggregate Trees", got)
	}
	assertNoRuntimeScatter(t, raw)
	pc.AssertNoTaskName(t, tasks, "bowtie2", "chromap", "star", "idr", "motif", "footprinting")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "OSMOTIC_STRESS_T0_PE.replicate_1.samtools_view").Outputs, "filtered_bam", "results/atacseq/samples/OSMOTIC_STRESS_T0_PE/replicate_1/alignment/OSMOTIC_STRESS_T0_PE_R1.filtered.bam")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "consensus.replicates.atac_consensus_peaks").Outputs, "bed", "results/atacseq/consensus/replicates/consensus.bed")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "igv.igv_session").Outputs, "xml", "results/atacseq/igv/igv_session.xml")
}

func TestControlBindingAndNarrowModeAreTyped(t *testing.T) {
	samples := []atacseq.Sample{
		{Name: "INPUT", Replicates: []atacseq.Replicate{{Number: 1, Runs: []atacseq.Run{{ID: "run_001", Fastq1: "in/i.fastq.gz"}}}}},
		{Name: "TREATMENT", Replicates: []atacseq.Replicate{{Number: 1, Runs: []atacseq.Run{{ID: "run_001", Fastq1: "in/t.fastq.gz"}}, Control: &atacseq.ControlRef{Sample: "INPUT", Replicate: 1}}}},
	}
	config := atacseq.DefaultConfig()
	config.PeakMode = atacseq.PeakNarrow
	raw := pc.MustPlanJSON(t, atacseq.Build(samples, config))
	peak := pc.TaskByID(t, raw, "TREATMENT.replicate_1.peaks.macs2_callpeak")
	pc.AssertIOPath(t, peak.Inputs, "control", "results/atacseq/samples/INPUT/replicate_1/alignment/INPUT_R1.filtered.bam")
	pc.AssertIOPath(t, peak.Outputs, "peaks", "results/atacseq/samples/TREATMENT/replicate_1/peaks/TREATMENT_R1_peaks.narrowPeak")
	if !pc.ContainsAll(peak.Command, "--format", "BAM", "--control") || slicesContains(peak.Command, "--broad") {
		t.Fatalf("typed narrow/control command = %#v", peak.Command)
	}
	if got := paramValue(peak.Params, "control"); got != "INPUT.R1" {
		t.Fatalf("control param = %q, want INPUT.R1", got)
	}
	if got := paramValue(pc.TaskByID(t, raw, "TREATMENT.replicate_1.run_001.bwa_mem").Params, "control"); got != "" {
		t.Fatalf("control link leaked into reusable alignment identity: %q", got)
	}
}

func TestChangingControlLinkPreservesAlignmentAndChangesPeakIdentity(t *testing.T) {
	control := func(name, read string) atacseq.Sample {
		return atacseq.Sample{Name: name, Replicates: []atacseq.Replicate{{Number: 1, Runs: []atacseq.Run{{ID: "run_001", Fastq1: read}}}}}
	}
	samples := []atacseq.Sample{
		control("INPUT_A", "in/a.fastq.gz"),
		control("INPUT_B", "in/b.fastq.gz"),
		{Name: "TREATMENT", Replicates: []atacseq.Replicate{{Number: 1, Runs: []atacseq.Run{{ID: "run_001", Fastq1: "in/t.fastq.gz"}}, Control: &atacseq.ControlRef{Sample: "INPUT_A", Replicate: 1}}}},
	}
	plain := pc.MustPlanJSON(t, atacseq.Build(samples, atacseq.DefaultConfig()))
	samples[2].Replicates[0].Control = &atacseq.ControlRef{Sample: "INPUT_B", Replicate: 1}
	changed := pc.MustPlanJSON(t, atacseq.Build(samples, atacseq.DefaultConfig()))
	plainAlignment := pc.TaskByID(t, plain, "TREATMENT.replicate_1.run_001.bwa_mem")
	changedAlignment := pc.TaskByID(t, changed, "TREATMENT.replicate_1.run_001.bwa_mem")
	if !bytes.Equal(mustJSON(t, plainAlignment), mustJSON(t, changedAlignment)) {
		t.Fatal("control-only change altered reusable alignment identity")
	}
	plainPeak := pc.TaskByID(t, plain, "TREATMENT.replicate_1.peaks.macs2_callpeak")
	changedPeak := pc.TaskByID(t, changed, "TREATMENT.replicate_1.peaks.macs2_callpeak")
	if bytes.Equal(mustJSON(t, plainPeak), mustJSON(t, changedPeak)) || paramValue(changedPeak.Params, "control") != "INPUT_B.R1" {
		t.Fatal("control-only change did not alter the associated peak identity")
	}
}

func TestProtectedAliasesAndInvalidPublicationFailCompose(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*atacseq.Config)
		unit   string
	}{
		{name: "MACS short control", unit: "macs2_callpeak", mutate: func(c *atacseq.Config) { c.MACS2.ExtraArgs = []string{"-cother.bam"} }},
		{name: "MACS output", unit: "macs2_callpeak", mutate: func(c *atacseq.Config) { c.MACS2.ExtraArgs = []string{"--outdir=elsewhere"} }},
		{name: "Picard output alias", unit: "picard_markduplicates", mutate: func(c *atacseq.Config) { c.MarkDuplicates.ExtraArgs = []string{"O=elsewhere.bam"} }},
		{name: "aligner route", unit: "bwa_mem", mutate: func(c *atacseq.Config) { c.BWAMem.ExtraArgs = []string{"--aligner=bowtie2"} }},
		{name: "publication", unit: "publication", mutate: func(c *atacseq.Config) { c.Publication.ConsensusMatrix = false }},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := atacseq.DefaultConfig()
			test.mutate(&config)
			_, err := gobble.Compose(atacseq.Build(loadSamples(t), config))
			requireDefect(t, err, gobble.DefectInvalidValue, test.unit)
		})
	}
}

func TestReadyBWAIndexAndBlacklistFilteringAreTyped(t *testing.T) {
	config := atacseq.DefaultConfig()
	prefix := gobble.PathSpec{Dir: gobble.Dir("in/reference/bwa"), Base: "genome"}
	config.Reference.BWAIndex.Prefix = prefix
	for _, name := range []string{"amb", "ann", "bwt", "pac", "sa"} {
		config.Reference.BWAIndex.Members = append(config.Reference.BWAIndex.Members, gobble.Member{Name: name, Spec: prefix.AppendExt("." + name)})
	}
	config.Reference.Blacklist = gobble.PathSpec{Dir: gobble.Dir("in/reference"), Base: "blacklist", Ext: ".bed"}
	config.Filters.RemoveBlacklist = true
	raw := pc.MustPlanJSON(t, atacseq.Build(loadSamples(t), config))
	tasks := pc.AllTasks(t, raw)
	if got := pc.CountTasksNamed(tasks, "bwa_index"); got != 0 {
		t.Fatalf("ready BWA index preparation count = %d, want 0", got)
	}
	if got := pc.CountTasksNamed(tasks, "bedtools_intersect"); got != 16 {
		t.Fatalf("blacklist plus peak intersection count = %d, want 16", got)
	}
	filter := pc.TaskByID(t, raw, "OSMOTIC_STRESS_T0_PE.replicate_1.bedtools_intersect")
	pc.AssertIOPath(t, filter.Inputs, "intervals", "in/reference/blacklist.bed")
	pc.AssertIOPath(t, filter.Outputs, "selected_bam", "results/atacseq/samples/OSMOTIC_STRESS_T0_PE/replicate_1/alignment/OSMOTIC_STRESS_T0_PE_R1.filtered.bam")
}

func TestBuildCopiesCallerDataAndDefaultsAreFresh(t *testing.T) {
	samples := loadSamples(t)
	config := atacseq.DefaultConfig()
	pipeline := atacseq.Build(samples, config)
	want := pc.MustPlanJSON(t, pipeline)
	samples[0].Replicates[0].Runs[0].Fastq1 = "changed.fastq.gz"
	config.MACS2.ExtraArgs = append(config.MACS2.ExtraArgs, "--changed")
	config.Reference.BWAIndex.Members = append(config.Reference.BWAIndex.Members, gobble.Member{Name: "changed"})
	if got := pc.MustPlanJSON(t, pipeline); !bytes.Equal(got, want) {
		t.Fatal("Build retained caller-owned mutable data")
	}
	first := atacseq.DefaultConfig()
	first.MACS2.ExtraArgs = append(first.MACS2.ExtraArgs, "--changed")
	if second := atacseq.DefaultConfig(); len(second.MACS2.ExtraArgs) != 0 || second.PeakMode != atacseq.PeakBroad {
		t.Fatalf("DefaultConfig retained mutation or lost broad default: %#v", second)
	}
	sourcecheck.AssertNoCall(t, "../../../assets/pipelines/atacseq/build.go", "SampleSheetPath", "Load", "Open", "ReadFile", "Stat", "Lstat", "Getwd")
}

func TestAdapterAndLifecycleContract(t *testing.T) {
	previous := gobble.SampleSheetPath()
	gobble.SetSampleSheetPath(fixtureSheet)
	t.Cleanup(func() { gobble.SetSampleSheetPath(previous) })
	if got, want := pc.MustPlanJSON(t, atacseq.Pipeline()), pc.MustPlanJSON(t, atacseq.Build(loadSamples(t), atacseq.DefaultConfig())); !bytes.Equal(got, want) {
		t.Fatal("Pipeline adapter differs from typed Build")
	}
	lifecycle := atacseq.Lifecycle
	if lifecycle.GraphGeneration != atacseq.GraphGeneration || !lifecycle.Design || !lifecycle.Build || !lifecycle.Customize || !lifecycle.Run || !lifecycle.Resume || !lifecycle.Stop || !lifecycle.Failure || lifecycle.PreLiftResumable {
		t.Fatalf("Lifecycle = %#v, want complete first-generation participation", lifecycle)
	}
}

func loadSamples(t *testing.T) []atacseq.Sample {
	t.Helper()
	samples, err := atacseq.Load(fixtureSheet)
	if err != nil {
		t.Fatalf("Load(%s): %v", fixtureSheet, err)
	}
	return samples
}

func requireDefect(t *testing.T, err error, code gobble.DefectCode, unit string) {
	t.Helper()
	var composeErr *gobble.Error
	if !errors.As(err, &composeErr) {
		t.Fatalf("error = %T %v, want *gobble.Error", err, err)
	}
	for _, defect := range composeErr.Defects {
		if defect.Code == code && defect.Unit == unit {
			return
		}
	}
	t.Fatalf("defects = %#v, want code %s unit %s", composeErr.Defects, code, unit)
}

func countInputs(inputs []pc.IO, prefix string) int {
	count := 0
	for _, input := range inputs {
		if strings.HasPrefix(input.Name, prefix) {
			count++
		}
	}
	return count
}

func countTreeInputPathsContaining(t *testing.T, inputs []pc.IO, fragment string) int {
	t.Helper()
	count := 0
	for _, input := range inputs {
		if strings.Contains(input.Path, fragment) {
			if input.Kind != "tree" {
				t.Errorf("MultiQC input %s at %s has kind %q, want tree", input.Name, input.Path, input.Kind)
			}
			count++
		}
	}
	return count
}

func assertNoRuntimeScatter(t *testing.T, raw []byte) {
	t.Helper()
	var plan struct {
		Tasks []struct {
			ID      string `json:"id"`
			Scatter string `json:"scatter"`
			Gather  string `json:"gather"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &plan); err != nil {
		t.Fatal(err)
	}
	for _, task := range plan.Tasks {
		if task.Scatter != "" || task.Gather != "" {
			t.Errorf("known ATAC membership task %s has runtime scatter/gather %q/%q", task.ID, task.Scatter, task.Gather)
		}
	}
}

func slicesContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func paramValue(params []pc.Param, name string) string {
	for _, param := range params {
		if param.Name == name {
			return param.Value
		}
	}
	return ""
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
