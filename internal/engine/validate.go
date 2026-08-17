package engine

import (
	"math"
	"regexp"
	"strings"
)

var namePat = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

type bindKey struct {
	task string
	name string
	out  bool
}

type rendered struct {
	task string
	unit string
	path string
	out  bool
}

type checker struct {
	s        Snapshot
	plan     bool
	defects  []Defect
	tasks    map[string]*Task
	inputs   map[string]Input
	memo     map[bindKey]Path
	walking  map[bindKey]bool
	rendered []rendered
}

// ComposeCheck reports compose defects on s. It does not check conflicts
// or backends.
func ComposeCheck(s Snapshot) []Defect {
	return check(s, false)
}

// Validate reports compose defects, rendered-path conflicts, and
// unsupported backends on s.
func Validate(s Snapshot) []Defect {
	return check(s, true)
}

func check(s Snapshot, plan bool) []Defect {
	c := &checker{
		s:       s,
		plan:    plan,
		tasks:   make(map[string]*Task, len(s.Tasks)),
		inputs:  make(map[string]Input, len(s.Inputs)),
		memo:    make(map[bindKey]Path),
		walking: make(map[bindKey]bool),
	}
	for i := range s.Tasks {
		t := &s.Tasks[i]
		if t.ID != "" {
			c.tasks[t.ID] = t
		}
	}
	for _, in := range s.Inputs {
		if in.Name != "" {
			c.inputs[in.Name] = in
		}
	}
	c.checkName(s.Name, s.Name, "pipeline")
	c.checkInputs()
	if len(s.Nodes) > 0 {
		c.walkNodes("", s.Nodes)
	} else {
		for i := range s.Tasks {
			t := &s.Tasks[i]
			c.checkName(t.Name, t.ID, "unit")
			c.checkTask(t)
		}
	}
	c.checkCycles()
	c.checkPaths()
	if plan {
		c.checkConflicts()
	}
	return c.defects
}

func (c *checker) add(code, unit, message string, paths ...string) {
	c.defects = append(c.defects, Defect{
		Code:    code,
		Unit:    unit,
		Message: message,
		Paths:   paths,
	})
}

func (c *checker) checkName(name, unit, kind string) {
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

func (c *checker) checkInputs() {
	seen := make(map[string]bool)
	for _, in := range c.s.Inputs {
		c.checkName(in.Name, in.Name, "input")
		if in.Name != "" && seen[in.Name] {
			c.add(DefectInvalidName, in.Name, "duplicate name")
			continue
		}
		if in.Name != "" && namePat.MatchString(in.Name) {
			seen[in.Name] = true
		}
	}
}

func (c *checker) walkNodes(prefix string, nodes []Node) {
	seen := make(map[string]bool)
	for _, n := range nodes {
		id := childID(prefix, n.Name)
		c.checkName(n.Name, id, "unit")
		if n.Name != "" && seen[n.Name] {
			c.add(DefectInvalidName, id, "duplicate name")
		} else if n.Name != "" && namePat.MatchString(n.Name) {
			seen[n.Name] = true
		}
		switch n.Kind {
		case NodeModule, NodeBranch, NodeMerge:
			c.walkNodes(id, n.Children)
		case NodeTask:
			if t := c.lookupTask(n.TaskID, id); t != nil {
				c.checkTask(t)
			}
		}
	}
}

func (c *checker) lookupTask(taskID, id string) *Task {
	if t, ok := c.tasks[taskID]; ok {
		return t
	}
	if t, ok := c.tasks[id]; ok {
		return t
	}
	return nil
}

func (c *checker) checkTask(t *Task) {
	id := t.ID
	if id == "" {
		id = t.Name
	}
	if len(t.Command) == 0 {
		c.add(DefectMissingCommand, id, "missing command")
	}
	if len(t.Outputs) == 0 {
		c.add(DefectMissingOutput, id, "missing output")
	}
	if c.plan && t.Backend != "" && t.Backend != "local" {
		c.add(DefectUnsupportedBackend, id, "unsupported backend")
	}
	if c.plan && !finiteCPU(t.CPU) {
		c.add(DefectInvalidName, id, "non-finite cpu")
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
	for _, b := range t.Inputs {
		checkPort(b)
		if b.FromKind == FromZero || !c.fromInGraph(b) {
			c.add(DefectMissingInput, bindUnit(id, b.Name), "missing input")
		}
	}
	for _, b := range t.Outputs {
		checkPort(b)
		if b.FromKind != FromZero && !c.fromInGraph(b) {
			c.add(DefectMissingInput, bindUnit(id, b.Name), "missing input")
		}
	}
	seenOut := make(map[string]bool)
	for _, name := range t.OutCalls {
		if seenOut[name] {
			continue
		}
		seenOut[name] = true
		if !hasBind(t.Outputs, name) {
			c.add(DefectMissingOutput, bindUnit(id, name), "missing output")
		}
	}
	seenIn := make(map[string]bool)
	for _, name := range t.InCalls {
		if seenIn[name] {
			continue
		}
		seenIn[name] = true
		if !hasBind(t.Inputs, name) {
			c.add(DefectMissingInput, bindUnit(id, name), "missing input")
		}
	}
}

func (c *checker) fromInGraph(b Bind) bool {
	switch b.FromKind {
	case FromInput:
		_, ok := c.inputs[b.FromName]
		return ok
	case FromOut, FromIn:
		_, ok := c.tasks[b.FromTask]
		return ok
	default:
		return false
	}
}

func (c *checker) checkCycles() {
	adj := make(map[string][]string)
	addEdges := func(t *Task, binds []Bind) {
		for _, b := range binds {
			if b.FromTask == "" || c.tasks[b.FromTask] == nil {
				continue
			}
			adj[b.FromTask] = append(adj[b.FromTask], t.ID)
		}
	}
	for i := range c.s.Tasks {
		t := &c.s.Tasks[i]
		addEdges(t, t.Inputs)
		addEdges(t, t.Outputs)
	}
	for i := range c.s.Tasks {
		t := &c.s.Tasks[i]
		if reachesSelf(t.ID, adj) {
			c.add(DefectCycle, t.ID, "cycle")
		}
	}
}

func reachesSelf(start string, adj map[string][]string) bool {
	if start == "" {
		return false
	}
	seen := make(map[string]bool)
	var walk func(string) bool
	walk = func(cur string) bool {
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

func (c *checker) checkPaths() {
	for _, in := range c.s.Inputs {
		c.renderPath(in.Spec, in.Name)
	}
	for i := range c.s.Tasks {
		t := &c.s.Tasks[i]
		id := t.ID
		for _, b := range t.Inputs {
			spec, ok := c.resolveBind(t, b, false)
			if !ok {
				continue
			}
			c.recordPath(spec, bindUnit(id, b.Name), id, false)
		}
		for _, b := range t.Outputs {
			spec, ok := c.resolveBind(t, b, true)
			if !ok {
				continue
			}
			c.recordPath(spec, bindUnit(id, b.Name), id, true)
		}
	}
}

func (c *checker) recordPath(spec Path, unit, task string, out bool) {
	got, d := spec.Render()
	if d != nil {
		c.add(DefectInvalidPath, unit, d.Message, d.Paths...)
		return
	}
	c.rendered = append(c.rendered, rendered{
		task: task,
		unit: unit,
		path: comparablePath(got),
		out:  out,
	})
}

func (c *checker) renderPath(spec Path, unit string) {
	_, d := spec.Render()
	if d == nil {
		return
	}
	c.add(DefectInvalidPath, unit, d.Message, d.Paths...)
}

func (c *checker) resolveBind(t *Task, b Bind, out bool) (Path, bool) {
	if b.Resolved {
		return b.Spec.clone(), true
	}
	key := bindKey{task: t.ID, name: b.Name, out: out}
	if spec, ok := c.memo[key]; ok {
		return spec, true
	}
	if c.walking[key] {
		return Path{}, false
	}
	if b.FromKind == FromZero {
		spec := b.Spec.clone()
		c.memo[key] = spec
		return spec, true
	}
	c.walking[key] = true
	from, ok := c.resolveFrom(b)
	c.walking[key] = false
	if !ok {
		return Path{}, false
	}
	spec := classifyPath(b.Spec, from, b.Rule)
	c.memo[key] = spec
	return spec, true
}

func (c *checker) resolveFrom(b Bind) (Path, bool) {
	switch b.FromKind {
	case FromInput:
		in, ok := c.inputs[b.FromName]
		if !ok {
			return Path{}, false
		}
		return in.Spec.clone(), true
	case FromOut:
		src, ok := c.tasks[b.FromTask]
		if !ok {
			return Path{}, false
		}
		ob, ok := findBind(src.Outputs, b.FromName)
		if !ok {
			return Path{}, false
		}
		return c.resolveBind(src, ob, true)
	case FromIn:
		src, ok := c.tasks[b.FromTask]
		if !ok {
			return Path{}, false
		}
		ib, ok := findBind(src.Inputs, b.FromName)
		if !ok {
			return Path{}, false
		}
		return c.resolveBind(src, ib, false)
	default:
		return Path{}, false
	}
}

func (c *checker) checkConflicts() {
	byTaskOut := make(map[string]map[string]string)
	outputs := make(map[string]string)
	for _, r := range c.rendered {
		if r.path == "" {
			continue
		}
		if !r.out {
			continue
		}
		if byTaskOut[r.task] == nil {
			byTaskOut[r.task] = make(map[string]string)
		}
		byTaskOut[r.task][r.path] = r.unit
		if _, ok := outputs[r.path]; ok {
			c.add(DefectConflict, r.unit, "conflict", r.path)
		} else {
			outputs[r.path] = r.unit
		}
	}
	for _, r := range c.rendered {
		if r.out {
			continue
		}
		outs := byTaskOut[r.task]
		if outs == nil {
			continue
		}
		if outUnit, ok := outs[r.path]; ok {
			c.add(DefectConflict, outUnit, "conflict", r.path)
		}
	}
}

func comparablePath(p string) string {
	p = strings.ReplaceAll(p, `\`, "/")
	cleaned, escaped := cleanPath(p)
	if escaped || cleaned == "" {
		return p
	}
	return cleaned
}

func finiteCPU(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
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

func bindUnit(taskID, port string) string {
	if taskID == "" {
		return port
	}
	if port == "" {
		return taskID
	}
	return taskID + "." + port
}

func hasBind(binds []Bind, name string) bool {
	_, ok := findBind(binds, name)
	return ok
}

func findBind(binds []Bind, name string) (Bind, bool) {
	for _, b := range binds {
		if b.Name == name {
			return b, true
		}
	}
	return Bind{}, false
}
