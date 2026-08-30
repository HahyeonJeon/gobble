package samtoolssort_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	. "github.com/HahyeonJeon/gobble/assets/modules/samtools-sort"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestSamtoolsSortStandaloneComposeBuildPlan(t *testing.T) {
	sam := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "aligned", Ext: ".sam"}
	opts := SamtoolsSortOptions{
		ExtraArgs: []string{"-n"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := SamtoolsSortPipeline(sam, opts)
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "samtools_sort")
	if task.Name != "samtools_sort" {
		t.Fatalf("task name = %q, want samtools_sort", task.Name)
	}
	if task.Image != "quay.io/biocontainers/samtools:1.24--h9dcdb79_1" {
		t.Fatalf("image = %q, want locked samtools pin", task.Image)
	}
	if !pc.ContainsAll(task.Command,
		"samtools", "sort",
		"-@", "2",
		"-o", "work/samtools-sort/aligned.bam",
		"-n",
		"in/aligned.sam",
	) {
		t.Fatalf("command = %#v, want named flags, extra-args, then SAM", task.Command)
	}
	pc.AssertUniqueParamNames(t, task.Params)
	pc.AssertIOPath(t, task.Outputs, "bam", "work/samtools-sort/aligned.bam")
}

func TestSamtoolsSortNestedModule(t *testing.T) {
	sam := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "aligned", Ext: ".sam"}
	p := gobble.NewPipeline("assay")
	h := p.AddInput("sam", sam)
	mod := p.AddModule("align")
	ports := AddSamtoolsSort(mod, h, SamtoolsSortOptions{ExtraArgs: []string{"-n"}})
	if ports.BAM.IsZero() {
		t.Fatalf("ports.BAM IsZero = true, want false")
	}
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "align.samtools_sort")
	if task.Name != "samtools_sort" || task.Module != "align" {
		t.Fatalf("nested task = %+v, want name samtools_sort module align", task)
	}
	if !pc.ContainsAll(task.Command, "-n") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
}
