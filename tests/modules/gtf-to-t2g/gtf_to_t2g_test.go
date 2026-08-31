package gtftot2g_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	gtftot2g "github.com/HahyeonJeon/gobble/assets/modules/gtf-to-t2g"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestGTFToT2GDeclaresRelation(t *testing.T) {
	p := gtftot2g.Pipeline(gobble.PathSpec{Base: "genes", Ext: ".gtf"}, gtftot2g.Options{})
	task := pc.AllTasks(t, pc.MustPlanJSON(t, p))[0]
	if task.Name != "gtf_to_t2g" || task.Image != string(gtftot2g.DefaultImage) {
		t.Fatalf("task = %#v", task)
	}
	pc.AssertIOPath(t, task.Outputs, "t2g", "work/scrnaseq/reference/transcript_to_gene.tsv")
}
