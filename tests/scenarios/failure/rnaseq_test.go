package failure_test

import (
	"errors"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/rnaseqscenario"
)

func TestRNAInvalidRouteFlagFailsBeforeRun(t *testing.T) {
	samples, _ := rnaseqscenario.Samples(t)
	config := rnaseq.DefaultConfig()
	config.STAR.ExtraArgs = []string{"--outSAMtype", "SAM"}
	graph, err := gobble.Compose(rnaseq.Build(samples, config))
	var structured *gobble.Error
	if graph != nil || !errors.As(err, &structured) || structured == nil || !rnaseq.Lifecycle.Failure {
		t.Fatalf("invalid RNA route Compose() = (%v, %v), want structured pre-run failure", graph, err)
	}
}
