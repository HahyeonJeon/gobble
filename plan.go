package gobble

import (
	"errors"
	"io"

	"github.com/HahyeonJeon/gobble/internal/engine"
)

// Plan is the inspectable dry-run document for a valid graph.
type Plan struct {
	inner *engine.Plan
}

// PlanOption configures [BuildPlan].
type PlanOption func(*planConfig)

type planConfig struct {
	w io.Writer
}

// WriteTo returns a [PlanOption] that writes the same JSON a [*Plan]
// would write with [Plan.WriteJSON].
func WriteTo(w io.Writer) PlanOption {
	return func(c *planConfig) {
		c.w = w
	}
}

// BuildPlan validates g and returns an inspectable plan.
//
// On any defect it returns (nil, [*Error]) with Op "plan".
func BuildPlan(g *Graph, opts ...PlanOption) (*Plan, error) {
	var cfg planConfig
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if g == nil {
		return nil, &Error{Op: "plan", Defects: []Defect{{
			Code:    DefectInvalidName,
			Message: "nil graph",
		}}}
	}
	inner, defects := engine.BuildPlan(snapshotGraph(g), planDocument(g))
	if err := publicError("plan", defects); err != nil {
		return nil, err
	}
	p := &Plan{inner: inner}
	if cfg.w != nil {
		if err := p.WriteJSON(cfg.w); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// WriteJSON writes the plan as one JSON object to w, including a trailing newline.
func (p *Plan) WriteJSON(w io.Writer) error {
	if p == nil || p.inner == nil {
		return errors.New("nil plan")
	}
	return p.inner.WriteJSON(w)
}

// MarshalJSON returns the same plan JSON [WriteJSON] writes, without a
// trailing newline.
func (p *Plan) MarshalJSON() ([]byte, error) {
	if p == nil || p.inner == nil {
		return []byte("null"), nil
	}
	return p.inner.MarshalJSON()
}

func planDocument(g *Graph) engine.Document {
	doc := engine.Document{Name: g.name}
	for i := range g.tasks {
		t := &g.tasks[i]
		pt := engine.TaskPlan{
			ID:      t.id,
			Name:    t.name,
			Module:  t.module,
			Branch:  t.branch,
			Merge:   t.merge,
			Command: copyStrings(t.command),
			Image:   t.image,
			Backend: t.backend,
			Resources: engine.ResourcePlan{
				CPU:    t.resources.CPU,
				Memory: t.resources.Memory,
			},
		}
		for _, p := range t.params {
			pt.Params = append(pt.Params, engine.ParamPlan{Name: p.Name, Value: p.Value})
		}
		for _, b := range t.inputs {
			pt.Inputs = append(pt.Inputs, planIO(b))
		}
		for _, b := range t.outputs {
			pt.Outputs = append(pt.Outputs, planIO(b))
		}
		doc.Tasks = append(doc.Tasks, pt)
	}
	for _, e := range g.edges {
		doc.Edges = append(doc.Edges, engine.Edge{
			FromTask: e.fromTask,
			FromPort: e.fromPort,
			ToTask:   e.toTask,
			ToPort:   e.toPort,
		})
	}
	return doc
}

func planIO(b graphBind) engine.IO {
	path, err := b.spec.Render()
	if err != nil {
		path = ""
	}
	return engine.IO{
		Name: b.name,
		Path: path,
		Spec: snapshotPath(b.spec),
	}
}
