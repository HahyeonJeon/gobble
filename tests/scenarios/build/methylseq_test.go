package build_test

import (
	"bytes"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/methylseqscenario"
)

func TestMethylTypedBuildMatchesDefaultAdapter(t *testing.T) {
	samples, sheet := methylseqscenario.Samples(t)
	want := pc.MustPlanJSON(t, methylseq.Build(samples, methylseq.DefaultConfig()))
	previous := gobble.SampleSheetPath()
	gobble.SetSampleSheetPath(sheet)
	t.Cleanup(func() { gobble.SetSampleSheetPath(previous) })
	got := pc.MustPlanJSON(t, methylseq.Pipeline())
	if !bytes.Equal(got, want) || !methylseq.Lifecycle().Build {
		t.Fatal("Methyl default adapter differs from typed Load plus Build")
	}
}
