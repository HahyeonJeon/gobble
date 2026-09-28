package gobble

import (
	"github.com/HahyeonJeon/gobble/internal/engine"
	"github.com/HahyeonJeon/gobble/internal/pipelinereview"
)

// PipelineCreationReview is a complete first-version check, not an executable
// payload or permission to adopt. Gaps must be empty before it is reviewable.
type PipelineCreationReview struct {
	SchemaVersion int            `json:"schemaVersion"`
	Scope         string         `json:"scope"`
	InputPath     string         `json:"inputPath"`
	Review        PipelineReview `json:"review"`
	Gaps          []string       `json:"gaps"`
}

// InspectPipelineCreation qualifies the entire supported single-end design.
// Source evaluation still belongs in the caller's isolated runtime boundary.
func InspectPipelineCreation(g *Graph, inputPath string) (PipelineCreationReview, error) {
	review, err := InspectPipelineReview(g)
	if err != nil {
		return PipelineCreationReview{}, err
	}
	doc, err := planDocument(g)
	if err != nil {
		return PipelineCreationReview{}, err
	}
	gaps := engine.QualifyCreation(doc, inputPath)
	if len(review.Flow.Inputs) != 1 || review.Flow.Inputs[0].Name != "reads" || review.Flow.Inputs[0].Kind != "file" || review.Flow.Inputs[0].Path != inputPath || len(review.Flow.Inputs[0].Members) != 0 {
		gaps = append(gaps, "Creation requires exactly one declared single-end read input.")
	}
	if len(review.Definition.Steps) != 2 || review.Definition.Steps[0].Recipe != "trim-galore-v1" || review.Definition.Steps[1].Recipe != "fastqc-v1" {
		gaps = append(gaps, "The displayed settings must match the complete checked processing steps.")
	}
	return PipelineCreationReview{1, pipelinereview.CreationScope, inputPath, review, gaps}, nil
}
