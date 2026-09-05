package methylseqscenario

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
)

// fixtureConfig sizes the deterministic command doubles for small test hosts.
// Production defaults and plan assertions retain their real analysis budgets.
func fixtureConfig(config methylseq.Config) methylseq.Config {
	for _, option := range []*modules.Options{
		&config.CatFASTQ.Options,
		&config.FastQC.Options,
		&config.TrimGalore.Options,
		&config.BismarkGenome.Options,
		&config.BismarkAlign.Options,
		&config.Deduplicate.Options,
		&config.Extractor.Options,
		&config.Report.Options,
		&config.Summary.Options,
		&config.MultiQC.Options,
	} {
		option.Resources = gobble.Resources{CPU: 1, Memory: "16m"}
	}
	return config
}
