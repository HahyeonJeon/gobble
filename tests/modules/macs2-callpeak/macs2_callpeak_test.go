package macs2callpeak_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	macs2callpeak "github.com/HahyeonJeon/gobble/assets/modules/macs2-callpeak"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandDeclaresTypedSummits(t *testing.T) {
	task := cc.Task(t, pipeline(macs2callpeak.Options{Mode: macs2callpeak.Broad, EffectiveGenomeSize: "12157105", QValue: 0.05}), "macs2_callpeak")
	if !pc.ContainsAll(task.Command, "macs2", "callpeak", "--broad", "--call-summits", "--treatment", "in/treatment.bam", "--control", "in/control.bam") {
		t.Fatalf("command = %#v, want control-aware broad call with typed summits", task.Command)
	}
	if task.Image != string(macs2callpeak.DefaultImage) {
		t.Fatalf("image = %q, want exact MACS2 authority %q", task.Image, macs2callpeak.DefaultImage)
	}
	pc.AssertIOPath(t, task.Outputs, "peaks", "results/macs2/peaks_peaks.broadPeak")
	pc.AssertIOPath(t, task.Outputs, "summits", "results/macs2/peaks_summits.bed")
	pc.AssertIOPath(t, task.Outputs, "xls", "results/macs2/peaks_peaks.xls")
}

func TestExtraArgsCannotBypassSummitBinding(t *testing.T) {
	cc.Invalid(t, pipeline(macs2callpeak.Options{
		Options:             modules.Options{ExtraArgs: []string{"--call-summits=false"}},
		Mode:                macs2callpeak.Broad,
		EffectiveGenomeSize: "12157105",
		QValue:              0.05,
	}))
}

func pipeline(options macs2callpeak.Options) *gobble.Pipeline {
	p := gobble.NewPipeline("macs2-callpeak")
	treatment := p.AddInput("treatment", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "treatment", Ext: ".bam"})
	control := p.AddInput("control", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "control", Ext: ".bam"})
	_, err := macs2callpeak.Add(p, treatment, control, options)
	p.RecordComposeError(err)
	return p
}
