package customize_test

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/scrnaseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/scrnaseqscenario"
)

func TestSCRNATypedUMIAndQCatchCustomizationIsVisible(t *testing.T) {
	config := scrnaseq.DefaultConfig()
	config.UMIResolution = scrnaseq.ResolutionParsimonyGeneEM
	config.QCatch.RemoveDoublets = true
	config.QCatch.VisualizeDoublets = true
	raw := scrnaseqscenario.Plan(t, config)
	quant := pc.TaskByID(t, raw, "Sample_X.simpleaf_quant")
	qcatch := pc.TaskByID(t, raw, "Sample_X.qcatch")
	if !strings.Contains(quant.Script, "'--resolution' 'parsimony-gene-em'") || !pc.ContainsAll(qcatch.Command, "--remove_doublets", "--visualize_doublets") || !scrnaseq.Lifecycle().Customize {
		t.Fatalf("scRNA typed customization absent: quant=%q qcatch=%#v", quant.Script, qcatch.Command)
	}
}
