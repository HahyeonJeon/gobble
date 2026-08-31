package design_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/scrnaseq"
)

func TestSCRNAProductIsDiscoverable(t *testing.T) {
	if scrnaseq.Contract.Parse == nil || scrnaseq.Contract.Load == nil || scrnaseq.Contract.DefaultConfig == nil || scrnaseq.Contract.Build == nil || scrnaseq.Contract.Pipeline == nil {
		t.Fatal("scRNA typed product contract is incomplete")
	}
	if scrnaseq.BenchmarkRelease != "nf-core/scrnaseq 4.2.0" || scrnaseq.Lifecycle.GraphGeneration != scrnaseq.GraphGeneration || !scrnaseq.Lifecycle.Design || scrnaseq.Lifecycle.PreLiftResumable {
		t.Fatalf("scRNA identity/lifecycle = %+v, benchmark %q", scrnaseq.Lifecycle, scrnaseq.BenchmarkRelease)
	}
}
