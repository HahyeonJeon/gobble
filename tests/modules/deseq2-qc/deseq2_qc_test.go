package deseq2qc_test

import (
	"slices"
	"strings"
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
	if !pc.ContainsAll(task.Command, "Rscript", "-e", "in/counts.tsv", "results/deseq2-qc/pca.pdf", "results/deseq2-qc/sample_distance.pdf") || slices.Contains(task.Command, "--") || pc.ContainsAll(task.Command, "--contrast") {
		t.Fatalf("command = %#v, want QC-only DESeq2 argv", task.Command)
	}
	if task.Image != string(deseq2qc.DefaultImage) || !pc.ContainsAll([]string{task.Image}, "community.wave.seqera.io/library/r-base_r-optparse_r-ggplot2_r-rcolorbrewer_pruned:9e75394d0bc21987@sha256:afd00df7ce26f38ecb2a063f65d441fc20c0803e5c7319ee5cbe3a23732a30dd") {
		t.Fatalf("image = %q, want exact nf-core/rnaseq 3.26.0 DESeq2-QC image", task.Image)
	}
	if !strings.Contains(task.Command[2], "varianceStabilizingTransformation") || strings.Contains(task.Command[2], "vst(dds") {
		t.Fatalf("R script does not support the official small-data gene count: %s", task.Command[2])
	}
	pc.AssertIOPath(t, task.Outputs, "matrix", "results/deseq2-qc/vst_matrix.tsv")
	cc.Invalid(t, deseq2qc.Pipeline(matrix, deseq2qc.Options{Options: modules.Options{Image: "alpine:latest"}}))
}
