package gobble

import (
	"regexp"
	"strings"
)

var namePat = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

type composeChecker struct {
	defects    []Defect
	tasks      map[string]*Task
	inputs     map[string]pipeInput
	memo       map[bindKey]PathSpec
	memberMemo map[bindKey][]graphMember
	walking    map[bindKey]bool
}

func composeCheckPipeline(p *Pipeline) []Defect {
	c := &composeChecker{
		tasks:      make(map[string]*Task, len(p.tasks)),
		inputs:     make(map[string]pipeInput, len(p.inputs)),
		memo:       make(map[bindKey]PathSpec),
		memberMemo: make(map[bindKey][]graphMember),
		walking:    make(map[bindKey]bool),
	}
	for _, t := range p.tasks {
		if id := t.id(); id != "" {
			c.tasks[id] = t
		}
	}
	for _, in := range p.inputs {
		if in.name != "" {
			c.inputs[in.name] = in
		}
	}
	c.checkName(p.name, p.name, "pipeline")
	c.checkPipeInputs(p)
	if len(p.children) > 0 {
		c.walkNodes("", p.children)
	} else {
		for _, t := range p.tasks {
			c.checkName(t.spec.Name, t.id(), "unit")
			c.checkPipeTask(t)
		}
	}
	c.checkPipeCycles(p)
	c.checkPipePaths(p)
	return c.defects
}

func graphCheck(g *Graph) []Defect {
	var defects []Defect
	c := &graphChecker{
		g:       g,
		defects: nil,
		tasks:   make(map[string]*graphTask, len(g.tasks)),
		inputs:  make(map[string]graphInput, len(g.inputs)),
	}
	for i := range g.tasks {
		t := &g.tasks[i]
		if t.id != "" {
			c.tasks[t.id] = t
		}
	}
	for _, in := range g.inputs {
		if in.name != "" {
			c.inputs[in.name] = in
		}
	}
	c.checkName(g.name, g.name, "pipeline")
	c.checkInputs()
	for i := range g.tasks {
		t := &g.tasks[i]
		c.checkName(t.name, t.id, "unit")
		c.checkTask(t)
	}
	c.checkCycles()
	c.checkPaths()
	defects = c.defects
	return defects
}

type graphChecker struct {
	g       *Graph
	defects []Defect
	tasks   map[string]*graphTask
	inputs  map[string]graphInput
}

func (c *composeChecker) add(code DefectCode, unit, message string, paths ...string) {
	c.defects = append(c.defects, Defect{
		Code:    code,
		Unit:    unit,
		Message: message,
		Paths:   paths,
	})
}

func (c *graphChecker) add(code DefectCode, unit, message string, paths ...string) {
	c.defects = append(c.defects, Defect{
		Code:    code,
		Unit:    unit,
		Message: message,
		Paths:   paths,
	})
}

func checkNameInto(add func(DefectCode, string, string, ...string), name, unit, kind string) {
	switch {
	case name == "":
		if unit == "" {
			unit = kind
		}
		add(DefectInvalidName, unit, "empty name")
	case !namePat.MatchString(name):
		add(DefectInvalidName, unit, "invalid name")
	}
}

func (c *composeChecker) checkName(name, unit, kind string) {
	checkNameInto(c.add, name, unit, kind)
}

func (c *graphChecker) checkName(name, unit, kind string) {
	checkNameInto(c.add, name, unit, kind)
}

func (c *composeChecker) checkPipeInputs(p *Pipeline) {
	seen := make(map[string]bool)
	for _, in := range p.inputs {
		c.checkName(in.name, in.name, "input")
		if in.name != "" && seen[in.name] {
			c.add(DefectInvalidName, in.name, "duplicate name")
			continue
		}
		if in.name != "" && namePat.MatchString(in.name) {
			seen[in.name] = true
		}
		if in.members != nil {
			c.checkGroup(in.name, in.spec, in.members)
		}
	}
}

func (c *graphChecker) checkInputs() {
	seen := make(map[string]bool)
	for _, in := range c.g.inputs {
		c.checkName(in.name, in.name, "input")
		if in.name != "" && seen[in.name] {
			c.add(DefectInvalidName, in.name, "duplicate name")
			continue
		}
		if in.name != "" && namePat.MatchString(in.name) {
			seen[in.name] = true
		}
		if in.members != nil {
			c.checkGraphGroup(in.name, in.spec, in.members)
		}
	}
}

func (c *composeChecker) walkNodes(prefix string, nodes []node) {
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
			if n.module != nil {
				c.walkNodes(id, n.module.children)
			}
		case nodeBranch:
			if n.branch != nil {
				c.walkNodes(id, n.branch.children)
			}
		case nodeMerge:
			if n.merge != nil {
				c.walkNodes(id, n.merge.children)
			}
		case nodeTask:
			if n.task != nil {
				c.checkPipeTask(n.task)
			}
		}
	}
}

func (c *composeChecker) checkPipeTask(t *Task) {
	id := t.id()
	if id == "" {
		id = t.spec.Name
	}
	hasCmd := len(t.spec.Command) > 0
	hasScript := t.spec.Script != ""
	switch {
	case hasCmd && hasScript:
		c.add(DefectInvalidValue, id, "command and script both set")
	case !hasCmd && !hasScript:
		c.add(DefectMissingCommand, id, "missing command")
	}
	c.checkEnv(id, t.spec.Env)
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
		c.checkGroup(unit, b.Spec, b.Group)
	}
	for _, b := range t.spec.Inputs {
		checkPort(b)
		if b.From.IsZero() || !c.fromInPipeline(t, b) {
			c.add(DefectMissingInput, bindUnit(id, b.Name), "missing input")
			continue
		}
		if !c.groupFromOK(t, b) {
			c.add(DefectMissingInput, bindUnit(id, b.Name), "missing input")
		}
	}
	for _, b := range t.spec.Outputs {
		checkPort(b)
		if !b.From.IsZero() && !c.fromInPipeline(t, b) {
			c.add(DefectMissingInput, bindUnit(id, b.Name), "missing input")
			continue
		}
		if !b.From.IsZero() && c.fromInPipeline(t, b) && !c.groupFromOK(t, b) {
			c.add(DefectMissingInput, bindUnit(id, b.Name), "missing input")
		}
	}
	seenOut := make(map[string]bool)
	for _, name := range t.outCalls {
		if seenOut[name] {
			continue
		}
		seenOut[name] = true
		if !hasBind(t.spec.Outputs, name) {
			c.add(DefectMissingOutput, bindUnit(id, name), "missing output")
		}
	}
	seenIn := make(map[string]bool)
	for _, name := range t.inCalls {
		if seenIn[name] {
			continue
		}
		seenIn[name] = true
		if !hasBind(t.spec.Inputs, name) {
			c.add(DefectMissingInput, bindUnit(id, name), "missing input")
		}
	}
}

func (c *graphChecker) checkTask(t *graphTask) {
	id := t.id
	if id == "" {
		id = t.name
	}
	hasCmd := len(t.command) > 0
	hasScript := t.script != ""
	switch {
	case hasCmd && hasScript:
		c.add(DefectInvalidValue, id, "command and script both set")
	case !hasCmd && !hasScript:
		c.add(DefectMissingCommand, id, "missing command")
	}
	c.checkEnv(id, t.env)
	if len(t.outputs) == 0 {
		c.add(DefectMissingOutput, id, "missing output")
	}
	seen := make(map[string]bool)
	checkPort := func(b graphBind) {
		unit := bindUnit(id, b.name)
		c.checkName(b.name, unit, "bind")
		if b.name != "" && seen[b.name] {
			c.add(DefectInvalidName, unit, "duplicate name")
			return
		}
		if b.name != "" && namePat.MatchString(b.name) {
			seen[b.name] = true
		}
		c.checkGraphGroup(unit, b.spec, b.members)
	}
	for _, b := range t.inputs {
		checkPort(b)
		if b.fromKind == handleZero || !c.fromInGraph(b) {
			c.add(DefectMissingInput, bindUnit(id, b.name), "missing input")
			continue
		}
		if !c.groupFromOK(b) {
			c.add(DefectMissingInput, bindUnit(id, b.name), "missing input")
		}
	}
	for _, b := range t.outputs {
		checkPort(b)
		if b.fromKind != handleZero && !c.fromInGraph(b) {
			c.add(DefectMissingInput, bindUnit(id, b.name), "missing input")
			continue
		}
		if b.fromKind != handleZero && c.fromInGraph(b) && !c.groupFromOK(b) {
			c.add(DefectMissingInput, bindUnit(id, b.name), "missing input")
		}
	}
}

func (c *composeChecker) checkEnv(id string, env map[string]string) {
	for k, v := range env {
		if k == "" {
			c.add(DefectInvalidValue, id, "empty env key")
		} else if strings.Contains(k, "=") {
			c.add(DefectInvalidValue, id, "env key contains =")
		}
		if v == "" {
			c.add(DefectInvalidValue, id, "empty env value")
		}
	}
}

func (c *graphChecker) checkEnv(id string, env map[string]string) {
	for k, v := range env {
		if k == "" {
			c.add(DefectInvalidValue, id, "empty env key")
		} else if strings.Contains(k, "=") {
			c.add(DefectInvalidValue, id, "env key contains =")
		}
		if v == "" {
			c.add(DefectInvalidValue, id, "empty env value")
		}
	}
}

func (c *composeChecker) checkGroup(unit string, spec PathSpec, members Group) {
	if members == nil {
		return
	}
	if !isZeroSpec(spec) {
		c.add(DefectInvalidValue, unit, "group and spec both set")
	}
	if len(members) == 0 {
		c.add(DefectInvalidValue, unit, "empty group")
		return
	}
	seen := make(map[string]bool)
	for _, m := range members {
		munit := bindUnit(unit, m.Name)
		if m.Name == "" {
			munit = unit
		}
		c.checkName(m.Name, munit, "member")
		if m.Name != "" && seen[m.Name] {
			c.add(DefectInvalidName, munit, "duplicate name")
			continue
		}
		if m.Name != "" && namePat.MatchString(m.Name) {
			seen[m.Name] = true
		}
	}
}

func (c *graphChecker) checkGraphGroup(unit string, spec PathSpec, members []graphMember) {
	if members == nil {
		return
	}
	if !isZeroSpec(spec) {
		c.add(DefectInvalidValue, unit, "group and spec both set")
	}
	if len(members) == 0 {
		c.add(DefectInvalidValue, unit, "empty group")
		return
	}
	seen := make(map[string]bool)
	for _, m := range members {
		munit := bindUnit(unit, m.name)
		if m.name == "" {
			munit = unit
		}
		c.checkName(m.name, munit, "member")
		if m.name != "" && seen[m.name] {
			c.add(DefectInvalidName, munit, "duplicate name")
			continue
		}
		if m.name != "" && namePat.MatchString(m.name) {
			seen[m.name] = true
		}
	}
}

func (c *composeChecker) fromInPipeline(t *Task, b Bind) bool {
	if foreignFrom(b.From, t.pipe) {
		return false
	}
	switch b.From.kind {
	case handleInput:
		_, ok := c.inputs[b.From.name]
		return ok
	case handleOut, handleIn:
		return b.From.task != nil && c.tasks[b.From.task.id()] != nil
	default:
		return false
	}
}

func (c *graphChecker) fromInGraph(b graphBind) bool {
	switch b.fromKind {
	case handleInput:
		_, ok := c.inputs[b.fromName]
		return ok
	case handleOut, handleIn:
		_, ok := c.tasks[b.fromTask]
		return ok
	default:
		return false
	}
}

func (c *composeChecker) groupFromOK(t *Task, b Bind) bool {
	srcGroup, srcMembers, ok := c.sourceMembers(t, b)
	if b.Group != nil {
		return ok && srcGroup && sameMemberNames(b.Group, srcMembers)
	}
	return !srcGroup
}

func (c *graphChecker) groupFromOK(b graphBind) bool {
	srcGroup, srcMembers, ok := c.sourceMembers(b)
	if b.members != nil {
		return ok && srcGroup && sameGraphMemberNames(b.members, srcMembers)
	}
	return !srcGroup
}

func (c *composeChecker) sourceMembers(t *Task, b Bind) (bool, Group, bool) {
	if foreignFrom(b.From, t.pipe) {
		return false, nil, false
	}
	switch b.From.kind {
	case handleInput:
		in, ok := c.inputs[b.From.name]
		if !ok {
			return false, nil, false
		}
		return in.members != nil, in.members, true
	case handleOut:
		src := b.From.task
		if src == nil {
			return false, nil, false
		}
		ob, ok := findBind(src.spec.Outputs, b.From.name)
		if !ok {
			return false, nil, false
		}
		return ob.Group != nil, ob.Group, true
	case handleIn:
		src := b.From.task
		if src == nil {
			return false, nil, false
		}
		ib, ok := findBind(src.spec.Inputs, b.From.name)
		if !ok {
			return false, nil, false
		}
		return ib.Group != nil, ib.Group, true
	default:
		return false, nil, false
	}
}

func (c *graphChecker) sourceMembers(b graphBind) (bool, []graphMember, bool) {
	switch b.fromKind {
	case handleInput:
		in, ok := c.inputs[b.fromName]
		if !ok {
			return false, nil, false
		}
		return in.members != nil, in.members, true
	case handleOut:
		src, ok := c.tasks[b.fromTask]
		if !ok {
			return false, nil, false
		}
		for _, ob := range src.outputs {
			if ob.name == b.fromName {
				return ob.members != nil, ob.members, true
			}
		}
		return false, nil, false
	case handleIn:
		src, ok := c.tasks[b.fromTask]
		if !ok {
			return false, nil, false
		}
		for _, ib := range src.inputs {
			if ib.name == b.fromName {
				return ib.members != nil, ib.members, true
			}
		}
		return false, nil, false
	default:
		return false, nil, false
	}
}

func sameMemberNames(a Group, b Group) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[string]bool, len(a))
	for _, m := range a {
		set[m.Name] = true
	}
	for _, m := range b {
		if !set[m.Name] {
			return false
		}
	}
	return true
}

func sameGraphMemberNames(a, b []graphMember) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[string]bool, len(a))
	for _, m := range a {
		set[m.name] = true
	}
	for _, m := range b {
		if !set[m.name] {
			return false
		}
	}
	return true
}

func (c *composeChecker) checkPipeCycles(p *Pipeline) {
	adj := make(map[string][]string)
	for _, t := range p.tasks {
		id := t.id()
		add := func(b Bind) {
			if b.From.IsZero() || foreignFrom(b.From, t.pipe) || b.From.task == nil {
				return
			}
			fromID := b.From.task.id()
			if c.tasks[fromID] == nil {
				return
			}
			adj[fromID] = append(adj[fromID], id)
		}
		for _, b := range t.spec.Inputs {
			add(b)
		}
		for _, b := range t.spec.Outputs {
			add(b)
		}
	}
	for _, t := range p.tasks {
		if reachesSelf(t.id(), adj) {
			c.add(DefectCycle, t.id(), "cycle")
		}
	}
}

func (c *graphChecker) checkCycles() {
	adj := make(map[string][]string)
	for _, e := range c.g.edges {
		if e.fromTask == "" || c.tasks[e.fromTask] == nil {
			continue
		}
		adj[e.fromTask] = append(adj[e.fromTask], e.toTask)
	}
	for i := range c.g.tasks {
		t := &c.g.tasks[i]
		if reachesSelf(t.id, adj) {
			c.add(DefectCycle, t.id, "cycle")
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

func (c *composeChecker) checkPipePaths(p *Pipeline) {
	for _, in := range p.inputs {
		if in.members != nil {
			for _, m := range in.members {
				c.renderPath(m.Spec, bindUnit(in.name, m.Name))
			}
			continue
		}
		c.renderPath(in.spec, in.name)
	}
	for _, t := range p.tasks {
		id := t.id()
		for _, b := range t.spec.Inputs {
			c.recordBindPaths(t, b, id, false)
		}
		for _, b := range t.spec.Outputs {
			c.recordBindPaths(t, b, id, true)
		}
	}
}

func (c *graphChecker) checkPaths() {
	for _, in := range c.g.inputs {
		if in.members != nil {
			for _, m := range in.members {
				c.renderPath(m.spec, bindUnit(in.name, m.name))
			}
			continue
		}
		c.renderPath(in.spec, in.name)
	}
	for i := range c.g.tasks {
		t := &c.g.tasks[i]
		for _, b := range t.inputs {
			c.recordBindPaths(t, b, false)
		}
		for _, b := range t.outputs {
			c.recordBindPaths(t, b, true)
		}
	}
}

func (c *composeChecker) recordBindPaths(t *Task, b Bind, id string, out bool) {
	unit := bindUnit(id, b.Name)
	if b.Group != nil {
		members, ok := c.resolveMembers(t, b, out)
		if !ok {
			return
		}
		for _, m := range members {
			c.renderPath(m.spec, unit)
		}
		return
	}
	spec, ok := c.resolveBind(t, b, out)
	if !ok {
		return
	}
	c.renderPath(spec, unit)
}

func (c *graphChecker) recordBindPaths(t *graphTask, b graphBind, out bool) {
	unit := bindUnit(t.id, b.name)
	if b.members != nil {
		for _, m := range b.members {
			c.renderPath(m.spec, unit)
		}
		return
	}
	c.renderPath(b.spec, unit)
}

func (c *composeChecker) renderPath(spec PathSpec, unit string) {
	_, err := spec.Render()
	if err == nil {
		return
	}
	msg := err.Error()
	var paths []string
	var ge *Error
	if errorsAsRender(err, &ge) && len(ge.Defects) > 0 {
		msg = ge.Defects[0].Message
		paths = ge.Defects[0].Paths
	}
	c.add(DefectInvalidPath, unit, msg, paths...)
}

func (c *graphChecker) renderPath(spec PathSpec, unit string) {
	_, err := spec.Render()
	if err == nil {
		return
	}
	msg := err.Error()
	var paths []string
	var ge *Error
	if errorsAsRender(err, &ge) && len(ge.Defects) > 0 {
		msg = ge.Defects[0].Message
		paths = ge.Defects[0].Paths
	}
	c.add(DefectInvalidPath, unit, msg, paths...)
}

func errorsAsRender(err error, ge **Error) bool {
	if err == nil {
		return false
	}
	e, ok := err.(*Error)
	if !ok {
		return false
	}
	*ge = e
	return true
}

func (c *composeChecker) resolveBind(t *Task, b Bind, out bool) (PathSpec, bool) {
	key := bindKey{task: t, name: b.Name, out: out}
	if spec, ok := c.memo[key]; ok {
		return spec, true
	}
	if c.walking[key] {
		return PathSpec{}, false
	}
	if b.From.IsZero() || foreignFrom(b.From, t.pipe) {
		spec := b.Spec.clone()
		c.memo[key] = spec
		return spec, true
	}
	c.walking[key] = true
	from, ok := c.resolveFrom(t, b)
	c.walking[key] = false
	if !ok {
		return PathSpec{}, false
	}
	spec := classifySpec(b.Spec, from, b.Rule)
	c.memo[key] = spec
	return spec, true
}

func (c *composeChecker) resolveMembers(t *Task, b Bind, out bool) ([]graphMember, bool) {
	if b.Group == nil {
		return nil, true
	}
	key := bindKey{task: t, name: b.Name, out: out}
	if members, ok := c.memberMemo[key]; ok {
		return members, true
	}
	if c.walking[key] {
		return nil, false
	}
	if b.From.IsZero() || foreignFrom(b.From, t.pipe) {
		members := authoredMembers(b.Group)
		c.memberMemo[key] = members
		return members, true
	}
	c.walking[key] = true
	fromMembers, ok := c.resolveFromMembers(t, b)
	c.walking[key] = false
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
	c.memberMemo[key] = outm
	return outm, true
}

func (c *composeChecker) resolveFrom(t *Task, b Bind) (PathSpec, bool) {
	if foreignFrom(b.From, t.pipe) {
		return PathSpec{}, false
	}
	switch b.From.kind {
	case handleInput:
		in, ok := c.inputs[b.From.name]
		if !ok {
			return PathSpec{}, false
		}
		return in.spec.clone(), true
	case handleOut:
		src := b.From.task
		if src == nil {
			return PathSpec{}, false
		}
		ob, ok := findBind(src.spec.Outputs, b.From.name)
		if !ok {
			return PathSpec{}, false
		}
		return c.resolveBind(src, ob, true)
	case handleIn:
		src := b.From.task
		if src == nil {
			return PathSpec{}, false
		}
		ib, ok := findBind(src.spec.Inputs, b.From.name)
		if !ok {
			return PathSpec{}, false
		}
		return c.resolveBind(src, ib, false)
	default:
		return PathSpec{}, false
	}
}

func (c *composeChecker) resolveFromMembers(t *Task, b Bind) ([]graphMember, bool) {
	if foreignFrom(b.From, t.pipe) {
		return nil, false
	}
	switch b.From.kind {
	case handleInput:
		in, ok := c.inputs[b.From.name]
		if !ok || in.members == nil {
			return nil, false
		}
		return authoredMembers(in.members), true
	case handleOut:
		src := b.From.task
		if src == nil {
			return nil, false
		}
		ob, ok := findBind(src.spec.Outputs, b.From.name)
		if !ok || ob.Group == nil {
			return nil, false
		}
		return c.resolveMembers(src, ob, true)
	case handleIn:
		src := b.From.task
		if src == nil {
			return nil, false
		}
		ib, ok := findBind(src.spec.Inputs, b.From.name)
		if !ok || ib.Group == nil {
			return nil, false
		}
		return c.resolveMembers(src, ib, false)
	default:
		return nil, false
	}
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

func composeDefects(op string, defects []Defect) error {
	if len(defects) == 0 {
		return nil
	}
	out := make([]Defect, len(defects))
	copy(out, defects)
	return &Error{Op: op, Defects: out}
}
