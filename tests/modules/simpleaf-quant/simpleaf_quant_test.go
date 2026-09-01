package simpleafquant_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	simpleafquant "github.com/HahyeonJeon/gobble/assets/modules/simpleaf-quant"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestSimpleafQuantBindsCompleteTrees(t *testing.T) {
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
}

func TestSimpleafQuantRejectsEveryOwnedOptionAlias(t *testing.T) {
	tests := []struct {
		name  string
		extra string
	}{
		{name: "mapped directory", extra: "--map-dir=other"},
		{name: "index long", extra: "--index=other"},
		{name: "index short", extra: "-iother"},
		{name: "transcript relation long", extra: "--t2g-map=other.tsv"},
		{name: "transcript relation short", extra: "-mother.tsv"},
		{name: "chemistry long", extra: "--chemistry=10xv4-3p"},
		{name: "chemistry short", extra: "-c10xv4-3p"},
		{name: "read one long", extra: "--reads1=other.fastq.gz"},
		{name: "read one short", extra: "-1other.fastq.gz"},
		{name: "read two long", extra: "--reads2=other.fastq.gz"},
		{name: "read two short", extra: "-2other.fastq.gz"},
		{name: "resolution long", extra: "--resolution=parsimony"},
		{name: "resolution short", extra: "-rparsimony"},
		{name: "output long", extra: "--output=elsewhere"},
		{name: "output short", extra: "-oelsewhere"},
		{name: "threads long", extra: "--threads=1"},
		{name: "threads short", extra: "-t1"},
		{name: "anndata output", extra: "--anndata-out"},
		{name: "knee filtering long", extra: "--knee"},
		{name: "knee filtering short", extra: "-k"},
		{name: "forced cells long", extra: "--forced-cells=100"},
		{name: "forced cells short", extra: "-f100"},
		{name: "explicit permit list long", extra: "--explicit-pl=other.txt"},
		{name: "explicit permit list short", extra: "-xother.txt"},
		{name: "expected cells long", extra: "--expect-cells=100"},
		{name: "expected cells short", extra: "-e100"},
		{name: "unfiltered permit list long", extra: "--unfiltered-pl=other.txt"},
		{name: "unfiltered permit list short", extra: "-uother.txt"},
		{name: "expected orientation long", extra: "--expected-ori=both"},
		{name: "expected orientation short", extra: "-dboth"},
		{name: "minimum reads", extra: "--min-reads=1"},
		{name: "disable piscem", extra: "--no-piscem"},
		{name: "select piscem", extra: "--use-piscem"},
		{name: "selective alignment long", extra: "--use-selective-alignment"},
		{name: "selective alignment short", extra: "-s"},
		{name: "generic aligner route", extra: "--aligner=star"},
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
			graph, err := gobble.Compose(pipeline)
			var composeErr *gobble.Error
			if graph != nil || !errors.As(err, &composeErr) || len(composeErr.Defects) != 1 || composeErr.Defects[0].Code != gobble.DefectInvalidValue || composeErr.Defects[0].Unit != "simpleaf_quant" {
				t.Fatalf("Compose() with ExtraArgs %q returned graph=%t, error=%v; want one simpleaf_quant invalid-value defect", test.extra, graph != nil, err)
			}
		})
	}
}
