package matrixtoh5ad_test

import (
	"strings"
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
	if len(task.Command) < 3 || task.Command[0] != "python" || task.Command[1] != "-c" {
		t.Fatalf("command = %q, want embedded Python conversion", task.Command)
	}
	last := -1
	for _, step := range []string{
		`adata.var["gene_versions"] = adata.var["gene_id"]`,
		`adata.var.index = adata.var["gene_versions"].str.split(".").str[0].values`,
		`simpleaf_map_info = json.loads(adata.uns["simpleaf_map_info"])`,
		`simpleaf_map_info.pop("runtime_seconds")`,
		`adata.uns["simpleaf_map_info"] = json.dumps(simpleaf_map_info, sort_keys=True)`,
	} {
		position := strings.Index(task.Command[2], step)
		if position <= last {
			t.Fatalf("conversion step %q position = %d after %d; script:\n%s", step, position, last, task.Command[2])
		}
		last = position
	}
	pc.AssertTreeIO(t, task.Inputs, "quant", "in/quant")
	pc.AssertIOPath(t, task.Outputs, "h5ad", "results/scrnaseq/matrices/sample_raw_matrix.h5ad")
}
