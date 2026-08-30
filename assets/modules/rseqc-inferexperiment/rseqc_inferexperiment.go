// Package rseqcinferexperiment owns one RSeQC infer_experiment.py command.
package rseqcinferexperiment

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 RSeQC image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/rseqc_r-base:2e29d2dfda9cef15@sha256:4f5926eeef842405756ca3453cd534b72d941c119b8e4785565c7918e05cd8ab"

// Options controls one RSeQC infer_experiment.py command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the strandedness evidence report.
type Ports struct{ Report gobble.Handle }

// Add records one validated RSeQC infer_experiment.py command.
func Add(parent modules.Parent, bam, bai, bed gobble.Handle, options Options) (Ports, error) {
	const unit = "rseqc_inferexperiment"
	bamPath, err := modules.HandlePath(unit, bam)
	if err != nil {
		return Ports{}, err
	}
	if _, err = modules.HandlePath(unit, bai); err != nil {
		return Ports{}, err
	}
	bedPath, err := modules.HandlePath(unit, bed)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/rseqc")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "sample"
	}
	report := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".infer_experiment.txt"}
	reportPath, _ := report.Render()
	command := []string{"infer_experiment.py", "-i", bamPath, "-r", bedPath}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "2g"}, command, []string{"-i", "-r"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: modules.ShellRedirect(command, reportPath), Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bam", From: bam}, {Name: "bai", From: bai}, {Name: "bed", From: bed}}, Outputs: []gobble.Bind{{Name: "report", Spec: report}}})
	return Ports{Report: task.Out("report")}, nil
}

// Pipeline returns a standalone validated RSeQC infer-experiment module.
func Pipeline(bam, bai, bed gobble.PathSpec, options Options) *gobble.Pipeline {
	inputs := []modules.Input{{Name: "bam", Spec: bam}, {Name: "bai", Spec: bai}, {Name: "bed", Spec: bed}}
	return modules.StandaloneChecked("rseqc-inferexperiment", inputs, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], handles[2], options)
		return err
	})
}
