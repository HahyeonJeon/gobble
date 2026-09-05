package modules

import "github.com/HahyeonJeon/gobble"

// WithDisplay decorates a task parent without changing commands or binds.
func WithDisplay(parent Parent, display gobble.TaskDisplay) Parent {
	display.Samples = append([]string(nil), display.Samples...)
	return displayParent{parent: parent, display: display}
}

type displayParent struct {
	parent  Parent
	display gobble.TaskDisplay
}

func (p displayParent) AddTask(spec gobble.TaskSpec) *gobble.Task {
	spec.Display = p.display
	return p.parent.AddTask(spec)
}
