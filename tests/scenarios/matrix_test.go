package scenarios_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines"
	"github.com/HahyeonJeon/gobble/assets/pipelines/atacseq"
	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
	"github.com/HahyeonJeon/gobble/assets/pipelines/scrnaseq"
	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
)

func TestFiveProductSevenScenarioMatrix(t *testing.T) {
	products := []struct {
		name       string
		benchmark  string
		generation string
		lifecycle  func() pipelines.LifecycleParticipation
	}{
		{name: "wgs", benchmark: wgs.BenchmarkRelease, generation: wgs.GraphGeneration, lifecycle: wgs.Lifecycle},
		{name: "rnaseq", benchmark: rnaseq.BenchmarkRelease, generation: rnaseq.GraphGeneration, lifecycle: rnaseq.Lifecycle},
		{name: "methylseq", benchmark: methylseq.BenchmarkRelease, generation: methylseq.GraphGeneration, lifecycle: methylseq.Lifecycle},
		{name: "atacseq", benchmark: atacseq.BenchmarkRelease, generation: atacseq.GraphGeneration, lifecycle: atacseq.Lifecycle},
		{name: "scrnaseq", benchmark: scrnaseq.BenchmarkRelease, generation: scrnaseq.GraphGeneration, lifecycle: scrnaseq.Lifecycle},
	}
	if len(products) != 5 {
		t.Fatalf("products = %d, want 5", len(products))
	}
	for _, product := range products {
		t.Run(product.name, func(t *testing.T) {
			participation := product.lifecycle()
			if product.benchmark == "" || participation.GraphGeneration != product.generation || !participation.Complete() || participation.PreLiftResumable {
				t.Fatalf("benchmark %q lifecycle %+v, want dated complete non-pre-lift baseline", product.benchmark, participation)
			}
			participation.Design = false
			if !product.lifecycle().Design {
				t.Fatal("Lifecycle() retained caller mutation")
			}
		})
	}
	if pipelines.SupportPlatform != "linux/amd64" || pipelines.SupportExecutionBoundary != "trusted-local Docker" || pipelines.SupportClaim != "engineering-only" {
		t.Fatalf("support baseline = %q %q %q", pipelines.SupportPlatform, pipelines.SupportExecutionBoundary, pipelines.SupportClaim)
	}
}
