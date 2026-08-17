package gobble

import (
	"errors"
	"regexp"
)

var namePat = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

// Compose builds an immutable Graph from p.
//
// On any compose defect it returns (nil, *Error) with Op "compose".
func Compose(p *Pipeline) (*Graph, error) {
	g, defects := composeCheck(p)
	if len(defects) > 0 {
		return nil, &Error{Op: "compose", Defects: defects}
	}
	return g, nil
}

type composer struct {
	p       *Pipeline
	defects []Defect
	memo    map[bindKey]PathSpec
	walking map[bindKey]bool
}

type bindKey struct {
	task *Task
	name string
	out  bool
}

// composeCheck is the unexported compose-defect walk.
func composeCheck(p *Pipeline) (*Graph, []Defect) {
	if p == nil {
		return nil, []Defect{{
			Code:    DefectInvalidName,
			Message: "nil pipeline",
		}}
	}
	c := &composer{
		p:       p,
		memo:    make(map[bindKey]PathSpec),
		walking: make(map[bindKey]bool),
	}
	c.checkName(p.name, p.name, "pipeline")
	c.checkInputs()
	c.walkNodes("", p.children)
	c.checkCycles()
	c.checkPaths()
	if len(c.defects) > 0 {
		return nil, c.defects
	}
	return c.buildGraph(), nil
}

func (c *composer) add(code DefectCode, unit, message string, paths ...string) {
	c.defects = append(c.defects, Defect{
		Code:    code,
		Unit:    unit,
		Message: message,
		Paths:   paths,
	})
}

func (c *composer) checkName(name, unit, kind string) {
	switch {
	case name == "":
		if unit == "" {
			unit = kind
		}
		c.add(DefectInvalidName, unit, "empty name")
	case !namePat.MatchString(name):
		c.add(DefectInvalidName, unit, "invalid name")
	}
}

func (c *composer) checkInputs() {
	seen := make(map[string]bool)
	for _, in := range c.p.inputs {
		c.checkName(in.name, in.name, "input")
		if in.name != "" && seen[in.name] {
			c.add(DefectInvalidName, in.name, "duplicate name")
			continue
		}
		if in.name != "" && namePat.MatchString(in.name) {
			seen[in.name] = true
		}
	}
}

func (c *composer) walkNodes(prefix string, nodes []node) {
	seen := make(map[string]bool)
	for _, n := range nodes {
		id := childID(prefix, n.name)
		c.checkName(n.name, id, "unit")
		if n.name != "" && seen[n.name] {
			c.add(DefectInvalidName, id, "duplicate name")
		} else if n.name != "" && namePat.MatchString(n.name) {
			seen[n.name] = true
		}
		switch n.kind {
		case nodeModule:
			c.walkNodes(id, n.module.children)
		case nodeBranch:
			c.walkNodes(id, n.branch.children)
		case nodeMerge:
			c.walkNodes(id, n.merge.children)
		case nodeTask:
			c.checkTask(n.task)
		}
	}
}

func (c *composer) checkTask(t *Task) {
	id := t.id()
	if len(t.spec.Command) == 0 {
		c.add(DefectMissingCommand, id, "missing command")
	}
	if len(t.spec.Outputs) == 0 {
		c.add(DefectMissingOutput, id, "missing output")
	}
	seen := make(map[string]bool)
	checkPort := func(b Bind) {
		unit := bindUnit(id, b.Name)
		c.checkName(b.Name, unit, "bind")
		if b.Name != "" && seen[b.Name] {
			c.add(DefectInvalidName, unit, "duplicate name")
			return
		}
		if b.Name != "" && namePat.MatchString(b.Name) {
			seen[b.Name] = true
		}
	}
	for _, b := range t.spec.Inputs {
		checkPort(b)
		if b.From.IsZero() {
			c.add(DefectMissingInput, bindUnit(id, b.Name), "missing input")
		} else if !c.fromInGraph(b.From) {
			c.add(DefectMissingInput, bindUnit(id, b.Name), "missing input")
		}
	}
	for _, b := range t.spec.Outputs {
		checkPort(b)
	}
	seenOut := make(map[string]bool)
	for _, name := range t.outCalls {
		if seenOut[name] {
			continue
		}
		seenOut[name] = true
		if _, ok := findBind(t.spec.Outputs, name); !ok {
			c.add(DefectMissingOutput, bindUnit(id, name), "missing output")
		}
	}
	seenIn := make(map[string]bool)
	for _, name := range t.inCalls {
		if seenIn[name] {
			continue
		}
		seenIn[name] = true
		if _, ok := findBind(t.spec.Inputs, name); !ok {
			c.add(DefectMissingInput, bindUnit(id, name), "missing input")
		}
	}
}

func (c *composer) fromInGraph(h Handle) bool {
	switch h.kind {
	case handleInput:
		if h.pipe != c.p {
			return false
		}
		for _, in := range c.p.inputs {
			if in.name == h.name {
				return true
			}
		}
		return false
	case handleOut, handleIn:
		if h.task == nil || h.task.pipe != c.p {
			return false
		}
		return true
	default:
		return false
	}
}

func (c *composer) checkCycles() {
	adj := make(map[*Task][]*Task)
	addEdges := func(t *Task, binds []Bind) {
		for _, b := range binds {
			src := b.From.task
			if src == nil || src.pipe != c.p {
				continue
			}
			adj[src] = append(adj[src], t)
		}
	}
	for _, t := range c.p.tasks {
		addEdges(t, t.spec.Inputs)
		addEdges(t, t.spec.Outputs)
	}
	for _, t := range c.p.tasks {
		if reachesSelf(t, adj) {
			c.add(DefectCycle, t.id(), "cycle")
		}
	}
}

func reachesSelf(start *Task, adj map[*Task][]*Task) bool {
	seen := make(map[*Task]bool)
	var walk func(*Task) bool
	walk = func(cur *Task) bool {
		for _, n := range adj[cur] {
			if n == start {
				return true
			}
			if seen[n] {
				continue
			}
			seen[n] = true
			if walk(n) {
				return true
			}
		}
		return false
	}
	return walk(start)
}

func (c *composer) checkPaths() {
	for _, in := range c.p.inputs {
		c.renderPath(in.spec, in.name)
	}
	for _, t := range c.p.tasks {
		id := t.id()
		for _, b := range t.spec.Inputs {
			spec, ok := c.resolveBind(t, b, false)
			if !ok {
				continue
			}
			c.renderPath(spec, bindUnit(id, b.Name))
		}
		for _, b := range t.spec.Outputs {
			spec, ok := c.resolveBind(t, b, true)
			if !ok {
				continue
			}
			c.renderPath(spec, bindUnit(id, b.Name))
		}
	}
}

func (c *composer) renderPath(spec PathSpec, unit string) {
	_, err := spec.Render()
	if err == nil {
		return
	}
	var ge *Error
	if errors.As(err, &ge) && len(ge.Defects) > 0 {
		d := ge.Defects[0]
		c.add(DefectInvalidPath, unit, d.Message, d.Paths...)
		return
	}
	c.add(DefectInvalidPath, unit, "invalid path")
}

func (c *composer) resolveBind(t *Task, b Bind, out bool) (PathSpec, bool) {
	key := bindKey{task: t, name: b.Name, out: out}
	if spec, ok := c.memo[key]; ok {
		return spec, true
	}
	if c.walking[key] {
		return PathSpec{}, false
	}
	if b.From.IsZero() {
		spec := b.Spec.clone()
		c.memo[key] = spec
		return spec, true
	}
	c.walking[key] = true
	from, ok := c.resolveFrom(b.From)
	c.walking[key] = false
	if !ok {
		return PathSpec{}, false
	}
	spec := classifySpec(b.Spec, from, b.Rule)
	c.memo[key] = spec
	return spec, true
}

func (c *composer) resolveFrom(h Handle) (PathSpec, bool) {
	switch h.kind {
	case handleInput:
		if h.pipe != c.p {
			return PathSpec{}, false
		}
		for _, in := range c.p.inputs {
			if in.name == h.name {
				return in.spec.clone(), true
			}
		}
		return PathSpec{}, false
	case handleOut:
		if h.task == nil {
			return PathSpec{}, false
		}
		b, ok := findBind(h.task.spec.Outputs, h.name)
		if !ok {
			return PathSpec{}, false
		}
		return c.resolveBind(h.task, b, true)
	case handleIn:
		if h.task == nil {
			return PathSpec{}, false
		}
		b, ok := findBind(h.task.spec.Inputs, h.name)
		if !ok {
			return PathSpec{}, false
		}
		return c.resolveBind(h.task, b, false)
	default:
		return PathSpec{}, false
	}
}

func (c *composer) buildGraph() *Graph {
	g := &Graph{name: c.p.name}
	for _, in := range c.p.inputs {
		g.inputs = append(g.inputs, graphInput{name: in.name, spec: in.spec.clone()})
	}
	for _, t := range c.p.tasks {
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
			gt.inputs = append(gt.inputs, c.graphBind(t, b, false))
		}
		for _, b := range t.spec.Outputs {
			gt.outputs = append(gt.outputs, c.graphBind(t, b, true))
		}
		g.tasks = append(g.tasks, gt)
		g.edges = append(g.edges, fromEdges(t)...)
	}
	return g
}

func (c *composer) graphBind(t *Task, b Bind, out bool) graphBind {
	gb := graphBind{
		name:     b.Name,
		fromKind: b.From.kind,
		fromName: b.From.name,
	}
	if spec, ok := c.resolveBind(t, b, out); ok {
		gb.spec = spec.clone()
	} else {
		gb.spec = b.Spec.clone()
	}
	if b.From.task != nil {
		gb.fromTask = b.From.task.id()
	}
	return gb
}

func childID(prefix, name string) string {
	switch {
	case prefix != "" && name != "":
		return prefix + "." + name
	case prefix != "":
		return prefix
	default:
		return name
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
