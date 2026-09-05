package monitor

import (
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func sampleTask(id, name, sample, status string) Task {
	return Task{Identity: id, TaskID: id, Name: name, Status: status, Display: gobble.TaskDisplay{Samples: []string{sample}}}
}

func TestDashboardExactSamplesAndSharedContext(t *testing.T) {
	s := Snapshot{Tasks: []Task{
		{Identity: "ref", TaskID: "ref", Name: "reference", Status: "succeeded", Display: gobble.TaskDisplay{Scope: gobble.DisplayShared}},
		sampleTask("a", "align", "S01", "running"),
		sampleTask("b", "align", "S010", "failed"),
		{Identity: "report", TaskID: "report", Name: "report", Status: "blocked", Display: gobble.TaskDisplay{Scope: gobble.DisplayCohort}},
	}, Edges: []Edge{{"ref", "a"}, {"ref", "b"}, {"a", "report"}, {"b", "report"}}}
	s.Tasks[1].Display.Samples = []string{"S01", "S01"}
	d, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	if d.Total.Total != 4 || d.Shared.Total != 2 || len(d.Stages) != 3 || len(d.Edges) != 2 {
		t.Fatalf("incorrect aggregation: %+v", d)
	}
	if got := d.SampleTasks("S01"); len(got) != 1 || got[0] != 1 {
		t.Fatalf("sample membership: %v", got)
	}
	if got := d.SearchSamples("s01"); len(got) != 2 {
		t.Fatalf("search: %v", got)
	}
	if len(d.SampleTasks("S0")) != 0 {
		t.Fatal("membership used a substring")
	}
	if d.Samples[0].Counts.Total != 1 || d.Samples[1].Counts.Total != 1 {
		t.Fatal("shared work counted per sample")
	}
	if len(d.StageTasks(d.Stages[0], "S01")) != 1 {
		t.Fatal("shared graph context lost")
	}
	if len(d.Attention) != 2 || d.Attention[0] != 2 || d.Attention[1] != 3 {
		t.Fatalf("attention: %v", d.Attention)
	}
}

func TestCountsPartitionAndExpansion(t *testing.T) {
	var c Counts
	for _, status := range []string{"succeeded", "running", "failed", "not-started", "blocked", "skipped", "incomplete", "unknown", "published-unfinalized"} {
		c.Add(Task{Status: status})
	}
	c.Add(Task{Status: "succeeded", Decision: "reused"})
	c.Add(Task{Template: true, Status: "not-started"})
	c.Add(Task{Template: true, Expanded: true, Status: "succeeded"})
	if c.Total != 10 || c.Templates != 2 || c.Unexpanded != 1 || c.Reused != 1 || c.Percent() != 20 || c.Attention() != 5 {
		t.Fatalf("counts: %+v", c)
	}
	if c.Total != c.Succeeded+c.Running+c.Failed+c.Pending+c.Blocked+c.Skipped+c.Incomplete+c.Unknown+c.Unfinalized {
		t.Fatal("states do not partition total")
	}
	if (Counts{Total: 1, Succeeded: 1, Unexpanded: 1}).Successful() {
		t.Fatal("unexpanded work marked complete")
	}
	if (Counts{}).Successful() {
		t.Fatal("empty workload marked complete")
	}
}

func TestGroupingCannotIntroduceCycles(t *testing.T) {
	s := Snapshot{Tasks: []Task{sampleTask("a", "align", "S01", "succeeded"), sampleTask("b", "sort", "S01", "running"), sampleTask("c", "align", "S01", "not-started")}, Edges: []Edge{{"a", "b"}, {"b", "c"}, {"a", "b"}}}
	d, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Stages) != 3 || len(d.Edges) != 2 {
		t.Fatalf("unsafe contraction: %+v", d.Stages)
	}
	for i, stage := range d.Stages {
		if stage.Rank != i {
			t.Fatalf("ranks: %+v", d.Stages)
		}
	}
	s.Edges = append(s.Edges, Edge{"c", "a"})
	if _, err := Build(s); err == nil {
		t.Fatal("cycle accepted")
	}
	s.Edges = []Edge{{"a", "missing"}}
	if _, err := Build(s); err == nil {
		t.Fatal("dangling edge accepted")
	}
}

func TestScatterInstancesShareAuthoredPosition(t *testing.T) {
	s := Snapshot{Tasks: []Task{{Identity: "scatter", TaskID: "scatter", Name: "align", Template: true, Expanded: true}, {Identity: "scatter#a", TaskID: "scatter", Name: "align", Status: "succeeded"}, {Identity: "scatter#b", TaskID: "scatter", Name: "align", Status: "running"}}}
	d, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Stages) != 1 || d.Total.Total != 2 || d.Total.Templates != 1 {
		t.Fatalf("scatter: %+v", d)
	}
	s.Tasks = append(s.Tasks, s.Tasks[1])
	if _, err := Build(s); err == nil {
		t.Fatal("duplicate identity accepted")
	}
}

func TestSkippedTemplateIsNotPendingExpansion(t *testing.T) {
	var counts Counts
	counts.Add(Task{Template: true, Status: "skipped"})
	counts.Add(Task{Status: "succeeded"})
	if counts.Unexpanded != 0 || counts.SkippedTemplates != 1 || counts.Total != 1 || counts.Successful() {
		t.Fatalf("skipped template misreported: %+v", counts)
	}
}
