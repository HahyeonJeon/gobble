package stop_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/rnaseqscenario"
)

func TestRNAUsesOrdinaryCancellableTasks(t *testing.T) {
	tasks := pc.AllTasks(t, rnaseqscenario.Plan(t, rnaseq.DefaultConfig()))
	for _, task := range tasks {
		for _, token := range task.Command {
			if token == "retry" || token == "cancel" {
				t.Fatalf("RNA task %s adds assay-specific control token %q", task.ID, token)
			}
		}
	}
	if !rnaseq.Lifecycle.Stop {
		t.Fatal("RNA stop participation is false")
	}
}
