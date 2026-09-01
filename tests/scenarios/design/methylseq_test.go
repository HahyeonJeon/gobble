package design_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
)

func TestMethylSeqProductIsDiscoverable(t *testing.T) {
	lifecycle := methylseq.Lifecycle()
	if methylseq.BenchmarkRelease != "nf-core/methylseq 4.2.0" || lifecycle.GraphGeneration != methylseq.GraphGeneration || !lifecycle.Design || lifecycle.PreLiftResumable {
		t.Fatalf("Methyl product identity/lifecycle = %+v, benchmark %q", lifecycle, methylseq.BenchmarkRelease)
	}
}
