package build_test

import (
	"bytes"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/atacseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/atacseqscenario"
)

func TestATACTypedBuildMatchesDefaultAdapter(t *testing.T) {
	samples, sheet := atacseqscenario.Samples(t)
	want := pc.MustPlanJSON(t, atacseq.Build(samples, atacseq.DefaultConfig()))
	previous := gobble.SampleSheetPath()
	gobble.SetSampleSheetPath(sheet)
	t.Cleanup(func() { gobble.SetSampleSheetPath(previous) })
	got := pc.MustPlanJSON(t, atacseq.Pipeline())
	if !bytes.Equal(got, want) || !atacseq.Lifecycle().Build {
		t.Fatal("ATAC default adapter differs from typed Load plus Build")
	}
}
