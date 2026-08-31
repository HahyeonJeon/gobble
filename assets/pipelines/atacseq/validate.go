package atacseq

import (
	"strconv"
	"strings"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

func validateBuild(samples []Sample, config Config) []gobble.Defect {
	var defects []gobble.Defect
	if countReplicates(samples) < 2 {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: "samplesheet", Message: "ATAC consensus requires at least two biological replicate members"})
	}
	seenSamples := make(map[string]bool, len(samples))
	replicateKeys := make(map[string]bool)
	for _, sample := range samples {
		if !identityPattern.MatchString(sample.Name) || seenSamples[sample.Name] || len(sample.Replicates) == 0 {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: sample.Name, Message: "ATAC sample identity is invalid, duplicated, or has no replicates"})
		}
		seenSamples[sample.Name] = true
		var samplePaired *bool
		var aggregateControlSample string
		var aggregateHasControl *bool
		for i, replicate := range sample.Replicates {
			key := stateKey(sample.Name, replicate.Number)
			if replicate.Number != i+1 || replicateKeys[key] || len(replicate.Runs) == 0 {
				defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: sample.Name, Message: "ATAC replicates must start at 1 without gaps and contain technical runs"})
			}
			replicateKeys[key] = true
			if len(replicate.Runs) == 0 {
				continue
			}
			paired := replicate.Runs[0].Fastq2 != ""
			if samplePaired == nil {
				samplePaired = new(bool)
				*samplePaired = paired
			} else if len(sample.Replicates) > 1 && *samplePaired != paired {
				defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: sample.Name, Message: "biological-replicate aggregation requires one read mode per sample"})
			}
			seenRuns := make(map[string]bool, len(replicate.Runs))
			for runIndex, run := range replicate.Runs {
				if !identityPattern.MatchString(run.ID) || seenRuns[run.ID] || run.ID != "run_"+leftPad3(runIndex+1) || !validWorkspacePath(run.Fastq1) || run.Fastq2 != "" && !validWorkspacePath(run.Fastq2) || (run.Fastq2 != "") != paired {
					defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: sample.Name, Message: "ATAC technical-run identity, order, read mode, or path is invalid", Paths: []string{run.Fastq1, run.Fastq2}})
				}
				seenRuns[run.ID] = true
			}
			hasControl := replicate.Control != nil
			if aggregateHasControl == nil {
				aggregateHasControl = new(bool)
				*aggregateHasControl = hasControl
				if hasControl {
					aggregateControlSample = replicate.Control.Sample
				}
			} else if len(sample.Replicates) > 1 && (*aggregateHasControl != hasControl || hasControl && replicate.Control.Sample != aggregateControlSample) {
				defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: sample.Name, Message: "replicate aggregation requires consistently absent controls or one explicit control sample"})
			}
		}
	}
	for _, sample := range samples {
		for _, replicate := range sample.Replicates {
			if replicate.Control != nil && !replicateKeys[stateKey(replicate.Control.Sample, replicate.Control.Replicate)] {
				defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidSampleSheet, Unit: sample.Name, Message: "ATAC control link does not resolve to an existing sample replicate"})
			}
		}
	}
	for _, value := range []struct {
		unit string
		spec gobble.PathSpec
	}{{"reference.fasta", config.Reference.FASTA}, {"reference.annotation", config.Reference.Annotation}} {
		if rendered, err := value.spec.Render(); err != nil || !validWorkspacePath(rendered) {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidPath, Unit: value.unit, Message: "ATAC reference path must be workspace-relative", Paths: []string{rendered}})
		}
	}
	if !pathSpecUnset(config.Reference.Blacklist) {
		if rendered, err := config.Reference.Blacklist.Render(); err != nil || !validWorkspacePath(rendered) {
			defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidPath, Unit: "reference.blacklist", Message: "ATAC blacklist path must be workspace-relative", Paths: []string{rendered}})
		}
	}
	if config.Filters.RemoveBlacklist && pathSpecUnset(config.Reference.Blacklist) {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "reference.blacklist", Message: "blacklist filtering requires an explicit blacklist"})
	}
	if !pathSpecUnset(config.Reference.BWAIndex.Prefix) || config.Reference.BWAIndex.Members != nil {
		if defect := readyBWAIndexDefect(config.Reference.BWAIndex); defect != nil {
			defects = append(defects, *defect)
		}
	}
	if config.Reference.MitoName == "" || config.Reference.Organism == "" || config.Reference.ReadLength < 1 || strings.TrimSpace(config.Reference.EffectiveGenomeSize) == "" {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "reference", Message: "ATAC mitochondrial name, organism, positive read length, and effective genome size are required"})
	}
	if config.Results.IsZero() || !validWorkspacePath(config.Results.String()+"/result") {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidPath, Unit: "results", Message: "ATAC results directory must be workspace-relative"})
	}
	if config.PeakMode != PeakBroad && config.PeakMode != PeakNarrow {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "peak_mode", Message: "ATAC peak mode must be broad or narrow"})
	}
	if config.Filters.MinimumMAPQ < 0 {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "filters", Message: "ATAC minimum MAPQ must not be negative"})
	}
	if config.ConsensusMinimum < 1 || config.ConsensusMinimum > countReplicates(samples) {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "consensus", Message: "ATAC consensus threshold is outside declared replicate membership"})
	}
	publication := config.Publication
	if !publication.FilteredAlignments || !publication.Tracks || !publication.ReplicatePeaks || !publication.ConsensusMatrix || !publication.QC || !publication.Reports || !publication.IGVSession {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "publication", Message: "ATAC required alignment, track, peak, matrix, QC, report, and IGV outputs cannot be disabled"})
	}
	if unit, flag := protectedExtra(config); flag != "" {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: unit, Message: "ATAC ExtraArgs contains protected option " + flag})
	}
	if len(config.PlotMACS2QC.ExtraArgs) != 0 {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "plot_macs2_qc", Message: "ATAC MACS2 QC plot operands are typed and ExtraArgs are unsupported"})
	}
	if len(config.PlotHOMERAnnotatePeaks.ExtraArgs) != 0 {
		defects = append(defects, gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "plot_homer_annotatepeaks", Message: "ATAC HOMER annotation-QC operands are typed and ExtraArgs are unsupported"})
	}
	return defects
}

func protectedExtra(config Config) (string, string) {
	sets := []struct {
		unit  string
		args  []string
		flags []string
	}{
		{unit: "bwa_index", args: config.BWAIndex.ExtraArgs, flags: []string{"--aligner", "--bowtie2", "--chromap", "--star"}},
		{unit: "bwa_mem", args: config.BWAMem.ExtraArgs, flags: []string{"--aligner", "--bowtie2", "--chromap", "--star", "-R", "-t", "-o"}},
		{unit: "bedtools_genomecov", args: config.GenomeCoverage.ExtraArgs, flags: []string{"-ibam", "-bg", "-scale", "-pc"}},
		{unit: "macs2_callpeak", args: config.MACS2.ExtraArgs, flags: []string{"--format", "-f", "--name", "-n", "--treatment", "-t", "--control", "-c", "--outdir", "--broad", "--call-summits"}},
		{unit: "featurecounts_atac", args: config.FeatureCounts.ExtraArgs, flags: []string{"-F", "-a", "-o", "-p", "--countReadPairs"}},
		{unit: "deseq2_qc", args: config.DESeq2QC.ExtraArgs, flags: []string{"--design", "--contrast"}},
	}
	for _, set := range sets {
		if flag := modules.MatchProtectedExtraArg(set.args, set.flags); flag != "" {
			return set.unit, flag
		}
	}
	return "", ""
}

func readyBWAIndexDefect(index ReadyBWAIndex) *gobble.Defect {
	want := []string{"amb", "ann", "bwt", "pac", "sa"}
	if len(index.Members) != len(want) {
		return &gobble.Defect{Code: gobble.DefectInvalidValue, Unit: "reference.bwa_index", Message: "ready BWA index must contain every fixed sidecar"}
	}
	prefix, err := index.Prefix.Render()
	if err != nil || !validWorkspacePath(prefix) {
		return &gobble.Defect{Code: gobble.DefectInvalidPath, Unit: "reference.bwa_index", Message: "ready BWA index prefix must be workspace-relative", Paths: []string{prefix}}
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
	clone(&config.SamtoolsFAIDX.Options)
	clone(&config.ReferenceIntervals.Options)
	clone(&config.FastQC.Options)
	clone(&config.TrimGalore.Options)
	clone(&config.BWAMem.Options)
	clone(&config.SamtoolsSort.Options)
	clone(&config.SamtoolsIndex.Options)
	clone(&config.SamtoolsStats.Options)
	clone(&config.SamtoolsFlagstat.Options)
	clone(&config.SamtoolsIdxstats.Options)
	clone(&config.MergeRuns.Options)
	clone(&config.MergeReplicates.Options)
	clone(&config.MarkDuplicates.Options)
	clone(&config.FilterBAM.Options)
	clone(&config.BlacklistFilter.Options)
	clone(&config.CollectMultipleMetrics.Options)
	clone(&config.GenomeCoverage.Options)
	clone(&config.ScaleCoverage.Options)
	clone(&config.BedGraphToBigWig.Options)
	clone(&config.ComputeMatrix.Options)
	clone(&config.PlotProfile.Options)
	clone(&config.PlotFingerprint.Options)
	clone(&config.MACS2.Options)
	clone(&config.HOMER.Options)
	clone(&config.PlotMACS2QC.Options)
	clone(&config.PlotHOMERAnnotatePeaks.Options)
	clone(&config.PeakCount.Options)
	clone(&config.PeakIntersect.Options)
	clone(&config.ReadCount.Options)
	clone(&config.FRiP.Options)
	clone(&config.Consensus.Options)
	clone(&config.FeatureCounts.Options)
	clone(&config.FeatureCountsMerge.Options)
	clone(&config.DESeq2QC.Options)
	clone(&config.Ataqv.Options)
	clone(&config.Mkarv.Options)
	clone(&config.IGV.Options)
	clone(&config.MultiQC.Options)
	config.Reference.FASTA = cloneSpec(config.Reference.FASTA)
	config.Reference.Annotation = cloneSpec(config.Reference.Annotation)
	config.Reference.Blacklist = cloneSpec(config.Reference.Blacklist)
	config.Reference.BWAIndex.Prefix = cloneSpec(config.Reference.BWAIndex.Prefix)
	config.Reference.BWAIndex.Members = cloneGroup(config.Reference.BWAIndex.Members)
	config.BWAMem.IndexPrefix = cloneSpec(config.BWAMem.IndexPrefix)
	return config
}

func cloneGroup(group gobble.Group) gobble.Group {
	if group == nil {
		return nil
	}
	out := make(gobble.Group, len(group))
	for i, member := range group {
		out[i] = member
		out[i].Spec = cloneSpec(member.Spec)
	}
	return out
}

func cloneSpec(spec gobble.PathSpec) gobble.PathSpec {
	spec.Suffixes = append([]string(nil), spec.Suffixes...)
	return spec
}

func pathSpecUnset(spec gobble.PathSpec) bool {
	return spec.Dir.IsZero() && spec.Prefix == "" && spec.Base == "" && len(spec.Suffixes) == 0 && spec.Ext == ""
}

func controlLabel(control *ControlRef) string {
	if control == nil {
		return ""
	}
	return control.Sample + ".R" + strconv.Itoa(control.Replicate)
}
