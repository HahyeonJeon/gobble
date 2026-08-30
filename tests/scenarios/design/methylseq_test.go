package design_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
)

func TestMethylSeqProductIsDiscoverable(t *testing.T) {
	if methylseq.Contract.Parse == nil || methylseq.Contract.Load == nil || methylseq.Contract.DefaultConfig == nil || methylseq.Contract.Build == nil || methylseq.Contract.Pipeline == nil {
		t.Fatal("Methyl typed product contract is incomplete")
	}
	if methylseq.BenchmarkRelease != "nf-core/methylseq 4.2.0" || methylseq.Lifecycle.GraphGeneration != methylseq.GraphGeneration || !methylseq.Lifecycle.Design || methylseq.Lifecycle.PreLiftResumable {
		t.Fatalf("Methyl product identity/lifecycle = %+v, benchmark %q", methylseq.Lifecycle, methylseq.BenchmarkRelease)
	}
}
