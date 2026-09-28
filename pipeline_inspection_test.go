package gobble_test

import (
	"encoding/json"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func TestPipelineInspectionPortsAndOwnership(t *testing.T) {
	g := mustCompose(func() *gobble.Pipeline {
		p := gobble.NewPipeline("Analysis")
		in := p.AddInput("reads", gobble.PathSpec{Base: "sample", Ext: ".fq"})
		trim := p.AddTask(gobble.TaskSpec{Name: "Trim", Command: []string{"true"}, Inputs: []gobble.Bind{{Name: "reads", From: in}}, Outputs: []gobble.Bind{{Name: "trimmed", Spec: gobble.PathSpec{Base: "trimmed", Ext: ".fq"}}}})
		for _, name := range []string{"Align", "Quality"} {
			p.AddTask(gobble.TaskSpec{Name: name, Command: []string{"true"}, Inputs: []gobble.Bind{{Name: "reads", From: trim.Out("trimmed")}}, Outputs: []gobble.Bind{{Name: "report", Spec: gobble.PathSpec{Base: name, Ext: ".txt"}}}})
		}
		return p
	})(t)
	before, err := gobble.BuildPlan(g)
	if err != nil {
		t.Fatal(err)
	}
	old, _ := json.Marshal(before)
	flow, err := gobble.InspectPipeline(g)
	if err != nil {
		t.Fatal(err)
	}
	if flow.SchemaVersion != 2 || len(flow.Steps) == 0 || len(flow.Inputs) == 0 {
		t.Fatalf("incomplete flow: %+v", flow)
	}
	edges := before.Edges()
	if len(flow.Connections) != len(edges) {
		t.Fatal("input or port edges lost")
	}
	inputEdges := 0
	for i, edge := range flow.Connections {
		want := edges[i]
		if edge.FromTask != want.FromTask || edge.FromPort != want.FromPort || edge.ToTask != want.ToTask || edge.ToPort != want.ToPort {
			t.Fatal("endpoint identity changed")
		}
		if edge.FromTask == "" {
			inputEdges++
		}
	}
	if inputEdges == 0 {
		t.Fatal("fixture has no boundary edge")
	}
	flow.Steps[0].Inputs = nil
	flow.Connections[0].FromPort = "changed"
	again, err := gobble.InspectPipeline(g)
	if err != nil || again.Connections[0].FromPort == "changed" {
		t.Fatal("inspection aliases graph")
	}
	after, _ := json.Marshal(before)
	if string(old) != string(after) {
		t.Fatal("existing Plan JSON changed")
	}
}

func TestPipelineInspectionRejectsInvalidGraph(t *testing.T) {
	if _, err := gobble.InspectPipeline(nil); err == nil {
		t.Fatal("nil graph accepted")
	}
}

func TestPipelineInspectionDeclaredControlAndDataKinds(t *testing.T) {
	for _, test := range []struct {
		name     string
		pipeline func() *gobble.Pipeline
		check    func(t *testing.T, flow gobble.PipelineInspection)
	}{
		{"branch", workflowCasePipeline, func(t *testing.T, flow gobble.PipelineInspection) {
			branches, merges := 0, 0
			for _, step := range flow.Steps {
				if step.Control.Branch != "" {
					branches++
				}
				if step.Control.Merge != "" {
					merges++
				}
			}
			if branches < 2 || merges == 0 {
				t.Fatal("declared branch/merge lost")
			}
		}},
		{"scatter", func() *gobble.Pipeline { return scatterGroupPipeline("true") }, func(t *testing.T, flow gobble.PipelineInspection) {
			if flow.Inputs[0].Kind != "group" || len(flow.Inputs[0].Members) != 2 {
				t.Fatal("group member identities lost")
			}
			first := flow.Steps[0].Control
			if first.Scatter != "each" || first.ScatterFromPort != "samples" || first.ScatterFromTask != "" || flow.Steps[1].Control.Gather != "all" {
				t.Fatal("scatter source or gather lost", flow)
			}
		}},
		{"condition", func() *gobble.Pipeline { return whenPredWithAfter("keep") }, func(t *testing.T, flow gobble.PipelineInspection) {
			if flow.Steps[0].Control.When != "opt" || flow.Steps[0].Control.SkipIfFalse != "keep" {
				t.Fatal("condition identity lost")
			}
		}},
		{"tree", treeFromPipelineInputPipeline, func(t *testing.T, flow gobble.PipelineInspection) {
			if flow.Inputs[0].Kind != "tree" || flow.Inputs[0].Path == "" {
				t.Fatal("tree input lost")
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			flow, err := gobble.InspectPipeline(mustCompose(test.pipeline)(t))
			if err != nil {
				t.Fatal(err)
			}
			test.check(t, flow)
		})
	}
}

func TestPipelineInspectionSettingOwnership(t *testing.T) {
	value := 25
	fields := []gobble.IntegerSetting{{Key: "quality", Label: "Quality", Unit: "Phred", Value: &value}}
	p := gobble.NewPipeline("Settings")
	p.AddTask(gobble.TaskSpec{Name: "step", Command: []string{"must-not-run"}, InspectionSettings: fields, Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Base: "result", Ext: ".txt"}}}})
	value = 99
	fields[0].Key = "changed"
	g, err := gobble.Compose(p)
	if err != nil {
		t.Fatal(err)
	}
	flow, err := gobble.InspectPipeline(g)
	if err != nil {
		t.Fatal(err)
	}
	if flow.Steps[0].Settings[0].Key != "quality" || *flow.Steps[0].Settings[0].Value != 25 {
		t.Fatal("TaskSpec setting aliases caller")
	}
	*flow.Steps[0].Settings[0].Value = 88
	again, err := gobble.InspectPipeline(g)
	if err != nil || *again.Steps[0].Settings[0].Value != 25 {
		t.Fatal("inspection aliases graph")
	}
	for _, bad := range [][]gobble.IntegerSetting{{{Key: "", Label: "bad"}}, {{Key: "same", Label: "one"}, {Key: "same", Label: "two"}}} {
		p := gobble.NewPipeline("Invalid")
		p.AddTask(gobble.TaskSpec{Name: "step", Command: []string{"true"}, InspectionSettings: bad, Outputs: []gobble.Bind{{Name: "out", Spec: gobble.PathSpec{Base: "result", Ext: ".txt"}}}})
		g, err := gobble.Compose(p)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := gobble.InspectPipeline(g); err == nil {
			t.Fatal("invalid metadata accepted")
		}
	}
}
