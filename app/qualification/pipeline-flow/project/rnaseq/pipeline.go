package rnaseq

import gobble "github.com/HahyeonJeon/gobble"

// Pipeline is a non-executing qualification fixture. Its declared commands must
// never be run during flow inspection; it contains a real fan-out and join.
func Pipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("RNA_analysis")
	reads := p.AddInput("reads", gobble.PathSpec{Base: "sample", Ext: ".fastq"})
	reference := p.AddInput("reference", gobble.PathSpec{Base: "reference", Ext: ".fa"})
	trim := p.AddTask(gobble.TaskSpec{Name: "trim", Display: gobble.TaskDisplay{Stage: "Trim adapters"}, Command: []string{"must-not-run"}, Image: "trim-galore:0.6.10", Resources: gobble.Resources{CPU: 2, Memory: "4g"}, Inputs: []gobble.Bind{{Name: "reads", From: reads}}, Outputs: []gobble.Bind{{Name: "trimmed", Spec: gobble.PathSpec{Base: "trimmed", Ext: ".fastq"}}}})
	align := p.AddTask(gobble.TaskSpec{Name: "align", Display: gobble.TaskDisplay{Stage: "Align reads"}, Command: []string{"must-not-run"}, Inputs: []gobble.Bind{{Name: "reads", From: trim.Out("trimmed")}, {Name: "genome", From: reference}}, Outputs: []gobble.Bind{{Name: "alignment", Spec: gobble.PathSpec{Base: "aligned", Ext: ".bam"}}}})
	quality := p.AddTask(gobble.TaskSpec{Name: "quality", Display: gobble.TaskDisplay{Stage: "Check read quality"}, Command: []string{"must-not-run"}, Inputs: []gobble.Bind{{Name: "reads", From: trim.Out("trimmed")}}, Outputs: []gobble.Bind{{Name: "report", Spec: gobble.PathSpec{Base: "quality", Ext: ".html"}}}})
	count := p.AddTask(gobble.TaskSpec{Name: "count", Display: gobble.TaskDisplay{Stage: "Count genes"}, Command: []string{"must-not-run"}, Inputs: []gobble.Bind{{Name: "alignment", From: align.Out("alignment")}}, Outputs: []gobble.Bind{{Name: "counts", Spec: gobble.PathSpec{Base: "counts", Ext: ".tsv"}}}})
	p.AddTask(gobble.TaskSpec{Name: "summary", Display: gobble.TaskDisplay{Stage: "Analysis summary"}, Command: []string{"must-not-run"}, Inputs: []gobble.Bind{{Name: "counts", From: count.Out("counts")}, {Name: "quality", From: quality.Out("report")}}, Outputs: []gobble.Bind{{Name: "summary", Spec: gobble.PathSpec{Base: "summary", Ext: ".html"}}}})
	return p
}
