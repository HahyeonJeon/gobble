// Package scrnaseqscenario provides shared read-only access to the scRNA
// product fixture for lifecycle scenario owners.
package scrnaseqscenario

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/scrnaseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

// Samples loads the pipeline-owned typed scRNA fixture.
func Samples(t *testing.T) ([]scrnaseq.Sample, string) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "..", "pipelines", "scrnaseq", "testdata", "scrnaseq-samplesheet.csv")
	samples, err := scrnaseq.Load(path)
	if err != nil {
		t.Fatalf("Load scRNA fixture: %v", err)
	}
	return samples, path
}

// Plan builds the real scRNA graph with config and returns its plan JSON.
func Plan(t *testing.T, config scrnaseq.Config) []byte {
	t.Helper()
	samples, _ := Samples(t)
	return pc.MustPlanJSON(t, scrnaseq.Build(samples, config))
}
