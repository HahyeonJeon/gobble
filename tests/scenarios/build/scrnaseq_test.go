package build_test

import (
	"bytes"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/scrnaseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/scrnaseqscenario"
)

func TestSCRNATypedBuildMatchesDefaultAdapter(t *testing.T) {
	samples, sheet := scrnaseqscenario.Samples(t)
	want := pc.MustPlanJSON(t, scrnaseq.Build(samples, scrnaseq.DefaultConfig()))
	previous := gobble.SampleSheetPath()
	gobble.SetSampleSheetPath(sheet)
	t.Cleanup(func() { gobble.SetSampleSheetPath(previous) })
	got := pc.MustPlanJSON(t, scrnaseq.Pipeline())
	if !bytes.Equal(got, want) || !scrnaseq.Lifecycle().Build {
		t.Fatal("scRNA default adapter differs from typed Load plus Build")
	}
}
