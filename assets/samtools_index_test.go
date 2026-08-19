package assets

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestSamtoolsIndexStandaloneComposeBuildPlan(t *testing.T) {
	bam := gobble.PathSpec{Dir: gobble.Dir("work"), Base: "aligned", Ext: ".bam"}
	opts := SamtoolsIndexOptions{
		ExtraArgs: []string{"-b"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := SamtoolsIndexPipeline(bam, opts)
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "samtools_index")
	if task.Name != "samtools_index" {
		t.Fatalf("task name = %q, want samtools_index", task.Name)
	}
	if task.Image != samtoolsImage {
		t.Fatalf("image = %q, want %q", task.Image, samtoolsImage)
	}
	if !containsAll(task.Command,
		"samtools", "index",
		"-@", "2",
		"-b",
		"work/aligned.bam",
		"work/aligned.bam.bai",
	) {
		t.Fatalf("command = %#v, want named flags, extra-args, BAM, BAI", task.Command)
	}
	assertUniqueParamNames(t, task.Params)
	assertIOPath(t, task.Outputs, "bai", "work/aligned.bam.bai")
}

func TestSamtoolsIndexNestedModule(t *testing.T) {
	bam := gobble.PathSpec{Dir: gobble.Dir("work"), Base: "aligned", Ext: ".bam"}
	p := gobble.NewPipeline("assay")
	h := p.AddInput("bam", bam)
	mod := AddModule(p, "align")
	ports := AddSamtoolsIndex(mod, h, SamtoolsIndexOptions{ExtraArgs: []string{"-b"}})
	if ports.BAI.IsZero() {
		t.Fatalf("ports.BAI IsZero = true, want false")
	}
	raw := mustPlanJSON(t, p)
	task := planTask(t, raw, "align.samtools_index")
	if task.Name != "samtools_index" || task.Module != "align" {
		t.Fatalf("nested task = %+v, want name samtools_index module align", task)
	}
	if !containsAll(task.Command, "-b") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
	assertIOPath(t, task.Outputs, "bai", "work/aligned.bam.bai")
}

func TestSamtoolsIndexSecondTaskFromSort(t *testing.T) {
	sam := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "aligned", Ext: ".sam"}
	p := gobble.NewPipeline("assay")
	h := p.AddInput("sam", sam)
	sortPorts := AddSamtoolsSort(p, h, SamtoolsSortOptions{})
	indexPorts := AddSamtoolsIndex(p, sortPorts.BAM, SamtoolsIndexOptions{})
	if indexPorts.BAI.IsZero() {
		t.Fatalf("BAI IsZero = true, want false")
	}
	raw := mustPlanJSON(t, p)
	sortTask := planTask(t, raw, "samtools_sort")
	indexTask := planTask(t, raw, "samtools_index")
	assertIOPath(t, sortTask.Outputs, "bam", "work/samtools-sort/aligned.bam")
	assertIOPath(t, indexTask.Outputs, "bai", "work/samtools-sort/aligned.bam.bai")
	assertIOPath(t, indexTask.Inputs, "bam", "work/samtools-sort/aligned.bam")
}
