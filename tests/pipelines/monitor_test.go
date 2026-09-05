package pipelineevidence_test

import (
	"encoding/json"
	"github.com/HahyeonJeon/gobble"
	"testing"
)

func TestAllProductTasksHaveMonitorOwnership(t *testing.T) {
	root := familyModuleRoot(t)
	for name, build := range map[string]func(*testing.T, string) []byte{"rnaseq": rnaseqFamilyPlan, "scrnaseq": scrnaseqFamilyPlan, "atacseq": atacseqFamilyPlan, "methylseq": methylseqFamilyPlan, "wgs": wgsFamilyPlan} {
		t.Run(name, func(t *testing.T) {
			var plan struct {
				Tasks []struct {
					ID      string
					Display gobble.TaskDisplay
				}
			}
			if err := json.Unmarshal(build(t, root), &plan); err != nil {
				t.Fatal(err)
			}
			samples := map[string]bool{}
			scopes := map[string]int{}
			for _, task := range plan.Tasks {
				d := task.Display
				scopes[d.Scope]++
				switch d.Scope {
				case gobble.DisplaySample:
					if len(d.Samples) == 0 {
						t.Errorf("%s has no sample ID", task.ID)
					}
					for _, id := range d.Samples {
						samples[id] = true
					}
				case gobble.DisplayShared, gobble.DisplayCohort:
					if len(d.Samples) > 0 {
						t.Errorf("%s incorrectly enters sample completion", task.ID)
					}
				default:
					t.Errorf("%s has no valid monitor scope: %q", task.ID, d.Scope)
				}
			}
			if len(samples) < 2 || scopes[gobble.DisplayCohort] == 0 {
				t.Fatal("sample or cohort work missing")
			}
		})
	}
}
