package simpleafquant_test

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	simpleafquant "github.com/HahyeonJeon/gobble/assets/modules/simpleaf-quant"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestSimpleafQuantBindsCompleteTreesAndProtectsInputs(t *testing.T) {
	p := simpleafquant.Pipeline(
		gobble.DeclareTree(gobble.Dir("in/index")),
		gobble.PathSpec{Base: "t2g", Ext: ".tsv"},
		gobble.PathSpec{Base: "whitelist", Ext: ".txt.gz"},
		gobble.PathSpec{Base: "r1", Ext: ".fastq.gz"},
		gobble.PathSpec{Base: "r2", Ext: ".fastq.gz"},
		simpleafquant.Options{Chemistry: "10xv2", Resolution: "cr-like"},
	)
	task := pc.AllTasks(t, pc.MustPlanJSON(t, p))[0]
	if task.Name != "simpleaf_quant" || task.Image != string(simpleafquant.DefaultImage) || !strings.Contains(task.Script, "'--unfiltered-pl'") {
		t.Fatalf("task = %#v", task)
	}
	pc.AssertTreeIO(t, task.Inputs, "index", "in/index")
	pc.AssertTreeIO(t, task.Outputs, "map", "results/scrnaseq/simpleaf/af_map")
	pc.AssertTreeIO(t, task.Outputs, "quant", "results/scrnaseq/simpleaf/af_quant")
	badOptions := simpleafquant.Options{Options: modules.Options{ExtraArgs: []string{"--map-dir=other"}}, Chemistry: "10xv2", Resolution: "cr-like"}
	bad := simpleafquant.Pipeline(gobble.DeclareTree(gobble.Dir("in/index")), gobble.PathSpec{Base: "t2g"}, gobble.PathSpec{Base: "wl"}, gobble.PathSpec{Base: "r1"}, gobble.PathSpec{Base: "r2"}, badOptions)
	if graph, err := gobble.Compose(bad); graph != nil || err == nil {
		t.Fatalf("protected map input compose = (%v, %v), want defect", graph, err)
	}
}
