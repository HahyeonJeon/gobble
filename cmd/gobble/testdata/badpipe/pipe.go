package badpipe

import "github.com/HahyeonJeon/gobble"

func Pipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("bad")
	p.AddTask(gobble.TaskSpec{Name: "copy"})
	return p
}
