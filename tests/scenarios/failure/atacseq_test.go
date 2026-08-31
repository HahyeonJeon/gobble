package failure_test

import (
	"errors"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/atacseq"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/atacseqscenario"
)

func TestATACMissingControlFailsBeforeRun(t *testing.T) {
	samples, _ := atacseqscenario.Samples(t)
	samples[0].Replicates[0].Control = &atacseq.ControlRef{Sample: "MISSING", Replicate: 1}
	graph, err := gobble.Compose(atacseq.Build(samples, atacseq.DefaultConfig()))
	var structured *gobble.Error
	if graph != nil || !errors.As(err, &structured) || structured == nil || !atacseq.Lifecycle.Failure {
		t.Fatalf("Compose(missing ATAC control) = (%v, %#v), want structured failure", graph, err)
	}
}
