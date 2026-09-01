package fastqcevidence_test

import (
	"errors"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	. "github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestFastQCStandaloneComposeBuildPlan(t *testing.T) {
	reads := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	opts := FastQCOptions{
		ExtraArgs: []string{"--kmers", "7"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := FastQCPipeline(reads, opts)
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "fastqc")
	if task.Name != "fastqc" {
		t.Fatalf("task name = %q, want fastqc", task.Name)
	}
	if task.Image != "quay.io/biocontainers/fastqc:0.12.1--hdfd78af_0" {
		t.Fatalf("image = %q, want locked FastQC pin", task.Image)
	}
	if !pc.ContainsAll(task.Command, "fastqc", "--outdir", "work/fastqc", "--noextract", "--threads", "2", "in/test_1.fastq.gz", "--kmers", "7") {
		t.Fatalf("command = %#v, want named flags, input, extra-args", task.Command)
	}
	if lastN := task.Command[len(task.Command)-2:]; lastN[0] != "--kmers" || lastN[1] != "7" {
		t.Fatalf("extra-args last tokens = %#v, want [--kmers 7]", lastN)
	}
	pc.AssertUniqueParamNames(t, task.Params)
	pc.AssertIOPath(t, task.Outputs, "html", "work/fastqc/test_1_fastqc.html")
	pc.AssertIOPath(t, task.Outputs, "zip", "work/fastqc/test_1_fastqc.zip")
}

func TestFastQCNestedModule(t *testing.T) {
	reads := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
	p := gobble.NewPipeline("assay")
	h := p.AddInput("reads", reads)
	mod := p.AddModule("raw")
	ports := AddFastQC(mod, h, FastQCOptions{ExtraArgs: []string{"--quiet"}})
	if ports.HTML.IsZero() || ports.Zip.IsZero() {
		t.Fatalf("ports HTML/Zip IsZero = %v/%v, want false", ports.HTML.IsZero(), ports.Zip.IsZero())
	}
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "raw.fastqc")
	if task.Name != "fastqc" {
		t.Fatalf("nested name = %q, want fastqc", task.Name)
	}
	if task.Module != "raw" {
		t.Fatalf("nested module = %q, want raw", task.Module)
	}
	if !pc.ContainsAll(task.Command, "--quiet") {
		t.Fatalf("command = %#v, want extra-args --quiet", task.Command)
	}
}

func TestFastQCProductRejectsPathBearingExtraArgs(t *testing.T) {
	tests := []struct {
		name  string
		extra string
	}{
		{name: "output long", extra: "--outdir=elsewhere"},
		{name: "output short", extra: "-oelsewhere"},
		{name: "adapters long", extra: "--adapters=in/adapters.txt"},
		{name: "adapters short", extra: "-ain/adapters.txt"},
		{name: "contaminants long", extra: "--contaminants=in/contaminants.txt"},
		{name: "contaminants short", extra: "-cin/contaminants.txt"},
		{name: "limits long", extra: "--limits=in/limits.txt"},
		{name: "limits short", extra: "-lin/limits.txt"},
		{name: "temporary directory long", extra: "--dir=elsewhere"},
		{name: "temporary directory short", extra: "-delsewhere"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pipeline := Pipeline(
				gobble.PathSpec{Dir: gobble.Dir("in"), Base: "reads", Ext: ".fastq.gz"},
				Options{Options: modules.Options{ExtraArgs: []string{test.extra}}},
			)
			graph, err := gobble.Compose(pipeline)
			var composeErr *gobble.Error
			if graph != nil || !errors.As(err, &composeErr) || len(composeErr.Defects) != 1 || composeErr.Defects[0].Code != gobble.DefectInvalidValue || composeErr.Defects[0].Unit != "fastqc" {
				t.Fatalf("Compose() with ExtraArgs %q returned graph=%t, error=%v; want one fastqc invalid-value defect", test.extra, graph != nil, err)
			}
		})
	}
}
