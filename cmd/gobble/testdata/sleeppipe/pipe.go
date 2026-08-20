package sleeppipe

import "github.com/HahyeonJeon/gobble"

func Pipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("sleep")
	p.AddTask(gobble.TaskSpec{
		Name:    "sleep",
		Command: []string{"sleep", "30"},
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Base: "out", Ext: ".txt"},
		}},
	})
	return p
}
