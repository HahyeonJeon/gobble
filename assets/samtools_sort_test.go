package assets

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestSamtoolsSortStandaloneComposeBuildPlan(t *testing.T) {
	sam := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "aligned", Ext: ".sam"}
	opts := SamtoolsSortOptions{
		ExtraArgs: []string{"-n"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := SamtoolsSortPipeline(sam, opts)
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "samtools_sort")
	if task.Name != "samtools_sort" {
		t.Fatalf("task name = %q, want samtools_sort", task.Name)
	}
	if task.Image != samtoolsImage {
		t.Fatalf("image = %q, want %q", task.Image, samtoolsImage)
	}
	if !containsAll(task.Command,
		"samtools", "sort",
		"-@", "2",
		"-o", "work/samtools-sort/aligned.bam",
		"-n",
		"in/aligned.sam",
	) {
		t.Fatalf("command = %#v, want named flags, extra-args, then SAM", task.Command)
	}
	assertUniqueParamNames(t, task.Params)
	assertIOPath(t, task.Outputs, "bam", "work/samtools-sort/aligned.bam")
}

func TestSamtoolsSortNestedModule(t *testing.T) {
	sam := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "aligned", Ext: ".sam"}
	p := gobble.NewPipeline("assay")
	h := p.AddInput("sam", sam)
	mod := AddModule(p, "align")
	ports := AddSamtoolsSort(mod, h, SamtoolsSortOptions{ExtraArgs: []string{"-n"}})
	if ports.BAM.IsZero() {
		t.Fatalf("ports.BAM IsZero = true, want false")
	}
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "align.samtools_sort")
	if task.Name != "samtools_sort" || task.Module != "align" {
		t.Fatalf("nested task = %+v, want name samtools_sort module align", task)
	}
	if !containsAll(task.Command, "-n") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
}
