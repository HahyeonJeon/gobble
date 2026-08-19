package gobble_test

import (
	"errors"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestGraphReaders(t *testing.T) {
	g, err := gobble.Compose(workflowCasePipeline())
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if g.Name() != "workflow-case" {
		t.Fatalf("Name() got %q, want workflow-case", g.Name())
	}
	wantIDs := []string{"prep.fastp", "call.align.bwa", "call.qc.fastqc", "call.join.report"}
	gotIDs := g.TaskIDs()
	if len(gotIDs) != len(wantIDs) {
		t.Fatalf("TaskIDs() got %v, want %v", gotIDs, wantIDs)
	}
	for i, id := range wantIDs {
		if gotIDs[i] != id {
			t.Fatalf("TaskIDs() got %v, want %v", gotIDs, wantIDs)
		}
	}
	wantInputs := []string{"reads_r1", "reads_r2"}
	gotInputs := g.InputNames()
	if len(gotInputs) != len(wantInputs) || gotInputs[0] != wantInputs[0] || gotInputs[1] != wantInputs[1] {
		t.Fatalf("InputNames() got %v, want %v", gotInputs, wantInputs)
	}
	edges := g.Edges()
	if len(edges) == 0 {
		t.Fatalf("Edges() empty")
	}
	found := false
	for _, e := range edges {
		if e.FromTask == "call.align.bwa" && e.FromPort == "bam" && e.ToTask == "call.join.report" && e.ToPort == "bam" {
			found = true
			if len(e.Wait) != 0 {
				t.Fatalf("Graph Edge.Wait got %#v, want empty", e.Wait)
			}
		}
	}
	if !found {
		t.Fatalf("Edges() missing bwa.bam -> join.report.bam: %#v", edges)
	}
	if g.BindKind("call.align.bwa", "bam") != gobble.ArtifactFile {
		t.Fatalf("BindKind(bam) got %q, want %s", g.BindKind("call.align.bwa", "bam"), gobble.ArtifactFile)
	}
	if g.BindPath("call.align.bwa", "bam") != "work/align/sample.sorted.bam" {
		t.Fatalf("BindPath(bam) got %q", g.BindPath("call.align.bwa", "bam"))
	}
}

func TestPlanReaders(t *testing.T) {
	g, err := gobble.Compose(workflowCasePipeline())
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	plan, err := gobble.BuildPlan(g)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if plan.Pipeline() != "workflow-case" {
		t.Fatalf("Pipeline() got %q, want workflow-case", plan.Pipeline())
	}
	ids := plan.TaskIDs()
	if len(ids) != 4 || ids[0] != "prep.fastp" {
		t.Fatalf("TaskIDs() got %v", ids)
	}
	edges := plan.Edges()
	found := false
	for _, e := range edges {
		if e.FromTask == "call.align.bwa" && e.FromPort == "bam" && e.ToTask == "call.join.report" && e.ToPort == "bam" {
			found = true
			if len(e.Wait) != 1 || e.Wait[0] != "work/align/sample.sorted.bam" {
				t.Fatalf("Plan Edge.Wait got %#v", e.Wait)
			}
		}
	}
	if !found {
		t.Fatalf("Plan Edges() missing bwa.bam -> join.report.bam: %#v", edges)
	}
	if plan.BindKind("call.align.bwa", "bam") != gobble.ArtifactFile {
		t.Fatalf("BindKind(bam) got %q", plan.BindKind("call.align.bwa", "bam"))
	}
	if plan.BindPath("call.align.bwa", "bam") != "work/align/sample.sorted.bam" {
		t.Fatalf("BindPath(bam) got %q", plan.BindPath("call.align.bwa", "bam"))
	}
}

func TestComposeNilPipeline(t *testing.T) {
	g, err := gobble.Compose(nil)
	if g != nil {
		t.Fatalf("Compose(nil) graph != nil")
	}
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		t.Fatalf("Compose(nil) error = %v, want *Error", err)
	}
	if ge.Op != "compose" {
		t.Fatalf("Compose(nil) Op got %q, want compose", ge.Op)
	}
	found := false
	for _, d := range ge.Defects {
		if d.Code == gobble.DefectInvalidRequest {
			found = true
		}
	}
	if !found {
		t.Fatalf("Compose(nil) defects %#v, want invalid-request", ge.Defects)
	}
}
