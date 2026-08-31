package wgsevidence_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/internal/sourcecheck"
)

const fixtureSheet = "testdata/wgs-samplesheet.csv"

func TestWGSParseTypedCohort(t *testing.T) {
	samples := loadSamples(t)
	if len(samples) != 2 || samples[0].Patient != "patient1" || samples[0].Name != "testN" || samples[0].Sex != "XX" || len(samples[0].Lanes) != 2 || samples[1].Name != "testT" || len(samples[1].Lanes) != 1 {
		t.Fatalf("Load() = %#v, want two-sample cohort with repeated testN lanes", samples)
	}
	if samples[0].Lanes[0].ID != "L001" || samples[0].Lanes[1].ID != "L002" {
		t.Fatalf("testN lanes = %#v, want stable samplesheet order", samples[0].Lanes)
	}
}

func TestWGSParserRejectsIncompleteOrConflictingCohorts(t *testing.T) {
	tests := []struct {
		name  string
		sheet string
		unit  string
	}{
		{name: "one sample", sheet: "patient,sample,lane,fastq_1,fastq_2\np1,s1,L1,in/a.fastq.gz,in/b.fastq.gz\n", unit: "samplesheet"},
		{name: "missing mate", sheet: "patient,sample,lane,fastq_1,fastq_2\np1,s1,L1,in/a.fastq.gz,\np2,s2,L1,in/c.fastq.gz,in/d.fastq.gz\n", unit: "fastq_2"},
		{name: "duplicate lane", sheet: "patient,sample,lane,fastq_1,fastq_2\np1,s1,L1,in/a.fastq.gz,in/b.fastq.gz\np1,s1,L1,in/c.fastq.gz,in/d.fastq.gz\np2,s2,L1,in/e.fastq.gz,in/f.fastq.gz\n", unit: "s1"},
		{name: "conflicting sex", sheet: "patient,sample,lane,fastq_1,fastq_2,sex\np1,s1,L1,in/a.fastq.gz,in/b.fastq.gz,XX\np1,s1,L2,in/c.fastq.gz,in/d.fastq.gz,XY\np2,s2,L1,in/e.fastq.gz,in/f.fastq.gz,XX\n", unit: "s1"},
		{name: "unknown column", sheet: "patient,sample,lane,fastq_1,fastq_2,status\np1,s1,L1,in/a.fastq.gz,in/b.fastq.gz,normal\np2,s2,L1,in/c.fastq.gz,in/d.fastq.gz,normal\n", unit: "samplesheet"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := wgs.Parse(strings.NewReader(test.sheet))
			requireDefect(t, err, gobble.DefectInvalidSampleSheet, test.unit)
		})
	}
}

func TestWGSParserKeepsPatientDistinctSamplePairs(t *testing.T) {
	sheet := "patient,sample,lane,fastq_1,fastq_2\np1,same,L1,in/a.fastq.gz,in/b.fastq.gz\np2,same,L1,in/c.fastq.gz,in/d.fastq.gz\n"
	samples, err := wgs.Parse(strings.NewReader(sheet))
	if err != nil {
		t.Fatalf("Parse() error = %v, want distinct patient/sample pairs", err)
	}
	if len(samples) != 2 || samples[0].Patient != "p1" || samples[1].Patient != "p2" || samples[0].Name != "same" || samples[1].Name != "same" {
		t.Fatalf("Parse() = %#v, want both patient-distinct same-name samples", samples)
	}
	tasks := pc.AllTasks(t, pc.MustPlanJSON(t, wgs.Build(samples, wgs.DefaultConfig())))
	pc.MustHaveTaskID(t, tasks, "p1.same.L1.bwa_mem")
	pc.MustHaveTaskID(t, tasks, "p2.same.L1.bwa_mem")
}

func TestWGSComposeBuildPlanSelectedVertical(t *testing.T) {
	pipeline := wgs.Build(loadSamples(t), wgs.DefaultConfig())
	if _, err := gobble.Compose(pipeline); err != nil {
		var composeErr *gobble.Error
		if errors.As(err, &composeErr) {
			t.Fatalf("Compose defects = %#v", composeErr.Defects)
		}
		t.Fatal(err)
	}
	raw := pc.MustPlanJSON(t, pipeline)
	tasks := pc.AllTasks(t, raw)
	wantCounts := map[string]int{
		"bwa_index": 1, "fastqc": 6, "fastp": 3, "bwa_mem": 3,
		"samtools_sort": 3, "samtools_merge": 3, "samtools_index": 6,
		"gatk4_markduplicates": 2, "gatk4_baserecalibrator": 2,
		"gatk4_applybqsr": 2, "gatk4_gather_bqsr_reports": 2,
		"samtools_stats":    2,
		"samtools_flagstat": 2, "samtools_idxstats": 2, "mosdepth": 2,
		"gatk4_haplotypecaller": 2, "gatk4_mergevcfs": 3,
		"gatk4_genomicsdbimport": 1, "gatk4_genotypegvcfs": 1,
		"bcftools_sort": 1, "bcftools_stats": 1, "multiqc": 1,
	}
	for name, want := range wantCounts {
		if got := pc.CountTasksNamed(tasks, name); got != want {
			t.Errorf("%s count = %d, want %d", name, got, want)
		}
	}
	for _, id := range []string{
		"reference.bwa_index",
		"patient1.testN.L001.fastp", "patient1.testN.L002.bwa_mem", "patient2.testT.L001.samtools_sort",
		"bqsr_intervals.patient1.testN.gatk4_baserecalibrator",
		"patient1.testN.bqsr_gather.samtools_merge",
		"haplotype_intervals.patient2.testT.gatk4_haplotypecaller",
		"patient2.testT.gvcf_gather.gatk4_mergevcfs",
		"joint_intervals.database.gatk4_genomicsdbimport",
		"joint_intervals.genotype.gatk4_genotypegvcfs",
		"joint_intervals.genotype.bcftools_sort",
		"joint_gather.joint.gatk4_mergevcfs",
		"callset_qc.bcftools_stats", "multiqc",
	} {
		pc.MustHaveTaskID(t, tasks, id)
	}

	pc.AssertIOPath(t, pc.TaskByID(t, raw, "patient1.testN.L001.fastp").Inputs, "read1", "in/reads/test_1.fastq.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "patient2.testT.L001.fastp").Inputs, "read1", "in/reads/test2_1.fastq.gz")
	bqsrGather := pc.TaskByID(t, raw, "patient1.testN.bqsr_gather.samtools_merge")
	pc.AssertIOPath(t, bqsrGather.Outputs, "bam", "results/wgs/samples/patient1/testN/alignment/testN.recalibrated.bam")
	if !pc.ContainsAll(bqsrGather.Command, "samtools", "merge", "work/patient1/testN/bqsr/bams/interval_001.bam", "work/patient1/testN/bqsr/bams/interval_002.bam") {
		t.Fatalf("BQSR BAM gather does not use Sarek's complete samtools merge path: %#v", bqsrGather.Command)
	}
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "patient1.testN.gvcf_gather.gatk4_mergevcfs").Outputs, "vcf", "results/wgs/samples/patient1/testN/gvcf/testN.g.vcf.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "joint_gather.joint.gatk4_mergevcfs").Outputs, "vcf", "results/wgs/joint/joint_germline.vcf.gz")
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "joint_gather.joint.gatk4_mergevcfs").Outputs, "tbi", "results/wgs/joint/joint_germline.vcf.gz.tbi")
	pc.AssertTreeIO(t, pc.TaskByID(t, raw, "joint_intervals.database.gatk4_genomicsdbimport").Outputs, "database", "work/joint/genomicsdb")

	genomicsDB := pc.TaskByID(t, raw, "joint_intervals.database.gatk4_genomicsdbimport")
	pc.AssertIOPath(t, genomicsDB.Inputs, "gvcf_0", "results/wgs/samples/patient1/testN/gvcf/testN.g.vcf.gz")
	pc.AssertIOPath(t, genomicsDB.Inputs, "gvcf_1", "results/wgs/samples/patient2/testT/gvcf/testT.g.vcf.gz")
	if !pc.ContainsAll(pc.TaskByID(t, raw, "joint_gather.joint.gatk4_mergevcfs").Command, "MergeVcfs", "--INPUT", "work/joint/sorted/interval_001.sorted.vcf.gz", "work/joint/sorted/interval_002.sorted.vcf.gz") {
		t.Fatalf("joint gather command does not require every interval: %#v", pc.TaskByID(t, raw, "joint_gather.joint.gatk4_mergevcfs").Command)
	}
	pc.AssertNoTaskName(t, tasks, "VariantRecalibrator", "ApplyVQSR", "Mutect2", "Strelka", "DeepVariant")
	assertScatterGatherFacts(t, raw)
}

func TestWGSPatientIdentityChangesGraphAndReuseIdentity(t *testing.T) {
	samples := loadSamples(t)
	plainPipeline := wgs.Build(samples, wgs.DefaultConfig())
	plain := pc.MustPlanJSON(t, plainPipeline)
	samples[0].Patient = "changed_patient"
	changedPipeline := wgs.Build(samples, wgs.DefaultConfig())
	changed := pc.MustPlanJSON(t, changedPipeline)
	if bytes.Equal(plain, changed) {
		t.Fatal("changing patient identity left the WGS plan unchanged")
	}
	plainGraph, err := gobble.Compose(plainPipeline)
	if err != nil {
		t.Fatalf("Compose(plain patient) error = %v", err)
	}
	changedGraph, err := gobble.Compose(changedPipeline)
	if err != nil {
		t.Fatalf("Compose(changed patient) error = %v", err)
	}
	if !slices.Contains(plainGraph.InputNames(), "p8_patient1_s5_testN_l4_L001_r1") || !slices.Contains(changedGraph.InputNames(), "p15_changed_patient_s5_testN_l4_L001_r1") {
		t.Fatal("patient identity is absent from pipeline input identity")
	}

	plainTask := pc.TaskByID(t, plain, "patient1.testN.L001.bwa_mem")
	changedTask := pc.TaskByID(t, changed, "changed_patient.testN.L001.bwa_mem")
	if !strings.Contains(plainTask.Script, "ID:patient1.testN.L001") || !strings.Contains(changedTask.Script, "ID:changed_patient.testN.L001") {
		t.Fatalf("patient identity is absent from read groups:\nplain: %s\nchanged: %s", plainTask.Script, changedTask.Script)
	}
	assertParam(t, plainTask.Params, "patient", "patient1")
	assertParam(t, changedTask.Params, "patient", "changed_patient")
	plainJoint := pc.TaskByID(t, plain, "joint_intervals.database.gatk4_genomicsdbimport")
	changedJoint := pc.TaskByID(t, changed, "joint_intervals.database.gatk4_genomicsdbimport")
	if bytes.Equal(mustJSON(t, plainJoint), mustJSON(t, changedJoint)) {
		t.Fatal("changing patient identity left cohort import reuse identity unchanged")
	}
	assertParam(t, plainJoint.Params, "cohort", "patient1.testN,patient2.testT")
	assertParam(t, changedJoint.Params, "cohort", "changed_patient.testN,patient2.testT")
}

func TestWGSBuildIsPureAndCopiesCallerData(t *testing.T) {
	samples := loadSamples(t)
	config := wgs.DefaultConfig()
	pipeline := wgs.Build(samples, config)
	want := pc.MustPlanJSON(t, pipeline)
	samples[0].Name = "changed"
	samples[0].Lanes[0].Fastq1 = "changed.fastq.gz"
	config.Reference.Intervals[0].Name = "changed"
	config.HaplotypeCaller.ExtraArgs = append(config.HaplotypeCaller.ExtraArgs, "--changed")
	if got := pc.MustPlanJSON(t, pipeline); !bytes.Equal(got, want) {
		t.Fatalf("Build retained caller-owned mutable data")
	}
	sourcecheck.AssertNoCall(t, "../../../assets/pipelines/wgs/build.go", "SampleSheetPath", "Load", "Open", "ReadFile", "Stat", "Lstat", "Getwd")
}

func TestWGSRejectsIntervalAndProtectedOptionDefects(t *testing.T) {
	samples := loadSamples(t)

	t.Run("empty intervals", func(t *testing.T) {
		config := wgs.DefaultConfig()
		config.Reference.Intervals = nil
		_, err := gobble.Compose(wgs.Build(samples, config))
		requireDefect(t, err, gobble.DefectInvalidValue, "reference.intervals")
	})

	t.Run("duplicate interval", func(t *testing.T) {
		config := wgs.DefaultConfig()
		config.Reference.Intervals[1] = config.Reference.Intervals[0]
		_, err := gobble.Compose(wgs.Build(samples, config))
		requireDefect(t, err, gobble.DefectInvalidValue, "reference.intervals")
	})

	t.Run("HaplotypeCaller output prefix", func(t *testing.T) {
		config := wgs.DefaultConfig()
		config.HaplotypeCaller.ExtraArgs = []string{"--out"}
		_, err := gobble.Compose(wgs.Build(samples, config))
		requireDefect(t, err, gobble.DefectInvalidValue, "gatk4_haplotypecaller")
	})

	t.Run("FastP route prefix", func(t *testing.T) {
		config := wgs.DefaultConfig()
		config.FastP.ExtraArgs = []string{"--inter"}
		_, err := gobble.Compose(wgs.Build(samples, config))
		requireDefect(t, err, gobble.DefectInvalidValue, "fastp")
	})

	t.Run("MultiQC output prefix", func(t *testing.T) {
		config := wgs.DefaultConfig()
		config.MultiQC.ExtraArgs = []string{"--outd"}
		_, err := gobble.Compose(wgs.Build(samples, config))
		requireDefect(t, err, gobble.DefectInvalidValue, "multiqc")
	})
}

func TestWGSReadyBWAIndexIsCompleteAndSkipsPreparation(t *testing.T) {
	config := wgs.DefaultConfig()
	prefix := gobble.PathSpec{Dir: gobble.Dir("in/reference/bwa"), Base: "genome"}
	config.Reference.BWAIndex.Prefix = prefix
	for _, name := range []string{"amb", "ann", "bwt", "pac", "sa"} {
		config.Reference.BWAIndex.Members = append(config.Reference.BWAIndex.Members, gobble.Member{Name: name, Spec: prefix.AppendExt("." + name)})
	}
	raw := pc.MustPlanJSON(t, wgs.Build(loadSamples(t), config))
	tasks := pc.AllTasks(t, raw)
	if got := pc.CountTasksNamed(tasks, "bwa_index"); got != 0 {
		t.Fatalf("ready-index bwa_index count = %d, want 0", got)
	}
	mem := pc.TaskByID(t, raw, "patient1.testN.L001.bwa_mem")
	pc.AssertGroupMembers(t, mem.Inputs, "index", []pc.Member{
		{Name: "amb", Path: "in/reference/bwa/genome.amb"},
		{Name: "ann", Path: "in/reference/bwa/genome.ann"},
		{Name: "bwt", Path: "in/reference/bwa/genome.bwt"},
		{Name: "pac", Path: "in/reference/bwa/genome.pac"},
		{Name: "sa", Path: "in/reference/bwa/genome.sa"},
	})
	if !strings.Contains(mem.Script, "'in/reference/bwa/genome'") {
		t.Fatalf("ready-index bwa mem script does not use configured prefix: %s", mem.Script)
	}

	config.Reference.BWAIndex.Members = config.Reference.BWAIndex.Members[:4]
	_, err := gobble.Compose(wgs.Build(loadSamples(t), config))
	requireDefect(t, err, gobble.DefectInvalidValue, "reference.bwa_index")
}

func TestWGSDefaultsAreFreshAndLifecycleIsComplete(t *testing.T) {
	first := wgs.DefaultConfig()
	first.Reference.KnownSites[0].Name = "changed"
	first.Reference.Intervals[0].Name = "changed"
	first.HaplotypeCaller.ExtraArgs = append(first.HaplotypeCaller.ExtraArgs, "--changed")
	second := wgs.DefaultConfig()
	if second.Reference.KnownSites[0].Name != "dbsnp" || second.Reference.Intervals[0].Name != "interval_001" || len(second.HaplotypeCaller.ExtraArgs) != 0 {
		t.Fatalf("DefaultConfig retained caller mutation: %#v", second)
	}
	if wgs.GraphGeneration != "wgs-joint-germline-v1" || !wgs.Lifecycle.Design || !wgs.Lifecycle.Build || !wgs.Lifecycle.Customize || !wgs.Lifecycle.Run || !wgs.Lifecycle.Resume || !wgs.Lifecycle.Stop || !wgs.Lifecycle.Failure || wgs.Lifecycle.PreLiftResumable {
		t.Fatalf("Lifecycle = %#v, want complete non-resumable lift", wgs.Lifecycle)
	}
}

func TestWGSPipelineAdapterUsesInjectedSheet(t *testing.T) {
	previous := gobble.SampleSheetPath()
	gobble.SetSampleSheetPath(fixtureSheet)
	t.Cleanup(func() { gobble.SetSampleSheetPath(previous) })
	got := pc.MustPlanJSON(t, wgs.Pipeline())
	want := pc.MustPlanJSON(t, wgs.Build(loadSamples(t), wgs.DefaultConfig()))
	if !bytes.Equal(got, want) {
		t.Fatalf("Pipeline adapter graph differs from explicit Build")
	}
}

func assertScatterGatherFacts(t *testing.T, raw []byte) {
	t.Helper()
	var plan struct {
		Tasks []struct {
			ID      string `json:"id"`
			Scatter string `json:"scatter"`
			Gather  string `json:"gather"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &plan); err != nil {
		t.Fatalf("Unmarshal plan: %v", err)
	}
	want := map[string][2]string{
		"bqsr_intervals.patient1.testN.gatk4_baserecalibrator":     {"bqsr_intervals", ""},
		"patient1.testN.bqsr_gather.samtools_merge":                {"", "bqsr_gather"},
		"haplotype_intervals.patient1.testN.gatk4_haplotypecaller": {"haplotype_intervals", ""},
		"joint_intervals.database.gatk4_genomicsdbimport":          {"joint_intervals", ""},
		"joint_intervals.genotype.gatk4_genotypegvcfs":             {"joint_intervals", ""},
		"joint_gather.joint.gatk4_mergevcfs":                       {"", "joint_gather"},
	}
	for _, task := range plan.Tasks {
		if fact, ok := want[task.ID]; ok {
			if task.Scatter != fact[0] || task.Gather != fact[1] {
				t.Errorf("task %s scatter/gather = %q/%q, want %q/%q", task.ID, task.Scatter, task.Gather, fact[0], fact[1])
			}
			delete(want, task.ID)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing scatter/gather facts: %#v", want)
	}
}

func assertParam(t *testing.T, params []pc.Param, name, want string) {
	t.Helper()
	for _, param := range params {
		if param.Name == name {
			if param.Value != want {
				t.Fatalf("param %s = %q, want %q", name, param.Value, want)
			}
			return
		}
	}
	t.Fatalf("params %#v omit %s=%q", params, name, want)
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal(%T): %v", value, err)
	}
	return data
}

func loadSamples(t *testing.T) []wgs.Sample {
	t.Helper()
	samples, err := wgs.Load(fixtureSheet)
	if err != nil {
		t.Fatalf("Load(%s) error = %v", fixtureSheet, err)
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
