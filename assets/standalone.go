package assets

import "github.com/HahyeonJeon/gobble"

// Input is one required standalone pipeline input.
// A non-nil Group records AddInputGroup. A present Tree records
// AddInputTree. Otherwise AddInput uses Spec.
type Input struct {
	Name  string
	Spec  gobble.PathSpec
	Group gobble.Group
	Tree  gobble.Tree
}

// Standalone returns a pipeline named name. It records each input with
// AddInput, AddInputGroup, or AddInputTree, then calls build with the
// pipeline as Parent and the input Handles in input order. build must
// not call AddInput. Command stays whatever the builder assembled.
func Standalone(name string, inputs []Input, build func(Parent, []gobble.Handle)) *gobble.Pipeline {
	p := gobble.NewPipeline(name)
	handles := make([]gobble.Handle, len(inputs))
	for i, in := range inputs {
		switch {
		case in.Group != nil:
			handles[i] = p.AddInputGroup(in.Name, in.Group)
		case !in.Tree.IsZero():
			handles[i] = p.AddInputTree(in.Name, in.Tree)
		default:
			handles[i] = p.AddInput(in.Name, in.Spec)
		}
	}
	build(p, handles)
	return p
}
