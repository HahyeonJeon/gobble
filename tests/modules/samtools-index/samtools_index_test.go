package samtoolsindex_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	. "github.com/HahyeonJeon/gobble/assets/modules/samtools-index"
	samtoolssort "github.com/HahyeonJeon/gobble/assets/modules/samtools-sort"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestSamtoolsIndexStandaloneComposeBuildPlan(t *testing.T) {
	bam := gobble.PathSpec{Dir: gobble.Dir("work"), Base: "aligned", Ext: ".bam"}
	opts := SamtoolsIndexOptions{
		ExtraArgs: []string{"-b"},
		Resources: gobble.Resources{CPU: 2},
	}
	p := SamtoolsIndexPipeline(bam, opts)
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "samtools_index")
	if task.Name != "samtools_index" {
		t.Fatalf("task name = %q, want samtools_index", task.Name)
	}
	if task.Image != "quay.io/biocontainers/samtools:1.24--h9dcdb79_1" {
		t.Fatalf("image = %q, want locked samtools pin", task.Image)
	}
	if !pc.ContainsAll(task.Command,
		"samtools", "index",
		"-@", "2",
		"-b",
		"work/aligned.bam",
		"work/aligned.bam.bai",
	) {
		t.Fatalf("command = %#v, want named flags, extra-args, BAM, BAI", task.Command)
	}
	pc.AssertUniqueParamNames(t, task.Params)
	pc.AssertIOPath(t, task.Outputs, "bai", "work/aligned.bam.bai")
}

func TestSamtoolsIndexNestedModule(t *testing.T) {
	bam := gobble.PathSpec{Dir: gobble.Dir("work"), Base: "aligned", Ext: ".bam"}
	p := gobble.NewPipeline("assay")
	h := p.AddInput("bam", bam)
	mod := p.AddModule("align")
	ports := AddSamtoolsIndex(mod, h, SamtoolsIndexOptions{ExtraArgs: []string{"-b"}})
	if ports.BAI.IsZero() {
		t.Fatalf("ports.BAI IsZero = true, want false")
	}
	raw := pc.MustPlanJSON(t, p)
	task := pc.TaskByID(t, raw, "align.samtools_index")
	if task.Name != "samtools_index" || task.Module != "align" {
		t.Fatalf("nested task = %+v, want name samtools_index module align", task)
	}
	if !pc.ContainsAll(task.Command, "-b") {
		t.Fatalf("command = %#v, want extra-args", task.Command)
	}
	pc.AssertIOPath(t, task.Outputs, "bai", "work/aligned.bam.bai")
}

func TestSamtoolsIndexSecondTaskFromSort(t *testing.T) {
	sam := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "aligned", Ext: ".sam"}
	p := gobble.NewPipeline("assay")
	h := p.AddInput("sam", sam)
	sortPorts := samtoolssort.AddSamtoolsSort(p, h, samtoolssort.SamtoolsSortOptions{})
	indexPorts := AddSamtoolsIndex(p, sortPorts.BAM, SamtoolsIndexOptions{})
	if indexPorts.BAI.IsZero() {
		t.Fatalf("BAI IsZero = true, want false")
	}
	raw := pc.MustPlanJSON(t, p)
	sortTask := pc.TaskByID(t, raw, "samtools_sort")
	indexTask := pc.TaskByID(t, raw, "samtools_index")
	pc.AssertIOPath(t, sortTask.Outputs, "bam", "work/samtools-sort/aligned.bam")
	pc.AssertIOPath(t, indexTask.Outputs, "bai", "work/samtools-sort/aligned.bam.bai")
	pc.AssertIOPath(t, indexTask.Inputs, "bam", "work/samtools-sort/aligned.bam")
}
