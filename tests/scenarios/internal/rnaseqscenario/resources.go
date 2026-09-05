package rnaseqscenario

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
)

// fixtureConfig sizes the deterministic command doubles for small test hosts.
// Production defaults and plan assertions retain their real analysis budgets.
func fixtureConfig(config rnaseq.Config) rnaseq.Config {
	for _, option := range []*modules.Options{
		&config.GTFFilter.Options,
		&config.GFFRead.Options,
		&config.Gunzip.Options,
		&config.STARGenome.Options,
		&config.SalmonIndex.Options,
		&config.FAIDX.Options,
		&config.ChromSizes.Options,
		&config.CatFASTQ.Options,
		&config.FQLint.Options,
		&config.FastQC.Options,
		&config.TrimGalore.Options,
		&config.TrimmedRetention.Options,
		&config.STAR.Options,
		&config.MappedRetention.Options,
		&config.Salmon.Options,
		&config.Sort.Options,
		&config.MarkDuplicates.Options,
		&config.Index.Options,
		&config.Stats.Options,
		&config.StringTie.Options,
		&config.GenomeCov.Options,
		&config.BedClip.Options,
		&config.BedGraphToBigWig.Options,
		&config.RSeQC.Options,
		&config.Qualimap.Options,
		&config.DupRadar.Options,
		&config.BiotypeQC.Options,
		&config.TxImport.Options,
		&config.DESeq2QC.Options,
		&config.MultiQC.Options,
	} {
		option.Resources = gobble.Resources{CPU: 1, Memory: "16m"}
	}
	return config
}
