package customize_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/methylseqscenario"
)

func TestMethylExtractorCustomizationIsPlanVisible(t *testing.T) {
	config := methylseq.DefaultConfig()
	config.Extractor.CoverageCutoff = 3
	raw := methylseqscenario.Plan(t, config)
	if !pc.ContainsAll(pc.TaskByID(t, raw, "SRR389222_sub1.bismark_methylation_extractor").Command, "--cutoff", "3") || !methylseq.Lifecycle().Customize {
		t.Fatal("Methyl extractor customization is not visible in the selected command")
	}
}
