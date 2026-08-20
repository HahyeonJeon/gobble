package runlocal

import "github.com/HahyeonJeon/gobble"

func Pipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("run-local")
	in := p.AddInput("reads", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "sample", Ext: ".txt"})
	p.AddTask(gobble.TaskSpec{
		Name:    "image",
		Image:   "alpine:3.21",
		Command: []string{"sh", "-c", "pwd > out/docker/pwd.txt && cp in/sample.txt out/docker/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{
			{Name: "out", Spec: gobble.PathSpec{Dir: gobble.Dir("out/docker"), Base: "sample", Ext: ".txt"}},
			{Name: "pwd", Spec: gobble.PathSpec{Dir: gobble.Dir("out/docker"), Base: "pwd", Ext: ".txt"}},
		},
		Params:    []gobble.Param{{Name: "mode", Value: "fast"}},
		Resources: gobble.Resources{CPU: 1, Memory: "256m"},
	})
	p.AddTask(gobble.TaskSpec{
		Name:    "host",
		Command: []string{"cp", "in/sample.txt", "out/process/sample.txt"},
		Inputs:  []gobble.Bind{{Name: "in", From: in}},
		Outputs: []gobble.Bind{{
			Name: "out",
			Spec: gobble.PathSpec{Dir: gobble.Dir("out/process"), Base: "sample", Ext: ".txt"},
		}},
	})
	return p
}
