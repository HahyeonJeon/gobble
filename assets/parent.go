package assets

import "github.com/HahyeonJeon/gobble"

// Parent can record a task. Pipeline, Module, Branch, Merge, Scatter,
// Gather, and When implement it.
type Parent interface {
	AddTask(spec gobble.TaskSpec) *gobble.Task
}

// ModuleParent can record a child module. Pipeline, Module, and Branch
// implement it. Merge does not.
type ModuleParent interface {
	Parent
	AddModule(name string) *gobble.Module
}

var (
	_ Parent       = (*gobble.Pipeline)(nil)
	_ Parent       = (*gobble.Module)(nil)
	_ Parent       = (*gobble.Branch)(nil)
	_ Parent       = (*gobble.Merge)(nil)
	_ Parent       = (*gobble.Scatter)(nil)
	_ Parent       = (*gobble.Gather)(nil)
	_ Parent       = (*gobble.When)(nil)
	_ ModuleParent = (*gobble.Pipeline)(nil)
	_ ModuleParent = (*gobble.Module)(nil)
	_ ModuleParent = (*gobble.Branch)(nil)
)

// AddTask records spec on parent. First-party asset builders must call this
// helper rather than parent.AddTask. The helper does not call AddInput.
func AddTask(parent Parent, spec gobble.TaskSpec) *gobble.Task {
	return parent.AddTask(spec)
}

// AddModule records a child module named name. Only a multi-task asset may
// call AddModule.
func AddModule(parent ModuleParent, name string) *gobble.Module {
	return parent.AddModule(name)
}

// CommandPath renders spec as one Command token. Assets must build argv from
// the same PathSpec the binds declare. Tests call this helper; builders call
// mustCommandPath so a render error panics at author time.
func CommandPath(spec gobble.PathSpec) (string, error) {
	return spec.Render()
}

func mustCommandPath(spec gobble.PathSpec) string {
	path, err := CommandPath(spec)
	if err != nil {
		panic(err)
	}
	return path
}
