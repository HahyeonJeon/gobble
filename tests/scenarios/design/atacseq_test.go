package design_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/atacseq"
)

func TestATACProductIsDiscoverable(t *testing.T) {
	if atacseq.Contract.Parse == nil || atacseq.Contract.Load == nil || atacseq.Contract.DefaultConfig == nil || atacseq.Contract.Build == nil || atacseq.Contract.Pipeline == nil {
		t.Fatal("ATAC typed product contract is incomplete")
	}
	if atacseq.BenchmarkRelease != "nf-core/atacseq 2.1.2" || atacseq.Lifecycle.GraphGeneration != atacseq.GraphGeneration || !atacseq.Lifecycle.Design || atacseq.Lifecycle.PreLiftResumable {
		t.Fatalf("ATAC identity/lifecycle = %+v, benchmark %q", atacseq.Lifecycle, atacseq.BenchmarkRelease)
	}
}
