package customize_test

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/wgsscenario"
)

func TestWGSTypedCallerCustomizationIsVisible(t *testing.T) {
	config := wgs.DefaultConfig()
	config.HaplotypeCaller.ExtraArgs = []string{"--min-pruning", "1"}
	raw := wgsscenario.Plan(t, config)
	script := pc.TaskByID(t, raw, "haplotype_intervals.patient1.testN.gatk4_haplotypecaller").Script
	if !strings.Contains(script, "'--min-pruning' '1'") || !wgs.Lifecycle().Customize {
		t.Fatalf("WGS HaplotypeCaller customization is absent: %s", script)
	}
}
