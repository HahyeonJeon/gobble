package modules_test

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	"testing"
)

type captureParent struct{ spec gobble.TaskSpec }

func (p *captureParent) AddTask(spec gobble.TaskSpec) *gobble.Task { p.spec = spec; return nil }
func TestDisplayDefaultsPreserveTaskOverrides(t *testing.T) {
	capture := &captureParent{}
	owners := []string{"S01"}
	parent := modules.WithDisplay(capture, gobble.TaskDisplay{Stage: "Default", Samples: owners, Scope: gobble.DisplaySample})
	owners[0] = "mutated"
	parent.AddTask(gobble.TaskSpec{Display: gobble.TaskDisplay{Stage: "Alignment"}})
	if d := capture.spec.Display; d.Stage != "Alignment" || len(d.Samples) != 1 || d.Samples[0] != "S01" {
		t.Fatalf("task display overwritten: %+v", d)
	}
	capture.spec.Display.Samples[0] = "changed child"
	parent.AddTask(gobble.TaskSpec{})
	if capture.spec.Display.Samples[0] != "S01" {
		t.Fatal("child mutated future defaults")
	}
	parent.AddTask(gobble.TaskSpec{Display: gobble.TaskDisplay{Scope: gobble.DisplayCohort}})
	if d := capture.spec.Display; d.Scope != gobble.DisplayCohort || len(d.Samples) != 0 {
		t.Fatalf("cohort inherited sample owner: %+v", d)
	}
}
