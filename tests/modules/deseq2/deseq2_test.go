package deseq2_test

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	. "github.com/HahyeonJeon/gobble/assets/modules/deseq2"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestDESeq2StandaloneComposeBuildPlan(t *testing.T) {
	counts := gobble.PathSpec{Dir: gobble.Dir("work/deseq2"), Base: "counts", Ext: ".csv"}
	opts := DESeq2Options{ExtraArgs: []string{"--quiet"}}
	groups := []string{"ctrl", "ctrl", "treat", "treat"}
	p := DESeq2Pipeline(counts, groups, opts)
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "deseq2")
	if task.Name != "deseq2" {
		t.Fatalf("task name = %q, want deseq2", task.Name)
	}
	if task.Image != "quay.io/biocontainers/bioconductor-deseq2:1.50.2--r45ha27e39d_0" {
		t.Fatalf("image = %q, want locked DESeq2 pin", task.Image)
	}
	if len(task.Command) < 2 || task.Command[0] != "Rscript" || task.Command[1] != "-e" {
		t.Fatalf("command = %#v, want Rscript -e", task.Command)
	}
	if !pc.ContainsAll(task.Command,
		"Rscript",
		"--",
		"work/deseq2/counts.csv",
		"work/deseq2/results.csv",
		"4",
		"ctrl", "treat",
		"--quiet",
	) {
		t.Fatalf("command = %#v, want dest, groups in sample order, extra-args", task.Command)
	}
	script, ok := pc.FlagValue(task.Command, "-e")
	if !ok {
		t.Fatalf("command = %#v, missing -e", task.Command)
	}
	for _, pin := range []string{"~ group", "gene_id", "log2FoldChange", "baseMean", "lfcSE", "stat", "pvalue", "padj"} {
		if !strings.Contains(script, pin) {
			t.Fatalf("DESeq2 script missing %q", pin)
		}
	}
	n := len(task.Command)
	if n < 1 || task.Command[n-1] != "--quiet" {
		t.Fatalf("command tail = %#v, want extra-args last", task.Command[max(0, n-2):])
	}
	pc.AssertUniqueParamNames(t, task.Params)
	pc.AssertIOPath(t, task.Inputs, "counts", "work/deseq2/counts.csv")
	pc.AssertIOPath(t, task.Outputs, "results", "work/deseq2/results.csv")
}

func TestDESeq2NestedModule(t *testing.T) {
	counts := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "counts", Ext: ".csv"}
	p := gobble.NewPipeline("assay")
	h := p.AddInput("counts", counts)
	mod := p.AddModule("deg")
	ports := AddDESeq2(mod, h, []string{"ctrl", "treat"}, DESeq2Options{ExtraArgs: []string{"--quiet"}})
	if ports.Results.IsZero() {
		t.Fatalf("ports.Results IsZero = true, want false")
	}
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "deg.deseq2")
	if task.Name != "deseq2" || task.Module != "deg" {
		t.Fatalf("nested task = %+v, want name deseq2 module deg", task)
	}
	if !pc.ContainsAll(task.Command, "--quiet", "ctrl", "treat", "work/deseq2/results.csv") {
		t.Fatalf("command = %#v, want extra-args, groups, and dest", task.Command)
	}
}
