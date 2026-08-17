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
// On any defect it returns (nil, [*Error]) with Op "plan". After a valid
// plan is built, a [WriteTo] error still returns the [*Plan].
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
	doc, err := planDocument(g)
	if err != nil {
		return nil, err
	}
	inner, defects := engine.BuildPlan(snapshotGraph(g), doc)
	if pub := publicError("plan", defects); pub != nil {
		return nil, pub
	}
	p := &Plan{inner: inner}
	if cfg.w != nil {
		if werr := p.WriteJSON(cfg.w); werr != nil {
			return p, werr
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

func planDocument(g *Graph) (engine.Document, error) {
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
			io, err := planIO(b)
			if err != nil {
				return engine.Document{}, planPathError(bindUnit(t.id, b.name), err)
			}
			pt.Inputs = append(pt.Inputs, io)
		}
		for _, b := range t.outputs {
			io, err := planIO(b)
			if err != nil {
				return engine.Document{}, planPathError(bindUnit(t.id, b.name), err)
			}
			pt.Outputs = append(pt.Outputs, io)
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
	return doc, nil
}

func planIO(b graphBind) (engine.IO, error) {
	path, err := b.spec.Render()
	if err != nil {
		return engine.IO{}, err
	}
	return engine.IO{
		Name: b.name,
		Path: path,
		Spec: snapshotPath(b.spec),
	}, nil
}

func planPathError(unit string, err error) error {
	msg := "invalid path"
	var ge *Error
	if errors.As(err, &ge) {
		if len(ge.Defects) > 0 && ge.Defects[0].Message != "" {
			msg = ge.Defects[0].Message
		}
	} else if err != nil {
		msg = err.Error()
	}
	return &Error{Op: "plan", Defects: []Defect{{
		Code:    DefectInvalidPath,
		Unit:    unit,
		Message: msg,
	}}}
}
