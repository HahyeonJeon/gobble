package trimreview

import (
	gobble "github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	trimgalore "github.com/HahyeonJeon/gobble/assets/modules/trim-galore"
)

// Inspection-only: the real module declares its command, but no task is run.
func Pipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("read_quality_review")
	reads := p.AddInput("reads", gobble.PathSpec{Base: "sample", Ext: ".fastq"})
	trimmed, err := trimgalore.Add(modules.WithDisplay(p, gobble.TaskDisplay{Stage: "Trim adapters"}), reads, gobble.Handle{},
		trimgalore.Options{Quality: 25, Length: 40})
	if err != nil {
		panic(err)
	}
	p.AddTask(gobble.TaskSpec{Name: "quality", Display: gobble.TaskDisplay{Stage: "Check read quality"}, Command: []string{"must-not-run"},
		Inputs:  []gobble.Bind{{Name: "reads", From: trimmed.Read1}},
		Outputs: []gobble.Bind{{Name: "report", Spec: gobble.PathSpec{Base: "quality", Ext: ".html"}}}})
	return p
}
