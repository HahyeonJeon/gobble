package design_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
)

func TestWGSProductIsDiscoverable(t *testing.T) {
	if wgs.Contract.Parse == nil || wgs.Contract.Load == nil || wgs.Contract.DefaultConfig == nil || wgs.Contract.Build == nil || wgs.Contract.Pipeline == nil {
		t.Fatal("WGS typed product contract is incomplete")
	}
	if wgs.BenchmarkRelease != "nf-core/sarek 3.10.0" || wgs.Lifecycle.GraphGeneration != wgs.GraphGeneration || !wgs.Lifecycle.Design || wgs.Lifecycle.PreLiftResumable {
		t.Fatalf("WGS product identity/lifecycle = %+v, benchmark %q", wgs.Lifecycle, wgs.BenchmarkRelease)
	}
}
