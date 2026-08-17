package gobble

import "strings"

// Pipeline is the root builder for a compose graph.
type Pipeline struct {
	name     string
	inputs   []pipeInput
	children []node
	tasks    []*Task
}

type pipeInput struct {
	name string
	spec PathSpec
}

type nodeKind int

const (
	nodeModule nodeKind = iota
	nodeBranch
	nodeMerge
	nodeTask
)

type ancestor struct {
	kind nodeKind
	name string
}

type node struct {
	kind   nodeKind
	name   string
	module *Module
	branch *Branch
	merge  *Merge
	task   *Task
}

// Module is a named group of tasks, branches, and merges.
type Module struct {
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

// Merge is a named fan-in of known branches.
// The branch list records author intent. Edges come from Bind.From wiring,
// not from this list.
type Merge struct {
	pipe     *Pipeline
	anc      []ancestor
	name     string
	branches []*Branch
	children []node
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

// AddInput records a pipeline input and returns a non-zero Handle for it.
func (p *Pipeline) AddInput(name string, spec PathSpec) Handle {
	p.inputs = append(p.inputs, pipeInput{name: name, spec: spec})
	return Handle{kind: handleInput, name: name, spec: spec.clone(), pipe: p}
}

// AddModule records a child module and returns it.
func (p *Pipeline) AddModule(name string) *Module {
	m := &Module{pipe: p, name: name}
	p.children = append(p.children, node{kind: nodeModule, name: name, module: m})
	return m
}

// AddTask records a child task and returns it.
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

// Merge records a child merge of branches and returns it.
// The branch list records author intent. Edges come from Bind.From wiring,
// not from this list.
func (p *Pipeline) Merge(name string, branches ...*Branch) *Merge {
	mg := &Merge{pipe: p, name: name, branches: append([]*Branch(nil), branches...)}
	p.children = append(p.children, node{kind: nodeMerge, name: name, merge: mg})
	return mg
}

// AddModule records a child module and returns it.
func (m *Module) AddModule(name string) *Module {
	child := &Module{pipe: m.pipe, anc: childAnc(m.anc, nodeModule, m.name), name: name}
	m.children = append(m.children, node{kind: nodeModule, name: name, module: child})
	return child
}

// AddTask records a child task and returns it.
func (m *Module) AddTask(spec TaskSpec) *Task {
	t := m.pipe.newTask(childAnc(m.anc, nodeModule, m.name), spec)
	m.children = append(m.children, node{kind: nodeTask, name: spec.Name, task: t})
	return t
}

// Branch records a child branch and returns it.
func (m *Module) Branch(name string) *Branch {
	b := &Branch{pipe: m.pipe, anc: childAnc(m.anc, nodeModule, m.name), name: name}
	m.children = append(m.children, node{kind: nodeBranch, name: name, branch: b})
	return b
}

// Merge records a child merge of branches and returns it.
// The branch list records author intent. Edges come from Bind.From wiring,
// not from this list.
func (m *Module) Merge(name string, branches ...*Branch) *Merge {
	mg := &Merge{
		pipe:     m.pipe,
		anc:      childAnc(m.anc, nodeModule, m.name),
		name:     name,
		branches: append([]*Branch(nil), branches...),
	}
	m.children = append(m.children, node{kind: nodeMerge, name: name, merge: mg})
	return mg
}

// AddTask records a child task and returns it.
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

// AddTask records a child task and returns it.
func (mg *Merge) AddTask(spec TaskSpec) *Task {
	t := mg.pipe.newTask(childAnc(mg.anc, nodeMerge, mg.name), spec)
	mg.children = append(mg.children, node{kind: nodeTask, name: spec.Name, task: t})
	return t
}

// Out returns a non-zero Handle that records a request for output port name.
func (t *Task) Out(name string) Handle {
	t.outCalls = append(t.outCalls, name)
	spec := PathSpec{}
	if b, ok := findBind(t.spec.Outputs, name); ok {
		spec = b.Spec.clone()
	}
	return Handle{kind: handleOut, name: name, spec: spec, task: t}
}

// In returns a non-zero Handle that records a request for input port name.
func (t *Task) In(name string) Handle {
	t.inCalls = append(t.inCalls, name)
	spec := PathSpec{}
	if b, ok := findBind(t.spec.Inputs, name); ok {
		spec = b.Spec.clone()
	}
	return Handle{kind: handleIn, name: name, spec: spec, task: t}
}

// Spec returns the PathSpec recorded when the handle was created.
// For Out and In that is the authored Bind spec, not the path Compose
// later resolves for the plan.
func (h Handle) Spec() PathSpec {
	return h.spec.clone()
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
	t := &Task{pipe: p, anc: append([]ancestor(nil), anc...), spec: spec}
	p.tasks = append(p.tasks, t)
	return t
}

func childAnc(anc []ancestor, kind nodeKind, name string) []ancestor {
	out := make([]ancestor, len(anc)+1)
	copy(out, anc)
	out[len(anc)] = ancestor{kind: kind, name: name}
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

func bindUnit(taskID, port string) string {
	if taskID == "" {
		return port
	}
	if port == "" {
		return taskID
	}
	return taskID + "." + port
}
