package customize_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/atacseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/atacseqscenario"
)

func TestATACTypedPeakCustomizationIsVisible(t *testing.T) {
	config := atacseq.DefaultConfig()
	config.PeakMode = atacseq.PeakNarrow
	config.MACS2.QValue = 0.01
	raw := atacseqscenario.Plan(t, config)
	task := pc.TaskByID(t, raw, "OSMOTIC_STRESS_T0_PE.replicate_1.peaks.macs2_callpeak")
	if !pc.ContainsAll(task.Command, "--qvalue", "0.01") || contains(task.Command, "--broad") || !atacseq.Lifecycle.Customize {
		t.Fatalf("ATAC peak customization is absent: %#v", task.Command)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
