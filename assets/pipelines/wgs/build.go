package wgs

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	bcftoolssort "github.com/HahyeonJeon/gobble/assets/modules/bcftools-sort"
	bcftoolsstats "github.com/HahyeonJeon/gobble/assets/modules/bcftools-stats"
	bwaindex "github.com/HahyeonJeon/gobble/assets/modules/bwa-index"
	bwamem "github.com/HahyeonJeon/gobble/assets/modules/bwa-mem"
	"github.com/HahyeonJeon/gobble/assets/modules/fastp"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	gatk4applybqsr "github.com/HahyeonJeon/gobble/assets/modules/gatk4-applybqsr"
	gatk4baserecalibrator "github.com/HahyeonJeon/gobble/assets/modules/gatk4-baserecalibrator"
	gatk4gatherbqsrreports "github.com/HahyeonJeon/gobble/assets/modules/gatk4-gather-bqsr-reports"
	gatk4genomicsdbimport "github.com/HahyeonJeon/gobble/assets/modules/gatk4-genomicsdbimport"
	gatk4genotypegvcfs "github.com/HahyeonJeon/gobble/assets/modules/gatk4-genotypegvcfs"
	gatk4haplotypecaller "github.com/HahyeonJeon/gobble/assets/modules/gatk4-haplotypecaller"
	gatk4markduplicates "github.com/HahyeonJeon/gobble/assets/modules/gatk4-markduplicates"
	gatk4mergevcfs "github.com/HahyeonJeon/gobble/assets/modules/gatk4-mergevcfs"
	"github.com/HahyeonJeon/gobble/assets/modules/mosdepth"
	"github.com/HahyeonJeon/gobble/assets/modules/multiqc"
	samtoolsflagstat "github.com/HahyeonJeon/gobble/assets/modules/samtools-flagstat"
	samtoolsidxstats "github.com/HahyeonJeon/gobble/assets/modules/samtools-idxstats"
	samtoolsindex "github.com/HahyeonJeon/gobble/assets/modules/samtools-index"
	samtoolsmerge "github.com/HahyeonJeon/gobble/assets/modules/samtools-merge"
	samtoolssort "github.com/HahyeonJeon/gobble/assets/modules/samtools-sort"
	samtoolsstats "github.com/HahyeonJeon/gobble/assets/modules/samtools-stats"
)

type knownSiteHandles struct {
	vcf gobble.Handle
	tbi gobble.Handle
}

type sampleState struct {
	sample      Sample
	module      *gobble.Module
	markedBAM   gobble.Handle
	markedBAI   gobble.Handle
	recalBAM    gobble.Handle
	recalBAI    gobble.Handle
	gvcf        gobble.Handle
	gvcfTBI     gobble.Handle
	bqsrParts   gobble.Handle
	recalParts  gobble.Handle
	gvcfParts   gobble.Handle
	gvcfIndexes gobble.Handle
}

// Build constructs the selected WGS joint-germline graph only from supplied
// values. It copies samples, lanes, reference members, known sites, and every
// module ExtraArgs slice before use. Build performs no filesystem or network
// access.
func Build(inputSamples []Sample, inputConfig Config) *gobble.Pipeline {
	pipeline := gobble.NewPipeline("wgs")
	samples := cloneSamples(inputSamples)
	config := cloneConfig(inputConfig)
	if defects := validateBuild(samples, config); len(defects) > 0 {
		pipeline.RecordComposeError(composeError(defects))
		return pipeline
	}

	fasta := pipeline.AddInput("reference_fasta", config.Reference.FASTA)
	fai := pipeline.AddInput("reference_fai", config.Reference.FAI)
	dict := pipeline.AddInput("reference_dict", config.Reference.Dictionary)
	intervals := pipeline.AddInputGroup("intervals", config.Reference.Intervals)

	knownSites := make(map[string]knownSiteHandles, len(config.Reference.KnownSites))
	for _, site := range config.Reference.KnownSites {
		knownSites[site.Name] = knownSiteHandles{
			vcf: pipeline.AddInput("known_site_"+site.Name, site.VCF),
			tbi: pipeline.AddInput("known_site_"+site.Name+"_index", site.Index),
		}
	}
	dbsnp := knownSites[config.Reference.DBSNP]

	var bwaIndexHandle gobble.Handle
	var bwaPrefix gobble.PathSpec
	if config.Reference.BWAIndex.Members == nil {
		options := config.BWAIndex
		options.OutDir = gobble.Dir("work/reference/bwa")
		if config.Publication.GeneratedBWAIndex {
			options.OutDir = config.Results.Join("reference", "bwa")
		}
		prepared, err := bwaindex.Add(pipeline.AddModule("reference"), fasta, options)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		bwaIndexHandle, bwaPrefix = prepared.Index, prepared.Prefix
	} else {
		bwaIndexHandle = pipeline.AddInputGroup("bwa_index", config.Reference.BWAIndex.Members)
		bwaPrefix = config.Reference.BWAIndex.Prefix
	}

	reports := make([]gobble.Handle, 0, len(samples)*12)
	states := make([]sampleState, len(samples))
	patientModules := make(map[string]*gobble.Module)
	for sampleIndex, sample := range samples {
		module := sampleModule(pipeline, patientModules, sample)
		workDir := sampleWorkDir(sample)
		laneBAMs := make([]gobble.Handle, 0, len(sample.Lanes))
		laneBAIs := make([]gobble.Handle, 0, len(sample.Lanes))
		for _, lane := range sample.Lanes {
			laneModule := module.AddModule(lane.ID)
			laneParent := sampleTaskParent(laneModule, sample, lane.ID)
			inputName := sampleInputName(sample, lane.ID)
			read1 := pipeline.AddInput(inputName+"_r1", sheetFileSpec(lane.Fastq1))
			read2 := pipeline.AddInput(inputName+"_r2", sheetFileSpec(lane.Fastq2))
			for _, read := range []struct {
				name   string
				handle gobble.Handle
			}{{name: "r1", handle: read1}, {name: "r2", handle: read2}} {
				options := config.FastQC
				options.OutDir = gobble.Dir(workDir + "/" + lane.ID + "/raw-fastqc/" + read.name)
				qc, err := fastqc.Add(sampleTaskParent(laneModule.AddModule("raw_"+read.name), sample, lane.ID), read.handle, options)
				if recordModuleError(pipeline, err) {
					return pipeline
				}
				reports = append(reports, qc.HTML, qc.Zip)
			}
			fastpOptions := config.FastP
			fastpOptions.OutDir = gobble.Dir(workDir + "/" + lane.ID + "/fastp")
			if config.Publication.PreparedReads {
				fastpOptions.OutDir = config.Results.Join("intermediates", "prepared", sample.Patient, sample.Name, lane.ID)
			}
			fastpOptions.Prefix = sample.Name + "_" + lane.ID
			prepared, err := fastp.Add(laneParent, read1, read2, fastpOptions)
			if recordModuleError(pipeline, err) {
				return pipeline
			}
			reports = append(reports, prepared.JSON, prepared.HTML)
			memOptions := config.BWAMem
			memOptions.IndexPrefix = bwaPrefix
			memOptions.ReadGroup = readGroup(sample, lane)
			memOptions.OutDir = gobble.Dir(workDir + "/" + lane.ID + "/bwa-mem")
			memOptions.Prefix = sample.Name + "_" + lane.ID
			aligned, err := bwamem.Add(laneParent, fasta, bwaIndexHandle, prepared.Read1, prepared.Read2, memOptions)
			if recordModuleError(pipeline, err) {
				return pipeline
			}
			sortOptions := config.SamtoolsSort
			sortOptions.OutDir = gobble.Dir(workDir + "/" + lane.ID + "/sorted")
			sortOptions.Prefix = sample.Name + "_" + lane.ID
			sorted, err := samtoolssort.Add(laneParent, aligned.SAM, sortOptions)
			if recordModuleError(pipeline, err) {
				return pipeline
			}
			indexed, err := samtoolsindex.Add(laneParent, sorted.BAM, config.SamtoolsIndex)
			if recordModuleError(pipeline, err) {
				return pipeline
			}
			laneBAMs = append(laneBAMs, sorted.BAM)
			laneBAIs = append(laneBAIs, indexed.BAI)
		}

		normalizedBAM := laneBAMs[0]
		normalizedBAI := laneBAIs[0]
		if len(laneBAMs) > 1 {
			mergeOptions := config.SamtoolsMerge
			mergeOptions.OutDir = gobble.Dir(workDir + "/merged")
			mergeOptions.Prefix = sample.Name
			merged, err := samtoolsmerge.Add(sampleTaskParent(module, sample, ""), laneBAMs, laneBAIs, mergeOptions)
			if recordModuleError(pipeline, err) {
				return pipeline
			}
			indexed, err := samtoolsindex.Add(sampleTaskParent(module.AddModule("merged"), sample, ""), merged.BAM, config.SamtoolsIndex)
			if recordModuleError(pipeline, err) {
				return pipeline
			}
			normalizedBAM, normalizedBAI = merged.BAM, indexed.BAI
		}
		markOptions := config.MarkDuplicates
		markOptions.OutDir = gobble.Dir(workDir + "/markduplicates")
		markOptions.Prefix = sample.Name + ".marked"
		marked, err := gatk4markduplicates.Add(sampleTaskParent(module, sample, ""), normalizedBAM, normalizedBAI, markOptions)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		reports = append(reports, marked.Metrics)
		states[sampleIndex] = sampleState{sample: sample, module: module, markedBAM: marked.BAM, markedBAI: marked.BAI}
	}

	bqsrScatter := pipeline.Scatter("bqsr_intervals").From(intervals)
	bqsrPatients := make(map[string]*gobble.Module)
	for i := range states {
		state := &states[i]
		parent := sampleTaskParent(sampleModule(bqsrScatter, bqsrPatients, state.sample), state.sample, "")
		workDir := sampleWorkDir(state.sample)
		tableDir := gobble.Dir(workDir + "/bqsr/tables")
		baseOptions := config.BaseRecalibrator
		baseOptions.IntervalDir = intervalDirectory(config.Reference.Intervals)
		baseOptions.OutDir = tableDir
		sites := make([]gatk4baserecalibrator.KnownSite, 0, len(config.Reference.KnownSites))
		for _, site := range config.Reference.KnownSites {
			handles := knownSites[site.Name]
			sites = append(sites, gatk4baserecalibrator.KnownSite{VCF: handles.vcf, TBI: handles.tbi})
		}
		table, err := gatk4baserecalibrator.Add(parent, state.markedBAM, state.markedBAI, fasta, fai, dict, intervals, sites, baseOptions)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		applyOptions := config.ApplyBQSR
		applyOptions.IntervalDir = intervalDirectory(config.Reference.Intervals)
		applyOptions.TableDir = tableDir
		applyOptions.OutDir = gobble.Dir(workDir + "/bqsr/bams")
		applied, err := gatk4applybqsr.Add(parent, state.markedBAM, state.markedBAI, fasta, fai, dict, table.Table, intervals, applyOptions)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		state.bqsrParts, state.recalParts = table.Table, applied.BAM
	}

	for i := range states {
		state := &states[i]
		parent := sampleTaskParent(state.module.Gather("bqsr_gather"), state.sample, "")
		workDir := sampleWorkDir(state.sample)
		tableOptions := config.GatherBQSRReports
		tableOptions.InputPaths = intervalOutputs(config.Reference.Intervals, gobble.Dir(workDir+"/bqsr/tables"), ".table")
		tableOptions.OutDir = sampleResultsDir(config, state.sample).Join("bqsr")
		tableOptions.Prefix = state.sample.Name + ".recalibration"
		_, err := gatk4gatherbqsrreports.Add(parent, []gobble.Handle{state.bqsrParts}, tableOptions)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		bamOptions := config.GatherBAM
		bamOptions.InputPaths = intervalOutputs(config.Reference.Intervals, gobble.Dir(workDir+"/bqsr/bams"), ".bam")
		bamOptions.OutDir = sampleResultsDir(config, state.sample).Join("alignment")
		bamOptions.Prefix = state.sample.Name + ".recalibrated"
		gatheredBAM, err := samtoolsmerge.AddGather(parent, state.recalParts, bamOptions)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		indexed, err := samtoolsindex.Add(parent, gatheredBAM.BAM, config.SamtoolsIndex)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		state.recalBAM, state.recalBAI = gatheredBAM.BAM, indexed.BAI
	}

	for i := range states {
		state := &states[i]
		parent := sampleTaskParent(state.module.AddModule("alignment_qc"), state.sample, "")
		statsOptions := config.SamtoolsStats
		statsOptions.OutDir = config.Results.Join("qc", "alignment", state.sample.Patient, state.sample.Name)
		statsOptions.Prefix = state.sample.Name
		stats, err := samtoolsstats.Add(parent, state.recalBAM, statsOptions)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		flagOptions := config.SamtoolsFlagstat
		flagOptions.OutDir, flagOptions.Prefix = statsOptions.OutDir, state.sample.Name
		flagstat, err := samtoolsflagstat.Add(parent, state.recalBAM, state.recalBAI, flagOptions)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		idxOptions := config.SamtoolsIdxstats
		idxOptions.OutDir, idxOptions.Prefix = statsOptions.OutDir, state.sample.Name
		idxstats, err := samtoolsidxstats.Add(parent, state.recalBAM, state.recalBAI, idxOptions)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		mosOptions := config.Mosdepth
		mosOptions.OutDir, mosOptions.Prefix = statsOptions.OutDir, state.sample.Name
		coverage, err := mosdepth.Add(parent, state.recalBAM, state.recalBAI, fasta, mosOptions)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		reports = append(reports, stats.Stats, flagstat.Report, idxstats.Report, coverage.Summary, coverage.Global)
	}

	haplotypeScatter := pipeline.Scatter("haplotype_intervals").From(intervals)
	haplotypePatients := make(map[string]*gobble.Module)
	for i := range states {
		state := &states[i]
		workDir := sampleWorkDir(state.sample)
		options := config.HaplotypeCaller
		options.IntervalDir = intervalDirectory(config.Reference.Intervals)
		options.OutDir = gobble.Dir(workDir + "/haplotypecaller")
		parent := sampleTaskParent(sampleModule(haplotypeScatter, haplotypePatients, state.sample), state.sample, "")
		called, err := gatk4haplotypecaller.Add(parent, state.recalBAM, state.recalBAI, fasta, fai, dict, dbsnp.vcf, dbsnp.tbi, intervals, options)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		state.gvcfParts, state.gvcfIndexes = called.GVCF, called.TBI
	}

	for i := range states {
		state := &states[i]
		parent := sampleTaskParent(state.module.Gather("gvcf_gather"), state.sample, "")
		options := config.MergeGVCFs
		options.InputPaths = intervalOutputs(config.Reference.Intervals, gobble.Dir(sampleWorkDir(state.sample)+"/haplotypecaller"), ".g.vcf.gz")
		options.OutDir = sampleResultsDir(config, state.sample).Join("gvcf")
		options.Prefix = state.sample.Name + ".g"
		merged, err := gatk4mergevcfs.Add(parent, []gobble.Handle{state.gvcfParts}, []gobble.Handle{state.gvcfIndexes}, dict, options)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		state.gvcf, state.gvcfTBI = merged.VCF, merged.TBI
	}

	variants := make([]gatk4genomicsdbimport.Variant, len(states))
	for i, state := range states {
		variants[i] = gatk4genomicsdbimport.Variant{GVCF: state.gvcf, TBI: state.gvcfTBI}
	}
	jointScatter := pipeline.Scatter("joint_intervals").From(intervals)
	cohort := cohortIdentity(samples)
	jointWorkDir := gobble.Dir("work/joint").Join(
		"cohort-"+cohortWorkIdentity(cohort),
		"intervals-"+intervalWorkIdentity(config.Reference.Intervals),
	)
	databaseOptions := config.GenomicsDBImport
	databaseOptions.IntervalDir = intervalDirectory(config.Reference.Intervals)
	databaseOptions.OutDir = jointWorkDir.Join("genomicsdb")
	database, err := gatk4genomicsdbimport.Add(cohortTaskParent(jointScatter.AddModule("database"), cohort), variants, intervals, databaseOptions)
	if recordModuleError(pipeline, err) {
		return pipeline
	}
	jointParent := cohortTaskParent(jointScatter.AddModule("genotype"), cohort)
	genotypeOptions := config.GenotypeGVCFs
	genotypeOptions.IntervalDir = intervalDirectory(config.Reference.Intervals)
	genotypeOptions.OutDir = jointWorkDir.Join("genotype")
	genotyped, err := gatk4genotypegvcfs.Add(jointParent, database.Database, intervals, fasta, fai, dict, dbsnp.vcf, dbsnp.tbi, genotypeOptions)
	if recordModuleError(pipeline, err) {
		return pipeline
	}
	sortOptions := config.BCFToolsSort
	sortOptions.IntervalDir = intervalDirectory(config.Reference.Intervals)
	sortOptions.InputDir = genotypeOptions.OutDir
	sortOptions.OutDir = jointWorkDir.Join("sorted")
	sorted, err := bcftoolssort.Add(jointParent, genotyped.VCF, genotyped.TBI, intervals, sortOptions)
	if recordModuleError(pipeline, err) {
		return pipeline
	}

	jointParentGather := cohortTaskParent(pipeline.Gather("joint_gather").AddModule("joint"), cohort)
	jointOptions := config.MergeJointVCFs
	jointOptions.InputPaths = intervalOutputs(config.Reference.Intervals, sortOptions.OutDir, ".sorted.vcf.gz")
	jointOptions.OutDir = config.Results.Join("joint")
	jointOptions.Prefix = "joint_germline"
	joint, err := gatk4mergevcfs.Add(jointParentGather, []gobble.Handle{sorted.VCF}, []gobble.Handle{sorted.TBI}, dict, jointOptions)
	if recordModuleError(pipeline, err) {
		return pipeline
	}
	statsOptions := config.BCFToolsStats
	statsOptions.OutDir = config.Results.Join("qc", "callset")
	statsOptions.Prefix = "joint_germline"
	callStats, err := bcftoolsstats.Add(cohortTaskParent(pipeline.AddModule("callset_qc"), cohort), joint.VCF, joint.TBI, fasta, statsOptions)
	if recordModuleError(pipeline, err) {
		return pipeline
	}
	reports = append(reports, callStats.Stats)
	multiQCOptions := config.MultiQC
	multiQCOptions.OutDir = config.Results.Join("multiqc")
	if _, err = multiqc.Add(cohortTaskParent(pipeline, cohort), reports, multiQCOptions); recordModuleError(pipeline, err) {
		return pipeline
	}
	return pipeline
}

func readGroup(sample Sample, lane Lane) string {
	identity := sampleIdentity(sample)
	return "@RG\\tID:" + identity + "." + lane.ID + "\\tSM:" + identity + "\\tLB:" + identity + "\\tPL:ILLUMINA\\tPU:" + identity + "." + lane.ID
}

type moduleParent interface {
	AddModule(string) *gobble.Module
}

type parameterizedParent struct {
	parent modules.Parent
	params []gobble.Param
}

func (p parameterizedParent) AddTask(spec gobble.TaskSpec) *gobble.Task {
	spec.Params = append(append([]gobble.Param(nil), spec.Params...), p.params...)
	return p.parent.AddTask(spec)
}

func sampleModule(parent moduleParent, patients map[string]*gobble.Module, sample Sample) *gobble.Module {
	patient := patients[sample.Patient]
	if patient == nil {
		patient = parent.AddModule(sample.Patient)
		patients[sample.Patient] = patient
	}
	return patient.AddModule(sample.Name)
}

func sampleTaskParent(parent modules.Parent, sample Sample, lane string) modules.Parent {
	params := []gobble.Param{{Name: "patient", Value: sample.Patient}, {Name: "sample", Value: sample.Name}}
	if sample.Sex != "" {
		params = append(params, gobble.Param{Name: "sex", Value: sample.Sex})
	}
	if lane != "" {
		params = append(params, gobble.Param{Name: "lane", Value: lane})
	}
	return parameterizedParent{parent: parent, params: params}
}

func cohortTaskParent(parent modules.Parent, cohort string) modules.Parent {
	return parameterizedParent{parent: parent, params: []gobble.Param{{Name: "cohort", Value: cohort}}}
}

func sampleIdentity(sample Sample) string {
	identity := sampleKey(sample)
	if sample.Sex != "" {
		identity += "." + sample.Sex
	}
	return identity
}

func sampleKey(sample Sample) string {
	return sample.Patient + "." + sample.Name
}

func cohortIdentity(samples []Sample) string {
	identities := make([]string, len(samples))
	for i, sample := range samples {
		identities[i] = sampleIdentity(sample)
	}
	return strings.Join(identities, ",")
}

func cohortWorkIdentity(cohort string) string {
	digest := sha256.Sum256([]byte(cohort))
	return hex.EncodeToString(digest[:])
}

func intervalWorkIdentity(intervals gobble.Group) string {
	parts := make([]string, 0, len(intervals)*2)
	for _, interval := range intervals {
		path, _ := interval.Spec.Render()
		parts = append(parts, interval.Name, path)
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(digest[:])
}

func sampleInputName(sample Sample, lane string) string {
	name := "p" + strconv.Itoa(len(sample.Patient)) + "_" + sample.Patient +
		"_s" + strconv.Itoa(len(sample.Name)) + "_" + sample.Name
	if sample.Sex != "" {
		name += "_x" + strconv.Itoa(len(sample.Sex)) + "_" + sample.Sex
	}
	return name + "_l" + strconv.Itoa(len(lane)) + "_" + lane
}

func sampleWorkDir(sample Sample) string {
	return "work/" + sample.Patient + "/" + sample.Name
}

func sampleResultsDir(config Config, sample Sample) gobble.Directory {
	return config.Results.Join("samples", sample.Patient, sample.Name)
}

func intervalDirectory(intervals gobble.Group) gobble.Directory {
	return intervals[0].Spec.Dir
}

func intervalOutputs(intervals gobble.Group, dir gobble.Directory, extension string) []gobble.PathSpec {
	outputs := make([]gobble.PathSpec, len(intervals))
	for i, member := range intervals {
		outputs[i] = gobble.PathSpec{Dir: dir, Base: member.Spec.Base, Ext: extension}
	}
	return outputs
}

func recordModuleError(pipeline *gobble.Pipeline, err error) bool {
	if err == nil {
		return false
	}
	pipeline.RecordComposeError(err)
	return true
}

func sheetFileSpec(value string) gobble.PathSpec {
	dir, file := "", value
	if split := strings.LastIndex(value, "/"); split >= 0 {
		dir, file = value[:split], value[split+1:]
	}
	base, ext := file, ""
	for _, suffix := range []string{".fastq.gz", ".fq.gz", ".fastq", ".fq"} {
		if strings.HasSuffix(strings.ToLower(file), suffix) {
			base, ext = file[:len(file)-len(suffix)], file[len(file)-len(suffix):]
			break
		}
	}
	spec := gobble.PathSpec{Base: base, Ext: ext}
	if dir != "" {
		spec.Dir = gobble.Dir(dir)
	}
	return spec
}

func validateBuild(samples []Sample, config Config) []gobble.Defect {
	var defects []gobble.Defect
	if len(samples) < 2 {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: "samplesheet", Message: "WGS joint germline requires at least two distinct samples"})
	}
	seenSamples := make(map[string]bool, len(samples))
	for _, sample := range samples {
		identity := sampleKey(sample)
		if !identityPattern.MatchString(sample.Patient) || !identityPattern.MatchString(sample.Name) || seenSamples[identity] {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: identity, Message: "WGS patient/sample identity is invalid or duplicated"})
		}
		seenSamples[identity] = true
		if sample.Sex != "" && !identityPattern.MatchString(sample.Sex) {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: sample.Name, Message: "WGS sex identity is invalid"})
		}
		if len(sample.Lanes) == 0 {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: sample.Name, Message: "WGS sample requires at least one paired lane"})
			continue
		}
		seenLanes := make(map[string]bool, len(sample.Lanes))
		for _, lane := range sample.Lanes {
			if !identityPattern.MatchString(lane.ID) || seenLanes[lane.ID] || !validWorkspacePath(lane.Fastq1) || !validWorkspacePath(lane.Fastq2) {
				defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: sample.Name, Message: "WGS lane identity or paired-read paths are invalid", Paths: []string{lane.Fastq1, lane.Fastq2}})
			}
			seenLanes[lane.ID] = true
		}
	}
	for _, spec := range []struct {
		unit string
		spec gobble.PathSpec
	}{{"reference.fasta", config.Reference.FASTA}, {"reference.fai", config.Reference.FAI}, {"reference.dictionary", config.Reference.Dictionary}} {
		if rendered, err := spec.spec.Render(); err != nil || !validWorkspacePath(rendered) {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidPath, Unit: spec.unit, Message: "WGS reference path must be workspace-relative", Paths: []string{rendered}})
		}
	}
	if !config.Reference.FAI.Equal(config.Reference.FASTA.AppendExt(".fai")) || !config.Reference.Dictionary.Equal(config.Reference.FASTA.WithExt(".dict")) {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "reference", Message: "WGS FASTA, FAI, and dictionary paths do not form one reference bundle"})
	}
	if len(config.Reference.KnownSites) == 0 {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "reference.known_sites", Message: "WGS BQSR known sites must not be empty"})
	}
	seenSites := make(map[string]bool, len(config.Reference.KnownSites))
	for _, site := range config.Reference.KnownSites {
		vcf, vcfErr := site.VCF.Render()
		index, indexErr := site.Index.Render()
		if !identityPattern.MatchString(site.Name) || seenSites[site.Name] || vcfErr != nil || indexErr != nil || !validWorkspacePath(vcf) || !validWorkspacePath(index) || !site.Index.Equal(site.VCF.AppendExt(".tbi")) {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidPath, Unit: "reference.known_sites", Message: "WGS known-site identity or path is invalid", Paths: []string{vcf, index}})
		}
		seenSites[site.Name] = true
	}
	if !seenSites[config.Reference.DBSNP] {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "reference.dbsnp", Message: "WGS dbSNP name must resolve to one known site"})
	}
	if defect := intervalDefect(config.Reference.Intervals); defect != nil {
		defects = append(defects, *defect)
	}
	if config.Reference.BWAIndex.Members != nil {
		if defect := readyBWAIndexDefect(config.Reference.BWAIndex); defect != nil {
			defects = append(defects, *defect)
		}
	}
	if config.Results.IsZero() || !validWorkspacePath(config.Results.String()+"/result") {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidPath, Unit: "results", Message: "WGS results directory must be workspace-relative"})
	}
	if config.Format != OutputBAM {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "output_format", Message: "WGS output format must be BAM"})
	}
	if !config.Publication.RecalibratedAlignments || !config.Publication.SampleGVCFs || !config.Publication.JointCallset || !config.Publication.Reports {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "publication", Message: "WGS recalibrated alignment, sample gVCF, joint callset, and report publication cannot be disabled"})
	}
	if unit, flag := protectedExtra(config); flag != "" {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: unit, Message: "WGS ExtraArgs contains protected option " + flag})
	}
	return defects
}

func protectedExtra(config Config) (string, string) {
	sets := []struct {
		unit  string
		args  []string
		flags []string
	}{
		{unit: "fastqc", args: config.FastQC.ExtraArgs, flags: []string{"--outdir", "--noextract", "--threads", "--extract", "--version", "--help"}},
		{unit: "samtools_sort", args: config.SamtoolsSort.ExtraArgs, flags: []string{"-o", "-@"}},
		{unit: "samtools_index", args: config.SamtoolsIndex.ExtraArgs, flags: []string{"-@"}},
		{unit: "multiqc", args: config.MultiQC.ExtraArgs, flags: []string{"--force", "--outdir", "--filename", "--no-data-dir", "--zip-data-dir", "--version", "--help"}},
	}
	for _, set := range sets {
		if flag := modules.MatchProtectedExtraArg(set.args, set.flags); flag != "" {
			return set.unit, flag
		}
	}
	return "", ""
}

func intervalDefect(intervals gobble.Group) *gobble.Defect {
	if len(intervals) == 0 {
		return &gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "reference.intervals", Message: "WGS interval membership must not be empty"}
	}
	seen := make(map[string]bool, len(intervals))
	dir := intervals[0].Spec.Dir.String()
	for _, member := range intervals {
		path, err := member.Spec.Render()
		if !identityPattern.MatchString(member.Name) || seen[member.Name] || member.Spec.Base != member.Name || member.Spec.Dir.String() != dir || member.Spec.Ext != ".bed" || member.Spec.Prefix != "" || len(member.Spec.Suffixes) != 0 || err != nil || !validWorkspacePath(path) {
			return &gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "reference.intervals", Message: "WGS interval membership is incomplete, duplicated, or unstable", Paths: []string{path}}
		}
		seen[member.Name] = true
	}
	return nil
}

func readyBWAIndexDefect(index ReadyBWAIndex) *gobble.Defect {
	prefix, err := index.Prefix.Render()
	if err != nil || !validWorkspacePath(prefix) {
		return &gobble.Defect{Code: gobble.DefectInvalidPath, Unit: "reference.bwa_index", Message: "ready BWA index prefix must be workspace-relative", Paths: []string{prefix}}
	}
	want := []string{"amb", "ann", "bwt", "pac", "sa"}
	if len(index.Members) != len(want) {
		return &gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "reference.bwa_index", Message: "ready BWA index must contain every fixed sidecar"}
	}
	for i, member := range index.Members {
		path, pathErr := member.Spec.Render()
		if member.Name != want[i] || pathErr != nil || !validWorkspacePath(path) || !member.Spec.Equal(index.Prefix.AppendExt("."+want[i])) {
			return &gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "reference.bwa_index", Message: "ready BWA index members are incomplete or out of order", Paths: []string{path}}
		}
	}
	return nil
}

func cloneConfig(config Config) Config {
	clone := func(options *modules.Options) { *options = options.Clone() }
	clone(&config.BWAIndex.Options)
	clone(&config.FastQC.Options)
	clone(&config.FastP.Options)
	clone(&config.BWAMem.Options)
	clone(&config.SamtoolsSort.Options)
	clone(&config.SamtoolsMerge.Options)
	clone(&config.MarkDuplicates.Options)
	clone(&config.BaseRecalibrator.Options)
	clone(&config.GatherBQSRReports.Options)
	clone(&config.ApplyBQSR.Options)
	clone(&config.GatherBAM.Options)
	clone(&config.SamtoolsIndex.Options)
	clone(&config.SamtoolsStats.Options)
	clone(&config.SamtoolsFlagstat.Options)
	clone(&config.SamtoolsIdxstats.Options)
	clone(&config.Mosdepth.Options)
	clone(&config.HaplotypeCaller.Options)
	clone(&config.MergeGVCFs.Options)
	clone(&config.GenomicsDBImport.Options)
	clone(&config.GenotypeGVCFs.Options)
	clone(&config.BCFToolsSort.Options)
	clone(&config.MergeJointVCFs.Options)
	clone(&config.BCFToolsStats.Options)
	clone(&config.MultiQC.Options)
	config.Reference.FASTA = cloneSpec(config.Reference.FASTA)
	config.Reference.FAI = cloneSpec(config.Reference.FAI)
	config.Reference.Dictionary = cloneSpec(config.Reference.Dictionary)
	config.Reference.BWAIndex.Prefix = cloneSpec(config.Reference.BWAIndex.Prefix)
	config.Reference.KnownSites = append([]KnownSite(nil), config.Reference.KnownSites...)
	for i := range config.Reference.KnownSites {
		config.Reference.KnownSites[i].VCF = cloneSpec(config.Reference.KnownSites[i].VCF)
		config.Reference.KnownSites[i].Index = cloneSpec(config.Reference.KnownSites[i].Index)
	}
	config.Reference.Intervals = cloneGroup(config.Reference.Intervals)
	config.Reference.BWAIndex.Members = cloneGroup(config.Reference.BWAIndex.Members)
	config.SamtoolsMerge.InputPaths = cloneSpecs(config.SamtoolsMerge.InputPaths)
	config.GatherBQSRReports.InputPaths = cloneSpecs(config.GatherBQSRReports.InputPaths)
	config.GatherBAM.InputPaths = cloneSpecs(config.GatherBAM.InputPaths)
	config.MergeGVCFs.InputPaths = cloneSpecs(config.MergeGVCFs.InputPaths)
	config.MergeJointVCFs.InputPaths = cloneSpecs(config.MergeJointVCFs.InputPaths)
	return config
}

func cloneGroup(group gobble.Group) gobble.Group {
	if group == nil {
		return nil
	}
	out := make(gobble.Group, len(group))
	for i, member := range group {
		out[i] = member
		out[i].Spec.Suffixes = append([]string(nil), member.Spec.Suffixes...)
	}
	return out
}

func cloneSpecs(specs []gobble.PathSpec) []gobble.PathSpec {
	out := make([]gobble.PathSpec, len(specs))
	for i, spec := range specs {
		out[i] = spec
		out[i].Suffixes = append([]string(nil), spec.Suffixes...)
	}
	return out
}

func cloneSpec(spec gobble.PathSpec) gobble.PathSpec {
	spec.Suffixes = append([]string(nil), spec.Suffixes...)
	return spec
}
