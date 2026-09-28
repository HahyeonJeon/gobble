package trimgalore_test

import (
	gobble "github.com/HahyeonJeon/gobble"
	trimgalore "github.com/HahyeonJeon/gobble/assets/modules/trim-galore"
	"testing"
)

func TestInspectionSettings(t *testing.T) {
	for _, quality := range []int{0, 25} {
		p := trimgalore.Pipeline(gobble.PathSpec{Base: "sample", Ext: ".fq"}, gobble.PathSpec{}, trimgalore.Options{Quality: quality, Length: 40})
		g, err := gobble.Compose(p)
		if err != nil {
			t.Fatal(err)
		}
		flow, err := gobble.InspectPipeline(g)
		if err != nil {
			t.Fatal(err)
		}
		fields := flow.Steps[0].Settings
		if len(fields) != 2 || fields[0].Key != "quality" || fields[0].Label != "Quality threshold" || fields[1].Key != "length" || *fields[1].Value != 40 {
			t.Fatalf("metadata: %+v", fields)
		}
		if quality == 0 && fields[0].Value != nil {
			t.Fatal("invented tool default")
		}
		if quality != 0 && (fields[0].Value == nil || *fields[0].Value != quality) {
			t.Fatal("quality option lost")
		}
	}
}
