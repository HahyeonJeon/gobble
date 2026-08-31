package anndatarconvert_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	anndatarconvert "github.com/HahyeonJeon/gobble/assets/modules/anndatar-convert"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestAnnDataRConvertDeclaresBothRDSFiles(t *testing.T) {
	p := anndatarconvert.Pipeline(gobble.PathSpec{Base: "sample", Ext: ".h5ad"}, anndatarconvert.Options{Prefix: "sample_raw_matrix"})
	task := pc.AllTasks(t, pc.MustPlanJSON(t, p))[0]
	if task.Name != "anndatar_convert" || task.Image != string(anndatarconvert.DefaultImage) || !pc.ContainsAll(task.Command, "Rscript", "-e") {
		t.Fatalf("task = %#v", task)
	}
	pc.AssertIOPath(t, task.Outputs, "seurat_rds", "results/scrnaseq/matrices/sample_raw_matrix.seurat.rds")
	pc.AssertIOPath(t, task.Outputs, "sce_rds", "results/scrnaseq/matrices/sample_raw_matrix.sce.rds")
}
