package gobble

import (
	"errors"
	"io"
	"strings"

	"github.com/HahyeonJeon/gobble/internal/engine"
)

// Plan is the inspectable dry-run document for a valid graph.
type Plan struct {
	inner *engine.Plan
	name  string
	tasks []planTaskRead
	edges []Edge
}

type planTaskRead struct {
	id      string
	inputs  []planBindRead
	outputs []planBindRead
}

type planBindRead struct {
	name    string
	kind    string
	path    string
	members []planMemberRead
}

type planMemberRead struct {
	name string
	path string
}

// PlanOption configures [BuildPlan].
type PlanOption func(*planConfig)

type planConfig struct {
	w io.Writer
}

// WriteTo returns a [PlanOption] that writes the same JSON a [*Plan]
// would write with [Plan.WriteJSON].
func WriteTo(w io.Writer) PlanOption {
	return func(c *planConfig) {
		c.w = w
	}
}

// BuildPlan validates g first, then returns an inspectable plan.
//
// On any defect it returns (nil, [*Error]) with Op "plan". After a valid
// plan is built, a [WriteTo] error still returns the [*Plan] and is the
// writer's own error, not an [*Error].
func BuildPlan(g *Graph, opts ...PlanOption) (*Plan, error) {
	var cfg planConfig
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if g == nil {
		return nil, &Error{Op: "plan", Defects: []Defect{{
			Code:    DefectInvalidRequest,
			Message: "nil graph",
		}}}
	}
	if err := composeDefects("plan", graphCheck(g)); err != nil {
		return nil, err
	}
	doc, err := planDocument(g)
	if err != nil {
		return nil, err
	}
	inner, defects := engine.BuildPlan(doc)
	if pub := publicError("plan", defects); pub != nil {
		return nil, pub
	}
	p := &Plan{inner: inner}
	p.load(doc)
	if cfg.w != nil {
		if werr := p.WriteJSON(cfg.w); werr != nil {
			return p, werr
		}
	}
	return p, nil
}

// WriteJSON writes the plan as one JSON object to w, including a trailing newline.
// A nil [*Plan] returns a non-[*Error].
func (p *Plan) WriteJSON(w io.Writer) error {
	if p == nil || p.inner == nil {
		return errors.New("nil plan")
	}
	return p.inner.WriteJSON(w)
}

// MarshalJSON returns the same plan JSON [WriteJSON] writes, without a
// trailing newline. A nil [*Plan] returns the JSON null value.
func (p *Plan) MarshalJSON() ([]byte, error) {
	if p == nil || p.inner == nil {
		return []byte("null"), nil
	}
	return p.inner.MarshalJSON()
}

// Pipeline returns the pipeline name recorded on the plan.
func (p *Plan) Pipeline() string {
	if p == nil {
		return ""
	}
	return p.name
}

// TaskIDs returns task ids in plan order.
func (p *Plan) TaskIDs() []string {
	if p == nil {
		return nil
	}
	out := make([]string, len(p.tasks))
	for i, t := range p.tasks {
		out[i] = t.id
	}
	return out
}

// Edges returns a copy of plan edges, including Wait paths.
func (p *Plan) Edges() []Edge {
	if p == nil {
		return nil
	}
	out := make([]Edge, len(p.edges))
	for i, e := range p.edges {
		out[i] = Edge{
			FromTask: e.FromTask,
			FromPort: e.FromPort,
			ToTask:   e.ToTask,
			ToPort:   e.ToPort,
			Wait:     copyStrings(e.Wait),
		}
	}
	return out
}

// TaskInputNames returns input bind names for taskID.
func (p *Plan) TaskInputNames(taskID string) []string {
	t, ok := p.lookupTask(taskID)
	if !ok {
		return nil
	}
	return planBindNames(t.inputs)
}

// TaskOutputNames returns output bind names for taskID.
func (p *Plan) TaskOutputNames(taskID string) []string {
	t, ok := p.lookupTask(taskID)
	if !ok {
		return nil
	}
	return planBindNames(t.outputs)
}

// BindKind returns ArtifactFile, ArtifactGroup, ArtifactTree, or empty
// if the bind is missing.
func (p *Plan) BindKind(taskID, name string) string {
	b, ok := p.lookupBind(taskID, name)
	if !ok {
		return ""
	}
	return b.kind
}

// BindPath returns the recorded file path for the named bind, or the
// declared directory for a Tree bind. Group binds and missing binds
// return empty.
func (p *Plan) BindPath(taskID, name string) string {
	b, ok := p.lookupBind(taskID, name)
	if !ok {
		return ""
	}
	return b.path
}

// MemberNames returns Group member names for the named bind.
func (p *Plan) MemberNames(taskID, bind string) []string {
	b, ok := p.lookupBind(taskID, bind)
	if !ok {
		return nil
	}
	out := make([]string, len(b.members))
	for i, m := range b.members {
		out[i] = m.name
	}
	return out
}

// MemberPath returns the recorded path for one Group member.
func (p *Plan) MemberPath(taskID, bind, member string) string {
	b, ok := p.lookupBind(taskID, bind)
	if !ok {
		return ""
	}
	for _, m := range b.members {
		if m.name == member {
			return m.path
		}
	}
	return ""
}

func (p *Plan) load(doc engine.Document) {
	p.name = doc.Name
	p.tasks = make([]planTaskRead, 0, len(doc.Tasks))
	for _, t := range doc.Tasks {
		p.tasks = append(p.tasks, planTaskRead{
			id:      t.ID,
			inputs:  planBindsFromIO(t.Inputs),
			outputs: planBindsFromIO(t.Outputs),
		})
	}
	p.edges = make([]Edge, 0, len(doc.Edges))
	for _, e := range doc.Edges {
		p.edges = append(p.edges, Edge{
			FromTask: e.FromTask,
			FromPort: e.FromPort,
			ToTask:   e.ToTask,
			ToPort:   e.ToPort,
			Wait:     copyStrings(e.Wait),
		})
	}
}

func planBindsFromIO(in []engine.IO) []planBindRead {
	out := make([]planBindRead, 0, len(in))
	for _, b := range in {
		kind := b.Kind
		if kind == "" {
			kind = ArtifactFile
			if b.Members != nil {
				kind = ArtifactGroup
			} else if b.Manifest != "" {
				kind = ArtifactTree
			}
		}
		rec := planBindRead{name: b.Name, path: b.Path, kind: kind}
		if kind == ArtifactGroup || b.Members != nil {
			rec.kind = ArtifactGroup
			rec.path = ""
			rec.members = make([]planMemberRead, 0, len(b.Members))
			for _, m := range b.Members {
				rec.members = append(rec.members, planMemberRead{name: m.Name, path: m.Path})
			}
		}
		out = append(out, rec)
	}
	return out
}

func (p *Plan) lookupTask(taskID string) (planTaskRead, bool) {
	if p == nil {
		return planTaskRead{}, false
	}
	for _, t := range p.tasks {
		if t.id == taskID {
			return t, true
		}
	}
	return planTaskRead{}, false
}

func (p *Plan) lookupBind(taskID, name string) (planBindRead, bool) {
	t, ok := p.lookupTask(taskID)
	if !ok {
		return planBindRead{}, false
	}
	for _, b := range t.inputs {
		if b.name == name {
			return b, true
		}
	}
	for _, b := range t.outputs {
		if b.name == name {
			return b, true
		}
	}
	return planBindRead{}, false
}

func planBindNames(binds []planBindRead) []string {
	out := make([]string, len(binds))
	for i, b := range binds {
		out[i] = b.name
	}
	return out
}

func planDocument(g *Graph) (engine.Document, error) {
	doc := engine.Document{Name: g.name}
	byID := make(map[string]*engine.TaskPlan, len(g.tasks))
	for i := range g.tasks {
		t := &g.tasks[i]
		command := copyStrings(t.command)
		if t.script != "" {
			command = scriptArgv(t.script)
		}
		pt := engine.TaskPlan{
			ID:      t.id,
			Name:    t.name,
			Module:  t.module,
			Branch:  t.branch,
			Merge:   t.merge,
			Command: command,
			Script:  t.script,
			Image:   t.image,
			Backend: t.backend,
			Resources: engine.ResourcePlan{
				CPU:    t.resources.CPU,
				Memory: t.resources.Memory,
			},
			Env: copyEnv(t.env),
		}
		for _, p := range t.params {
			pt.Params = append(pt.Params, engine.ParamPlan{Name: p.Name, Value: p.Value})
		}
		for _, b := range t.inputs {
			io, err := planIO(g, b, true)
			if err != nil {
				return engine.Document{}, planPathError(bindUnit(t.id, b.name), err)
			}
			pt.Inputs = append(pt.Inputs, io)
		}
		for _, b := range t.outputs {
			io, err := planIO(g, b, false)
			if err != nil {
				return engine.Document{}, planPathError(bindUnit(t.id, b.name), err)
			}
			pt.Outputs = append(pt.Outputs, io)
		}
		doc.Tasks = append(doc.Tasks, pt)
	}
	for i := range doc.Tasks {
		byID[doc.Tasks[i].ID] = &doc.Tasks[i]
	}
	for _, e := range g.edges {
		wait, err := planWait(byID, e)
		if err != nil {
			return engine.Document{}, err
		}
		doc.Edges = append(doc.Edges, engine.Edge{
			FromTask: e.fromTask,
			FromPort: e.fromPort,
			ToTask:   e.toTask,
			ToPort:   e.toPort,
			Wait:     wait,
		})
	}
	return doc, nil
}

func scriptArgv(script string) []string {
	return []string{"sh", "-c", "set -eu\n" + script}
}

func planIO(g *Graph, b graphBind, asInput bool) (engine.IO, error) {
	if !b.tree.IsZero() {
		dir := b.tree.Dir.String()
		io := engine.IO{
			Name:     b.name,
			Kind:     engine.ArtifactTree,
			Path:     dir,
			Manifest: treeManifestRel(dir),
		}
		if asInput {
			if from, ok := fromResolvedTree(g, b); ok {
				src := from.Dir.String()
				if src != "" && src != dir {
					io.Source = src
				}
			}
		}
		return io, nil
	}
	if b.members != nil {
		io := engine.IO{
			Name:    b.name,
			Kind:    engine.ArtifactGroup,
			Spec:    snapshotPath(b.spec),
			Members: make([]engine.IOMember, 0, len(b.members)),
		}
		fromMembers := fromResolvedMembers(g, b)
		for _, m := range b.members {
			path, err := m.spec.Render()
			if err != nil {
				return engine.IO{}, err
			}
			member := engine.IOMember{
				Name: m.name,
				Path: path,
				Spec: snapshotPath(m.spec),
			}
			if asInput {
				if from, ok := findGraphMember(fromMembers, m.name); ok {
					if src, err := from.spec.Render(); err == nil && src != path {
						member.Source = src
					}
				}
			}
			io.Members = append(io.Members, member)
		}
		return io, nil
	}
	path, err := b.spec.Render()
	if err != nil {
		return engine.IO{}, err
	}
	io := engine.IO{
		Name: b.name,
		Kind: engine.ArtifactFile,
		Path: path,
		Spec: snapshotPath(b.spec),
	}
	if asInput {
		if from, ok := fromResolvedSpec(g, b); ok {
			if src, err := from.Render(); err == nil && src != path {
				io.Source = src
			}
		}
	}
	return io, nil
}

func treeManifestRel(dir string) string {
	name := ".gobble-tree.json"
	if dir == "" {
		return name
	}
	cleaned := strings.ReplaceAll(dir, `\`, "/")
	return strings.TrimSuffix(cleaned, "/") + "/" + name
}

func fromResolvedTree(g *Graph, b graphBind) (Tree, bool) {
	switch b.fromKind {
	case handleInput:
		for _, in := range g.inputs {
			if in.name == b.fromName {
				if in.tree.IsZero() {
					return Tree{}, false
				}
				return in.tree, true
			}
		}
	case handleOut:
		for i := range g.tasks {
			t := &g.tasks[i]
			if t.id != b.fromTask {
				continue
			}
			for _, ob := range t.outputs {
				if ob.name == b.fromName {
					if ob.tree.IsZero() {
						return Tree{}, false
					}
					return ob.tree, true
				}
			}
		}
	case handleIn:
		for i := range g.tasks {
			t := &g.tasks[i]
			if t.id != b.fromTask {
				continue
			}
			for _, ib := range t.inputs {
				if ib.name == b.fromName {
					if ib.tree.IsZero() {
						return Tree{}, false
					}
					return ib.tree, true
				}
			}
		}
	}
	return Tree{}, false
}

func fromResolvedSpec(g *Graph, b graphBind) (PathSpec, bool) {
	switch b.fromKind {
	case handleInput:
		for _, in := range g.inputs {
			if in.name != b.fromName {
				continue
			}
			if in.members != nil || !in.tree.IsZero() {
				return PathSpec{}, false
			}
			return in.spec, true
		}
	case handleOut:
		for i := range g.tasks {
			t := &g.tasks[i]
			if t.id != b.fromTask {
				continue
			}
			for _, ob := range t.outputs {
				if ob.name != b.fromName {
					continue
				}
				if ob.members != nil || !ob.tree.IsZero() {
					return PathSpec{}, false
				}
				return ob.spec, true
			}
		}
	case handleIn:
		for i := range g.tasks {
			t := &g.tasks[i]
			if t.id != b.fromTask {
				continue
			}
			for _, ib := range t.inputs {
				if ib.name != b.fromName {
					continue
				}
				if ib.members != nil || !ib.tree.IsZero() {
					return PathSpec{}, false
				}
				return ib.spec, true
			}
		}
	}
	return PathSpec{}, false
}

func fromResolvedMembers(g *Graph, b graphBind) []graphMember {
	switch b.fromKind {
	case handleInput:
		for _, in := range g.inputs {
			if in.name == b.fromName {
				return in.members
			}
		}
	case handleOut:
		for i := range g.tasks {
			t := &g.tasks[i]
			if t.id != b.fromTask {
				continue
			}
			for _, ob := range t.outputs {
				if ob.name == b.fromName {
					return ob.members
				}
			}
		}
	case handleIn:
		for i := range g.tasks {
			t := &g.tasks[i]
			if t.id != b.fromTask {
				continue
			}
			for _, ib := range t.inputs {
				if ib.name == b.fromName {
					return ib.members
				}
			}
		}
	}
	return nil
}

func planWait(byID map[string]*engine.TaskPlan, e graphEdge) ([]string, error) {
	to, ok := byID[e.toTask]
	if !ok {
		return nil, neverReadyError(bindUnit(e.toTask, e.toPort))
	}
	toIO, toIsInput, ok := findTaskIO(to, e.toPort)
	if !ok {
		return nil, neverReadyError(bindUnit(e.toTask, e.toPort))
	}
	if e.fromTask == "" || toIsInput {
		return waitPaths(toIO, bindUnit(e.toTask, e.toPort))
	}
	from, ok := byID[e.fromTask]
	if !ok {
		return nil, neverReadyError(bindUnit(e.toTask, e.toPort))
	}
	fromIO, _, ok := findPublishedIO(from, e.fromPort)
	if !ok {
		return nil, neverReadyError(bindUnit(e.toTask, e.toPort))
	}
	return waitPaths(fromIO, bindUnit(e.toTask, e.toPort))
}

func findTaskIO(t *engine.TaskPlan, port string) (engine.IO, bool, bool) {
	for _, in := range t.Inputs {
		if in.Name == port {
			return in, true, true
		}
	}
	for _, out := range t.Outputs {
		if out.Name == port {
			return out, false, true
		}
	}
	return engine.IO{}, false, false
}

func findPublishedIO(t *engine.TaskPlan, port string) (engine.IO, bool, bool) {
	for _, out := range t.Outputs {
		if out.Name == port {
			return out, false, true
		}
	}
	for _, in := range t.Inputs {
		if in.Name == port {
			return in, true, true
		}
	}
	return engine.IO{}, false, false
}

func waitPaths(io engine.IO, unit string) ([]string, error) {
	if io.Members != nil {
		waits := make([]string, 0, len(io.Members))
		for _, m := range io.Members {
			p := m.Path
			if m.Source != "" {
				p = m.Source
			}
			if p == "" {
				return nil, neverReadyError(unit)
			}
			waits = append(waits, p)
		}
		if len(waits) == 0 {
			return nil, neverReadyError(unit)
		}
		return waits, nil
	}
	p := io.Path
	if io.Source != "" {
		p = io.Source
	}
	if p == "" {
		return nil, neverReadyError(unit)
	}
	return []string{p}, nil
}

func neverReadyError(unit string) error {
	return &Error{Op: "plan", Defects: []Defect{{
		Code:    DefectNeverReady,
		Unit:    unit,
		Message: "never-ready",
	}}}
}

func planPathError(unit string, err error) error {
	msg := "invalid path"
	var ge *Error
	if errors.As(err, &ge) {
		if len(ge.Defects) > 0 && ge.Defects[0].Message != "" {
			msg = ge.Defects[0].Message
		}
	} else if err != nil {
		msg = err.Error()
	}
	return &Error{Op: "plan", Defects: []Defect{{
		Code:    DefectInvalidPath,
		Unit:    unit,
		Message: msg,
	}}}
}
