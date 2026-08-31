package stop_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble/assets/pipelines/atacseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	"github.com/HahyeonJeon/gobble/tests/scenarios/internal/atacseqscenario"
)

func TestATACStopParticipationCoversBranchAndCohortWork(t *testing.T) {
	tasks := pc.AllTasks(t, atacseqscenario.Plan(t, atacseq.DefaultConfig()))
	pc.MustHaveTaskID(t, tasks, "OSMOTIC_STRESS_T15_PE.replicate_1.run_002.bwa_mem")
	pc.MustHaveTaskID(t, tasks, "consensus.replicates.featurecounts_merge_matrices")
	pc.MustHaveTaskID(t, tasks, "ataqv.ataqv_mkarv")
	if !atacseq.Lifecycle.Stop {
		t.Fatal("ATAC stop participation is false")
	}
}
