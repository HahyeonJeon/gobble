package stop_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/wgs"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/wgsscenario"
)

func TestWGSStopParticipationCoversIntervalWork(t *testing.T) {
	tasks := pc.AllTasks(t, wgsscenario.Plan(t, wgs.DefaultConfig()))
	pc.MustHaveTaskID(t, tasks, "bqsr_intervals.patient1.testN.gatk4_applybqsr")
	pc.MustHaveTaskID(t, tasks, "joint_intervals.genotype.gatk4_genotypegvcfs")
	if !wgs.Lifecycle().Stop {
		t.Fatal("WGS stop participation is false")
	}
}
