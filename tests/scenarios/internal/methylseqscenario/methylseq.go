// Package methylseqscenario provides shared read-only access to the Methyl
// product and its sole fixture owner for lifecycle scenario evidence.
package methylseqscenario

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	methylseqevidence "github.com/HahyeonJeon/gobble/tests/pipelines/methylseq"
)

// Samples loads the Methyl pipeline owner's localized official fixture.
func Samples(t *testing.T) ([]methylseq.Sample, string) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "..", "..", filepath.FromSlash(methylseqevidence.FixtureSheet))
	samples, err := methylseq.Load(path)
	if err != nil {
		t.Fatalf("Load Methyl fixture: %v", err)
	}
	return samples, path
}

// Plan builds config over the Methyl owner's official fixture.
func Plan(t *testing.T, config methylseq.Config) []byte {
	t.Helper()
	samples, _ := Samples(t)
	return pc.MustPlanJSON(t, methylseq.Build(samples, config))
}
