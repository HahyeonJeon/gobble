package gobble

// Graph is the immutable result of Compose. It is not a plan.
type Graph struct {
	name   string
	inputs []graphInput
	tasks  []graphTask
	edges  []graphEdge
}

type graphInput struct {
	name string
	spec PathSpec
}

type graphTask struct {
	id        string
	name      string
	module    string
	branch    string
	merge     string
	command   []string
	script    string
	image     string
	backend   string
	resources Resources
	params    []Param
	env       map[string]string
	inputs    []graphBind
	outputs   []graphBind
}

type graphBind struct {
	name     string
	spec     PathSpec
	fromKind handleKind
	fromName string
	fromTask string
	members  []graphMember
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
