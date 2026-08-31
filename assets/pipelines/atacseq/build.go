package atacseq

import (
	"strconv"
	"strings"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	atacconsensuspeaks "github.com/HahyeonJeon/gobble/assets/modules/atac-consensus-peaks"
	atacfripscore "github.com/HahyeonJeon/gobble/assets/modules/atac-frip-score"
	atacreferenceintervals "github.com/HahyeonJeon/gobble/assets/modules/atac-reference-intervals"
	"github.com/HahyeonJeon/gobble/assets/modules/ataqv"
	ataqvmkarv "github.com/HahyeonJeon/gobble/assets/modules/ataqv-mkarv"
	bedgraphscale "github.com/HahyeonJeon/gobble/assets/modules/bedgraph-scale"
	bedtoolsgenomecov "github.com/HahyeonJeon/gobble/assets/modules/bedtools-genomecov"
	bedtoolsintersect "github.com/HahyeonJeon/gobble/assets/modules/bedtools-intersect"
	bwaindex "github.com/HahyeonJeon/gobble/assets/modules/bwa-index"
	bwamem "github.com/HahyeonJeon/gobble/assets/modules/bwa-mem"
	deeptoolscomputematrix "github.com/HahyeonJeon/gobble/assets/modules/deeptools-compute-matrix"
	deeptoolsplotfingerprint "github.com/HahyeonJeon/gobble/assets/modules/deeptools-plot-fingerprint"
	deeptoolsplotprofile "github.com/HahyeonJeon/gobble/assets/modules/deeptools-plot-profile"
	deseq2qc "github.com/HahyeonJeon/gobble/assets/modules/deseq2-qc"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	"github.com/HahyeonJeon/gobble/assets/modules/featurecounts"
	featurecountsmergematrices "github.com/HahyeonJeon/gobble/assets/modules/featurecounts-merge-matrices"
	homerannotatepeaks "github.com/HahyeonJeon/gobble/assets/modules/homer-annotate-peaks"
	igvsession "github.com/HahyeonJeon/gobble/assets/modules/igv-session"
	macs2callpeak "github.com/HahyeonJeon/gobble/assets/modules/macs2-callpeak"
	"github.com/HahyeonJeon/gobble/assets/modules/multiqc"
	picardcollectmultiplemetrics "github.com/HahyeonJeon/gobble/assets/modules/picard-collect-multiple-metrics"
	picardmarkduplicates "github.com/HahyeonJeon/gobble/assets/modules/picard-markduplicates"
	picardmergesamfiles "github.com/HahyeonJeon/gobble/assets/modules/picard-merge-sam-files"
	samtoolsfaidx "github.com/HahyeonJeon/gobble/assets/modules/samtools-faidx"
	samtoolsflagstat "github.com/HahyeonJeon/gobble/assets/modules/samtools-flagstat"
	samtoolsidxstats "github.com/HahyeonJeon/gobble/assets/modules/samtools-idxstats"
	samtoolsindex "github.com/HahyeonJeon/gobble/assets/modules/samtools-index"
	samtoolssort "github.com/HahyeonJeon/gobble/assets/modules/samtools-sort"
	samtoolsstats "github.com/HahyeonJeon/gobble/assets/modules/samtools-stats"
	samtoolsview "github.com/HahyeonJeon/gobble/assets/modules/samtools-view"
	samtoolsviewcount "github.com/HahyeonJeon/gobble/assets/modules/samtools-view-count"
	trimgalore "github.com/HahyeonJeon/gobble/assets/modules/trim-galore"
	ucscbedgraphtobigwig "github.com/HahyeonJeon/gobble/assets/modules/ucsc-bedgraphtobigwig"
	wclines "github.com/HahyeonJeon/gobble/assets/modules/wc-lines"
)

type referenceHandles struct {
	fasta     gobble.Handle
	fai       gobble.Handle
	gtf       gobble.Handle
	blacklist gobble.Handle
	index     gobble.Handle
	prefix    gobble.PathSpec
	intervals atacreferenceintervals.Ports
}

type replicateState struct {
	sample          string
	replicate       int
	paired          bool
	control         *ControlRef
	controlIdentity string
	module          *gobble.Module
	resultDir       gobble.Directory
	bam             gobble.Handle
	bai             gobble.Handle
	track           gobble.Handle
	peaks           gobble.Handle
	ataqv           gobble.Handle
}

type alignmentProduct struct {
	bam     gobble.Handle
	bai     gobble.Handle
	track   gobble.Handle
	reports []gobble.Handle
}

type samtoolsReports struct {
	stats    gobble.Handle
	flagstat gobble.Handle
	idxstats gobble.Handle
}

func (r samtoolsReports) handles() []gobble.Handle {
	return []gobble.Handle{r.stats, r.flagstat, r.idxstats}
}

// Build constructs the selected BWA ATAC-seq graph only from supplied values.
// It copies every caller-owned slice, control link, path member, and ExtraArgs
// list. Build performs no filesystem or network access.
func Build(inputSamples []Sample, inputConfig Config) *gobble.Pipeline {
	pipeline := gobble.NewPipeline("atacseq")
	samples := cloneSamples(inputSamples)
	config := cloneConfig(inputConfig)
	if defects := validateBuild(samples, config); len(defects) > 0 {
		pipeline.RecordComposeError(composeError(defects))
		return pipeline
	}

	reference, err := addReference(pipeline, config)
	if recordModuleError(pipeline, err) {
		return pipeline
	}
	var reports []gobble.Handle
	states := make([]*replicateState, 0, countReplicates(samples))
	statesByKey := make(map[string]*replicateState, countReplicates(samples))
	sampleModules := make(map[string]*gobble.Module, len(samples))
	samplesByName := make(map[string]Sample, len(samples))
	for _, sample := range samples {
		samplesByName[sample.Name] = sample
		sampleModule := pipeline.AddModule(sample.Name)
		sampleModules[sample.Name] = sampleModule
		for _, replicate := range sample.Replicates {
			replicateModule := sampleModule.AddModule("replicate_" + strconv.Itoa(replicate.Number))
			paired := replicate.Runs[0].Fastq2 != ""
			workDir := gobble.Dir("work/atacseq").Join(sample.Name, "replicate_"+strconv.Itoa(replicate.Number))
			resultDir := config.Results.Join("samples", sample.Name, "replicate_"+strconv.Itoa(replicate.Number))
			runBAMs := make([]gobble.Handle, 0, len(replicate.Runs))
			for _, run := range replicate.Runs {
				runModule := replicateModule.AddModule(run.ID)
				parent := stateTaskParent(runModule, sample.Name, replicate.Number, run.ID, "", false)
				inputPrefix := inputName(sample.Name, replicate.Number, run.ID)
				read1 := pipeline.AddInput(inputPrefix+"_r1", sheetFileSpec(run.Fastq1))
				var read2 gobble.Handle
				if run.Fastq2 != "" {
					read2 = pipeline.AddInput(inputPrefix+"_r2", sheetFileSpec(run.Fastq2))
				}
				for _, read := range presentReads(read1, read2) {
					options := config.FastQC
					options.OutDir = workDir.Join(run.ID, "raw-fastqc", read.name)
					qcParent := stateTaskParent(runModule.AddModule("raw_fastqc_"+read.name), sample.Name, replicate.Number, run.ID, "", false)
					qc, addErr := fastqc.Add(qcParent, read.handle, options)
					if recordModuleError(pipeline, addErr) {
						return pipeline
					}
					reports = append(reports, qc.HTML, qc.Zip)
				}
				trimOptions := config.TrimGalore
				trimOptions.OutDir = workDir.Join(run.ID, "trim-galore")
				trimOptions.Prefix = sample.Name + "_R" + strconv.Itoa(replicate.Number) + "_" + run.ID
				trimmed, addErr := trimgalore.Add(parent, read1, read2, trimOptions)
				if recordModuleError(pipeline, addErr) {
					return pipeline
				}
				reports = append(reports, trimmed.Report1)
				if !trimmed.Report2.IsZero() {
					reports = append(reports, trimmed.Report2)
				}
				for _, read := range presentReads(trimmed.Read1, trimmed.Read2) {
					options := config.FastQC
					options.OutDir = workDir.Join(run.ID, "post-trim-fastqc", read.name)
					qcParent := stateTaskParent(runModule.AddModule("post_trim_fastqc_"+read.name), sample.Name, replicate.Number, run.ID, "", false)
					qc, addErr := fastqc.Add(qcParent, read.handle, options)
					if recordModuleError(pipeline, addErr) {
						return pipeline
					}
					reports = append(reports, qc.HTML, qc.Zip)
				}
				memOptions := config.BWAMem
				memOptions.IndexPrefix = reference.prefix
				memOptions.ReadGroup = readGroup(sample.Name, replicate.Number, run.ID)
				memOptions.OutDir = workDir.Join(run.ID, "bwa")
				memOptions.Prefix = "aligned"
				aligned, addErr := bwamem.Add(parent, reference.fasta, reference.index, trimmed.Read1, trimmed.Read2, memOptions)
				if recordModuleError(pipeline, addErr) {
					return pipeline
				}
				sortOptions := config.SamtoolsSort
				sortOptions.OutDir = workDir.Join(run.ID, "sorted")
				sortOptions.Prefix = "aligned"
				sorted, addErr := samtoolssort.Add(parent, aligned.SAM, sortOptions)
				if recordModuleError(pipeline, addErr) {
					return pipeline
				}
				indexed, addErr := samtoolsindex.Add(parent, sorted.BAM, config.SamtoolsIndex)
				if recordModuleError(pipeline, addErr) {
					return pipeline
				}
				runBAMs = append(runBAMs, sorted.BAM)
				statsDir := workDir.Join(run.ID, "alignment-qc")
				runReports, addErr := addSamtoolsReports(parent, sorted.BAM, indexed.BAI, statsDir, "library", config)
				if recordModuleError(pipeline, addErr) {
					return pipeline
				}
				reports = append(reports, runReports.handles()...)
			}

			mergeOptions := config.MergeRuns
			mergeOptions.OutDir = workDir.Join("technical-run-merge")
			mergeOptions.Prefix = sample.Name + "_R" + strconv.Itoa(replicate.Number)
			merged, addErr := picardmergesamfiles.Add(stateTaskParent(replicateModule.AddModule("technical_run_merge"), sample.Name, replicate.Number, "", "", false), runBAMs, mergeOptions)
			if recordModuleError(pipeline, addErr) {
				return pipeline
			}
			finished, addErr := finalizeAlignment(stateTaskParent(replicateModule, sample.Name, replicate.Number, "", "", false), merged.BAM, paired, workDir, resultDir, sample.Name+"_R"+strconv.Itoa(replicate.Number), reference, config)
			if recordModuleError(pipeline, addErr) {
				return pipeline
			}
			reports = append(reports, finished.reports...)
			state := &replicateState{sample: sample.Name, replicate: replicate.Number, paired: paired, control: replicate.Control, controlIdentity: controlLabel(replicate.Control), module: replicateModule, resultDir: resultDir, bam: finished.bam, bai: finished.bai, track: finished.track}
			states = append(states, state)
			statesByKey[stateKey(sample.Name, replicate.Number)] = state
		}
	}

	for _, state := range states {
		var control gobble.Handle
		if state.control != nil {
			control = statesByKey[stateKey(state.control.Sample, state.control.Replicate)].bam
		}
		peakReports, addErr := addPeakQC(state, control, reference, config)
		if recordModuleError(pipeline, addErr) {
			return pipeline
		}
		reports = append(reports, peakReports...)
	}

	coverageReports, addErr := addCoverageQC(pipeline.AddModule("coverage_qc"), states, reference, config)
	if recordModuleError(pipeline, addErr) {
		return pipeline
	}
	reports = append(reports, coverageReports...)
	consensusModule := pipeline.AddModule("consensus")
	replicateConsensus, replicateConsensusReports, addErr := addConsensus(consensusModule.AddModule("replicates"), "replicates", states, reference, config)
	if recordModuleError(pipeline, addErr) {
		return pipeline
	}
	reports = append(reports, replicateConsensusReports...)

	aggregates := make([]*replicateState, 0, len(samples))
	aggregatesBySample := make(map[string]*replicateState)
	for _, sample := range samples {
		if len(sample.Replicates) < 2 {
			continue
		}
		members := make([]gobble.Handle, len(sample.Replicates))
		for i, replicate := range sample.Replicates {
			members[i] = statesByKey[stateKey(sample.Name, replicate.Number)].bam
		}
		module := sampleModules[sample.Name].AddModule("aggregate")
		workDir := gobble.Dir("work/atacseq").Join(sample.Name, "aggregate")
		resultDir := config.Results.Join("aggregates", sample.Name)
		options := config.MergeReplicates
		options.OutDir, options.Prefix = workDir.Join("replicate-merge"), sample.Name
		merged, mergeErr := picardmergesamfiles.Add(stateTaskParent(module.AddModule("replicate_merge"), sample.Name, 0, "", "", true), members, options)
		if recordModuleError(pipeline, mergeErr) {
			return pipeline
		}
		paired := sample.Replicates[0].Runs[0].Fastq2 != ""
		finished, finishErr := finalizeAlignment(stateTaskParent(module, sample.Name, 0, "", "", true), merged.BAM, paired, workDir, resultDir, sample.Name, reference, config)
		if recordModuleError(pipeline, finishErr) {
			return pipeline
		}
		reports = append(reports, finished.reports...)
		state := &replicateState{sample: sample.Name, paired: paired, controlIdentity: aggregateControlIdentity(sample), module: module, resultDir: resultDir, bam: finished.bam, bai: finished.bai, track: finished.track}
		aggregates = append(aggregates, state)
		aggregatesBySample[sample.Name] = state
	}
	for _, state := range aggregates {
		control, controlErr := aggregateControlBAM(state, samplesByName[state.sample], samplesByName, statesByKey, aggregatesBySample, config)
		if recordModuleError(pipeline, controlErr) {
			return pipeline
		}
		peakReports, peakErr := addPeakQC(state, control, reference, config)
		if recordModuleError(pipeline, peakErr) {
			return pipeline
		}
		reports = append(reports, peakReports...)
	}

	var aggregateConsensus gobble.Handle
	if len(aggregates) > 1 {
		aggregateConsensus, replicateConsensusReports, addErr = addConsensus(consensusModule.AddModule("aggregates"), "aggregates", aggregates, reference, config)
		if recordModuleError(pipeline, addErr) {
			return pipeline
		}
		reports = append(reports, replicateConsensusReports...)
	}

	allStates := append(append([]*replicateState(nil), states...), aggregates...)
	ataqvReports := make([]gobble.Handle, 0, len(allStates))
	igvResources := make([]gobble.Handle, 0, len(allStates)*2+2)
	for _, state := range allStates {
		ataqvReports = append(ataqvReports, state.ataqv)
		igvResources = append(igvResources, state.track, state.peaks)
	}
	igvResources = append(igvResources, replicateConsensus)
	if !aggregateConsensus.IsZero() {
		igvResources = append(igvResources, aggregateConsensus)
	}
	mkarvOptions := config.Mkarv
	mkarvOptions.OutDir = config.Results.Join("ataqv", "html")
	if _, addErr = ataqvmkarv.Add(pipeline.AddModule("ataqv"), ataqvReports, mkarvOptions); recordModuleError(pipeline, addErr) {
		return pipeline
	}
	igvOptions := config.IGV
	igvOptions.OutDir = config.Results.Join("igv")
	if _, addErr = igvsession.Add(pipeline.AddModule("igv"), reference.fasta, reference.fai, igvResources, igvOptions); recordModuleError(pipeline, addErr) {
		return pipeline
	}
	multiQCOptions := config.MultiQC
	multiQCOptions.OutDir = config.Results.Join("multiqc")
	if _, addErr = multiqc.Add(pipeline, reports, multiQCOptions); recordModuleError(pipeline, addErr) {
		return pipeline
	}
	return pipeline
}

func addReference(pipeline *gobble.Pipeline, config Config) (referenceHandles, error) {
	module := pipeline.AddModule("reference")
	fasta := pipeline.AddInput("reference_fasta", config.Reference.FASTA)
	gtf := pipeline.AddInput("reference_annotation", config.Reference.Annotation)
	faidx, err := samtoolsfaidx.Add(module, fasta, config.SamtoolsFAIDX)
	if err != nil {
		return referenceHandles{}, err
	}
	intervalOptions := config.ReferenceIntervals
	intervalOptions.OutDir = gobble.Dir("work/atacseq/reference")
	intervalOptions.MitoName = config.Reference.MitoName
	intervalOptions.ReadLength = config.Reference.ReadLength
	intervals, err := atacreferenceintervals.Add(module, faidx.FAI, gtf, intervalOptions)
	if err != nil {
		return referenceHandles{}, err
	}
	var index gobble.Handle
	var prefix gobble.PathSpec
	if config.Reference.BWAIndex.Members == nil {
		options := config.BWAIndex
		options.OutDir, options.Prefix = gobble.Dir("work/atacseq/reference/bwa"), "genome"
		prepared, addErr := bwaindex.Add(module, fasta, options)
		if addErr != nil {
			return referenceHandles{}, addErr
		}
		index, prefix = prepared.Index, prepared.Prefix
	} else {
		index = pipeline.AddInputGroup("bwa_index", config.Reference.BWAIndex.Members)
		prefix = config.Reference.BWAIndex.Prefix
	}
	var blacklist gobble.Handle
	if config.Filters.RemoveBlacklist {
		blacklist = pipeline.AddInput("reference_blacklist", config.Reference.Blacklist)
	}
	return referenceHandles{fasta: fasta, fai: faidx.FAI, gtf: gtf, blacklist: blacklist, index: index, prefix: prefix, intervals: intervals}, nil
}

func finalizeAlignment(parent modules.Parent, merged gobble.Handle, paired bool, workDir, resultDir gobble.Directory, prefix string, reference referenceHandles, config Config) (alignmentProduct, error) {
	markOptions := config.MarkDuplicates
	markOptions.OutDir, markOptions.Prefix = workDir.Join("markduplicates"), prefix+".marked"
	marked, err := picardmarkduplicates.Add(parent, merged, markOptions)
	if err != nil {
		return alignmentProduct{}, err
	}
	filterOptions := config.FilterBAM
	filterOptions.OutDir = resultDir.Join("alignment")
	if config.Filters.RemoveBlacklist {
		filterOptions.OutDir = workDir.Join("filter-before-blacklist")
	}
	filterOptions.Prefix = prefix + ".filtered"
	filterOptions.MinimumMAPQ = config.Filters.MinimumMAPQ
	filterOptions.Paired = paired
	filterOptions.RemoveOrphan = config.Filters.RemoveOrphans
	filterOptions.RemoveMito = config.Filters.RemoveMito
	filterOptions.MitoName = config.Reference.MitoName
	filtered, err := samtoolsview.Add(parent, marked.BAM, filterOptions)
	if err != nil {
		return alignmentProduct{}, err
	}
	finalBAM := filtered.BAM
	if config.Filters.RemoveBlacklist {
		blacklistOptions := config.BlacklistFilter
		blacklistOptions.OutDir = resultDir.Join("alignment")
		blacklistOptions.Prefix = prefix + ".filtered"
		blacklistOptions.Invert = true
		blacklisted, addErr := bedtoolsintersect.Add(parent, filtered.BAM, reference.blacklist, blacklistOptions)
		if addErr != nil {
			return alignmentProduct{}, addErr
		}
		finalBAM = blacklisted.BAM
	}
	indexed, err := samtoolsindex.Add(parent, finalBAM, config.SamtoolsIndex)
	if err != nil {
		return alignmentProduct{}, err
	}
	qcDir := resultDir.Join("qc", "alignment")
	reports, err := addSamtoolsReports(parent, finalBAM, indexed.BAI, qcDir, prefix, config)
	if err != nil {
		return alignmentProduct{}, err
	}
	metricsOptions := config.CollectMultipleMetrics
	metricsOptions.OutDir, metricsOptions.Prefix = qcDir.Join("picard"), prefix
	metrics, err := picardcollectmultiplemetrics.Add(parent, finalBAM, indexed.BAI, reference.fasta, reference.fai, metricsOptions)
	if err != nil {
		return alignmentProduct{}, err
	}
	coverageOptions := config.GenomeCoverage
	coverageOptions.OutDir, coverageOptions.Prefix = workDir.Join("coverage"), prefix
	if paired {
		coverageOptions.ExtraArgs = append(coverageOptions.ExtraArgs, "-pc")
	}
	coverage, err := bedtoolsgenomecov.Add(parent, finalBAM, coverageOptions)
	if err != nil {
		return alignmentProduct{}, err
	}
	scaleOptions := config.ScaleCoverage
	scaleOptions.OutDir, scaleOptions.Prefix = workDir.Join("coverage"), prefix+".normalized"
	scaled, err := bedgraphscale.Add(parent, coverage.BedGraph, reports.flagstat, scaleOptions)
	if err != nil {
		return alignmentProduct{}, err
	}
	trackOptions := config.BedGraphToBigWig
	trackOptions.OutDir, trackOptions.Prefix = resultDir.Join("tracks"), prefix
	track, err := ucscbedgraphtobigwig.Add(parent, scaled.BedGraph, reference.intervals.ChromSizes, trackOptions)
	if err != nil {
		return alignmentProduct{}, err
	}
	reportHandles := append([]gobble.Handle{marked.Metrics, metrics.Metrics}, reports.handles()...)
	return alignmentProduct{bam: finalBAM, bai: indexed.BAI, track: track.BigWig, reports: reportHandles}, nil
}

func addSamtoolsReports(parent modules.Parent, bam, bai gobble.Handle, outDir gobble.Directory, prefix string, config Config) (samtoolsReports, error) {
	statsOptions := config.SamtoolsStats
	statsOptions.OutDir, statsOptions.Prefix = outDir, prefix
	stats, err := samtoolsstats.Add(parent, bam, statsOptions)
	if err != nil {
		return samtoolsReports{}, err
	}
	flagOptions := config.SamtoolsFlagstat
	flagOptions.OutDir, flagOptions.Prefix = outDir, prefix
	flagstat, err := samtoolsflagstat.Add(parent, bam, bai, flagOptions)
	if err != nil {
		return samtoolsReports{}, err
	}
	idxOptions := config.SamtoolsIdxstats
	idxOptions.OutDir, idxOptions.Prefix = outDir, prefix
	idxstats, err := samtoolsidxstats.Add(parent, bam, bai, idxOptions)
	if err != nil {
		return samtoolsReports{}, err
	}
	return samtoolsReports{stats: stats.Stats, flagstat: flagstat.Report, idxstats: idxstats.Report}, nil
}

func addPeakQC(state *replicateState, control gobble.Handle, reference referenceHandles, config Config) ([]gobble.Handle, error) {
	peakModule := state.module.AddModule("peaks")
	parent := stateTaskParent(peakModule, state.sample, state.replicate, "", state.controlIdentity, state.replicate == 0)
	prefix := state.sample
	if state.replicate > 0 {
		prefix += "_R" + strconv.Itoa(state.replicate)
	}
	macsOptions := config.MACS2
	macsOptions.OutDir, macsOptions.Prefix = state.resultDir.Join("peaks"), prefix
	macsOptions.Paired = state.paired
	macsOptions.EffectiveGenomeSize = config.Reference.EffectiveGenomeSize
	if config.PeakMode == PeakBroad {
		macsOptions.Mode = macs2callpeak.Broad
	} else {
		macsOptions.Mode = macs2callpeak.Narrow
	}
	peaks, err := macs2callpeak.Add(parent, state.bam, control, macsOptions)
	if err != nil {
		return nil, err
	}
	homerOptions := config.HOMER
	homerOptions.OutDir, homerOptions.Prefix = state.resultDir.Join("peak-annotation"), prefix
	annotation, err := homerannotatepeaks.Add(parent, peaks.Peaks, reference.fasta, reference.gtf, homerOptions)
	if err != nil {
		return nil, err
	}
	countOptions := config.PeakCount
	countOptions.OutDir, countOptions.Prefix = state.resultDir.Join("peak-qc"), prefix
	peakCount, err := wclines.Add(parent, peaks.Peaks, countOptions)
	if err != nil {
		return nil, err
	}
	intersectOptions := config.PeakIntersect
	intersectOptions.OutDir, intersectOptions.Prefix, intersectOptions.Invert = state.resultDir.Join("peak-qc"), prefix+".in_peaks", false
	overlap, err := bedtoolsintersect.Add(parent, state.bam, peaks.Peaks, intersectOptions)
	if err != nil {
		return nil, err
	}
	totalOptions := config.ReadCount
	totalOptions.OutDir, totalOptions.Prefix = state.resultDir.Join("peak-qc"), prefix+".total"
	totalParent := stateTaskParent(peakModule.AddModule("total_count"), state.sample, state.replicate, "", state.controlIdentity, state.replicate == 0)
	total, err := samtoolsviewcount.Add(totalParent, state.bam, totalOptions)
	if err != nil {
		return nil, err
	}
	inPeakOptions := config.ReadCount
	inPeakOptions.OutDir, inPeakOptions.Prefix = state.resultDir.Join("peak-qc"), prefix+".in_peaks"
	inPeakParent := stateTaskParent(peakModule.AddModule("in_peak_count"), state.sample, state.replicate, "", state.controlIdentity, state.replicate == 0)
	inPeaks, err := samtoolsviewcount.Add(inPeakParent, overlap.BAM, inPeakOptions)
	if err != nil {
		return nil, err
	}
	fripOptions := config.FRiP
	fripOptions.OutDir, fripOptions.Prefix = state.resultDir.Join("peak-qc"), prefix
	frip, err := atacfripscore.Add(parent, total.Count, inPeaks.Count, fripOptions)
	if err != nil {
		return nil, err
	}
	ataqvOptions := config.Ataqv
	ataqvOptions.OutDir, ataqvOptions.Prefix = state.resultDir.Join("ataqv"), prefix
	ataqvOptions.Organism, ataqvOptions.MitoName = config.Reference.Organism, config.Reference.MitoName
	ataqvReport, err := ataqv.Add(parent, state.bam, state.bai, peaks.Peaks, reference.intervals.TSS, reference.intervals.Autosomes, ataqvOptions)
	if err != nil {
		return nil, err
	}
	state.peaks, state.ataqv = peaks.Peaks, ataqvReport.JSON
	return []gobble.Handle{peaks.XLS, annotation.Annotation, peakCount.Count, frip.Report, ataqvReport.JSON}, nil
}

func addCoverageQC(parent *gobble.Module, states []*replicateState, reference referenceHandles, config Config) ([]gobble.Handle, error) {
	tracks := make([]gobble.Handle, len(states))
	bams := make([]gobble.Handle, len(states))
	bais := make([]gobble.Handle, len(states))
	for i, state := range states {
		tracks[i], bams[i], bais[i] = state.track, state.bam, state.bai
	}
	matrixOptions := config.ComputeMatrix
	matrixOptions.OutDir, matrixOptions.Prefix = config.Results.Join("qc", "coverage"), "replicate_coverage"
	matrix, err := deeptoolscomputematrix.Add(parent, tracks, []gobble.Handle{reference.intervals.Genes, reference.intervals.TSS}, matrixOptions)
	if err != nil {
		return nil, err
	}
	profileOptions := config.PlotProfile
	profileOptions.OutDir, profileOptions.Prefix = config.Results.Join("qc", "coverage"), "replicate_coverage"
	profile, err := deeptoolsplotprofile.Add(parent, matrix.Matrix, profileOptions)
	if err != nil {
		return nil, err
	}
	fingerprintOptions := config.PlotFingerprint
	fingerprintOptions.OutDir, fingerprintOptions.Prefix = config.Results.Join("qc", "coverage"), "replicate_fingerprint"
	fingerprint, err := deeptoolsplotfingerprint.Add(parent, bams, bais, fingerprintOptions)
	if err != nil {
		return nil, err
	}
	return []gobble.Handle{matrix.Table, profile.PDF, profile.Table, fingerprint.PDF, fingerprint.Raw, fingerprint.Metrics}, nil
}

func addConsensus(parent *gobble.Module, label string, states []*replicateState, reference referenceHandles, config Config) (gobble.Handle, []gobble.Handle, error) {
	peaks := make([]gobble.Handle, len(states))
	bams := make([]gobble.Handle, len(states))
	for i, state := range states {
		peaks[i], bams[i] = state.peaks, state.bam
	}
	consensusOptions := config.Consensus
	consensusOptions.OutDir, consensusOptions.Prefix, consensusOptions.Minimum = config.Results.Join("consensus", label), "consensus", config.ConsensusMinimum
	consensus, err := atacconsensuspeaks.Add(parent, peaks, consensusOptions)
	if err != nil {
		return gobble.Handle{}, nil, err
	}
	homerOptions := config.HOMER
	homerOptions.OutDir, homerOptions.Prefix = config.Results.Join("consensus", label, "annotation"), "consensus"
	annotation, err := homerannotatepeaks.Add(parent, consensus.BED, reference.fasta, reference.gtf, homerOptions)
	if err != nil {
		return gobble.Handle{}, nil, err
	}
	pairedBAMs := make([]gobble.Handle, 0, len(bams))
	singleBAMs := make([]gobble.Handle, 0, len(bams))
	for i, state := range states {
		if state.paired {
			pairedBAMs = append(pairedBAMs, bams[i])
		} else {
			singleBAMs = append(singleBAMs, bams[i])
		}
	}
	var countMatrices []gobble.Handle
	var countSummaries []gobble.Handle
	for _, group := range []struct {
		name   string
		paired bool
		bams   []gobble.Handle
	}{{name: "paired_end", paired: true, bams: pairedBAMs}, {name: "single_end", bams: singleBAMs}} {
		if len(group.bams) == 0 {
			continue
		}
		featureOptions := config.FeatureCounts
		featureOptions.OutDir, featureOptions.Prefix, featureOptions.Paired = config.Results.Join("consensus", label, "featurecounts", group.name), "consensus_"+group.name, group.paired
		modeCounts, countErr := featurecounts.AddATAC(parent.AddModule(group.name), group.bams, consensus.SAF, featureOptions)
		if countErr != nil {
			return gobble.Handle{}, nil, countErr
		}
		countMatrices = append(countMatrices, modeCounts.Counts)
		countSummaries = append(countSummaries, modeCounts.Summary)
	}
	mergeOptions := config.FeatureCountsMerge
	mergeOptions.OutDir, mergeOptions.Prefix = config.Results.Join("consensus", label, "featurecounts"), "consensus"
	counts, err := featurecountsmergematrices.Add(parent, countMatrices, mergeOptions)
	if err != nil {
		return gobble.Handle{}, nil, err
	}
	deseqOptions := config.DESeq2QC
	deseqOptions.OutDir = config.Results.Join("consensus", label, "deseq2-qc")
	qc, err := deseq2qc.AddATAC(parent, counts.Counts, deseqOptions)
	if err != nil {
		return gobble.Handle{}, nil, err
	}
	reports := append([]gobble.Handle{consensus.Presence, annotation.Annotation}, countSummaries...)
	reports = append(reports, qc.PCA, qc.Distance, qc.Matrix, qc.SizeFactors)
	return consensus.BED, reports, nil
}

type namedHandle struct {
	name   string
	handle gobble.Handle
}

func presentReads(read1, read2 gobble.Handle) []namedHandle {
	reads := []namedHandle{{name: "r1", handle: read1}}
	if !read2.IsZero() {
		reads = append(reads, namedHandle{name: "r2", handle: read2})
	}
	return reads
}

func aggregateControlIdentity(sample Sample) string {
	controls := make([]string, 0, len(sample.Replicates))
	for _, replicate := range sample.Replicates {
		if replicate.Control != nil {
			controls = append(controls, controlLabel(replicate.Control))
		}
	}
	return strings.Join(controls, ",")
}

func aggregateControlBAM(state *replicateState, sample Sample, samplesByName map[string]Sample, statesByKey map[string]*replicateState, aggregatesBySample map[string]*replicateState, config Config) (gobble.Handle, error) {
	unique := make([]ControlRef, 0, len(sample.Replicates))
	seen := make(map[string]bool)
	for _, replicate := range sample.Replicates {
		if replicate.Control == nil {
			continue
		}
		key := stateKey(replicate.Control.Sample, replicate.Control.Replicate)
		if !seen[key] {
			unique = append(unique, *replicate.Control)
			seen[key] = true
		}
	}
	if len(unique) == 0 {
		return gobble.Handle{}, nil
	}
	target := samplesByName[unique[0].Sample]
	if aggregate := aggregatesBySample[unique[0].Sample]; aggregate != nil && len(unique) == len(target.Replicates) {
		complete := true
		for _, replicate := range target.Replicates {
			complete = complete && seen[stateKey(target.Name, replicate.Number)]
		}
		if complete {
			return aggregate.bam, nil
		}
	}
	handles := make([]gobble.Handle, len(unique))
	for i, control := range unique {
		handles[i] = statesByKey[stateKey(control.Sample, control.Replicate)].bam
	}
	if len(handles) == 1 {
		return handles[0], nil
	}
	options := config.MergeReplicates
	options.OutDir = gobble.Dir("work/atacseq").Join(state.sample, "aggregate", "control-merge")
	options.Prefix = state.sample + ".control"
	parent := stateTaskParent(state.module.AddModule("control_merge"), state.sample, 0, "", state.controlIdentity, true)
	merged, err := picardmergesamfiles.Add(parent, handles, options)
	if err != nil {
		return gobble.Handle{}, err
	}
	return merged.BAM, nil
}

func readGroup(sample string, replicate int, run string) string {
	identity := sample + ".R" + strconv.Itoa(replicate)
	return "@RG\\tID:" + identity + "." + run + "\\tSM:" + identity + "\\tLB:" + identity + "\\tPL:ILLUMINA\\tPU:" + identity + "." + run
}

func stateKey(sample string, replicate int) string { return sample + "#" + strconv.Itoa(replicate) }

func inputName(sample string, replicate int, run string) string {
	return "s" + strconv.Itoa(len(sample)) + "_" + sample + "_r" + strconv.Itoa(replicate) + "_t" + strconv.Itoa(len(run)) + "_" + run
}

type parameterizedParent struct {
	parent modules.Parent
	params []gobble.Param
}

func (p parameterizedParent) AddTask(spec gobble.TaskSpec) *gobble.Task {
	spec.Params = append(append([]gobble.Param(nil), spec.Params...), p.params...)
	return p.parent.AddTask(spec)
}

func stateTaskParent(parent modules.Parent, sample string, replicate int, run, control string, aggregate bool) modules.Parent {
	params := []gobble.Param{{Name: "sample", Value: sample}}
	if aggregate {
		params = append(params, gobble.Param{Name: "level", Value: "aggregate"})
	} else {
		params = append(params, gobble.Param{Name: "replicate", Value: strconv.Itoa(replicate)})
	}
	if run != "" {
		params = append(params, gobble.Param{Name: "technical_run", Value: run})
	}
	if control != "" {
		params = append(params, gobble.Param{Name: "control", Value: control})
	}
	return parameterizedParent{parent: parent, params: params}
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
