package scrnaseqscenario

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	"github.com/HahyeonJeon/gobble/assets/pipelines/scrnaseq"
)

// fixtureConfig sizes the deterministic command doubles for small test hosts.
// Production defaults and plan assertions retain their real analysis budgets.
func fixtureConfig(config scrnaseq.Config) scrnaseq.Config {
	for _, option := range []*modules.Options{
		&config.Consolidate.Options,
		&config.FastQC.Options,
		&config.GTFFilter.Options,
		&config.Transcriptome.Options,
		&config.TranscriptToGene.Options,
		&config.SimpleafIndex.Options,
		&config.SimpleafQuant.Options,
		&config.QCatch.Options,
		&config.MatrixToH5AD.Options,
		&config.AnnDataR.Options,
		&config.H5ADConcat.Options,
		&config.MultiQC.Options,
	} {
		option.Resources = gobble.Resources{CPU: 1, Memory: "16m"}
	}
	return config
}
