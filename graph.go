package gobble

// ArtifactFile, ArtifactGroup, and ArtifactTree are bind kinds returned
// by Graph and Plan readers.
const (
	ArtifactFile  = "file"
	ArtifactGroup = "group"
	ArtifactTree  = "tree"
)

// Edge is a read-only directed bind edge. An empty FromTask is a pipeline
// input. Wait is set on Plan edges and empty on Graph edges.
type Edge struct {
	FromTask string
	FromPort string
	ToTask   string
	ToPort   string
	Wait     []string
}

// Graph is the immutable result of Compose. It is not a plan.
type Graph struct {
	name   string
	inputs []graphInput
	tasks  []graphTask
	edges  []graphEdge
}

type graphInput struct {
	name    string
	spec    PathSpec
	members []graphMember
	tree    Tree
}

type graphTask struct {
	id                 string
	name               string
	module             string
	branch             string
	merge              string
	scatter            string
	gather             string
	when               string
	scatterFromKind    handleKind
	scatterFromName    string
	scatterFromTask    string
	scatterFromPath    string
	scatterMembers     []string
	scatterMemberPaths []string
	skipMissingKind    handleKind
	skipMissingName    string
	skipMissingTask    string
	skipMissingPath    string
	skipFalse          string
	command            []string
	script             string
	image              string
	backend            string
	resources          Resources
	params             []Param
	env                map[string]string
	inputs             []graphBind
	outputs            []graphBind
}

type graphBind struct {
	name     string
	spec     PathSpec
	fromKind handleKind
	fromName string
	fromTask string
	members  []graphMember
	tree     Tree
}

type graphMember struct {
	name string
	spec PathSpec
}

type graphEdge struct {
	fromTask string
	fromPort string
	toTask   string
	toPort   string
}

// Name returns the pipeline name recorded on g.
func (g *Graph) Name() string {
	if g == nil {
		return ""
	}
	return g.name
}

// TaskIDs returns task ids in compose order.
func (g *Graph) TaskIDs() []string {
	if g == nil {
		return nil
	}
	out := make([]string, len(g.tasks))
	for i, t := range g.tasks {
		out[i] = t.id
	}
	return out
}

// InputNames returns pipeline input names in author order.
func (g *Graph) InputNames() []string {
	if g == nil {
		return nil
	}
	out := make([]string, len(g.inputs))
	for i, in := range g.inputs {
		out[i] = in.name
	}
	return out
}

// Edges returns a copy of the graph's bind edges. Wait is empty.
func (g *Graph) Edges() []Edge {
	if g == nil {
		return nil
	}
	out := make([]Edge, len(g.edges))
	for i, e := range g.edges {
		out[i] = Edge{
			FromTask: e.fromTask,
			FromPort: e.fromPort,
			ToTask:   e.toTask,
			ToPort:   e.toPort,
		}
	}
	return out
}

// TaskInputNames returns input bind names for taskID.
func (g *Graph) TaskInputNames(taskID string) []string {
	t, ok := g.lookupTask(taskID)
	if !ok {
		return nil
	}
	return bindNames(t.inputs)
}

// TaskOutputNames returns output bind names for taskID.
func (g *Graph) TaskOutputNames(taskID string) []string {
	t, ok := g.lookupTask(taskID)
	if !ok {
		return nil
	}
	return bindNames(t.outputs)
}

// BindKind returns ArtifactFile, ArtifactGroup, ArtifactTree, or empty
// if the bind is missing.
func (g *Graph) BindKind(taskID, name string) string {
	b, ok := g.lookupBind(taskID, name)
	if !ok {
		return ""
	}
	if !b.tree.IsZero() {
		return ArtifactTree
	}
	if b.members != nil {
		return ArtifactGroup
	}
	return ArtifactFile
}

// BindPath returns the rendered file path for the named bind, or the
// declared directory for a Tree bind. Group binds and missing binds
// return empty.
func (g *Graph) BindPath(taskID, name string) string {
	b, ok := g.lookupBind(taskID, name)
	if !ok || b.members != nil {
		return ""
	}
	if !b.tree.IsZero() {
		return b.tree.Dir.String()
	}
	s, err := b.spec.Render()
	if err != nil {
		return ""
	}
	return s
}

// MemberNames returns Group member names for the named bind.
func (g *Graph) MemberNames(taskID, bind string) []string {
	b, ok := g.lookupBind(taskID, bind)
	if !ok || b.members == nil {
		return nil
	}
	out := make([]string, len(b.members))
	for i, m := range b.members {
		out[i] = m.name
	}
	return out
}

// MemberPath returns the rendered path for one Group member.
func (g *Graph) MemberPath(taskID, bind, member string) string {
	b, ok := g.lookupBind(taskID, bind)
	if !ok {
		return ""
	}
	for _, m := range b.members {
		if m.name != member {
			continue
		}
		s, err := m.spec.Render()
		if err != nil {
			return ""
		}
		return s
	}
	return ""
}

func (g *Graph) lookupTask(taskID string) (graphTask, bool) {
	if g == nil {
		return graphTask{}, false
	}
	for i := range g.tasks {
		if g.tasks[i].id == taskID {
			return g.tasks[i], true
		}
	}
	return graphTask{}, false
}

func (g *Graph) lookupBind(taskID, name string) (graphBind, bool) {
	t, ok := g.lookupTask(taskID)
	if !ok {
		return graphBind{}, false
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
	return graphBind{}, false
}

func bindNames(binds []graphBind) []string {
	out := make([]string, len(binds))
	for i, b := range binds {
		out[i] = b.name
	}
	return out
}
