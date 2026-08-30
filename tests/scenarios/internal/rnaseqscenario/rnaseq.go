// Package rnaseqscenario provides shared read-only access to the RNA product
// and its sole fixture owner for lifecycle scenario evidence.
package rnaseqscenario

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

// Samples loads the RNA pipeline owner's localized official fixture.
func Samples(t *testing.T) ([]rnaseq.Sample, string) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "..", "pipelines", "rnaseq", "testdata", "rnaseq-samplesheet.csv")
	samples, err := rnaseq.Load(path)
	if err != nil {
		t.Fatalf("Load RNA fixture: %v", err)
	}
	return samples, path
}

// Plan builds the supplied config over the RNA owner's official fixture.
func Plan(t *testing.T, config rnaseq.Config) []byte {
	t.Helper()
	samples, _ := Samples(t)
	return pc.MustPlanJSON(t, rnaseq.Build(samples, config))
}
