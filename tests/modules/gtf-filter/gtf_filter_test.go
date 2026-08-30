package gtffilter_test

import (
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	gtffilter "github.com/HahyeonJeon/gobble/assets/modules/gtf-filter"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContract(t *testing.T) {
	gtf := gobble.Literal("in/genes.gtf")
	fasta := gobble.Literal("in/genome.fasta")
	task := cc.Task(t, gtffilter.Pipeline(gtf, fasta, gtffilter.Options{}), "gtf_filter")
	if !pc.ContainsAll(task.Command, "python", "-c", "in/genes.gtf", "in/genome.fasta", "work/reference/genes.filtered.gtf") {
		t.Fatalf("command = %#v, want exact GTF, FASTA, and filtered output operands", task.Command)
	}
	if script := task.Command[2]; !strings.Contains(script, "sequence_names") || !strings.Contains(script, "transcript_id") || !strings.Contains(script, "kept == 0") {
		t.Fatalf("filter script omits sequence, transcript-id, or empty-result validation: %s", script)
	}
	pc.AssertIOPath(t, task.Outputs, "filtered_gtf", "work/reference/genes.filtered.gtf")
	cc.Invalid(t, gtffilter.Pipeline(gtf, fasta, gtffilter.Options{Options: modules.Options{ExtraArgs: []string{"--skip_transcript_id_check"}}}))
	cc.Invalid(t, gtffilter.Pipeline(gtf, fasta, gtffilter.Options{Options: modules.Options{Image: "alpine:latest"}}))
}
