package design_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
)

func TestRNASeqProductIsDiscoverable(t *testing.T) {
	if rnaseq.Contract.Parse == nil || rnaseq.Contract.Load == nil || rnaseq.Contract.DefaultConfig == nil || rnaseq.Contract.Build == nil || rnaseq.Contract.Pipeline == nil {
		t.Fatal("RNA typed product contract is incomplete")
	}
	if rnaseq.BenchmarkRelease != "nf-core/rnaseq 3.26.0" || rnaseq.Lifecycle.GraphGeneration != rnaseq.GraphGeneration || !rnaseq.Lifecycle.Design || rnaseq.Lifecycle.PreLiftResumable {
		t.Fatalf("RNA product identity/lifecycle = %+v, benchmark %q", rnaseq.Lifecycle, rnaseq.BenchmarkRelease)
	}
}
