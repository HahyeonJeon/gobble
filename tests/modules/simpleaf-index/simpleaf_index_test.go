package simpleafindex_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	simpleafindex "github.com/HahyeonJeon/gobble/assets/modules/simpleaf-index"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestSimpleafIndexDeclaresCompleteTree(t *testing.T) {
	p := simpleafindex.Pipeline(gobble.PathSpec{Base: "transcripts", Ext: ".fa"}, simpleafindex.Options{})
	task := pc.AllTasks(t, pc.MustPlanJSON(t, p))[0]
	if task.Name != "simpleaf_index" || task.Image != string(simpleafindex.DefaultImage) || !strings.Contains(task.Script, "'simpleaf' 'index'") || !strings.Contains(task.Script, "'simpleaf' 'set-paths'") {
		t.Fatalf("task = %#v", task)
	}
	pc.AssertTreeIO(t, task.Outputs, "index", "results/scrnaseq/reference/simpleaf_index/index")
}

func TestSimpleafIndexRejectsEveryOwnedOptionAlias(t *testing.T) {
	tests := []struct {
		name  string
		extra string
	}{
		{name: "threads long", extra: "--threads=1"},
		{name: "threads short", extra: "-t1"},
		{name: "direct reference long", extra: "--ref-seq=other.fa"},
		{name: "direct reference long alias", extra: "--refseq=other.fa"},
		{name: "FASTA long", extra: "--fasta=other.fa"},
		{name: "FASTA short", extra: "-fother.fa"},
		{name: "GTF long", extra: "--gtf=other.gtf"},
		{name: "GTF short", extra: "-gother.gtf"},
		{name: "GFF3 format", extra: "--gff3-format"},
		{name: "read length long", extra: "--rlen=100"},
		{name: "read length short", extra: "-r100"},
		{name: "deduplicate", extra: "--dedup"},
		{name: "spliced sequence", extra: "--spliced=other.fa"},
		{name: "unspliced sequence", extra: "--unspliced=other.fa"},
		{name: "feature CSV", extra: "--feature-csv=features.csv"},
		{name: "probe CSV", extra: "--probe-csv=probes.csv"},
		{name: "output long", extra: "--output=elsewhere"},
		{name: "output short", extra: "-oelsewhere"},
		{name: "disable piscem", extra: "--no-piscem"},
		{name: "select piscem", extra: "--use-piscem"},
		{name: "sparse salmon index long", extra: "--sparse"},
		{name: "sparse salmon index short", extra: "-p"},
		{name: "selective alignment route", extra: "--use-selective-alignment"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bad := simpleafindex.Pipeline(
				gobble.PathSpec{Base: "transcripts", Ext: ".fa"},
				simpleafindex.Options{Options: modules.Options{ExtraArgs: []string{test.extra}}},
			)
			graph, err := gobble.Compose(bad)
			var composeErr *gobble.Error
			if graph != nil || !errors.As(err, &composeErr) || len(composeErr.Defects) != 1 || composeErr.Defects[0].Code != gobble.DefectInvalidValue || composeErr.Defects[0].Unit != "simpleaf_index" {
				t.Fatalf("Compose() with ExtraArgs %q returned graph=%t, error=%v; want one simpleaf_index invalid-value defect", test.extra, graph != nil, err)
			}
		})
	}
}
