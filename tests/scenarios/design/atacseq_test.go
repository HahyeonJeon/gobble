package design_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/atacseq"
)

func TestATACProductIsDiscoverable(t *testing.T) {
	lifecycle := atacseq.Lifecycle()
	if atacseq.BenchmarkRelease != "nf-core/atacseq 2.1.2" || lifecycle.GraphGeneration != atacseq.GraphGeneration || !lifecycle.Design || lifecycle.PreLiftResumable {
		t.Fatalf("ATAC identity/lifecycle = %+v, benchmark %q", lifecycle, atacseq.BenchmarkRelease)
	}
}
