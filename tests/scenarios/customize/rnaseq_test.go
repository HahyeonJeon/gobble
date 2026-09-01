package customize_test

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/rnaseqscenario"
)

func TestRNASalmonCustomizationIsPlanVisible(t *testing.T) {
	config := rnaseq.DefaultConfig()
	config.Salmon.ExtraArgs = []string{"--validateMappings"}
	raw := rnaseqscenario.Plan(t, config)
	if !strings.Contains(pc.TaskByID(t, raw, "WT_REP1.salmon_quant").Script, "'--validateMappings'") || !rnaseq.Lifecycle().Customize {
		t.Fatal("RNA Salmon customization is not visible in the selected command")
	}
}
