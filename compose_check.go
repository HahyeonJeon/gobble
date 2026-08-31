package gobble

import (
	"path/filepath"
	"regexp"
	"strings"

	intpath "github.com/HahyeonJeon/gobble/internal/path"
)

var namePat = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

type composeChecker struct {
	defects    []Defect
	tasks      map[string]*Task
	inputs     map[string]pipeInput
	memo       map[bindKey]PathSpec
	memberMemo map[bindKey][]graphMember
	treeMemo   map[bindKey]Directory
	walking    map[bindKey]bool
}

func composeCheckPipeline(p *Pipeline) []Defect {
	c := &composeChecker{
		tasks:      make(map[string]*Task, len(p.tasks)),
		inputs:     make(map[string]pipeInput, len(p.inputs)),
		memo:       make(map[bindKey]PathSpec),
		memberMemo: make(map[bindKey][]graphMember),
		treeMemo:   make(map[bindKey]Directory),
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
	c.checkGatherScatters(p)
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
	c.checkGatherScatters()
	c.checkScatterFrom()
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
		c.checkArtifactXOR(in.name, in.spec, in.members, in.tree)
		if !in.tree.IsZero() && in.tree.Dir.IsZero() {
			c.add(DefectInvalidPath, in.name, "workspace-root tree")
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
		c.checkArtifactXOR(in.name, in.spec, graphMembersAsGroup(in.members), in.tree)
		if !in.tree.IsZero() && in.tree.Dir.IsZero() {
			c.add(DefectInvalidPath, in.name, "workspace-root tree")
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
		case nodeScatter:
			if n.scatter != nil {
				c.checkScatterFrom(id, n.scatter)
				c.walkNodes(id, n.scatter.children)
			}
		case nodeGather:
			if n.gather != nil {
				c.walkNodes(id, n.gather.children)
			}
		case nodeWhen:
			if n.when != nil {
				c.checkWhenPreds(id, n.when)
				c.walkNodes(id, n.when.children)
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
		c.checkArtifactXOR(unit, b.Spec, b.Group, b.Tree)
	}
	for _, b := range t.spec.Inputs {
		checkPort(b)
		if b.From.IsZero() || !c.fromInPipeline(t, b) {
			c.add(DefectMissingInput, bindUnit(id, b.Name), "missing input")
			continue
		}
		if !c.groupFromOK(t, b, false) {
			c.add(DefectMissingInput, bindUnit(id, b.Name), "missing input")
		}
	}
	for _, b := range t.spec.Outputs {
		checkPort(b)
		if !b.From.IsZero() && !c.fromInPipeline(t, b) {
			c.add(DefectMissingInput, bindUnit(id, b.Name), "missing input")
			continue
		}
		if !b.From.IsZero() && c.fromInPipeline(t, b) && !c.groupFromOK(t, b, true) {
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
		c.checkArtifactXOR(unit, b.spec, graphMembersAsGroup(b.members), b.tree)
	}
	for _, b := range t.inputs {
		checkPort(b)
		if b.fromKind == handleZero || !c.fromInGraph(b) {
			c.add(DefectMissingInput, bindUnit(id, b.name), "missing input")
			continue
		}
		if !c.groupFromOK(t, b, false) {
			c.add(DefectMissingInput, bindUnit(id, b.name), "missing input")
		}
	}
	for _, b := range t.outputs {
		checkPort(b)
		if b.fromKind != handleZero && !c.fromInGraph(b) {
			c.add(DefectMissingInput, bindUnit(id, b.name), "missing input")
			continue
		}
		if b.fromKind != handleZero && c.fromInGraph(b) && !c.groupFromOK(t, b, true) {
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

func (c *composeChecker) checkArtifactXOR(unit string, spec PathSpec, members Group, tree Tree) {
	checkArtifactXORInto(c.add, unit, spec, members, tree)
	if members != nil {
		c.checkGroupMembers(unit, members)
	}
}

func (c *graphChecker) checkArtifactXOR(unit string, spec PathSpec, members Group, tree Tree) {
	checkArtifactXORInto(c.add, unit, spec, members, tree)
	if members != nil {
		c.checkGroupMembers(unit, members)
	}
}

func checkArtifactXORInto(add func(DefectCode, string, string, ...string), unit string, spec PathSpec, members Group, tree Tree) {
	hasGroup := members != nil
	hasTree := !tree.IsZero()
	hasSpec := !isZeroSpec(spec)
	if hasGroup && hasSpec {
		add(DefectInvalidValue, unit, "group and spec both set")
	}
	if hasGroup && hasTree {
		add(DefectInvalidValue, unit, "group and tree both set")
	}
	if hasSpec && hasTree {
		add(DefectInvalidValue, unit, "spec and tree both set")
	}
}

func (c *composeChecker) checkGroupMembers(unit string, members Group) {
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

func (c *graphChecker) checkGroupMembers(unit string, members Group) {
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

func graphMembersAsGroup(members []graphMember) Group {
	if members == nil {
		return nil
	}
	out := make(Group, len(members))
	for i, m := range members {
		out[i] = Member{Name: m.name, Spec: m.spec}
	}
	return out
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

func (c *composeChecker) groupFromOK(t *Task, b Bind, output bool) bool {
	srcGroup, srcMembers, srcTree, ok := c.sourceKind(t, b)
	if !ok {
		return false
	}
	if !b.Tree.IsZero() {
		if output && scatterMemberTreeFromProducer(t, b) {
			return !b.Tree.Dir.IsZero()
		}
		return srcTree && !srcGroup
	}
	if b.Group != nil {
		return srcGroup && !srcTree && sameMemberNames(b.Group, srcMembers)
	}
	if scatterFileFromProducer(t, b) {
		return true
	}
	return !srcGroup && !srcTree
}

func (c *graphChecker) groupFromOK(t *graphTask, b graphBind, output bool) bool {
	srcGroup, srcMembers, srcTree, ok := c.sourceKind(b)
	if !ok {
		return false
	}
	if !b.tree.IsZero() {
		if output && graphScatterMemberTreeFromProducer(t, b) {
			return !b.tree.Dir.IsZero()
		}
		return srcTree && !srcGroup
	}
	if b.members != nil {
		return srcGroup && !srcTree && sameGraphMemberNames(b.members, srcMembers)
	}
	if graphScatterFileFromProducer(t, b) {
		return true
	}
	return !srcGroup && !srcTree
}

func graphScatterFileFromProducer(t *graphTask, b graphBind) bool {
	if t == nil || t.scatter == "" || b.members != nil || !b.tree.IsZero() {
		return false
	}
	if t.scatterFromKind == handleZero {
		return false
	}
	return b.fromKind == t.scatterFromKind && b.fromName == t.scatterFromName && b.fromTask == t.scatterFromTask
}

func graphScatterMemberTreeFromProducer(t *graphTask, b graphBind) bool {
	if t == nil || t.scatter == "" || b.tree.IsZero() {
		return false
	}
	if t.scatterFromKind == handleZero {
		return false
	}
	return b.fromKind == t.scatterFromKind && b.fromName == t.scatterFromName && b.fromTask == t.scatterFromTask
}

func (c *graphChecker) fromScatterChild(b graphBind) bool {
	if b.fromTask == "" {
		return false
	}
	src, ok := c.tasks[b.fromTask]
	return ok && src.scatter != ""
}

func (c *composeChecker) checkScatterFrom(unit string, s *Scatter) {
	if s == nil {
		return
	}
	if s.from.IsZero() || foreignFrom(s.from, s.pipe) {
		c.add(DefectMissingInput, unit, "missing input")
		return
	}
	if !c.scatterFromOK(s) {
		c.add(DefectMissingInput, unit, "missing input")
	}
}

func (c *composeChecker) scatterFromOK(s *Scatter) bool {
	switch s.from.kind {
	case handleInput:
		_, ok := c.inputs[s.from.name]
		return ok && s.from.pipe == s.pipe
	case handleOut, handleIn:
		return s.from.task != nil && c.tasks[s.from.task.id()] != nil
	default:
		return false
	}
}

func (c *graphChecker) checkScatterFrom() {
	seen := make(map[string]bool)
	for i := range c.g.tasks {
		t := &c.g.tasks[i]
		if t.scatter == "" || seen[t.scatter] {
			continue
		}
		seen[t.scatter] = true
		if t.scatterFromKind == handleZero {
			c.add(DefectMissingInput, t.scatter, "missing input")
		}
	}
}

func (c *composeChecker) checkWhenPreds(unit string, w *When) {
	if w == nil || w.skipMissing.IsZero() {
		return
	}
	if foreignFrom(w.skipMissing, w.pipe) || !c.skipMissingFileOK(w.skipMissing) {
		c.add(DefectInvalidValue, unit, "skip-if-missing is not a file")
	}
}

func (c *composeChecker) skipMissingFileOK(h Handle) bool {
	switch h.kind {
	case handleInput:
		in, ok := c.inputs[h.name]
		return ok && in.members == nil && in.tree.IsZero()
	case handleOut:
		if h.task == nil {
			return false
		}
		b, ok := findBind(h.task.spec.Outputs, h.name)
		return ok && b.Group == nil && b.Tree.IsZero()
	case handleIn:
		if h.task == nil {
			return false
		}
		b, ok := findBind(h.task.spec.Inputs, h.name)
		return ok && b.Group == nil && b.Tree.IsZero()
	default:
		return false
	}
}

func (c *composeChecker) checkGatherScatters(p *Pipeline) {
	for _, t := range p.tasks {
		if t.gatherOp() == nil {
			continue
		}
		names := make(map[string]bool)
		add := func(b Bind) {
			if b.From.IsZero() || foreignFrom(b.From, t.pipe) || b.From.task == nil {
				return
			}
			src := b.From.task
			if sc := src.scatterOp(); sc != nil {
				names[sc.name] = true
			}
		}
		for _, b := range t.spec.Inputs {
			add(b)
		}
		for _, b := range t.spec.Outputs {
			add(b)
		}
		if len(names) > 1 {
			c.add(DefectInvalidValue, t.id(), "ambiguous gather scatter")
		}
	}
}

func (c *graphChecker) checkGatherScatters() {
	byID := c.tasks
	for i := range c.g.tasks {
		t := &c.g.tasks[i]
		if t.gather == "" {
			continue
		}
		names := make(map[string]bool)
		add := func(b graphBind) {
			if b.fromTask == "" {
				return
			}
			src, ok := byID[b.fromTask]
			if !ok || src.scatter == "" {
				return
			}
			names[src.scatter] = true
		}
		for _, b := range t.inputs {
			add(b)
		}
		for _, b := range t.outputs {
			add(b)
		}
		if len(names) > 1 {
			c.add(DefectInvalidValue, t.id, "ambiguous gather scatter")
		}
	}
}

func (c *composeChecker) sourceKind(t *Task, b Bind) (bool, Group, bool, bool) {
	if foreignFrom(b.From, t.pipe) {
		return false, nil, false, false
	}
	switch b.From.kind {
	case handleInput:
		in, ok := c.inputs[b.From.name]
		if !ok {
			return false, nil, false, false
		}
		return in.members != nil, in.members, !in.tree.IsZero(), true
	case handleOut:
		src := b.From.task
		if src == nil {
			return false, nil, false, false
		}
		ob, ok := findBind(src.spec.Outputs, b.From.name)
		if !ok {
			return false, nil, false, false
		}
		return ob.Group != nil, ob.Group, !ob.Tree.IsZero(), true
	case handleIn:
		src := b.From.task
		if src == nil {
			return false, nil, false, false
		}
		ib, ok := findBind(src.spec.Inputs, b.From.name)
		if !ok {
			return false, nil, false, false
		}
		return ib.Group != nil, ib.Group, !ib.Tree.IsZero(), true
	default:
		return false, nil, false, false
	}
}

func (c *graphChecker) sourceKind(b graphBind) (bool, []graphMember, bool, bool) {
	switch b.fromKind {
	case handleInput:
		in, ok := c.inputs[b.fromName]
		if !ok {
			return false, nil, false, false
		}
		return in.members != nil, in.members, !in.tree.IsZero(), true
	case handleOut:
		src, ok := c.tasks[b.fromTask]
		if !ok {
			return false, nil, false, false
		}
		for _, ob := range src.outputs {
			if ob.name == b.fromName {
				return ob.members != nil, ob.members, !ob.tree.IsZero(), true
			}
		}
		return false, nil, false, false
	case handleIn:
		src, ok := c.tasks[b.fromTask]
		if !ok {
			return false, nil, false, false
		}
		for _, ib := range src.inputs {
			if ib.name == b.fromName {
				return ib.members != nil, ib.members, !ib.tree.IsZero(), true
			}
		}
		return false, nil, false, false
	default:
		return false, nil, false, false
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
		if !in.tree.IsZero() {
			c.renderTreeDir(in.tree.Dir, in.name)
			continue
		}
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
		if !in.tree.IsZero() {
			c.renderTreeDir(in.tree.Dir, in.name)
			continue
		}
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
	if scatterFileFromProducer(t, b) || fromScatterChild(t, b) {
		return
	}
	if !b.Tree.IsZero() {
		dir, ok := c.resolveTree(t, b, out)
		if !ok {
			return
		}
		c.renderTreeDir(dir, unit)
		return
	}
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
	if graphScatterFileFromProducer(t, b) || c.fromScatterChild(b) {
		return
	}
	if !b.tree.IsZero() {
		c.renderTreeDir(b.tree.Dir, unit)
		return
	}
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

func (c *composeChecker) renderTreeDir(dir Directory, unit string) {
	if msg, paths := treeDirDefect(dir); msg != "" {
		c.add(DefectInvalidPath, unit, msg, paths...)
	}
}

func (c *graphChecker) renderTreeDir(dir Directory, unit string) {
	if msg, paths := treeDirDefect(dir); msg != "" {
		c.add(DefectInvalidPath, unit, msg, paths...)
	}
}

func treeDirDefect(dir Directory) (string, []string) {
	raw := strings.ReplaceAll(dir.String(), `\`, "/")
	if raw == "" || raw == "." {
		paths := []string(nil)
		if raw != "" {
			paths = []string{raw}
		}
		return "workspace-root tree", paths
	}
	if filepath.IsAbs(raw) || strings.HasPrefix(raw, "/") {
		return "absolute plan path", []string{raw}
	}
	cleaned, escaped := intpath.Clean(raw)
	if escaped {
		return "path escapes directory", []string{raw}
	}
	if cleaned == "" || cleaned == "." {
		return "workspace-root tree", []string{raw}
	}
	if cleaned == ".gobble" || strings.HasPrefix(cleaned, ".gobble/") {
		return "path is under .gobble", []string{raw}
	}
	if cleaned == ".git" || strings.HasPrefix(cleaned, ".git/") {
		return "path is under .git", []string{raw}
	}
	return "", nil
}

func (c *composeChecker) resolveTree(t *Task, b Bind, out bool) (Directory, bool) {
	if b.Tree.IsZero() {
		return Directory{}, true
	}
	key := bindKey{task: t, name: b.Name, out: out}
	if dir, ok := c.treeMemo[key]; ok {
		return dir, true
	}
	if c.walking[key] {
		return Directory{}, false
	}
	if b.From.IsZero() || foreignFrom(b.From, t.pipe) {
		c.treeMemo[key] = b.Tree.Dir
		return b.Tree.Dir, true
	}
	c.walking[key] = true
	from, ok := c.resolveFromTree(t, b)
	c.walking[key] = false
	if !ok {
		return Directory{}, false
	}
	dir := b.Tree.Dir
	if dir.IsZero() {
		dir = from
	}
	c.treeMemo[key] = dir
	return dir, true
}

func (c *composeChecker) resolveFromTree(t *Task, b Bind) (Directory, bool) {
	if foreignFrom(b.From, t.pipe) {
		return Directory{}, false
	}
	switch b.From.kind {
	case handleInput:
		in, ok := c.inputs[b.From.name]
		if !ok || in.tree.IsZero() {
			return Directory{}, false
		}
		return in.tree.Dir, true
	case handleOut:
		src := b.From.task
		if src == nil {
			return Directory{}, false
		}
		ob, ok := findBind(src.spec.Outputs, b.From.name)
		if !ok || ob.Tree.IsZero() {
			return Directory{}, false
		}
		return c.resolveTree(src, ob, true)
	case handleIn:
		src := b.From.task
		if src == nil {
			return Directory{}, false
		}
		ib, ok := findBind(src.spec.Inputs, b.From.name)
		if !ok || ib.Tree.IsZero() {
			return Directory{}, false
		}
		return c.resolveTree(src, ib, false)
	default:
		return Directory{}, false
	}
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
	if b.From.IsZero() || foreignFrom(b.From, t.pipe) || scatterFileFromProducer(t, b) || fromScatterChild(t, b) {
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
