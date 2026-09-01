package samtoolssort_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	samtoolssort "github.com/HahyeonJeon/gobble/assets/modules/samtools-sort"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestSelectedSamtoolsSortStandalone(t *testing.T) {
	alignment := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "aligned", Ext: ".sam"}
	p := samtoolssort.ProductPipeline(alignment, samtoolssort.Options{
		Options: modules.Options{Resources: gobble.Resources{CPU: 2}, ExtraArgs: []string{"-n"}},
		Prefix:  "sample",
	})
	task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "samtools_sort")
	if task.Image != string(samtoolssort.DefaultImage) {
		t.Fatalf("image = %q, want selected immutable image %q", task.Image, samtoolssort.DefaultImage)
	}
	if !pc.ContainsAll(task.Command, "samtools", "sort", "-o", "work/samtools-sort/sample.bam", "-@", "2", "in/aligned.sam", "-n") {
		t.Fatalf("command = %#v, want selected samtools sort argv", task.Command)
	}
	pc.AssertIOPath(t, task.Outputs, "bam", "work/samtools-sort/sample.bam")
}

func TestSelectedSamtoolsSortNestedModule(t *testing.T) {
	p := gobble.NewPipeline("assay")
	h := p.AddInput("alignment", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "aligned", Ext: ".sam"})
	ports, err := samtoolssort.Add(p.AddModule("alignment"), h, samtoolssort.Options{})
	if err != nil || ports.BAM.IsZero() {
		t.Fatalf("Add selected samtools sort = (%+v, %v)", ports, err)
	}
	task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "alignment.samtools_sort")
	if task.Module != "alignment" || task.Image != string(samtoolssort.DefaultImage) {
		t.Fatalf("nested selected task = %+v", task)
	}
}
