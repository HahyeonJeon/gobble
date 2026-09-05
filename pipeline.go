package gobble

import (
	"errors"
	"strings"
)

// Pipeline is the root builder for a compose graph.
type Pipeline struct {
	name       string
	inputs     []pipeInput
	children   []node
	tasks      []*Task
	composeErr *Error
}

type pipeInput struct {
	name    string
	spec    PathSpec
	members Group
	tree    Tree
}

type nodeKind int

const (
	nodeModule nodeKind = iota
	nodeBranch
	nodeMerge
	nodeTask
	nodeScatter
	nodeGather
	nodeWhen
)

type ancestor struct {
	display TaskDisplay
	kind    nodeKind
	name    string
	scatter *Scatter
	gather  *Gather
	when    *When
}

type node struct {
	kind    nodeKind
	name    string
	module  *Module
	branch  *Branch
	merge   *Merge
	scatter *Scatter
	gather  *Gather
	when    *When
	task    *Task
}

// Module is a named group of tasks, branches, and merges.
type Module struct {
	display TaskDisplay
	pipe     *Pipeline
	anc      []ancestor
	name     string
	children []node
}

// Branch is a named fan-out of a known subgraph.
type Branch struct {
	pipe     *Pipeline
	anc      []ancestor
	name     string
	children []node
}

// Merge is a named fan-in scope. Edges come from Bind.From wiring.
type Merge struct {
	pipe     *Pipeline
	anc      []ancestor
	name     string
	children []node
}

// Scatter is a named membership fan-out. Membership is From a Group,
// Tree, or File Handle. Child tasks stay the authored task list.
type Scatter struct {
	pipe     *Pipeline
	anc      []ancestor
	name     string
	from     Handle
	children []node
}

// Gather is a named fan-in of scatter members. Edges come from
// Bind.From wiring, like Merge.
type Gather struct {
	pipe     *Pipeline
	anc      []ancestor
	name     string
	children []node
}

// When is a named conditional. SkipIfMissing and SkipIfFalse are the
// only predicates. A When with no predicate never skips.
type When struct {
	pipe        *Pipeline
	anc         []ancestor
	name        string
	skipMissing Handle
	skipFalse   string
	children    []node
}

// Task is a recorded task builder.
type Task struct {
	pipe     *Pipeline
	anc      []ancestor
	spec     TaskSpec
	outCalls []string
	inCalls  []string
}

type handleKind int

const (
	handleZero handleKind = iota
	handleInput
	handleOut
	handleIn
)

// Handle is a wiring token for a pipeline input or task port.
//
// Only the zero Handle has IsZero true. It means From was never set.
type Handle struct {
	kind handleKind
	name string
	spec PathSpec
	tree Tree
	task *Task
	pipe *Pipeline
}

// NewPipeline returns a pipeline builder named name.
func NewPipeline(name string) *Pipeline {
	return &Pipeline{name: name}
}

// Name returns the pipeline name.
func (p *Pipeline) Name() string {
	return p.name
}

// RecordComposeError records a copy of err so Compose returns a caller-owned
// copy and does not build a graph. A nil receiver or nil err is a no-op. err
// is stored as [*Error] with Op "compose". The first recorded error is kept.
func (p *Pipeline) RecordComposeError(err error) {
	if p == nil || err == nil || p.composeErr != nil {
		return
	}
	if ge := normalizeComposeError(err); ge != nil {
		p.composeErr = ge
	}
}

func normalizeComposeError(err error) *Error {
	var ge *Error
	if errors.As(err, &ge) {
		if ge == nil {
			return nil
		}
		out := &Error{Op: "compose", Defects: make([]Defect, len(ge.Defects))}
		for i, d := range ge.Defects {
			out.Defects[i] = Defect{
				Code:    d.Code,
				Unit:    d.Unit,
				Message: d.Message,
				Paths:   copyStrings(d.Paths),
			}
		}
		return out
	}
	return &Error{
		Op: "compose",
		Defects: []Defect{{
			Code:    DefectInvalidRequest,
			Message: err.Error(),
		}},
	}
}

// AddInput records a copied pipeline input and returns a non-zero Handle.
func (p *Pipeline) AddInput(name string, spec PathSpec) Handle {
	p.inputs = append(p.inputs, pipeInput{name: name, spec: spec.clone()})
	return Handle{kind: handleInput, name: name, spec: spec.clone(), pipe: p}
}

// AddInputGroup records a copied Group pipeline input named name and returns
// a non-zero Handle for it. Members must be a non-empty Group. A nil or empty
// Group is invalid at compose time, matching other Group rules.
func (p *Pipeline) AddInputGroup(name string, members Group) Handle {
	if members == nil {
		members = Group{}
	}
	p.inputs = append(p.inputs, pipeInput{name: name, members: copyGroup(members)})
	return Handle{kind: handleInput, name: name, pipe: p}
}

// AddInputTree records a Tree pipeline input named name and returns a
// non-zero Handle for it. tree.Dir must be non-zero at compose time.
func (p *Pipeline) AddInputTree(name string, tree Tree) Handle {
	p.inputs = append(p.inputs, pipeInput{name: name, tree: tree})
	return Handle{kind: handleInput, name: name, tree: tree, pipe: p}
}

// AddModule records a child module and returns it.
func (p *Pipeline) AddModule(name string) *Module {
	m := &Module{pipe: p, name: name}
	p.children = append(p.children, node{kind: nodeModule, name: name, module: m})
	return m
}

// AddTask records a child task and returns it. Mutable fields are copied.
func (p *Pipeline) AddTask(spec TaskSpec) *Task {
	t := p.newTask(nil, spec)
	p.children = append(p.children, node{kind: nodeTask, name: spec.Name, task: t})
	return t
}

// Branch records a child branch and returns it.
func (p *Pipeline) Branch(name string) *Branch {
	b := &Branch{pipe: p, name: name}
	p.children = append(p.children, node{kind: nodeBranch, name: name, branch: b})
	return b
}

// Merge records a child merge and returns it. Edges come from Bind.From
// wiring, not from a branch list.
func (p *Pipeline) Merge(name string) *Merge {
	mg := &Merge{pipe: p, name: name}
	p.children = append(p.children, node{kind: nodeMerge, name: name, merge: mg})
	return mg
}

// Scatter records a child scatter and returns it.
func (p *Pipeline) Scatter(name string) *Scatter {
	s := &Scatter{pipe: p, name: name}
	p.children = append(p.children, node{kind: nodeScatter, name: name, scatter: s})
	return s
}

// Gather records a child gather and returns it. Edges come from
// Bind.From wiring, not from a scatter list.
func (p *Pipeline) Gather(name string) *Gather {
	g := &Gather{pipe: p, name: name}
	p.children = append(p.children, node{kind: nodeGather, name: name, gather: g})
	return g
}

// When records a child conditional and returns it.
func (p *Pipeline) When(name string) *When {
	w := &When{pipe: p, name: name}
	p.children = append(p.children, node{kind: nodeWhen, name: name, when: w})
	return w
}

// AddModule records a child module and returns it.
func (m *Module) AddModule(name string) *Module {
	child := &Module{pipe: m.pipe, anc: m.ancestors(), name: name}
	m.children = append(m.children, node{kind: nodeModule, name: name, module: child})
	return child
}

// AddTask records a child task and returns it. Mutable fields are copied.
func (m *Module) AddTask(spec TaskSpec) *Task {
	t := m.pipe.newTask(m.ancestors(), spec)
	m.children = append(m.children, node{kind: nodeTask, name: spec.Name, task: t})
	return t
}

// Branch records a child branch and returns it.
func (m *Module) Branch(name string) *Branch {
	b := &Branch{pipe: m.pipe, anc: m.ancestors(), name: name}
	m.children = append(m.children, node{kind: nodeBranch, name: name, branch: b})
	return b
}

// Merge records a child merge and returns it. Edges come from Bind.From
// wiring, not from a branch list.
func (m *Module) Merge(name string) *Merge {
	mg := &Merge{
		pipe: m.pipe,
		anc:  m.ancestors(),
		name: name,
	}
	m.children = append(m.children, node{kind: nodeMerge, name: name, merge: mg})
	return mg
}

// Scatter records a child scatter and returns it.
func (m *Module) Scatter(name string) *Scatter {
	s := &Scatter{pipe: m.pipe, anc: m.ancestors(), name: name}
	m.children = append(m.children, node{kind: nodeScatter, name: name, scatter: s})
	return s
}

// Gather records a child gather and returns it. Edges come from
// Bind.From wiring, not from a scatter list.
func (m *Module) Gather(name string) *Gather {
	g := &Gather{pipe: m.pipe, anc: m.ancestors(), name: name}
	m.children = append(m.children, node{kind: nodeGather, name: name, gather: g})
	return g
}

// When records a child conditional and returns it.
func (m *Module) When(name string) *When {
	w := &When{pipe: m.pipe, anc: m.ancestors(), name: name}
	m.children = append(m.children, node{kind: nodeWhen, name: name, when: w})
	return w
}

// AddTask records a child task and returns it. Mutable fields are copied.
func (b *Branch) AddTask(spec TaskSpec) *Task {
	t := b.pipe.newTask(childAnc(b.anc, nodeBranch, b.name), spec)
	b.children = append(b.children, node{kind: nodeTask, name: spec.Name, task: t})
	return t
}

// AddModule records a child module and returns it.
func (b *Branch) AddModule(name string) *Module {
	m := &Module{pipe: b.pipe, anc: childAnc(b.anc, nodeBranch, b.name), name: name}
	b.children = append(b.children, node{kind: nodeModule, name: name, module: m})
	return m
}

// AddTask records a child task and returns it. Mutable fields are copied.
func (mg *Merge) AddTask(spec TaskSpec) *Task {
	t := mg.pipe.newTask(childAnc(mg.anc, nodeMerge, mg.name), spec)
	mg.children = append(mg.children, node{kind: nodeTask, name: spec.Name, task: t})
	return t
}

// From records the Group, Tree, or File membership source.
func (s *Scatter) From(h Handle) *Scatter {
	s.from = h
	return s
}

// AddTask records a child task and returns it. Mutable fields are copied.
func (s *Scatter) AddTask(spec TaskSpec) *Task {
	t := s.pipe.newTask(childAncOp(s.anc, nodeScatter, s.name, s, nil, nil), spec)
	s.children = append(s.children, node{kind: nodeTask, name: spec.Name, task: t})
	return t
}

// AddModule records a child module and returns it.
func (s *Scatter) AddModule(name string) *Module {
	m := &Module{pipe: s.pipe, anc: childAncOp(s.anc, nodeScatter, s.name, s, nil, nil), name: name}
	s.children = append(s.children, node{kind: nodeModule, name: name, module: m})
	return m
}

// AddTask records a child task and returns it. Mutable fields are copied.
func (g *Gather) AddTask(spec TaskSpec) *Task {
	t := g.pipe.newTask(childAncOp(g.anc, nodeGather, g.name, nil, g, nil), spec)
	g.children = append(g.children, node{kind: nodeTask, name: spec.Name, task: t})
	return t
}

// AddModule records a child module and returns it.
func (g *Gather) AddModule(name string) *Module {
	m := &Module{pipe: g.pipe, anc: childAncOp(g.anc, nodeGather, g.name, nil, g, nil), name: name}
	g.children = append(g.children, node{kind: nodeModule, name: name, module: m})
	return m
}

// SkipIfMissing records a File Handle skip predicate.
func (w *When) SkipIfMissing(h Handle) *When {
	w.skipMissing = h
	return w
}

// SkipIfFalse records a declared boolean param skip predicate.
func (w *When) SkipIfFalse(param string) *When {
	w.skipFalse = param
	return w
}

// AddTask records a child task and returns it. Mutable fields are copied.
func (w *When) AddTask(spec TaskSpec) *Task {
	t := w.pipe.newTask(childAncOp(w.anc, nodeWhen, w.name, nil, nil, w), spec)
	w.children = append(w.children, node{kind: nodeTask, name: spec.Name, task: t})
	return t
}

// AddModule records a child module and returns it.
func (w *When) AddModule(name string) *Module {
	m := &Module{pipe: w.pipe, anc: childAncOp(w.anc, nodeWhen, w.name, nil, nil, w), name: name}
	w.children = append(w.children, node{kind: nodeModule, name: name, module: m})
	return m
}

// Out returns a non-zero Handle that records a request for output port name.
func (t *Task) Out(name string) Handle {
	t.outCalls = append(t.outCalls, name)
	spec := PathSpec{}
	var tree Tree
	if b, ok := findBind(t.spec.Outputs, name); ok {
		spec = b.Spec.clone()
		tree = b.Tree
	}
	return Handle{kind: handleOut, name: name, spec: spec, tree: tree, task: t}
}

// In returns a non-zero Handle that records a request for input port name.
func (t *Task) In(name string) Handle {
	t.inCalls = append(t.inCalls, name)
	spec := PathSpec{}
	var tree Tree
	if b, ok := findBind(t.spec.Inputs, name); ok {
		spec = b.Spec.clone()
		tree = b.Tree
	}
	return Handle{kind: handleIn, name: name, spec: spec, tree: tree, task: t}
}

// Spec returns the PathSpec recorded when the handle was created.
// For Out and In that is the authored Bind spec, not the path Compose
// later resolves for the plan. Group and Tree handles return a zero Spec.
func (h Handle) Spec() PathSpec {
	return h.spec.clone()
}

// Tree returns the authored Tree recorded when the handle was created.
// File and Group handles return a zero Tree.
func (h Handle) Tree() Tree {
	return h.tree
}

// Name returns the input or port name recorded on this handle.
func (h Handle) Name() string {
	return h.name
}

// IsZero reports whether h is the zero Handle.
func (h Handle) IsZero() bool {
	return h.kind == handleZero
}

func (p *Pipeline) newTask(anc []ancestor, spec TaskSpec) *Task {
	var display TaskDisplay
	for _, a := range anc {
		display = inheritDisplay(display, a.display)
	}
	spec.Display = inheritDisplay(display, spec.Display)
	t := &Task{pipe: p, anc: append([]ancestor(nil), anc...), spec: copyTaskSpec(spec)}
	p.tasks = append(p.tasks, t)
	return t
}

func copyTaskSpec(spec TaskSpec) TaskSpec {
	out := spec
	out.Display = cloneDisplay(spec.Display)
	out.Command = copyStrings(spec.Command)
	out.Inputs = copyBinds(spec.Inputs)
	out.Outputs = copyBinds(spec.Outputs)
	out.Params = copyParams(spec.Params)
	out.Env = copyEnv(spec.Env)
	return out
}

func copyBinds(in []Bind) []Bind {
	if in == nil {
		return nil
	}
	out := make([]Bind, len(in))
	for i, bind := range in {
		out[i] = bind
		out[i].Spec = bind.Spec.clone()
		out[i].Group = copyGroup(bind.Group)
	}
	return out
}

func copyGroup(in Group) Group {
	if in == nil {
		return nil
	}
	out := make(Group, len(in))
	for i, member := range in {
		out[i] = member
		out[i].Spec = member.Spec.clone()
	}
	return out
}

func childAnc(anc []ancestor, kind nodeKind, name string) []ancestor {
	return childAncOp(anc, kind, name, nil, nil, nil)
}

func childAncOp(anc []ancestor, kind nodeKind, name string, sc *Scatter, g *Gather, w *When) []ancestor {
	out := make([]ancestor, len(anc)+1)
	copy(out, anc)
	out[len(anc)] = ancestor{kind: kind, name: name, scatter: sc, gather: g, when: w}
	return out
}

func unitID(anc []ancestor, name string) string {
	parts := make([]string, 0, len(anc)+1)
	for _, a := range anc {
		if a.name != "" {
			parts = append(parts, a.name)
		}
	}
	if name != "" {
		parts = append(parts, name)
	}
	return strings.Join(parts, ".")
}

func (t *Task) id() string {
	return unitID(t.anc, t.spec.Name)
}

func (t *Task) nearest(kind nodeKind) string {
	for i := len(t.anc) - 1; i >= 0; i-- {
		if t.anc[i].kind == kind {
			return t.anc[i].name
		}
	}
	return ""
}

func (t *Task) scatterOp() *Scatter {
	for i := len(t.anc) - 1; i >= 0; i-- {
		if t.anc[i].kind == nodeScatter {
			return t.anc[i].scatter
		}
	}
	return nil
}

func (t *Task) gatherOp() *Gather {
	for i := len(t.anc) - 1; i >= 0; i-- {
		if t.anc[i].kind == nodeGather {
			return t.anc[i].gather
		}
	}
	return nil
}

func (t *Task) whenOp() *When {
	for i := len(t.anc) - 1; i >= 0; i-- {
		if t.anc[i].kind == nodeWhen {
			return t.anc[i].when
		}
	}
	return nil
}

func sameHandle(a, b Handle) bool {
	if a.kind != b.kind || a.name != b.name {
		return false
	}
	switch a.kind {
	case handleInput:
		return a.pipe == b.pipe
	case handleOut, handleIn:
		return a.task == b.task
	default:
		return a.kind == handleZero && b.kind == handleZero
	}
}

func bindUnit(taskID, port string) string {
	if taskID == "" {
		return port
	}
	if port == "" {
		return taskID
	}
	return taskID + "." + port
}
