package gtfgenefilter_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	gtfgenefilter "github.com/HahyeonJeon/gobble/assets/modules/gtf-gene-filter"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestGTFGeneFilterOwnsTypedOperands(t *testing.T) {
	p := gtfgenefilter.Pipeline(gobble.PathSpec{Base: "genome", Ext: ".fa"}, gobble.PathSpec{Base: "genes", Ext: ".gtf"}, gtfgenefilter.Options{})
	task := pc.AllTasks(t, pc.MustPlanJSON(t, p))[0]
	if task.Name != "gtf_gene_filter" || task.Image != string(gtfgenefilter.DefaultImage) {
		t.Fatalf("task = %#v", task)
	}
	pc.AssertIOPath(t, task.Outputs, "filtered_gtf", "work/scrnaseq/reference/genes.filtered.gtf")
	bad := gtfgenefilter.Pipeline(gobble.PathSpec{Base: "genome", Ext: ".fa"}, gobble.PathSpec{Base: "genes", Ext: ".gtf"}, gtfgenefilter.Options{Options: modules.Options{ExtraArgs: []string{"--output"}}})
	if graph, err := gobble.Compose(bad); graph != nil || err == nil {
		t.Fatalf("ExtraArgs compose = (%v, %v), want defect", graph, err)
	}
}
