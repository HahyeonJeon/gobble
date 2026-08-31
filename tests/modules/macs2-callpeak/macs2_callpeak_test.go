package macs2callpeak_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	macs2callpeak "github.com/HahyeonJeon/gobble/assets/modules/macs2-callpeak"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandsConformToMACS2271PeakModes(t *testing.T) {
	for _, test := range []struct {
		name        string
		mode        macs2callpeak.Mode
		peakPath    string
		wantBroad   bool
		wantSummits bool
	}{
		{name: "broad", mode: macs2callpeak.Broad, peakPath: "results/macs2/peaks_peaks.broadPeak", wantBroad: true},
		{name: "narrow", mode: macs2callpeak.Narrow, peakPath: "results/macs2/peaks_peaks.narrowPeak", wantSummits: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			task := cc.Task(t, pipeline(macs2callpeak.Options{Mode: test.mode, EffectiveGenomeSize: "12157105", QValue: 0.05}), "macs2_callpeak")
			if len(task.Command) < 2 || task.Command[0] != "macs2" || task.Command[1] != "callpeak" ||
				!pc.ContainsAll(task.Command, "--gsize", "12157105", "--format", "BAM", "--qvalue", "0.05", "--treatment", "in/treatment.bam", "--control", "in/control.bam") {
				t.Fatalf("command = %#v, want control-aware MACS2 callpeak", task.Command)
			}
			if hasArg(task.Command, "--broad") != test.wantBroad || hasArg(task.Command, "--call-summits") != test.wantSummits {
				t.Fatalf("command = %#v, want broad=%t call-summits=%t", task.Command, test.wantBroad, test.wantSummits)
			}
			if hasArg(task.Command, "--broad") && hasArg(task.Command, "--call-summits") {
				t.Fatalf("command = %#v, MACS2 2.2.7.1 rejects broad peak calls with summits", task.Command)
			}
			if task.Image != string(macs2callpeak.DefaultImage) {
				t.Fatalf("image = %q, want exact MACS2 authority %q", task.Image, macs2callpeak.DefaultImage)
			}
			pc.AssertIOPath(t, task.Outputs, "peaks", test.peakPath)
			pc.AssertIOPath(t, task.Outputs, "xls", "results/macs2/peaks_peaks.xls")
			if hasOutput(task.Outputs, "summits") != test.wantSummits {
				t.Fatalf("outputs = %#v, want summits=%t", task.Outputs, test.wantSummits)
			}
			if test.wantSummits {
				pc.AssertIOPath(t, task.Outputs, "summits", "results/macs2/peaks_summits.bed")
			}
		})
	}
}

func TestExtraArgsCannotAliasTypedMACS2Options(t *testing.T) {
	protected := []struct {
		long  string
		short string
	}{
		{long: "--gsize", short: "-g"},
		{long: "--format", short: "-f"},
		{long: "--name", short: "-n"},
		{long: "--treatment", short: "-t"},
		{long: "--control", short: "-c"},
		{long: "--outdir"},
		{long: "--qvalue", short: "-q"},
		{long: "--broad"},
		{long: "--call-summits"},
	}
	for _, option := range protected {
		for length := 3; length <= len(option.long); length++ {
			alias := option.long[:length]
			for _, extra := range []string{alias, alias + "=override"} {
				t.Run(extra, func(t *testing.T) {
					assertInvalidExtraArg(t, extra)
				})
			}
		}
		if option.short != "" {
			for _, extra := range []string{option.short, option.short + "override", option.short + "=override"} {
				t.Run(extra, func(t *testing.T) {
					assertInvalidExtraArg(t, extra)
				})
			}
		}
	}
}

func assertInvalidExtraArg(t *testing.T, extra string) {
	t.Helper()
	cc.Invalid(t, pipeline(macs2callpeak.Options{
		Options:             modules.Options{ExtraArgs: []string{extra}},
		Mode:                macs2callpeak.Broad,
		EffectiveGenomeSize: "12157105",
		QValue:              0.05,
	}))
}

func hasArg(command []string, want string) bool {
	for _, arg := range command {
		if arg == want {
			return true
		}
	}
	return false
}

func hasOutput(outputs []pc.IO, want string) bool {
	for _, output := range outputs {
		if output.Name == want {
			return true
		}
	}
	return false
}

func pipeline(options macs2callpeak.Options) *gobble.Pipeline {
	p := gobble.NewPipeline("macs2-callpeak")
	treatment := p.AddInput("treatment", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "treatment", Ext: ".bam"})
	control := p.AddInput("control", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "control", Ext: ".bam"})
	_, err := macs2callpeak.Add(p, treatment, control, options)
	p.RecordComposeError(err)
	return p
}
