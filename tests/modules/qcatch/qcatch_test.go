package qcatch_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	"github.com/HahyeonJeon/gobble/assets/modules/qcatch"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestQCatchTypedChemistryOutputsAndConflicts(t *testing.T) {
	p := qcatch.Pipeline(gobble.DeclareTree(gobble.Dir("in/quant")), qcatch.Options{Chemistry: "10X_3p_v2", RemoveDoublets: true, VisualizeDoublets: true})
	task := pc.AllTasks(t, pc.MustPlanJSON(t, p))[0]
	if task.Name != "qcatch" || task.Image != string(qcatch.DefaultImage) || !pc.ContainsAll(task.Command, "--chemistry", "10X_3p_v2", "--remove_doublets", "--visualize_doublets") {
		t.Fatalf("task = %#v", task)
	}
	pc.AssertTreeIO(t, task.Inputs, "quant", "in/quant")
	pc.AssertIOPath(t, task.Outputs, "filtered_h5ad", "results/scrnaseq/qcatch/filtered_quants.h5ad")
	for _, options := range []qcatch.Options{
		{Chemistry: "10X_3p_v2", NPartitions: 10},
		{Chemistry: "10X_3p_v2", VisualizeDoublets: true},
		{Options: modules.Options{ExtraArgs: []string{"-iother"}}, Chemistry: "10X_3p_v2"},
		{Options: modules.Options{ExtraArgs: []string{"--export_summary_table"}}, Chemistry: "10X_3p_v2"},
		{Options: modules.Options{ExtraArgs: []string{"-e"}}, Chemistry: "10X_3p_v2"},
		{Options: modules.Options{ExtraArgs: []string{"-eother"}}, Chemistry: "10X_3p_v2"},
	} {
		if graph, err := gobble.Compose(qcatch.Pipeline(gobble.DeclareTree(gobble.Dir("in/quant")), options)); graph != nil || err == nil {
			t.Fatalf("invalid QCatch compose = (%v, %v), want defect", graph, err)
		}
	}

	xTask := pc.AllTasks(t, pc.MustPlanJSON(t, qcatch.Pipeline(
		gobble.DeclareTree(gobble.Dir("in/quant")),
		qcatch.Options{Options: modules.Options{ExtraArgs: []string{"-x"}}, Chemistry: "10X_3p_v2"},
	)))[0]
	if !pc.ContainsAll(xTask.Command, "-x") {
		t.Fatalf("QCatch command = %#v, want non-alias ExtraArgs -x", xTask.Command)
	}
}
