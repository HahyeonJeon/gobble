package build_test

import (
	"bytes"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/rnaseqscenario"
)

func TestRNATypedBuildMatchesDefaultAdapter(t *testing.T) {
	samples, sheet := rnaseqscenario.Samples(t)
	want := pc.MustPlanJSON(t, rnaseq.Build(samples, rnaseq.DefaultConfig()))
	previous := gobble.SampleSheetPath()
	gobble.SetSampleSheetPath(sheet)
	t.Cleanup(func() { gobble.SetSampleSheetPath(previous) })
	got := pc.MustPlanJSON(t, rnaseq.Pipeline())
	if !bytes.Equal(got, want) || !rnaseq.Lifecycle.Build {
		t.Fatal("RNA default adapter differs from typed Load plus Build")
	}
}
