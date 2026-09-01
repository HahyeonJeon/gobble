// Package atacseqscenario provides shared read-only access to the ATAC product
// fixture for lifecycle scenario owners.
package atacseqscenario

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/atacseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	atacseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/atacseq"
)

// Samples loads the pipeline-owned typed ATAC fixture.
func Samples(t *testing.T) ([]atacseq.Sample, string) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "..", "..", filepath.FromSlash(atacseqevidence.FixtureSheet))
	samples, err := atacseq.Load(path)
	if err != nil {
		t.Fatalf("Load ATAC fixture: %v", err)
	}
	return samples, path
}

// Plan builds the real ATAC graph with config and returns its plan JSON.
func Plan(t *testing.T, config atacseq.Config) []byte {
	t.Helper()
	samples, _ := Samples(t)
	return pc.MustPlanJSON(t, atacseq.Build(samples, config))
}
