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
		s.Inputs = append(s.Inputs, snapshotPipeInput(in))
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
		s.Inputs = append(s.Inputs, snapshotGraphInput(in))
	}
	for _, t := range g.tasks {
		gt := engine.Task{
			ID:      t.id,
			Name:    t.name,
			Command: copyStrings(t.command),
			Script:  t.script,
			Backend: t.backend,
			CPU:     t.resources.CPU,
			Memory:  t.resources.Memory,
			Env:     copyEnv(t.env),
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
		Script:   t.spec.Script,
		Backend:  t.spec.Backend,
		CPU:      t.spec.Resources.CPU,
		Env:      copyEnv(t.spec.Env),
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

func snapshotPipeInput(in pipeInput) engine.Input {
	ei := engine.Input{Name: in.name, Spec: snapshotPath(in.spec)}
	if in.members != nil {
		ei.Members = snapshotMembers(in.members)
	}
	return ei
}

func snapshotGraphInput(in graphInput) engine.Input {
	ei := engine.Input{Name: in.name, Spec: snapshotPath(in.spec)}
	if in.members != nil {
		ei.Members = snapshotGraphMembers(in.members)
	}
	return ei
}

func snapshotMembers(in Group) []engine.Member {
	if in == nil {
		return nil
	}
	out := make([]engine.Member, 0, len(in))
	for _, m := range in {
		out = append(out, engine.Member{Name: m.Name, Spec: snapshotPath(m.Spec)})
	}
	return out
}

func snapshotGraphMembers(in []graphMember) []engine.Member {
	if in == nil {
		return nil
	}
	out := make([]engine.Member, 0, len(in))
	for _, m := range in {
		out = append(out, engine.Member{Name: m.name, Spec: snapshotPath(m.spec)})
	}
	return out
}

func snapshotBind(b Bind, p *Pipeline) engine.Bind {
	eb := engine.Bind{
		Name: b.Name,
		Spec: snapshotPath(b.Spec),
		Rule: engine.DeriveRule(b.Rule),
	}
	eb.FromKind, eb.FromName, eb.FromTask = snapshotFrom(b.From, p)
	if b.Group != nil {
		eb.Members = snapshotMembers(b.Group)
	}
	return eb
}

func snapshotGraphBind(b graphBind) engine.Bind {
	eb := engine.Bind{
		Name:     b.name,
		Spec:     snapshotPath(b.spec),
		FromKind: snapshotFromKind(b.fromKind),
		FromName: b.fromName,
		FromTask: b.fromTask,
		Resolved: true,
	}
	if b.members != nil {
		eb.Members = snapshotGraphMembers(b.members)
	}
	return eb
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

func snapshotFrom(h Handle, p *Pipeline) (engine.FromKind, string, string) {
	// A non-zero authored Handle stays a From. Do not report FromZero
	// for a foreign handle, or an output bind with a complete Spec
	// looks like "no From".
	if foreignFrom(h, p) {
		return snapshotFromKind(h.kind), "", ""
	}
	switch h.kind {
	case handleInput:
		return engine.FromInput, h.name, ""
	case handleOut:
		return engine.FromOut, h.name, h.task.id()
	case handleIn:
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
