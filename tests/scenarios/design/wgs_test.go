package design_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
)

func TestWGSProductIsDiscoverable(t *testing.T) {
	lifecycle := wgs.Lifecycle()
	if wgs.BenchmarkRelease != "nf-core/sarek 3.10.0" || lifecycle.GraphGeneration != wgs.GraphGeneration || !lifecycle.Design || lifecycle.PreLiftResumable {
		t.Fatalf("WGS product identity/lifecycle = %+v, benchmark %q", lifecycle, wgs.BenchmarkRelease)
	}
}
