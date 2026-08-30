// Package fqlint owns one fq lint command.
package fqlint

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 fq image resolved for linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/fq:0.12.0--h9ee0642_0@sha256:74b59572f1d05b4829b45b599ee04311c8b3acec510f3cfb879f23b4bbd2090b"

// Options controls one fq lint command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the retained lint report.
type Ports struct{ Report gobble.Handle }

// Add records one validated fq lint command.
func Add(parent modules.Parent, fastq gobble.Handle, options Options) (Ports, error) {
	const unit = "fq_lint"
	input, err := modules.HandlePath(unit, fastq)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/fq-lint")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "reads"
	}
	report := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".fq_lint.txt"}
	reportPath, err := report.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "lint report path is invalid")
	}
	command := []string{"fq", "lint", input}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "512m"}, command, nil)
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: modules.ShellRedirect(command, reportPath), Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "fastq", From: fastq}}, Outputs: []gobble.Bind{{Name: "report", Spec: report}}})
	return Ports{Report: task.Out("report")}, nil
}

// Pipeline returns a standalone validated fq lint module.
func Pipeline(fastq gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("fq-lint", []modules.Input{{Name: "fastq", Spec: fastq}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], options)
		return err
	})
}
