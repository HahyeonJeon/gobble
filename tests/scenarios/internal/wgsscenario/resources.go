package wgsscenario

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
)

// fixtureConfig sizes the deterministic command doubles for small test hosts.
// Production defaults and plan assertions retain their real analysis budgets.
func fixtureConfig(config wgs.Config) wgs.Config {
	for _, option := range []*modules.Options{
		&config.BWAIndex.Options,
		&config.FastQC.Options,
		&config.FastP.Options,
		&config.BWAMem.Options,
		&config.SamtoolsSort.Options,
		&config.SamtoolsMerge.Options,
		&config.MarkDuplicates.Options,
		&config.BaseRecalibrator.Options,
		&config.GatherBQSRReports.Options,
		&config.ApplyBQSR.Options,
		&config.GatherBAM.Options,
		&config.SamtoolsIndex.Options,
		&config.SamtoolsStats.Options,
		&config.SamtoolsFlagstat.Options,
		&config.SamtoolsIdxstats.Options,
		&config.Mosdepth.Options,
		&config.HaplotypeCaller.Options,
		&config.MergeGVCFs.Options,
		&config.GenomicsDBImport.Options,
		&config.GenotypeGVCFs.Options,
		&config.BCFToolsSort.Options,
		&config.MergeJointVCFs.Options,
		&config.BCFToolsStats.Options,
		&config.MultiQC.Options,
	} {
		option.Resources = gobble.Resources{CPU: 1, Memory: "16m"}
	}
	return config
}
