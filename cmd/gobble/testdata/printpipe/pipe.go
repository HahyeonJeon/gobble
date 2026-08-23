package printpipe

import (
	"fmt"

	"github.com/HahyeonJeon/gobble"
)

func init() {
	fmt.Print("user init output\n")
}

func Pipeline() *gobble.Pipeline {
	fmt.Print("user Pipeline output\n")
	p := gobble.NewPipeline("printed")
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
