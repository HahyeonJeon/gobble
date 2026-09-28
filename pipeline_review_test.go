package gobble_test

import (
	"reflect"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	trim "github.com/HahyeonJeon/gobble/assets/modules/trim-galore"
	"github.com/HahyeonJeon/gobble/internal/pipelinereview"
)

func checkedReview(t *testing.T, quality int, add bool, extra []string) gobble.PipelineReview {
	t.Helper()
	p := gobble.NewPipeline("review_fixture")
	reads := p.AddInput("reads", gobble.PathSpec{Base: "sample", Ext: ".fastq"})
	ports, err := trim.Add(modules.WithDisplay(p, gobble.TaskDisplay{Stage: "Trim adapters"}), reads, gobble.Handle{}, trim.Options{Options: modules.Options{ExtraArgs: extra}, Quality: quality, Length: 40})
	if err != nil {
		t.Fatal(err)
	}
	p.AddTask(gobble.TaskSpec{Name: "align", Command: []string{"must-not-run"}, Inputs: []gobble.Bind{{Name: "reads", From: ports.Read1}}, Outputs: []gobble.Bind{{Name: "alignment", Spec: gobble.Literal("aligned.bam")}}})
	if add {
		if _, err := fastqc.Add(p, ports.Read1, fastqc.Options{}); err != nil {
			t.Fatal(err)
		}
	}
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatal(err)
	}
	out, err := gobble.InspectPipelineReview(g)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestPipelineReviewSettingAndBranch(t *testing.T) {
	base := checkedReview(t, 25, false, nil)
	next := checkedReview(t, 30, true, nil)
	if base.Definition.Steps[0].Recipe != "trim-galore-v1" {
		t.Fatalf("trim recipe not qualified: %+v", base.Definition.Steps[0])
	}
	if next.Definition.Steps[2].Recipe != "fastqc-v1" {
		t.Fatalf("FastQC recipe not qualified: %+v", next.Definition.Steps[2])
	}
	comparison := pipelinereview.Compare(base.Definition, next.Definition)
	if len(comparison.Gaps) != 0 || len(comparison.Changes) != 2 {
		t.Fatalf("comparison: %+v", comparison)
	}
	c := comparison.Changes[0]
	if c.Kind != "setting" || c.Key != "quality" || c.Before == nil || *c.Before != 25 || c.After == nil || *c.After != 30 {
		t.Fatalf("setting: %+v", c)
	}
	if comparison.Changes[1].Kind != "added-step" {
		t.Fatal(comparison)
	}
	if got := pipelinereview.Compare(base.Definition, checkedReview(t, 25, false, nil).Definition); len(got.Changes) != 0 || len(got.Gaps) != 0 {
		t.Fatal(got)
	}
	// Each returned check owns its settings values independently.
	*next.Definition.Steps[0].Settings[0].Value = 99
	if *base.Definition.Steps[0].Settings[0].Value != 25 {
		t.Fatal("aliased setting")
	}
}

func TestPipelineReviewRejectsUnexplainedBehavior(t *testing.T) {
	base := checkedReview(t, 25, false, nil)
	next := checkedReview(t, 30, false, []string{"--quiet"})
	if got := pipelinereview.Compare(base.Definition, next.Definition); len(got.Gaps) == 0 {
		t.Fatal("extra command behavior accepted", got)
	}
	next = checkedReview(t, 30, true, nil)
	// Reordering artifact-local edge IDs does not manufacture a connection diff.
	for i, j := 0, len(next.Definition.Edges)-1; i < j; i, j = i+1, j-1 {
		next.Definition.Edges[i], next.Definition.Edges[j] = next.Definition.Edges[j], next.Definition.Edges[i]
	}
	if got := pipelinereview.Compare(base.Definition, next.Definition); len(got.Gaps) != 0 {
		t.Fatal(got)
	}
	next.Definition.Edges = next.Definition.Edges[:len(next.Definition.Edges)-1]
	if got := pipelinereview.Compare(base.Definition, next.Definition); len(got.Gaps) == 0 {
		t.Fatal("removed input accepted")
	}
}

func TestPipelineReviewDoesNotChangeFlowContract(t *testing.T) {
	first := checkedReview(t, 25, false, nil)
	second := checkedReview(t, 25, false, nil)
	if first.Flow.SchemaVersion != 2 || !reflect.DeepEqual(first.Flow, second.Flow) {
		t.Fatal("ordinary flow changed")
	}
}
