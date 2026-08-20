package hostpipe

import "github.com/HahyeonJeon/gobble"

func Pipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("host")
	p.AddTask(gobble.TaskSpec{
		Name:    "touch",
		Command: []string{"sh", "-c", ": > out.txt"},
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Base: "out", Ext: ".txt"},
		}},
	})
	return p
}
