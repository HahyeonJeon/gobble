package gobble

import "github.com/HahyeonJeon/gobble/internal/engine"

// Compose builds an immutable Graph from p.
//
// A From handle must belong to p. A handle from another Pipeline is
// missing-input on that bind, including output binds.
// On any compose defect it returns (nil, *Error) with Op "compose".
func Compose(p *Pipeline) (*Graph, error) {
	if p == nil {
		return nil, &Error{Op: "compose", Defects: []Defect{{
			Code:    DefectInvalidRequest,
			Message: "nil pipeline",
		}}}
	}
	if err := composeDefects("compose", composeCheckPipeline(p)); err != nil {
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

func foreignFrom(h Handle, p *Pipeline) bool {
	switch h.kind {
	case handleZero:
		return false
	case handleInput:
		return h.pipe != p
	case handleOut, handleIn:
		return h.task == nil || h.task.pipe != p
	default:
		return true
	}
}

func snapshotPath(p PathSpec) engine.Path {
	return engine.Path{
		Dir:      p.Dir.String(),
		Prefix:   p.Prefix,
		Base:     p.Base,
		Suffixes: copyStrings(p.Suffixes),
		Ext:      p.Ext,
		Literal:  p.literal,
		Opaque:   p.opaque,
		BadLit:   p.badLit,
	}
}

type resolver struct {
	p          *Pipeline
	memo       map[bindKey]PathSpec
	memberMemo map[bindKey][]graphMember
	walking    map[bindKey]bool
}

type bindKey struct {
	task *Task
	name string
	out  bool
}

func buildGraph(p *Pipeline) *Graph {
	r := &resolver{
		p:          p,
		memo:       make(map[bindKey]PathSpec),
		memberMemo: make(map[bindKey][]graphMember),
		walking:    make(map[bindKey]bool),
	}
	return r.buildGraph()
}

func (r *resolver) buildGraph() *Graph {
	g := &Graph{name: r.p.name}
	for _, in := range r.p.inputs {
		gi := graphInput{name: in.name, spec: in.spec.clone()}
		if in.members != nil {
			gi.members = authoredMembers(in.members)
		}
		g.inputs = append(g.inputs, gi)
	}
	for _, t := range r.p.tasks {
		gt := graphTask{
			id:        t.id(),
			name:      t.spec.Name,
			module:    t.nearest(nodeModule),
			branch:    t.nearest(nodeBranch),
			merge:     t.nearest(nodeMerge),
			command:   copyStrings(t.spec.Command),
			script:    t.spec.Script,
			image:     t.spec.Image,
			backend:   t.spec.Backend,
			resources: t.spec.Resources,
			params:    copyParams(t.spec.Params),
			env:       copyEnv(t.spec.Env),
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
	if b.Group != nil {
		if members, ok := r.resolveMembers(t, b, out); ok {
			gb.members = members
		} else {
			gb.members = authoredMembers(b.Group)
		}
	} else if spec, ok := r.resolveBind(t, b, out); ok {
		gb.spec = spec.clone()
	} else {
		gb.spec = b.Spec.clone()
	}
	if foreignFrom(b.From, r.p) {
		gb.fromKind = handleZero
		gb.fromName = ""
		return gb
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

func (r *resolver) resolveMembers(t *Task, b Bind, out bool) ([]graphMember, bool) {
	if b.Group == nil {
		return nil, true
	}
	key := bindKey{task: t, name: b.Name, out: out}
	if members, ok := r.memberMemo[key]; ok {
		return members, true
	}
	if r.walking[key] {
		return nil, false
	}
	if b.From.IsZero() {
		members := authoredMembers(b.Group)
		r.memberMemo[key] = members
		return members, true
	}
	r.walking[key] = true
	fromMembers, ok := r.resolveFromMembers(b.From)
	r.walking[key] = false
	if !ok {
		return nil, false
	}
	outm := make([]graphMember, 0, len(b.Group))
	for _, m := range b.Group {
		from, ok := findGraphMember(fromMembers, m.Name)
		if !ok {
			return nil, false
		}
		outm = append(outm, graphMember{name: m.Name, spec: classifySpec(m.Spec, from.spec, b.Rule)})
	}
	r.memberMemo[key] = outm
	return outm, true
}

func (r *resolver) resolveFromMembers(h Handle) ([]graphMember, bool) {
	if foreignFrom(h, r.p) {
		return nil, false
	}
	switch h.kind {
	case handleInput:
		for _, in := range r.p.inputs {
			if in.name == h.name {
				if in.members == nil {
					return nil, false
				}
				return authoredMembers(in.members), true
			}
		}
		return nil, false
	case handleOut:
		b, ok := findBind(h.task.spec.Outputs, h.name)
		if !ok || b.Group == nil {
			return nil, false
		}
		return r.resolveMembers(h.task, b, true)
	case handleIn:
		b, ok := findBind(h.task.spec.Inputs, h.name)
		if !ok || b.Group == nil {
			return nil, false
		}
		return r.resolveMembers(h.task, b, false)
	default:
		return nil, false
	}
}

func authoredMembers(in Group) []graphMember {
	out := make([]graphMember, 0, len(in))
	for _, m := range in {
		out = append(out, graphMember{name: m.Name, spec: m.Spec.clone()})
	}
	return out
}

func findGraphMember(members []graphMember, name string) (graphMember, bool) {
	for _, m := range members {
		if m.name == name {
			return m, true
		}
	}
	return graphMember{}, false
}

func (r *resolver) resolveFrom(h Handle) (PathSpec, bool) {
	if foreignFrom(h, r.p) {
		return PathSpec{}, false
	}
	switch h.kind {
	case handleInput:
		for _, in := range r.p.inputs {
			if in.name == h.name {
				return in.spec.clone(), true
			}
		}
		return PathSpec{}, false
	case handleOut:
		b, ok := findBind(h.task.spec.Outputs, h.name)
		if !ok {
			return PathSpec{}, false
		}
		return r.resolveBind(h.task, b, true)
	case handleIn:
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
		if b.From.IsZero() || foreignFrom(b.From, t.pipe) {
			return
		}
		e := graphEdge{fromPort: b.From.name, toTask: t.id(), toPort: b.Name}
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
