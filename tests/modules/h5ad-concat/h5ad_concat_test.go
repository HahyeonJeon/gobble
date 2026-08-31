package h5adconcat_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	h5adconcat "github.com/HahyeonJeon/gobble/assets/modules/h5ad-concat"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestH5ADConcatRequiresCompleteUniqueMembership(t *testing.T) {
	p := h5adconcat.Pipeline([]gobble.PathSpec{{Base: "a", Ext: ".h5ad"}, {Base: "b", Ext: ".h5ad"}}, h5adconcat.Options{Labels: []string{"A", "B"}})
	task := pc.AllTasks(t, pc.MustPlanJSON(t, p))[0]
	if task.Name != "h5ad_concat" || task.Image != string(h5adconcat.DefaultImage) || !pc.ContainsAll(task.Command, "A", "a.h5ad", "B", "b.h5ad") {
		t.Fatalf("task = %#v", task)
	}
	pc.AssertIOPath(t, task.Outputs, "h5ad", "results/scrnaseq/matrices/combined_raw_matrix.h5ad")
	bad := h5adconcat.Pipeline([]gobble.PathSpec{{Base: "a", Ext: ".h5ad"}, {Base: "b", Ext: ".h5ad"}}, h5adconcat.Options{Labels: []string{"A"}})
	if graph, err := gobble.Compose(bad); graph != nil || err == nil {
		t.Fatalf("incomplete membership compose = (%v, %v), want defect", graph, err)
	}
}
