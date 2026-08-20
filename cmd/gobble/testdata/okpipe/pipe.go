package okpipe

import "github.com/HahyeonJeon/gobble"

func Pipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("copy")
	p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"true"},
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Base: "out", Ext: ".txt"},
		}},
	})
	return p
}
