package plotmacs2qc_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	plotmacs2qc "github.com/HahyeonJeon/gobble/assets/modules/plot-macs2-qc"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContract(t *testing.T) {
	peaks := []gobble.PathSpec{
		{Dir: gobble.Dir("in"), Base: "sample_a_peaks", Ext: ".broadPeak"},
		{Dir: gobble.Dir("in"), Base: "sample_b_peaks", Ext: ".broadPeak"},
	}
	options := plotmacs2qc.Options{OutDir: gobble.Dir("results/qc"), Prefix: "macs2"}
	task := cc.Task(t, plotmacs2qc.Pipeline(peaks, []string{"sample_a", "sample_b"}, options), "plot_macs2_qc")
	if !pc.ContainsAll(task.Command, "Rscript", "-e", "results/qc/macs2.summary.txt", "results/qc/macs2.plots.pdf", "sample_a", "in/sample_a_peaks.broadPeak", "sample_b", "in/sample_b_peaks.broadPeak") {
		t.Fatalf("command = %#v, want typed strict MACS2 plot fan-in", task.Command)
	}
	if task.Image != string(plotmacs2qc.DefaultImage) {
		t.Fatalf("image = %q, want exact peak-QC authority %q", task.Image, plotmacs2qc.DefaultImage)
	}
	pc.AssertIOPath(t, task.Outputs, "summary", "results/qc/macs2.summary.txt")
	pc.AssertIOPath(t, task.Outputs, "pdf", "results/qc/macs2.plots.pdf")
}

func TestMembershipAndExtraArgsFailClosed(t *testing.T) {
	peak := []gobble.PathSpec{{Dir: gobble.Dir("in"), Base: "sample", Ext: ".broadPeak"}}
	cc.Invalid(t, plotmacs2qc.Pipeline(peak, nil, plotmacs2qc.Options{}))
	cc.Invalid(t, plotmacs2qc.Pipeline(peak, []string{"sample"}, plotmacs2qc.Options{Options: modules.Options{ExtraArgs: []string{"--outdir", "elsewhere"}}}))
	cc.Invalid(t, plotmacs2qc.Pipeline(append(peak, peak...), []string{"sample", "sample"}, plotmacs2qc.Options{}))
}
