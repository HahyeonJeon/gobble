// Package commandcheck provides common command-module contract assertions.
package commandcheck

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

// Task composes one standalone module and checks its bounded task contract.
func Task(t *testing.T, pipeline *gobble.Pipeline, id string) pc.Task {
	t.Helper()
	task := pc.TaskByID(t, pc.MustPlanJSON(t, pipeline), id)
	if task.Image == "" || !strings.Contains(task.Image, "@sha256:") {
		t.Fatalf("task %s image = %q, want immutable image", id, task.Image)
	}
	if task.Resources.CPU <= 0 || len(task.Resources.Memory) == 0 {
		t.Fatalf("task %s resources = %+v, want bounded CPU and memory", id, task.Resources)
	}
	if len(task.Command) == 0 && task.Script == "" {
		t.Fatalf("task %s has no command or script", id)
	}
	return task
}

// Invalid checks that an invalid module option fails before a plan exists.
func Invalid(t *testing.T, pipeline *gobble.Pipeline) {
	t.Helper()
	graph, err := gobble.Compose(pipeline)
	if graph != nil || err == nil {
		t.Fatalf("Compose(invalid module) = (%v, %v), want nil graph and structured error", graph, err)
	}
}
