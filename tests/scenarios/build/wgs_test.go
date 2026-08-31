package build_test

import (
	"bytes"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/wgsscenario"
)

func TestWGSTypedBuildMatchesDefaultAdapter(t *testing.T) {
	samples, sheet := wgsscenario.Samples(t)
	want := pc.MustPlanJSON(t, wgs.Build(samples, wgs.DefaultConfig()))
	previous := gobble.SampleSheetPath()
	gobble.SetSampleSheetPath(sheet)
	t.Cleanup(func() { gobble.SetSampleSheetPath(previous) })
	got := pc.MustPlanJSON(t, wgs.Pipeline())
	if !bytes.Equal(got, want) || !wgs.Lifecycle.Build {
		t.Fatal("WGS default adapter differs from typed Load plus Build")
	}
}
