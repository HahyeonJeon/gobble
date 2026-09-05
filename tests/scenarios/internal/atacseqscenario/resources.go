package atacseqscenario

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	"github.com/HahyeonJeon/gobble/assets/pipelines/atacseq"
)

// fixtureConfig sizes the deterministic command doubles for small test hosts.
// Production defaults and plan assertions retain their real analysis budgets.
func fixtureConfig(config atacseq.Config) atacseq.Config {
	for _, option := range []*modules.Options{
		&config.BWAIndex.Options,
		&config.SamtoolsFAIDX.Options,
		&config.ReferenceIntervals.Options,
		&config.FastQC.Options,
		&config.TrimGalore.Options,
		&config.BWAMem.Options,
		&config.SamtoolsSort.Options,
		&config.SamtoolsIndex.Options,
		&config.SamtoolsStats.Options,
		&config.SamtoolsFlagstat.Options,
		&config.SamtoolsIdxstats.Options,
		&config.MergeRuns.Options,
		&config.MergeReplicates.Options,
		&config.MarkDuplicates.Options,
		&config.FilterBAM.Options,
		&config.BlacklistFilter.Options,
		&config.CollectMultipleMetrics.Options,
		&config.GenomeCoverage.Options,
		&config.ScaleCoverage.Options,
		&config.BedGraphToBigWig.Options,
		&config.ComputeMatrix.Options,
		&config.PlotProfile.Options,
		&config.PlotFingerprint.Options,
		&config.MACS2.Options,
		&config.HOMER.Options,
		&config.PlotMACS2QC.Options,
		&config.PlotHOMERAnnotatePeaks.Options,
		&config.PeakCount.Options,
		&config.PeakIntersect.Options,
		&config.ReadCount.Options,
		&config.FRiP.Options,
		&config.Consensus.Options,
		&config.FeatureCounts.Options,
		&config.FeatureCountsMerge.Options,
		&config.DESeq2QC.Options,
		&config.Ataqv.Options,
		&config.Mkarv.Options,
		&config.IGV.Options,
		&config.MultiQC.Options,
	} {
		option.Resources = gobble.Resources{CPU: 1, Memory: "16m"}
	}
	return config
}
