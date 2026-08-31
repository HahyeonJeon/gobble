package plothomerannotatepeaks_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	plothomerannotatepeaks "github.com/HahyeonJeon/gobble/assets/modules/plot-homer-annotatepeaks"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContract(t *testing.T) {
	annotations := []gobble.PathSpec{
		{Dir: gobble.Dir("in"), Base: "sample_a", Ext: ".annotatePeaks.txt"},
		{Dir: gobble.Dir("in"), Base: "sample_b", Ext: ".annotatePeaks.txt"},
	}
	options := plothomerannotatepeaks.Options{
		OutDir:         gobble.Dir("results/qc"),
		Prefix:         "homer",
		MultiQCID:      "mlib_peak_annotation",
		MultiQCSection: "MERGED LIB: HOMER peak annotation",
	}
	task := cc.Task(t, plothomerannotatepeaks.Pipeline(annotations, []string{"sample_a", "sample_b"}, options), "plot_homer_annotatepeaks")
	if !pc.ContainsAll(task.Command, "Rscript", "-e", "results/qc/homer.summary.txt", "results/qc/homer.plots.pdf", "results/qc/homer.summary_mqc.tsv", "mlib_peak_annotation", "MERGED LIB: HOMER peak annotation", "sample_a", "in/sample_a.annotatePeaks.txt", "sample_b", "in/sample_b.annotatePeaks.txt") {
		t.Fatalf("command = %#v, want typed strict HOMER plot fan-in", task.Command)
	}
	if task.Image != string(plothomerannotatepeaks.DefaultImage) {
		t.Fatalf("image = %q, want exact annotation-QC authority %q", task.Image, plothomerannotatepeaks.DefaultImage)
	}
	pc.AssertIOPath(t, task.Outputs, "summary", "results/qc/homer.summary.txt")
	pc.AssertIOPath(t, task.Outputs, "pdf", "results/qc/homer.plots.pdf")
	pc.AssertIOPath(t, task.Outputs, "multiqc", "results/qc/homer.summary_mqc.tsv")
}

func TestMembershipMetadataAndExtraArgsFailClosed(t *testing.T) {
	annotation := []gobble.PathSpec{{Dir: gobble.Dir("in"), Base: "sample", Ext: ".annotatePeaks.txt"}}
	cc.Invalid(t, plothomerannotatepeaks.Pipeline(annotation, nil, plothomerannotatepeaks.Options{}))
	cc.Invalid(t, plothomerannotatepeaks.Pipeline(annotation, []string{"sample"}, plothomerannotatepeaks.Options{MultiQCID: "bad id"}))
	cc.Invalid(t, plothomerannotatepeaks.Pipeline(annotation, []string{"sample"}, plothomerannotatepeaks.Options{Options: modules.Options{ExtraArgs: []string{"--homer_files", "other.txt"}}}))
}
