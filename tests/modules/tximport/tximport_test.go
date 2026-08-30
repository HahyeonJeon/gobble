package tximport_test

import (
	"slices"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	"github.com/HahyeonJeon/gobble/assets/modules/tximport"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContract(t *testing.T) {
	quants := []gobble.PathSpec{gobble.Literal("in/a/quant.sf"), gobble.Literal("in/b/quant.sf")}
	gtf := gobble.Literal("in/genes.gtf")
	task := cc.Task(t, tximport.Pipeline(quants, []string{"a", "b"}, gtf, tximport.Options{}), "tximport")
	if !pc.ContainsAll(task.Command, "Rscript", "-e", "in/genes.gtf", "results/salmon", "a", "in/a/quant.sf", "b", "in/b/quant.sf") || slices.Contains(task.Command, "--") {
		t.Fatalf("command = %#v, want ordered required cohort", task.Command)
	}
	pc.AssertIOPath(t, task.Outputs, "gene_length_scaled", "results/salmon/gene_counts_length_scaled.tsv")
	cc.Invalid(t, tximport.Pipeline(quants, []string{"a"}, gtf, tximport.Options{}))
	cc.Invalid(t, tximport.Pipeline(quants, []string{"a", "b"}, gtf, tximport.Options{Options: modules.Options{Image: "alpine:latest"}}))
}
