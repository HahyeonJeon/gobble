package design_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/scrnaseq"
)

func TestSCRNAProductIsDiscoverable(t *testing.T) {
	lifecycle := scrnaseq.Lifecycle()
	if scrnaseq.BenchmarkRelease != "nf-core/scrnaseq 4.2.0" || lifecycle.GraphGeneration != scrnaseq.GraphGeneration || !lifecycle.Design || lifecycle.PreLiftResumable {
		t.Fatalf("scRNA identity/lifecycle = %+v, benchmark %q", lifecycle, scrnaseq.BenchmarkRelease)
	}
}
