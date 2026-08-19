package assets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

// Documented nf-core modules-branch FastQC zip from test_1.fastq.gz.
// Public reuse: modules README lists illumina/fastqc/test_fastqc.zip;
// test-datasets LICENSE is MIT, copyright 2018 nf-core. Fixture origin
// is this pin, not a live FastQC run.
var pinSARSCoV2FastQCZip = Pin{
	Name:   "test_fastqc.zip",
	URL:    "https://raw.githubusercontent.com/nf-core/test-datasets/modules/data/genomics/sarscov2/illumina/fastqc/test_fastqc.zip",
	Bytes:  620149,
	SHA256: "3fb4ca7852b311f1ab542028cde1debd98edeac94384224b7d5bba78dd9612b6",
}

func TestMultiQCStandaloneComposeBuildPlan(t *testing.T) {
	report := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_fastqc", Ext: ".zip"}
	opts := MultiQCOptions{
		ExtraArgs: []string{"--title", "qc"},
		Resources: gobble.Resources{CPU: 1},
	}
	p := MultiQCPipeline([]gobble.PathSpec{report}, opts)
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "multiqc")
	if task.Name != "multiqc" {
		t.Fatalf("task name = %q, want multiqc", task.Name)
	}
	if task.Image != multiqcImage {
		t.Fatalf("image = %q, want %q", task.Image, multiqcImage)
	}
	if !containsAll(task.Command, "multiqc", "--force", "--outdir", "work/multiqc", "--zip-data-dir", "in/test_fastqc.zip", "--title", "qc") {
		t.Fatalf("command = %#v, want named flags, reports, extra-args", task.Command)
	}
	n := len(task.Command)
	if n < 2 || task.Command[n-2] != "--title" || task.Command[n-1] != "qc" {
		t.Fatalf("extra-args last tokens = %#v, want [--title qc]", task.Command[n-2:])
	}
	if containsAll(task.Command, "--threads") || containsAll(task.Command, "--thread") {
		t.Fatalf("command = %#v, MultiQC must not copy Resources.CPU into a thread flag", task.Command)
	}
	assertUniqueParamNames(t, task.Params)
	assertIOPath(t, task.Inputs, "report_0", "in/test_fastqc.zip")
	assertIOPath(t, task.Outputs, "html", "work/multiqc/multiqc_report.html")
	assertIOPath(t, task.Outputs, "data", "work/multiqc/multiqc_data.zip")
}

func TestMultiQCNestedModule(t *testing.T) {
	report := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_fastqc", Ext: ".zip"}
	p := gobble.NewPipeline("assay")
	h := p.AddInput("zip", report)
	mod := AddModule(p, "qc")
	ports := AddMultiQC(mod, []gobble.Handle{h}, MultiQCOptions{ExtraArgs: []string{"--fullnames"}})
	if ports.HTML.IsZero() || ports.Data.IsZero() {
		t.Fatalf("ports HTML/Data IsZero = %v/%v, want false", ports.HTML.IsZero(), ports.Data.IsZero())
	}
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "qc.multiqc")
	if task.Name != "multiqc" || task.Module != "qc" {
		t.Fatalf("nested task = %+v, want name multiqc module qc", task)
	}
	if !containsAll(task.Command, "--fullnames") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
}

func TestMultiQCStandaloneRun(t *testing.T) {
	requireDocker(t)
	src := cachePin(t, pinSARSCoV2FastQCZip)
	dir := t.TempDir()
	stageFile(t, dir, "in/test_fastqc.zip", src)
	report := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_fastqc", Ext: ".zip"}
	p := MultiQCPipeline([]gobble.PathSpec{report}, MultiQCOptions{})
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := gobble.Run(g, dir, 1); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, rel := range []string{"work/multiqc/multiqc_report.html", "work/multiqc/multiqc_data.zip"} {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("published %s: %v", rel, err)
		}
	}
}
