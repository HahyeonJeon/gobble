package failure_test

import (
	"errors"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/wgsscenario"
)

func TestWGSIncompleteIntervalMembershipFailsBeforeRun(t *testing.T) {
	samples, _ := wgsscenario.Samples(t)
	config := wgs.DefaultConfig()
	config.Reference.Intervals = nil
	graph, err := gobble.Compose(wgs.Build(samples, config))
	var structured *gobble.Error
	if graph != nil || !errors.As(err, &structured) || structured == nil || !wgs.Lifecycle.Failure {
		t.Fatalf("Compose(incomplete WGS intervals) = (%v, %#v), want structured failure", graph, err)
	}
}
