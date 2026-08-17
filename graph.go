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
	image     string
	backend   string
	resources Resources
	params    []Param
	inputs    []graphBind
	outputs   []graphBind
}

type graphBind struct {
	name     string
	spec     PathSpec
	fromKind handleKind
	fromName string
	fromTask string
}

type graphEdge struct {
	fromTask  string
	fromPort  string
	fromInput string
	toTask    string
	toPort    string
}
