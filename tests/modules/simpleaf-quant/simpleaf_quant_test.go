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
	for _, extra := range []string{"--map-dir=other", "--no-piscem", "--use-selective-alignment"} {
		badOptions := simpleafquant.Options{Options: modules.Options{ExtraArgs: []string{extra}}, Chemistry: "10xv2", Resolution: "cr-like"}
		bad := simpleafquant.Pipeline(gobble.DeclareTree(gobble.Dir("in/index")), gobble.PathSpec{Base: "t2g"}, gobble.PathSpec{Base: "wl"}, gobble.PathSpec{Base: "r1"}, gobble.PathSpec{Base: "r2"}, badOptions)
		if graph, err := gobble.Compose(bad); graph != nil || err == nil {
			t.Errorf("protected ExtraArgs %q compose = (%v, %v), want defect", extra, graph, err)
		}
	}
}

func TestSimpleafQuantStandaloneRejectsAlignerExtraArg(t *testing.T) {
	options := simpleafquant.Options{
		Options:    modules.Options{ExtraArgs: []string{"--aligner=star"}},
		Chemistry:  "10xv2",
		Resolution: "cr-like",
	}
	pipeline := simpleafquant.Pipeline(
		gobble.DeclareTree(gobble.Dir("in/index")),
		gobble.PathSpec{Base: "t2g"},
		gobble.PathSpec{Base: "whitelist"},
		gobble.PathSpec{Base: "r1"},
		gobble.PathSpec{Base: "r2"},
		options,
	)
	if graph, err := gobble.Compose(pipeline); graph != nil || err == nil {
		t.Fatalf("standalone Compose() = (%v, %v), want protected --aligner defect", graph, err)
	}
}

func TestSimpleafQuantRejectsProtocolAndTypedOptionAliases(t *testing.T) {
	tests := []struct {
		name  string
		extra string
	}{
		{name: "expected orientation long", extra: "--expected-ori=both"},
		{name: "expected orientation short attached", extra: "-dboth"},
		{name: "index short attached", extra: "-iother"},
		{name: "chemistry short attached", extra: "-c10xv4-3p"},
		{name: "read one short attached", extra: "-1other.fastq.gz"},
		{name: "read two short attached", extra: "-2other.fastq.gz"},
		{name: "resolution short attached", extra: "-rparsimony"},
		{name: "threads short attached", extra: "-t1"},
		{name: "permit list short attached", extra: "-uother.txt"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := simpleafquant.Options{
				Options:    modules.Options{ExtraArgs: []string{test.extra}},
				Chemistry:  "10xv2",
				Resolution: "cr-like",
			}
			pipeline := simpleafquant.Pipeline(
				gobble.DeclareTree(gobble.Dir("in/index")),
				gobble.PathSpec{Base: "t2g"},
				gobble.PathSpec{Base: "whitelist"},
				gobble.PathSpec{Base: "r1"},
				gobble.PathSpec{Base: "r2"},
				options,
			)
			if graph, err := gobble.Compose(pipeline); graph != nil || err == nil {
				t.Fatalf("Compose() with ExtraArgs %q = (%v, %v), want protected-option defect", test.extra, graph, err)
			}
		})
	}
}
