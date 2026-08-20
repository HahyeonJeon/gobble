package assets

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestMergeCountsStandaloneComposeBuildPlan(t *testing.T) {
	c0 := gobble.PathSpec{Dir: gobble.Dir("work/ctrl1/featurecounts"), Base: "counts", Ext: ".txt"}
	c1 := gobble.PathSpec{Dir: gobble.Dir("work/treat1/featurecounts"), Base: "counts", Ext: ".txt"}
	opts := MergeCountsOptions{
		ExtraArgs:   []string{"--quiet"},
		SampleNames: []string{"ctrl1", "treat1"},
	}
	p := MergeCountsPipeline([]gobble.PathSpec{c0, c1}, opts)
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "merge_counts")
	if task.Name != "merge_counts" {
		t.Fatalf("task name = %q, want merge_counts", task.Name)
	}
	if task.Image != deseq2Image || deseq2Image != "quay.io/biocontainers/bioconductor-deseq2:1.50.2--r45ha27e39d_0" {
		t.Fatalf("image = %q, want locked DESeq2 pin", task.Image)
	}
	if len(task.Command) < 2 || task.Command[0] != "Rscript" || task.Command[1] != "-e" {
		t.Fatalf("command = %#v, want Rscript -e", task.Command)
	}
	if !containsAll(task.Command,
		"Rscript",
		"--",
		"work/deseq2/counts.csv",
		"2",
		"ctrl1", "work/ctrl1/featurecounts/counts.txt",
		"treat1", "work/treat1/featurecounts/counts.txt",
		"--quiet",
	) {
		t.Fatalf("command = %#v, want dest, sample columns in handle order, extra-args", task.Command)
	}
	n := len(task.Command)
	if n < 1 || task.Command[n-1] != "--quiet" {
		t.Fatalf("command tail = %#v, want extra-args last", task.Command[max(0, n-2):])
	}
	for _, in := range task.Inputs {
		if in.Kind == "group" {
			t.Fatalf("inputs = %#v, merge must not use Group From", task.Inputs)
		}
	}
	assertUniqueParamNames(t, task.Params)
	assertIOPath(t, task.Inputs, "counts_0", "work/ctrl1/featurecounts/counts.txt")
	assertIOPath(t, task.Inputs, "counts_1", "work/treat1/featurecounts/counts.txt")
	assertIOPath(t, task.Outputs, "counts", "work/deseq2/counts.csv")
}

func TestMergeCountsNestedModule(t *testing.T) {
	c0 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "s0", Ext: ".txt"}
	c1 := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "s1", Ext: ".txt"}
	p := gobble.NewPipeline("assay")
	h0 := p.AddInput("c0", c0)
	h1 := p.AddInput("c1", c1)
	mod := AddModule(p, "deg")
	ports := AddMergeCounts(mod, []gobble.Handle{h0, h1}, MergeCountsOptions{
		ExtraArgs:   []string{"--quiet"},
		SampleNames: []string{"ctrl1", "treat1"},
	})
	if ports.Counts.IsZero() {
		t.Fatalf("ports.Counts IsZero = true, want false")
	}
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "deg.merge_counts")
	if task.Name != "merge_counts" || task.Module != "deg" {
		t.Fatalf("nested task = %+v, want name merge_counts module deg", task)
	}
	if !containsAll(task.Command, "--quiet", "ctrl1", "treat1") {
		t.Fatalf("command = %#v, want extra-args and sample names", task.Command)
	}
}
