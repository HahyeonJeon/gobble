package assets

import "github.com/HahyeonJeon/gobble"

// Input is one required standalone pipeline input.
// A non-nil Group records AddInputGroup; otherwise AddInput uses Spec.
type Input struct {
	Name  string
	Spec  gobble.PathSpec
	Group gobble.Group
}

// Standalone returns a pipeline named name. It records each input with
// AddInput or AddInputGroup, then calls build with the pipeline as
// Parent and the input Handles in input order. build must not call
// AddInput. Command stays whatever the builder assembled.
func Standalone(name string, inputs []Input, build func(Parent, []gobble.Handle)) *gobble.Pipeline {
	p := gobble.NewPipeline(name)
	handles := make([]gobble.Handle, len(inputs))
	for i, in := range inputs {
		if in.Group != nil {
			handles[i] = p.AddInputGroup(in.Name, in.Group)
		} else {
			handles[i] = p.AddInput(in.Name, in.Spec)
		}
	}
	build(p, handles)
	return p
}
