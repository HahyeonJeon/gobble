// Package qualimapbamqc owns one Qualimap rnaseq command.
package qualimapbamqc

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 Qualimap image resolved for
// linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/qualimap:2.3--hdfd78af_0@sha256:49d81e27bf995d0ef72ae46c79c22c8c779e21573cd76f7b60cf3f73af61b087"

// Options controls one Qualimap RNA-seq BAM QC command.
type Options struct {
	modules.Options
	OutDir       gobble.Directory
	Prefix       string
	Strandedness string
	Paired       bool
}

// Ports contains the primary Qualimap text report for aggregate reporting.
type Ports struct{ Report gobble.Handle }

// Add records one validated Qualimap rnaseq command.
func Add(parent modules.Parent, bam, gtf gobble.Handle, options Options) (Ports, error) {
	const unit = "qualimap_bamqc"
	bamPath, err := modules.HandlePath(unit, bam)
	if err != nil {
		return Ports{}, err
	}
	gtfPath, err := modules.HandlePath(unit, gtf)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/qualimap")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "sample"
	}
	resultDir := gobble.Dir(outDir.String() + "/" + prefix)
	protocol := "non-strand-specific"
	switch options.Strandedness {
	case gobble.StrandednessForward:
		protocol = "strand-specific-forward"
	case gobble.StrandednessReverse:
		protocol = "strand-specific-reverse"
	case "", "auto", gobble.StrandednessUnstranded:
	default:
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "strandedness must be unstranded, forward, reverse, or auto")
	}
	command := []string{"qualimap", "rnaseq", "-bam", bamPath, "-gtf", gtfPath, "-p", protocol, "-outdir", resultDir.String()}
	if options.Paired {
		command = append(command, "-pe")
	}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 2, Memory: "4g"}, command, []string{"-bam", "-gtf", "-p", "-outdir", "-pe"})
	if err != nil {
		return Ports{}, err
	}
	report := gobble.PathSpec{Dir: resultDir, Base: "rnaseq_qc_results", Ext: ".txt"}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bam", From: bam}, {Name: "gtf", From: gtf}}, Outputs: []gobble.Bind{{Name: "report", Spec: report}}})
	return Ports{Report: task.Out("report")}, nil
}

// Pipeline returns a standalone validated Qualimap RNA-seq module.
func Pipeline(bam, gtf gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("qualimap-bamqc", []modules.Input{{Name: "bam", Spec: bam}, {Name: "gtf", Spec: gtf}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], options)
		return err
	})
}
