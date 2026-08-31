package wgsevidence

import (
	"github.com/HahyeonJeon/gobble"
	bcftoolssort "github.com/HahyeonJeon/gobble/assets/modules/bcftools-sort"
	gatk4genomicsdbimport "github.com/HahyeonJeon/gobble/assets/modules/gatk4-genomicsdbimport"
	gatk4genotypegvcfs "github.com/HahyeonJeon/gobble/assets/modules/gatk4-genotypegvcfs"
	gatk4haplotypecaller "github.com/HahyeonJeon/gobble/assets/modules/gatk4-haplotypecaller"
	gatk4mergevcfs "github.com/HahyeonJeon/gobble/assets/modules/gatk4-mergevcfs"
	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
)

type jointMappedSample struct {
	patient string
	sample  string
	bam     gobble.PathSpec
	bai     gobble.PathSpec
}

type jointFixtureState struct {
	sample  jointMappedSample
	module  *gobble.Module
	gvcf    gobble.Handle
	gvcfTBI gobble.Handle
	parts   gobble.Handle
	indexes gobble.Handle
}

// JointFixturePipeline builds the Planning-bound J evidence path. It starts at
// both exact mapped BAM/BAI pairs and reaches the unfiltered joint callset. It
// is test-owned evidence, not a second product entry point.
func JointFixturePipeline() *gobble.Pipeline {
	pipeline := gobble.NewPipeline("wgs-joint-fixture")
	config := wgs.DefaultConfig()
	fasta := pipeline.AddInput("reference_fasta", config.Reference.FASTA)
	fai := pipeline.AddInput("reference_fai", config.Reference.FAI)
	dict := pipeline.AddInput("reference_dict", config.Reference.Dictionary)
	dbsnp := pipeline.AddInput("dbsnp", config.Reference.KnownSites[0].VCF)
	dbsnpTBI := pipeline.AddInput("dbsnp_tbi", config.Reference.KnownSites[0].Index)
	intervals := pipeline.AddInputGroup("intervals", config.Reference.Intervals)
	intervalDir := config.Reference.Intervals[0].Spec.Dir

	mapped := []jointMappedSample{
		{
			patient: "patient1", sample: "testN",
			bam: jointPath("testN", "test.paired_end.sorted", ".bam"),
			bai: jointPath("testN", "test.paired_end.sorted", ".bam.bai"),
		},
		{
			patient: "patient2", sample: "testT",
			bam: jointPath("testT", "test2.paired_end.sorted", ".bam"),
			bai: jointPath("testT", "test2.paired_end.sorted", ".bam.bai"),
		},
	}
	states := make([]jointFixtureState, len(mapped))
	patientModules := make(map[string]*gobble.Module)
	callScatter := pipeline.Scatter("joint_fixture_haplotype_intervals").From(intervals)
	callPatients := make(map[string]*gobble.Module)
	for i, sample := range mapped {
		module := fixtureSampleModule(pipeline, patientModules, sample)
		bam := pipeline.AddInput(sample.patient+"_"+sample.sample+"_bam", sample.bam)
		bai := pipeline.AddInput(sample.patient+"_"+sample.sample+"_bai", sample.bai)
		options := config.HaplotypeCaller
		options.IntervalDir = intervalDir
		options.OutDir = gobble.Dir("work/evidence/joint/" + sample.patient + "/" + sample.sample + "/haplotypecaller")
		called, err := gatk4haplotypecaller.Add(fixtureSampleModule(callScatter, callPatients, sample), bam, bai, fasta, fai, dict, dbsnp, dbsnpTBI, intervals, options)
		if jointFixtureError(pipeline, err) {
			return pipeline
		}
		states[i] = jointFixtureState{sample: sample, module: module, parts: called.GVCF, indexes: called.TBI}
	}

	for i := range states {
		state := &states[i]
		options := config.MergeGVCFs
		options.InputPaths = jointIntervalOutputs(config.Reference.Intervals, gobble.Dir("work/evidence/joint/"+state.sample.patient+"/"+state.sample.sample+"/haplotypecaller"), ".g.vcf.gz")
		options.OutDir = gobble.Dir("work/evidence/joint/" + state.sample.patient + "/" + state.sample.sample + "/gvcf")
		options.Prefix = state.sample.sample + ".g"
		merged, err := gatk4mergevcfs.Add(state.module.Gather("gvcf_gather"), []gobble.Handle{state.parts}, []gobble.Handle{state.indexes}, dict, options)
		if jointFixtureError(pipeline, err) {
			return pipeline
		}
		state.gvcf, state.gvcfTBI = merged.VCF, merged.TBI
	}

	variants := make([]gatk4genomicsdbimport.Variant, len(states))
	for i, state := range states {
		variants[i] = gatk4genomicsdbimport.Variant{GVCF: state.gvcf, TBI: state.gvcfTBI}
	}
	jointScatter := pipeline.Scatter("joint_fixture_intervals").From(intervals)
	databaseOptions := config.GenomicsDBImport
	databaseOptions.IntervalDir = intervalDir
	databaseOptions.OutDir = gobble.Dir("work/evidence/joint/genomicsdb")
	database, err := gatk4genomicsdbimport.Add(jointScatter.AddModule("database"), variants, intervals, databaseOptions)
	if jointFixtureError(pipeline, err) {
		return pipeline
	}
	jointParent := jointScatter.AddModule("genotype")
	genotypeOptions := config.GenotypeGVCFs
	genotypeOptions.IntervalDir = intervalDir
	genotypeOptions.OutDir = gobble.Dir("work/evidence/joint/genotype")
	genotyped, err := gatk4genotypegvcfs.Add(jointParent, database.Database, intervals, fasta, fai, dict, dbsnp, dbsnpTBI, genotypeOptions)
	if jointFixtureError(pipeline, err) {
		return pipeline
	}
	sortOptions := config.BCFToolsSort
	sortOptions.IntervalDir = intervalDir
	sortOptions.InputDir = genotypeOptions.OutDir
	sortOptions.OutDir = gobble.Dir("work/evidence/joint/sorted")
	sorted, err := bcftoolssort.Add(jointParent, genotyped.VCF, genotyped.TBI, intervals, sortOptions)
	if jointFixtureError(pipeline, err) {
		return pipeline
	}
	mergeOptions := config.MergeJointVCFs
	mergeOptions.InputPaths = jointIntervalOutputs(config.Reference.Intervals, sortOptions.OutDir, ".sorted.vcf.gz")
	mergeOptions.OutDir = gobble.Dir("evidence/wgs/joint")
	mergeOptions.Prefix = "joint_germline"
	_, err = gatk4mergevcfs.Add(pipeline.Gather("joint_fixture_gather").AddModule("joint"), []gobble.Handle{sorted.VCF}, []gobble.Handle{sorted.TBI}, dict, mergeOptions)
	jointFixtureError(pipeline, err)
	return pipeline
}

type fixtureModuleParent interface {
	AddModule(string) *gobble.Module
}

func fixtureSampleModule(parent fixtureModuleParent, patients map[string]*gobble.Module, sample jointMappedSample) *gobble.Module {
	patient := patients[sample.patient]
	if patient == nil {
		patient = parent.AddModule(sample.patient)
		patients[sample.patient] = patient
	}
	return patient.AddModule(sample.sample)
}

func jointPath(sample, base, ext string) gobble.PathSpec {
	return gobble.PathSpec{Dir: gobble.Dir("in/joint/" + sample), Base: base, Ext: ext}
}

func jointIntervalOutputs(intervals gobble.Group, dir gobble.Directory, extension string) []gobble.PathSpec {
	outputs := make([]gobble.PathSpec, len(intervals))
	for i, member := range intervals {
		outputs[i] = gobble.PathSpec{Dir: dir, Base: member.Spec.Base, Ext: extension}
	}
	return outputs
}

func jointFixtureError(pipeline *gobble.Pipeline, err error) bool {
	if err == nil {
		return false
	}
	pipeline.RecordComposeError(err)
	return true
}
