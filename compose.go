package gobble

import "github.com/HahyeonJeon/gobble/internal/engine"

// Compose builds an immutable Graph from p.
//
// A From handle must belong to p. A handle from another Pipeline is
// missing-input on that bind, including output binds.
// If p has a recorded compose error from [Pipeline.RecordComposeError],
// Compose returns (nil, that *Error) and does not build a graph.
// On any compose defect it returns (nil, *Error) with Op "compose".
func Compose(p *Pipeline) (*Graph, error) {
	if p == nil {
		return nil, &Error{Op: "compose", Defects: []Defect{{
			Code:    DefectInvalidRequest,
			Message: "nil pipeline",
		}}}
	}
	if p.composeErr != nil {
		return nil, cloneError(p.composeErr)
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
	treeMemo   map[bindKey]Directory
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
		treeMemo:   make(map[bindKey]Directory),
		walking:    make(map[bindKey]bool),
	}
	return r.buildGraph()
}

func (r *resolver) buildGraph() *Graph {
	g := &Graph{name: r.p.name}
	for _, in := range r.p.inputs {
		gi := graphInput{name: in.name, spec: in.spec.clone(), tree: in.tree}
		if in.members != nil {
			gi.members = authoredMembers(in.members)
		}
		g.inputs = append(g.inputs, gi)
	}
	for _, t := range r.p.tasks {
		gt := graphTask{
			display: cloneDisplay(t.spec.Display),
			id:        t.id(),
			name:      t.spec.Name,
			module:    t.nearest(nodeModule),
			branch:    t.nearest(nodeBranch),
			merge:     t.nearest(nodeMerge),
			scatter:   t.nearest(nodeScatter),
			gather:    t.nearest(nodeGather),
			when:      t.nearest(nodeWhen),
			command:   copyStrings(t.spec.Command),
			script:    t.spec.Script,
			image:     t.spec.Image,
			backend:   t.spec.Backend,
			resources: t.spec.Resources,
			params:    copyParams(t.spec.Params),
			env:       copyEnv(t.spec.Env),
		}
		r.copyOperatorFacts(t, &gt)
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

func (r *resolver) copyOperatorFacts(t *Task, gt *graphTask) {
	if sc := t.scatterOp(); sc != nil && !sc.from.IsZero() && !foreignFrom(sc.from, r.p) {
		gt.scatterFromKind = sc.from.kind
		gt.scatterFromName = sc.from.name
		if sc.from.task != nil {
			gt.scatterFromTask = sc.from.task.id()
		}
		gt.scatterFromPath = staticScatterFromPath(r.p, sc.from)
		gt.scatterMembers, gt.scatterMemberPaths = staticScatterMembers(r.p, sc.from)
	}
	if w := t.whenOp(); w != nil {
		gt.skipFalse = w.skipFalse
		if !w.skipMissing.IsZero() && !foreignFrom(w.skipMissing, r.p) {
			gt.skipMissingKind = w.skipMissing.kind
			gt.skipMissingName = w.skipMissing.name
			if w.skipMissing.task != nil {
				gt.skipMissingTask = w.skipMissing.task.id()
			}
			gt.skipMissingPath = skipMissingPath(r.p, w.skipMissing)
		}
	}
}

func staticScatterMembers(p *Pipeline, h Handle) ([]string, []string) {
	if h.kind != handleInput {
		return nil, nil
	}
	for _, in := range p.inputs {
		if in.name != h.name {
			continue
		}
		if in.members != nil {
			names := make([]string, 0, len(in.members))
			paths := make([]string, 0, len(in.members))
			for _, m := range in.members {
				names = append(names, m.Name)
				path, err := m.Spec.Render()
				if err != nil {
					paths = append(paths, "")
					continue
				}
				paths = append(paths, path)
			}
			return names, paths
		}
		if !in.tree.IsZero() {
			return nil, nil
		}
		path, err := in.spec.Render()
		if err != nil || path == "" {
			return nil, nil
		}
		return []string{path}, []string{path}
	}
	return nil, nil
}

func staticScatterFromPath(p *Pipeline, h Handle) string {
	if h.kind != handleInput {
		return ""
	}
	for _, in := range p.inputs {
		if in.name != h.name {
			continue
		}
		if !in.tree.IsZero() {
			return in.tree.Dir.String()
		}
		if in.members != nil {
			return ""
		}
		path, err := in.spec.Render()
		if err != nil {
			return ""
		}
		return path
	}
	return ""
}

func skipMissingPath(p *Pipeline, h Handle) string {
	switch h.kind {
	case handleInput:
		for _, in := range p.inputs {
			if in.name != h.name {
				continue
			}
			if in.members != nil || !in.tree.IsZero() {
				return ""
			}
			path, err := in.spec.Render()
			if err != nil {
				return ""
			}
			return path
		}
	case handleOut:
		if h.task == nil {
			return ""
		}
		b, ok := findBind(h.task.spec.Outputs, h.name)
		if !ok || b.Group != nil || !b.Tree.IsZero() {
			return ""
		}
		path, err := b.Spec.Render()
		if err != nil {
			return ""
		}
		return path
	}
	return ""
}

func scatterFileFromProducer(t *Task, b Bind) bool {
	if t == nil || b.Group != nil || !b.Tree.IsZero() {
		return false
	}
	sc := t.scatterOp()
	if sc == nil || sc.from.IsZero() {
		return false
	}
	return sameHandle(b.From, sc.from)
}

func scatterMemberTreeFromProducer(t *Task, b Bind) bool {
	if t == nil || b.Tree.IsZero() {
		return false
	}
	sc := t.scatterOp()
	if sc == nil || sc.from.IsZero() {
		return false
	}
	return sameHandle(b.From, sc.from)
}

func fromScatterChild(t *Task, b Bind) bool {
	if t == nil || b.From.task == nil {
		return false
	}
	return b.From.task.scatterOp() != nil
}

func (r *resolver) graphBind(t *Task, b Bind, out bool) graphBind {
	gb := graphBind{
		name:     b.Name,
		fromKind: b.From.kind,
		fromName: b.From.name,
	}
	if !b.Tree.IsZero() {
		if dir, ok := r.resolveTree(t, b, out); ok {
			gb.tree = DeclareTree(dir)
		} else {
			gb.tree = b.Tree
		}
	} else if b.Group != nil {
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
	if b.From.IsZero() || scatterFileFromProducer(t, b) || fromScatterChild(t, b) {
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

func (r *resolver) resolveTree(t *Task, b Bind, out bool) (Directory, bool) {
	if b.Tree.IsZero() {
		return Directory{}, true
	}
	key := bindKey{task: t, name: b.Name, out: out}
	if dir, ok := r.treeMemo[key]; ok {
		return dir, true
	}
	if r.walking[key] {
		return Directory{}, false
	}
	if b.From.IsZero() {
		r.treeMemo[key] = b.Tree.Dir
		return b.Tree.Dir, true
	}
	if out && scatterMemberTreeFromProducer(t, b) {
		r.treeMemo[key] = b.Tree.Dir
		return b.Tree.Dir, true
	}
	r.walking[key] = true
	from, ok := r.resolveFromTree(b.From)
	r.walking[key] = false
	if !ok {
		return Directory{}, false
	}
	dir := b.Tree.Dir
	if dir.IsZero() {
		dir = from
	}
	r.treeMemo[key] = dir
	return dir, true
}

func (r *resolver) resolveFromTree(h Handle) (Directory, bool) {
	if foreignFrom(h, r.p) {
		return Directory{}, false
	}
	switch h.kind {
	case handleInput:
		for _, in := range r.p.inputs {
			if in.name == h.name {
				if in.tree.IsZero() {
					return Directory{}, false
				}
				return in.tree.Dir, true
			}
		}
		return Directory{}, false
	case handleOut:
		b, ok := findBind(h.task.spec.Outputs, h.name)
		if !ok || b.Tree.IsZero() {
			return Directory{}, false
		}
		return r.resolveTree(h.task, b, true)
	case handleIn:
		b, ok := findBind(h.task.spec.Inputs, h.name)
		if !ok || b.Tree.IsZero() {
			return Directory{}, false
		}
		return r.resolveTree(h.task, b, false)
	default:
		return Directory{}, false
	}
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
