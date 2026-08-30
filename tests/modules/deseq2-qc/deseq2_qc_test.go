package deseq2qc_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	deseq2qc "github.com/HahyeonJeon/gobble/assets/modules/deseq2-qc"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContract(t *testing.T) {
	matrix := gobble.Literal("in/counts.tsv")
	task := cc.Task(t, deseq2qc.Pipeline(matrix, deseq2qc.Options{}), "deseq2_qc")
	if !pc.ContainsAll(task.Command, "Rscript", "-e", "--", "in/counts.tsv", "results/deseq2-qc/pca.pdf", "results/deseq2-qc/sample_distance.pdf") || pc.ContainsAll(task.Command, "--contrast") {
		t.Fatalf("command = %#v, want QC-only DESeq2 argv", task.Command)
	}
	pc.AssertIOPath(t, task.Outputs, "matrix", "results/deseq2-qc/vst_matrix.tsv")
	cc.Invalid(t, deseq2qc.Pipeline(matrix, deseq2qc.Options{Options: modules.Options{Image: "alpine:latest"}}))
}
