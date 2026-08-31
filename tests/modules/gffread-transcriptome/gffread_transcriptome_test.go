package gffreadtranscriptome_test

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	gffreadtranscriptome "github.com/HahyeonJeon/gobble/assets/modules/gffread-transcriptome"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestGFFReadTranscriptomeCommandAndAliasBoundary(t *testing.T) {
	p := gffreadtranscriptome.Pipeline(gobble.PathSpec{Base: "genes", Ext: ".gtf"}, gobble.PathSpec{Base: "genome", Ext: ".fa"}, gffreadtranscriptome.Options{})
	task := pc.AllTasks(t, pc.MustPlanJSON(t, p))[0]
	if task.Name != "gffread_transcriptome" || task.Image != string(gffreadtranscriptome.DefaultImage) || !pc.ContainsAll(task.Command, "gffread", "-F", "-w", "-g") {
		t.Fatalf("task = %#v", task)
	}
	pc.AssertIOPath(t, task.Outputs, "transcript_fasta", "work/scrnaseq/reference/transcripts.fasta")
	bad := gffreadtranscriptome.Pipeline(gobble.PathSpec{Base: "genes", Ext: ".gtf"}, gobble.PathSpec{Base: "genome", Ext: ".fa"}, gffreadtranscriptome.Options{Options: modules.Options{ExtraArgs: []string{"-wother.fa"}}})
	if graph, err := gobble.Compose(bad); graph != nil || err == nil {
		t.Fatalf("protected alias compose = (%v, %v), want defect", graph, err)
	}
}
