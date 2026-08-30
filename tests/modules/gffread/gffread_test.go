package gffread_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	"github.com/HahyeonJeon/gobble/assets/modules/gffread"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
	cc "github.com/HahyeonJeon/gobble/tests/modules/internal/commandcheck"
)

func TestCommandContracts(t *testing.T) {
	gtf, fasta := gobble.Literal("in/genes.gtf"), gobble.Literal("in/genome.fa")
	transcript := cc.Task(t, gffread.TranscriptomePipeline(gtf, fasta, gffread.Options{}), "gffread_transcriptome")
	if !pc.ContainsAll(transcript.Command, "gffread", "in/genes.gtf", "-g", "in/genome.fa", "-w", "work/reference/reference.transcripts.fasta") {
		t.Fatalf("transcript command = %#v", transcript.Command)
	}
	bed := cc.Task(t, gffread.BEDPipeline(gtf, fasta, gffread.Options{}), "gffread_bed")
	pc.AssertIOPath(t, bed.Outputs, "output", "work/reference/reference.genes.bed")
	cc.Invalid(t, gffread.TranscriptomePipeline(gtf, fasta, gffread.Options{Options: modules.Options{Image: "alpine:latest"}}))
}
