// Package wclines owns one wc -l command.
package wclines

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/atacseq 2.1.2 Python utility image, which includes coreutils.
const DefaultImage modules.Image = "quay.io/biocontainers/python:3.8.3@sha256:4965e8f9078ba50c7148d49dcbc41c1827f21cb74329013deeca366204f0e317"

// Options controls one line-count command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains one line-count report.
type Ports struct{ Count gobble.Handle }

// Add records one validated wc command.
func Add(parent modules.Parent, input gobble.Handle, options Options) (Ports, error) {
	const unit = "wc_lines"
	inputPath, err := modules.HandlePath(unit, input)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/peak-qc")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "peaks"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".peak_count.txt"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "line-count output path is invalid")
	}
	command := []string{"wc", "-l", inputPath}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "256m"}, command, []string{"-l", "-c", "-w"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: modules.ShellRedirect(command, outputPath), Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "input", From: input}}, Outputs: []gobble.Bind{{Name: "count", Spec: output}}})
	return Ports{Count: task.Out("count")}, nil
}
