package design_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
)

func TestRNASeqProductIsDiscoverable(t *testing.T) {
	lifecycle := rnaseq.Lifecycle()
	if rnaseq.BenchmarkRelease != "nf-core/rnaseq 3.26.0" || lifecycle.GraphGeneration != rnaseq.GraphGeneration || !lifecycle.Design || lifecycle.PreLiftResumable {
		t.Fatalf("RNA product identity/lifecycle = %+v, benchmark %q", lifecycle, rnaseq.BenchmarkRelease)
	}
}
