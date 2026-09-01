// Package wgsscenario provides shared read-only access to the WGS product
// fixture for lifecycle scenario owners.
package wgsscenario

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	wgsevidence "github.com/HahyeonJeon/gobble/tests/pipelines/wgs"
)

// Samples loads the pipeline-owned typed WGS fixture.
func Samples(t *testing.T) ([]wgs.Sample, string) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "..", "..", filepath.FromSlash(wgsevidence.FixtureSheet))
	samples, err := wgs.Load(path)
	if err != nil {
		t.Fatalf("Load WGS fixture: %v", err)
	}
	return samples, path
}

// Plan builds the real WGS graph with config and returns its plan JSON.
func Plan(t *testing.T, config wgs.Config) []byte {
	t.Helper()
	samples, _ := Samples(t)
	return pc.MustPlanJSON(t, wgs.Build(samples, config))
}
