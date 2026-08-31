package matrixtoh5ad_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	matrixtoh5ad "github.com/HahyeonJeon/gobble/assets/modules/matrix-to-h5ad"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestMatrixToH5ADConsumesQuantTreeAndPublishesOneFile(t *testing.T) {
	p := matrixtoh5ad.Pipeline(gobble.DeclareTree(gobble.Dir("in/quant")), matrixtoh5ad.Options{Sample: "sample", ExpectedCells: 5000, SeqCenter: "center"})
	task := pc.AllTasks(t, pc.MustPlanJSON(t, p))[0]
	if task.Name != "matrix_to_h5ad" || task.Image != string(matrixtoh5ad.DefaultImage) || !pc.ContainsAll(task.Command, "sample", "5000", "center") {
		t.Fatalf("task = %#v", task)
	}
	pc.AssertTreeIO(t, task.Inputs, "quant", "in/quant")
	pc.AssertIOPath(t, task.Outputs, "h5ad", "results/scrnaseq/matrices/sample_raw_matrix.h5ad")
}
