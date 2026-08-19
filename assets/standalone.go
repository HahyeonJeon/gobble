package assets

import "github.com/HahyeonJeon/gobble"

// Input is one required standalone pipeline input.
type Input struct {
	Name string
	Spec gobble.PathSpec
}

// Standalone returns a pipeline named name. It records each input with
// AddInput, then calls build with the pipeline as Parent and the input
// Handles in input order. build must not call AddInput. Command stays
// whatever the builder assembled.
func Standalone(name string, inputs []Input, build func(Parent, []gobble.Handle)) *gobble.Pipeline {
	p := gobble.NewPipeline(name)
	handles := make([]gobble.Handle, len(inputs))
	for i, in := range inputs {
		handles[i] = p.AddInput(in.Name, in.Spec)
	}
	build(p, handles)
	return p
}
