package badbackend

import "github.com/HahyeonJeon/gobble"

func Pipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("backend")
	p.AddTask(gobble.TaskSpec{
		Name:    "copy",
		Command: []string{"true"},
		Backend: "slurm",
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Base: "out", Ext: ".txt"},
		}},
	})
	return p
}
