package gobble

import "github.com/HahyeonJeon/gobble/internal/engine"

// Compose builds an immutable Graph from p.
//
// On any compose defect it returns (nil, *Error) with Op "compose".
func Compose(p *Pipeline) (*Graph, error) {
	if p == nil {
		return nil, &Error{Op: "compose", Defects: []Defect{{
			Code:    DefectInvalidName,
			Message: "nil pipeline",
		}}}
	}
	if err := publicError("compose", engine.ComposeCheck(snapshotPipeline(p))); err != nil {
		return nil, err
	}
	return buildGraph(p), nil
}

func publicError(op string, defects []engine.Defect) error {
	if len(defects) == 0 {
		return nil
	}
	out := make([]Defect, len(defects))
	for i, d := range defects {
		out[i] = Defect{
			Code:    DefectCode(d.Code),
			Unit:    d.Unit,
			Message: d.Message,
			Paths:   copyStrings(d.Paths),
		}
	}
	return &Error{Op: op, Defects: out}
}

func snapshotPipeline(p *Pipeline) engine.Snapshot {
	s := engine.Snapshot{Name: p.name}
	for _, in := range p.inputs {
		s.Inputs = append(s.Inputs, engine.Input{Name: in.name, Spec: snapshotPath(in.spec)})
	}
	for _, t := range p.tasks {
		s.Tasks = append(s.Tasks, snapshotTask(t))
	}
	s.Nodes = snapshotNodes(p.children)
	return s
}

func snapshotGraph(g *Graph) engine.Snapshot {
	s := engine.Snapshot{Name: g.name}
	for _, in := range g.inputs {
		s.Inputs = append(s.Inputs, engine.Input{Name: in.name, Spec: snapshotPath(in.spec)})
	}
	for _, t := range g.tasks {
		gt := engine.Task{
			ID:      t.id,
			Name:    t.name,
			Command: copyStrings(t.command),
			Backend: t.backend,
			CPU:     t.resources.CPU,
		}
		for _, b := range t.inputs {
			gt.Inputs = append(gt.Inputs, snapshotGraphBind(b))
		}
		for _, b := range t.outputs {
			gt.Outputs = append(gt.Outputs, snapshotGraphBind(b))
		}
		s.Tasks = append(s.Tasks, gt)
	}
	return s
}

func snapshotTask(t *Task) engine.Task {
	gt := engine.Task{
		ID:       t.id(),
		Name:     t.spec.Name,
		Command:  copyStrings(t.spec.Command),
		Backend:  t.spec.Backend,
		CPU:      t.spec.Resources.CPU,
		OutCalls: copyStrings(t.outCalls),
		InCalls:  copyStrings(t.inCalls),
	}
	for _, b := range t.spec.Inputs {
		gt.Inputs = append(gt.Inputs, snapshotBind(b, t.pipe))
	}
	for _, b := range t.spec.Outputs {
		gt.Outputs = append(gt.Outputs, snapshotBind(b, t.pipe))
	}
	return gt
}

func snapshotBind(b Bind, p *Pipeline) engine.Bind {
	eb := engine.Bind{
		Name: b.Name,
		Spec: snapshotPath(b.Spec),
		Rule: engine.DeriveRule(b.Rule),
	}
	eb.FromKind, eb.FromName, eb.FromTask = snapshotFrom(b.From, p)
	return eb
}

func snapshotGraphBind(b graphBind) engine.Bind {
	return engine.Bind{
		Name:     b.name,
		Spec:     snapshotPath(b.spec),
		FromKind: snapshotFromKind(b.fromKind),
		FromName: b.fromName,
		FromTask: b.fromTask,
		Resolved: true,
	}
}

func snapshotFrom(h Handle, p *Pipeline) (engine.FromKind, string, string) {
	switch h.kind {
	case handleInput:
		if h.pipe != p {
			return engine.FromZero, "", ""
		}
		return engine.FromInput, h.name, ""
	case handleOut:
		if h.task == nil || h.task.pipe != p {
			return engine.FromZero, "", ""
		}
		return engine.FromOut, h.name, h.task.id()
	case handleIn:
		if h.task == nil || h.task.pipe != p {
			return engine.FromZero, "", ""
		}
		return engine.FromIn, h.name, h.task.id()
	default:
		return engine.FromZero, "", ""
	}
}

func snapshotFromKind(k handleKind) engine.FromKind {
	switch k {
	case handleInput:
		return engine.FromInput
	case handleOut:
		return engine.FromOut
	case handleIn:
		return engine.FromIn
	default:
		return engine.FromZero
	}
}

func snapshotNodes(nodes []node) []engine.Node {
	if nodes == nil {
		return nil
	}
	out := make([]engine.Node, 0, len(nodes))
	for _, n := range nodes {
		en := engine.Node{Name: n.name}
		switch n.kind {
		case nodeModule:
			en.Kind = engine.NodeModule
			if n.module != nil {
				en.Children = snapshotNodes(n.module.children)
			}
		case nodeBranch:
			en.Kind = engine.NodeBranch
			if n.branch != nil {
				en.Children = snapshotNodes(n.branch.children)
			}
		case nodeMerge:
			en.Kind = engine.NodeMerge
			if n.merge != nil {
				en.Children = snapshotNodes(n.merge.children)
			}
		case nodeTask:
			en.Kind = engine.NodeTask
			if n.task != nil {
				en.TaskID = n.task.id()
			}
		}
		out = append(out, en)
	}
	return out
}

func snapshotPath(p PathSpec) engine.Path {
	return engine.Path{
		Dir:     p.Dir.String(),
		Lead:    p.Lead,
		Name:    p.Name,
		Steps:   copyStrings(p.Steps),
		Ext:     p.Ext,
		Literal: p.literal,
		Opaque:  p.opaque,
		BadLit:  p.badLit,
	}
}

type resolver struct {
	p       *Pipeline
	memo    map[bindKey]PathSpec
	walking map[bindKey]bool
}

type bindKey struct {
	task *Task
	name string
	out  bool
}

func buildGraph(p *Pipeline) *Graph {
	r := &resolver{
		p:       p,
		memo:    make(map[bindKey]PathSpec),
		walking: make(map[bindKey]bool),
	}
	return r.buildGraph()
}

func (r *resolver) buildGraph() *Graph {
	g := &Graph{name: r.p.name}
	for _, in := range r.p.inputs {
		g.inputs = append(g.inputs, graphInput{name: in.name, spec: in.spec.clone()})
	}
	for _, t := range r.p.tasks {
		gt := graphTask{
			id:        t.id(),
			name:      t.spec.Name,
			module:    t.nearest(nodeModule),
			branch:    t.nearest(nodeBranch),
			merge:     t.nearest(nodeMerge),
			command:   copyStrings(t.spec.Command),
			image:     t.spec.Image,
			backend:   t.spec.Backend,
			resources: t.spec.Resources,
			params:    copyParams(t.spec.Params),
		}
		for _, b := range t.spec.Inputs {
			gt.inputs = append(gt.inputs, r.graphBind(t, b, false))
		}
		for _, b := range t.spec.Outputs {
			gt.outputs = append(gt.outputs, r.graphBind(t, b, true))
		}
		g.tasks = append(g.tasks, gt)
		g.edges = append(g.edges, fromEdges(t)...)
	}
	return g
}

func (r *resolver) graphBind(t *Task, b Bind, out bool) graphBind {
	gb := graphBind{
		name:     b.Name,
		fromKind: b.From.kind,
		fromName: b.From.name,
	}
	if spec, ok := r.resolveBind(t, b, out); ok {
		gb.spec = spec.clone()
	} else {
		gb.spec = b.Spec.clone()
	}
	if b.From.task != nil {
		gb.fromTask = b.From.task.id()
	}
	return gb
}

func (r *resolver) resolveBind(t *Task, b Bind, out bool) (PathSpec, bool) {
	key := bindKey{task: t, name: b.Name, out: out}
	if spec, ok := r.memo[key]; ok {
		return spec, true
	}
	if r.walking[key] {
		return PathSpec{}, false
	}
	if b.From.IsZero() {
		spec := b.Spec.clone()
		r.memo[key] = spec
		return spec, true
	}
	r.walking[key] = true
	from, ok := r.resolveFrom(b.From)
	r.walking[key] = false
	if !ok {
		return PathSpec{}, false
	}
	spec := classifySpec(b.Spec, from, b.Rule)
	r.memo[key] = spec
	return spec, true
}

func (r *resolver) resolveFrom(h Handle) (PathSpec, bool) {
	switch h.kind {
	case handleInput:
		if h.pipe != r.p {
			return PathSpec{}, false
		}
		for _, in := range r.p.inputs {
			if in.name == h.name {
				return in.spec.clone(), true
			}
		}
		return PathSpec{}, false
	case handleOut:
		if h.task == nil || h.task.pipe != r.p {
			return PathSpec{}, false
		}
		b, ok := findBind(h.task.spec.Outputs, h.name)
		if !ok {
			return PathSpec{}, false
		}
		return r.resolveBind(h.task, b, true)
	case handleIn:
		if h.task == nil || h.task.pipe != r.p {
			return PathSpec{}, false
		}
		b, ok := findBind(h.task.spec.Inputs, h.name)
		if !ok {
			return PathSpec{}, false
		}
		return r.resolveBind(h.task, b, false)
	default:
		return PathSpec{}, false
	}
}

func fromEdges(t *Task) []graphEdge {
	var edges []graphEdge
	add := func(b Bind) {
		if b.From.IsZero() {
			return
		}
		e := graphEdge{fromPort: b.From.name, toTask: t.id(), toPort: b.Name}
		if b.From.kind == handleInput {
			e.fromInput = b.From.name
		}
		if b.From.task != nil {
			e.fromTask = b.From.task.id()
		}
		edges = append(edges, e)
	}
	for _, b := range t.spec.Inputs {
		add(b)
	}
	for _, b := range t.spec.Outputs {
		add(b)
	}
	return edges
}
