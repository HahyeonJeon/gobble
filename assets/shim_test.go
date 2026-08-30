package assets_test

import (
	"bytes"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets"
	"github.com/HahyeonJeon/gobble/assets/pipelines/methylseq"
	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
)

func TestConstructorShimsMatchPipelineOwners(t *testing.T) {
	tests := []struct {
		name  string
		sheet string
		shim  func() *gobble.Pipeline
		owner func() *gobble.Pipeline
	}{
		{name: "wgs", shim: assets.WGS, owner: wgs.Pipeline},
		{name: "rnaseq", sheet: "../tests/pipelines/rnaseq/testdata/rnaseq-samplesheet.csv", shim: assets.RNASeq, owner: rnaseq.Pipeline},
		{name: "methylseq", sheet: "../tests/pipelines/methylseq/testdata/methylseq-samplesheet.csv", shim: assets.MethylSeq, owner: methylseq.Pipeline},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.sheet != "" {
				previous := gobble.SampleSheetPath()
				gobble.SetSampleSheetPath(test.sheet)
				t.Cleanup(func() { gobble.SetSampleSheetPath(previous) })
			}
			got := planJSON(t, test.shim())
			want := planJSON(t, test.owner())
			if !bytes.Equal(got, want) {
				t.Fatalf("shim plan differs from %s pipeline owner", test.name)
			}
		})
	}
}

func planJSON(t *testing.T, pipeline *gobble.Pipeline) []byte {
	t.Helper()
	graph, err := gobble.Compose(pipeline)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	plan, err := gobble.BuildPlan(graph)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	data, err := plan.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	return data
}
