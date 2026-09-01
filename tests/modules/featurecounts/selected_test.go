package featurecounts_test

import (
	"errors"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	"github.com/HahyeonJeon/gobble/assets/modules/featurecounts"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

func TestSelectedFeatureCountsBiotypeStandalone(t *testing.T) {
	bam := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "aligned", Ext: ".bam"}
	gtf := gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genes", Ext: ".gtf"}
	p := featurecounts.BiotypePipeline(bam, gtf, featurecounts.BiotypeOptions{
		Options:      modules.Options{Resources: gobble.Resources{CPU: 2}},
		Strandedness: gobble.StrandednessReverse,
		Paired:       true,
	})
	task := pc.TaskByID(t, pc.MustPlanJSON(t, p), "featurecounts_biotype_qc")
	if task.Image != string(featurecounts.DefaultImage) {
		t.Fatalf("image = %q, want selected immutable image %q", task.Image, featurecounts.DefaultImage)
	}
	if !pc.ContainsAll(task.Command,
		"featureCounts", "-a", "in/genes.gtf", "-o", "work/featurecounts-biotype/biotype.featureCounts.tsv",
		"-s", "2", "-T", "2", "-p", "in/aligned.bam",
	) {
		t.Fatalf("command = %#v, want selected featureCounts biotype argv", task.Command)
	}
	pc.AssertIOPath(t, task.Outputs, "counts", "work/featurecounts-biotype/biotype.featureCounts.tsv")
	pc.AssertIOPath(t, task.Outputs, "summary", "work/featurecounts-biotype/biotype.featureCounts.tsv.summary")
}

func TestSelectedFeatureCountsATACRejectsUndeclaredBAMMembership(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
	}{
		{name: "bare BAM operand", args: []string{"in/undeclared.bam"}},
		{name: "BAM operand after switch", args: []string{"--primary", "in/undeclared.bam"}},
		{name: "option terminator before BAM", args: []string{"--", "in/undeclared.bam"}},
		{name: "missing value consumes typed BAM", args: []string{"-Q"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			graph, err := gobble.Compose(featureCountsATACPipeline(test.args))
			var composeErr *gobble.Error
			if graph != nil || !errors.As(err, &composeErr) || len(composeErr.Defects) != 1 || composeErr.Defects[0].Code != gobble.DefectInvalidValue || composeErr.Defects[0].Unit != "featurecounts_atac" {
				t.Fatalf("Compose(ExtraArgs %#v) = (%v, %#v), want one featurecounts_atac invalid-value defect", test.args, graph, err)
			}
		})
	}
}

func TestSelectedFeatureCountsATACAcceptsCompleteOptions(t *testing.T) {
	task := pc.TaskByID(t, pc.MustPlanJSON(t, featureCountsATACPipeline([]string{"-Q", "10", "--primary", "--minOverlap=1"})), "featurecounts_atac")
	wantTail := []string{"-Q", "10", "--primary", "--minOverlap=1", "in/a.bam", "in/b.bam"}
	if len(task.Command) < len(wantTail) {
		t.Fatalf("featureCounts command = %#v, want complete options followed by typed BAMs", task.Command)
	}
	gotTail := task.Command[len(task.Command)-len(wantTail):]
	for i := range wantTail {
		if gotTail[i] != wantTail[i] {
			t.Fatalf("featureCounts command tail = %#v, want %#v", gotTail, wantTail)
		}
	}
	if len(task.Inputs) != 3 || task.Inputs[1].Name != "bam_0" || task.Inputs[2].Name != "bam_1" {
		t.Fatalf("featureCounts inputs = %#v, want SAF plus two typed BAM members", task.Inputs)
	}
}

func featureCountsATACPipeline(extraArgs []string) *gobble.Pipeline {
	pipeline := gobble.NewPipeline("atac-featurecounts")
	saf := pipeline.AddInput("saf", gobble.Literal("in/consensus.saf"))
	bams := []gobble.Handle{
		pipeline.AddInput("bam_a", gobble.Literal("in/a.bam")),
		pipeline.AddInput("bam_b", gobble.Literal("in/b.bam")),
	}
	_, err := featurecounts.AddATAC(pipeline, bams, saf, featurecounts.ATACOptions{Options: modules.Options{ExtraArgs: extraArgs}})
	if err != nil {
		pipeline.RecordComposeError(err)
	}
	return pipeline
}
