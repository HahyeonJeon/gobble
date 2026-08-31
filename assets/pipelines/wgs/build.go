package wgs

import (
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
	gatk4gatherbamfiles "github.com/HahyeonJeon/gobble/assets/modules/gatk4-gatherbamfiles"
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
	intervalHandles := make(map[string]gobble.Handle, len(config.Reference.Intervals))
	for _, member := range config.Reference.Intervals {
		intervalHandles[member.Name] = pipeline.AddInput("interval_"+member.Name, member.Spec)
	}

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
	for sampleIndex, sample := range samples {
		module := pipeline.AddModule(sample.Name)
		laneBAMs := make([]gobble.Handle, 0, len(sample.Lanes))
		laneBAIs := make([]gobble.Handle, 0, len(sample.Lanes))
		for _, lane := range sample.Lanes {
			laneModule := module.AddModule(lane.ID)
			read1 := pipeline.AddInput(sample.Name+"_"+lane.ID+"_r1", sheetFileSpec(lane.Fastq1))
			read2 := pipeline.AddInput(sample.Name+"_"+lane.ID+"_r2", sheetFileSpec(lane.Fastq2))
			for _, read := range []struct {
				name   string
				handle gobble.Handle
			}{{name: "r1", handle: read1}, {name: "r2", handle: read2}} {
				options := config.FastQC
				options.OutDir = gobble.Dir("work/" + sample.Name + "/" + lane.ID + "/raw-fastqc/" + read.name)
				qc, err := fastqc.Add(laneModule.AddModule("raw_"+read.name), read.handle, options)
				if recordModuleError(pipeline, err) {
					return pipeline
				}
				reports = append(reports, qc.HTML, qc.Zip)
			}
			fastpOptions := config.FastP
			fastpOptions.OutDir = gobble.Dir("work/" + sample.Name + "/" + lane.ID + "/fastp")
			if config.Publication.PreparedReads {
				fastpOptions.OutDir = config.Results.Join("intermediates", "prepared", sample.Name, lane.ID)
			}
			fastpOptions.Prefix = sample.Name + "_" + lane.ID
			prepared, err := fastp.Add(laneModule, read1, read2, fastpOptions)
			if recordModuleError(pipeline, err) {
				return pipeline
			}
			reports = append(reports, prepared.JSON, prepared.HTML)
			memOptions := config.BWAMem
			memOptions.IndexPrefix = bwaPrefix
			memOptions.ReadGroup = readGroup(sample, lane)
			memOptions.OutDir = gobble.Dir("work/" + sample.Name + "/" + lane.ID + "/bwa-mem")
			memOptions.Prefix = sample.Name + "_" + lane.ID
			aligned, err := bwamem.Add(laneModule, fasta, bwaIndexHandle, prepared.Read1, prepared.Read2, memOptions)
			if recordModuleError(pipeline, err) {
				return pipeline
			}
			sortOptions := config.SamtoolsSort
			sortOptions.OutDir = gobble.Dir("work/" + sample.Name + "/" + lane.ID + "/sorted")
			sortOptions.Prefix = sample.Name + "_" + lane.ID
			sorted, err := samtoolssort.Add(laneModule, aligned.SAM, sortOptions)
			if recordModuleError(pipeline, err) {
				return pipeline
			}
			indexed, err := samtoolsindex.Add(laneModule, sorted.BAM, config.SamtoolsIndex)
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
			mergeOptions.OutDir = gobble.Dir("work/" + sample.Name + "/merged")
			mergeOptions.Prefix = sample.Name
			merged, err := samtoolsmerge.Add(module, laneBAMs, laneBAIs, mergeOptions)
			if recordModuleError(pipeline, err) {
				return pipeline
			}
			indexed, err := samtoolsindex.Add(module.AddModule("merged"), merged.BAM, config.SamtoolsIndex)
			if recordModuleError(pipeline, err) {
				return pipeline
			}
			normalizedBAM, normalizedBAI = merged.BAM, indexed.BAI
		}
		markOptions := config.MarkDuplicates
		markOptions.OutDir = gobble.Dir("work/" + sample.Name + "/markduplicates")
		markOptions.Prefix = sample.Name + ".marked"
		marked, err := gatk4markduplicates.Add(module, normalizedBAM, normalizedBAI, markOptions)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		reports = append(reports, marked.Metrics)
		states[sampleIndex] = sampleState{sample: sample, markedBAM: marked.BAM, markedBAI: marked.BAI}
	}

	bqsrScatter := pipeline.Scatter("bqsr_intervals").From(intervals)
	for i := range states {
		state := &states[i]
		parent := bqsrScatter.AddModule(state.sample.Name)
		tableDir := gobble.Dir("work/" + state.sample.Name + "/bqsr/tables")
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
		applyOptions.OutDir = gobble.Dir("work/" + state.sample.Name + "/bqsr/bams")
		applied, err := gatk4applybqsr.Add(parent, state.markedBAM, state.markedBAI, fasta, fai, dict, table.Table, intervals, applyOptions)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		state.bqsrParts, state.recalParts = table.Table, applied.BAM
	}

	for i := range states {
		state := &states[i]
		parent := pipeline.Gather(state.sample.Name + "_bqsr_gather").AddModule(state.sample.Name)
		tableOptions := config.GatherBQSRReports
		tableOptions.InputPaths = intervalOutputs(config.Reference.Intervals, gobble.Dir("work/"+state.sample.Name+"/bqsr/tables"), ".table")
		tableOptions.OutDir = config.Results.Join("samples", state.sample.Name, "bqsr")
		tableOptions.Prefix = state.sample.Name + ".recalibration"
		_, err := gatk4gatherbqsrreports.Add(parent, []gobble.Handle{state.bqsrParts}, tableOptions)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		bamOptions := config.GatherBAM
		bamOptions.InputPaths = intervalOutputs(config.Reference.Intervals, gobble.Dir("work/"+state.sample.Name+"/bqsr/bams"), ".bam")
		bamOptions.OutDir = config.Results.Join("samples", state.sample.Name, "alignment")
		bamOptions.Prefix = state.sample.Name + ".recalibrated"
		gatheredBAM, err := gatk4gatherbamfiles.Add(parent, []gobble.Handle{state.recalParts}, bamOptions)
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
		parent := pipeline.AddModule(state.sample.Name + "_alignment_qc")
		statsOptions := config.SamtoolsStats
		statsOptions.OutDir = config.Results.Join("qc", "alignment", state.sample.Name)
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
	for i := range states {
		state := &states[i]
		options := config.HaplotypeCaller
		options.IntervalDir = intervalDirectory(config.Reference.Intervals)
		options.OutDir = gobble.Dir("work/" + state.sample.Name + "/haplotypecaller")
		called, err := gatk4haplotypecaller.Add(haplotypeScatter.AddModule(state.sample.Name), state.recalBAM, state.recalBAI, fasta, fai, dict, dbsnp.vcf, dbsnp.tbi, intervals, options)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		state.gvcfParts, state.gvcfIndexes = called.GVCF, called.TBI
	}

	for i := range states {
		state := &states[i]
		parent := pipeline.Gather(state.sample.Name + "_gvcf_gather").AddModule(state.sample.Name)
		options := config.MergeGVCFs
		options.InputPaths = intervalOutputs(config.Reference.Intervals, gobble.Dir("work/"+state.sample.Name+"/haplotypecaller"), ".g.vcf.gz")
		options.OutDir = config.Results.Join("samples", state.sample.Name, "gvcf")
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
	databases := make([]gatk4genotypegvcfs.Database, 0, len(config.Reference.Intervals))
	for _, member := range config.Reference.Intervals {
		options := config.GenomicsDBImport
		options.OutDir = gobble.Dir("work/joint/genomicsdb/" + member.Name)
		database, err := gatk4genomicsdbimport.Add(pipeline.AddModule("joint_database_"+member.Name), variants, intervalHandles[member.Name], options)
		if recordModuleError(pipeline, err) {
			return pipeline
		}
		databases = append(databases, gatk4genotypegvcfs.Database{Interval: member.Name, Tree: database.Database})
	}

	jointScatter := pipeline.Scatter("joint_intervals").From(intervals)
	jointParent := jointScatter.AddModule("joint")
	genotypeOptions := config.GenotypeGVCFs
	genotypeOptions.IntervalDir = intervalDirectory(config.Reference.Intervals)
	genotypeOptions.OutDir = gobble.Dir("work/joint/genotype")
	genotyped, err := gatk4genotypegvcfs.Add(jointParent, databases, intervals, fasta, fai, dict, dbsnp.vcf, dbsnp.tbi, genotypeOptions)
	if recordModuleError(pipeline, err) {
		return pipeline
	}
	sortOptions := config.BCFToolsSort
	sortOptions.IntervalDir = intervalDirectory(config.Reference.Intervals)
	sortOptions.InputDir = genotypeOptions.OutDir
	sortOptions.OutDir = gobble.Dir("work/joint/sorted")
	sorted, err := bcftoolssort.Add(jointParent, genotyped.VCF, genotyped.TBI, intervals, sortOptions)
	if recordModuleError(pipeline, err) {
		return pipeline
	}

	jointParentGather := pipeline.Gather("joint_gather").AddModule("joint")
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
	callStats, err := bcftoolsstats.Add(pipeline.AddModule("callset_qc"), joint.VCF, joint.TBI, fasta, statsOptions)
	if recordModuleError(pipeline, err) {
		return pipeline
	}
	reports = append(reports, callStats.Stats)
	multiQCOptions := config.MultiQC
	multiQCOptions.OutDir = config.Results.Join("multiqc")
	if _, err = multiqc.Add(pipeline, reports, multiQCOptions); recordModuleError(pipeline, err) {
		return pipeline
	}
	return pipeline
}

func readGroup(sample Sample, lane Lane) string {
	return "@RG\\tID:" + sample.Name + "." + lane.ID + "\\tSM:" + sample.Name + "\\tLB:" + sample.Name + "\\tPL:ILLUMINA\\tPU:" + lane.ID
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
		if !identityPattern.MatchString(sample.Patient) || !identityPattern.MatchString(sample.Name) || seenSamples[sample.Name] {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: sample.Name, Message: "WGS patient or sample identity is invalid or duplicated"})
		}
		seenSamples[sample.Name] = true
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
