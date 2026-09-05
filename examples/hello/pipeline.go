// Package hello counts sequences in a tiny FASTA file using a local command.
// It demonstrates workspace inputs and outputs without downloading tools or data.
package hello

import "github.com/HahyeonJeon/gobble"

func Pipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("hello-gobble")
	reads := p.AddInput("sequences", gobble.PathSpec{
		Dir: gobble.Dir("inputs"), Base: "sequences", Ext: ".fasta",
	})
	p.AddTask(gobble.TaskSpec{
		Name: "count-sequences",
		Command: []string{"sh", "-c",
			"awk '/^>/{n++} END {print n+0}' inputs/sequences.fasta > results/sequence-count.txt"},
		Inputs: []gobble.Bind{{Name: "sequences", From: reads}},
		Outputs: []gobble.Bind{{Name: "count", Spec: gobble.PathSpec{
			Dir: gobble.Dir("results"), Base: "sequence-count", Ext: ".txt",
		}}},
	})
	return p
}
