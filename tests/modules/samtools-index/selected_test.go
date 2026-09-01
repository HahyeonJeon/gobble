package samtoolsindex_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	samtoolsindex "github.com/HahyeonJeon/gobble/assets/modules/samtools-index"
	samtoolssort "github.com/HahyeonJeon/gobble/assets/modules/samtools-sort"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestSelectedSamtoolsIndexStandalone(t *testing.T) {
	bam := gobble.PathSpec{Dir: gobble.Dir("work"), Base: "aligned", Ext: ".bam"}
	p := samtoolsindex.ProductPipeline(bam, samtoolsindex.Options{Options: modules.Options{Resources: gobble.Resources{CPU: 2}, ExtraArgs: []string{"-b"}}})
	task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "samtools_index")
	if task.Image != string(samtoolsindex.DefaultImage) {
		t.Fatalf("image = %q, want selected immutable image %q", task.Image, samtoolsindex.DefaultImage)
	}
	if !pc.ContainsAll(task.Command, "samtools", "index", "-@", "2", "work/aligned.bam", "work/aligned.bam.bai", "-b") {
		t.Fatalf("command = %#v, want selected samtools index argv", task.Command)
	}
	pc.AssertIOPath(t, task.Outputs, "bai", "work/aligned.bam.bai")
}

func TestSelectedSamtoolsIndexConsumesSelectedSort(t *testing.T) {
	p := gobble.NewPipeline("assay")
	h := p.AddInput("alignment", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "aligned", Ext: ".sam"})
	sorted, err := samtoolssort.Add(p, h, samtoolssort.Options{})
	if err != nil {
		t.Fatal(err)
	}
	indexed, err := samtoolsindex.Add(p, sorted.BAM, samtoolsindex.Options{})
	if err != nil || indexed.BAI.IsZero() {
		t.Fatalf("Add selected samtools index = (%+v, %v)", indexed, err)
	}
	raw := pc.MustPlanJSON(t, p)
	pc.AssertIOPath(t, pc.TaskByID(t, raw, "samtools_index").Inputs, "bam", "work/samtools-sort/aligned.bam")
}
