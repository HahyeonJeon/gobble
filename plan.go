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

// BuildPlan validates g first, then returns an inspectable plan.
//
// On any defect it returns (nil, [*Error]) with Op "plan". After a valid
// plan is built, a [WriteTo] error still returns the [*Plan] and is the
// writer's own error, not an [*Error].
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
	if pub := publicError("plan", engine.Validate(snapshotGraph(g))); pub != nil {
		return nil, pub
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
// A nil [*Plan] returns a non-[*Error].
func (p *Plan) WriteJSON(w io.Writer) error {
	if p == nil || p.inner == nil {
		return errors.New("nil plan")
	}
	return p.inner.WriteJSON(w)
}

// MarshalJSON returns the same plan JSON [WriteJSON] writes, without a
// trailing newline. A nil [*Plan] returns the JSON null value.
func (p *Plan) MarshalJSON() ([]byte, error) {
	if p == nil || p.inner == nil {
		return []byte("null"), nil
	}
	return p.inner.MarshalJSON()
}

func planDocument(g *Graph) (engine.Document, error) {
	doc := engine.Document{Name: g.name}
	byID := make(map[string]*engine.TaskPlan, len(g.tasks))
	for i := range g.tasks {
		t := &g.tasks[i]
		command := copyStrings(t.command)
		if t.script != "" {
			command = scriptArgv(t.script)
		}
		pt := engine.TaskPlan{
			ID:      t.id,
			Name:    t.name,
			Module:  t.module,
			Branch:  t.branch,
			Merge:   t.merge,
			Command: command,
			Script:  t.script,
			Image:   t.image,
			Backend: t.backend,
			Resources: engine.ResourcePlan{
				CPU:    t.resources.CPU,
				Memory: t.resources.Memory,
			},
			Env: copyEnv(t.env),
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
	for i := range doc.Tasks {
		byID[doc.Tasks[i].ID] = &doc.Tasks[i]
	}
	for _, e := range g.edges {
		wait, err := planWait(byID, e)
		if err != nil {
			return engine.Document{}, err
		}
		doc.Edges = append(doc.Edges, engine.Edge{
			FromTask: e.fromTask,
			FromPort: e.fromPort,
			ToTask:   e.toTask,
			ToPort:   e.toPort,
			Wait:     wait,
		})
	}
	return doc, nil
}

func scriptArgv(script string) []string {
	return []string{"sh", "-c", "set -eu\n" + script}
}

func planIO(b graphBind) (engine.IO, error) {
	if b.members != nil {
		io := engine.IO{
			Name:    b.name,
			Spec:    snapshotPath(b.spec),
			Members: make([]engine.IOMember, 0, len(b.members)),
		}
		for _, m := range b.members {
			path, err := m.spec.Render()
			if err != nil {
				return engine.IO{}, err
			}
			io.Members = append(io.Members, engine.IOMember{
				Name: m.name,
				Path: path,
				Spec: snapshotPath(m.spec),
			})
		}
		return io, nil
	}
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

func planWait(byID map[string]*engine.TaskPlan, e graphEdge) ([]string, error) {
	to, ok := byID[e.toTask]
	if !ok {
		return nil, neverReadyError(bindUnit(e.toTask, e.toPort))
	}
	toIO, toIsInput, ok := findTaskIO(to, e.toPort)
	if !ok {
		return nil, neverReadyError(bindUnit(e.toTask, e.toPort))
	}
	if e.fromTask == "" || toIsInput {
		return waitPaths(toIO, bindUnit(e.toTask, e.toPort))
	}
	from, ok := byID[e.fromTask]
	if !ok {
		return nil, neverReadyError(bindUnit(e.toTask, e.toPort))
	}
	fromIO, _, ok := findPublishedIO(from, e.fromPort)
	if !ok {
		return nil, neverReadyError(bindUnit(e.toTask, e.toPort))
	}
	return waitPaths(fromIO, bindUnit(e.toTask, e.toPort))
}

func findTaskIO(t *engine.TaskPlan, port string) (engine.IO, bool, bool) {
	for _, in := range t.Inputs {
		if in.Name == port {
			return in, true, true
		}
	}
	for _, out := range t.Outputs {
		if out.Name == port {
			return out, false, true
		}
	}
	return engine.IO{}, false, false
}

func findPublishedIO(t *engine.TaskPlan, port string) (engine.IO, bool, bool) {
	for _, out := range t.Outputs {
		if out.Name == port {
			return out, false, true
		}
	}
	for _, in := range t.Inputs {
		if in.Name == port {
			return in, true, true
		}
	}
	return engine.IO{}, false, false
}

func waitPaths(io engine.IO, unit string) ([]string, error) {
	if io.Members != nil {
		waits := make([]string, 0, len(io.Members))
		for _, m := range io.Members {
			if m.Path == "" {
				return nil, neverReadyError(unit)
			}
			waits = append(waits, m.Path)
		}
		if len(waits) == 0 {
			return nil, neverReadyError(unit)
		}
		return waits, nil
	}
	if io.Path == "" {
		return nil, neverReadyError(unit)
	}
	return []string{io.Path}, nil
}

func neverReadyError(unit string) error {
	return &Error{Op: "plan", Defects: []Defect{{
		Code:    DefectNeverReady,
		Unit:    unit,
		Message: "never-ready",
	}}}
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
