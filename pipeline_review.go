package gobble

import (
	"encoding/json"

	"github.com/HahyeonJeon/gobble/internal/engine"
	"github.com/HahyeonJeon/gobble/internal/pipelinereview"
)

// PipelineReview retains visual facts and a separate definition for comparing
// processing behavior. It neither serializes executable work nor approves a Run.
type PipelineReview struct {
	SchemaVersion int                      `json:"schemaVersion"`
	Flow          PipelineInspection       `json:"flow"`
	Definition    PipelineReviewDefinition `json:"definition"`
}

type PipelineReviewDefinition = pipelinereview.Definition

// InspectPipelineReview checks the same graph as InspectPipeline and adds
// complete task fingerprints and bounded, qualified review recipes. Sensitive
// command/environment values are hashed, not exposed as review prose.
func InspectPipelineReview(g *Graph) (PipelineReview, error) {
	flow, err := InspectPipeline(g)
	if err != nil {
		return PipelineReview{}, err
	}
	doc, err := planDocument(g)
	if err != nil {
		return PipelineReview{}, err
	}
	d := pipelinereview.Definition{SchemaVersion: 1, Context: pipelinereview.Digest([]any{flow.Name, flow.Inputs}), Steps: []pipelinereview.Step{}, Edges: []pipelinereview.Edge{}}
	for i, t := range doc.Tasks {
		step := engine.ReviewStep(t)
		// Module-authored display/settings cannot claim a different operation.
		if step.Recipe == "fastqc-v1" && len(flow.Steps[i].Settings) > 0 {
			step.Recipe = ""
		}
		if step.Recipe == "trim-galore-v1" {
			actual, _ := json.Marshal(flow.Steps[i].Settings)
			expected, _ := json.Marshal(step.Settings)
			if string(actual) != string(expected) {
				step.Recipe = ""
				step.Settings = []pipelinereview.Setting{}
			}
		}
		step.Fingerprint = pipelinereview.Digest([]any{step.Fingerprint, flow.Steps[i]})
		visual := flow.Steps[i]
		visual.Settings = []IntegerSetting{}
		step.Residual = pipelinereview.Digest([]any{step.Residual, visual})
		d.Steps = append(d.Steps, step)
	}
	for _, e := range flow.Connections {
		d.Edges = append(d.Edges, pipelinereview.Edge{FromTask: e.FromTask, FromPort: e.FromPort, ToTask: e.ToTask, ToPort: e.ToPort, Wait: e.Wait})
	}
	return PipelineReview{1, flow, d}, nil
}
