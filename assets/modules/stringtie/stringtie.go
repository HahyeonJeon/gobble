// Package stringtie owns one reference-guided StringTie command.
package stringtie

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 StringTie image resolved for
// linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/stringtie:2.2.3--h43eeafb_0@sha256:3ae64324a4729c6af09eb1246cd315920136758c0d5743845fd1980769087aa3"

// Options controls one reference-guided StringTie command.
type Options struct {
	modules.Options
	OutDir       gobble.Directory
	Prefix       string
	Strandedness string
}

// Ports are the reference-guided transcript and abundance files.
type Ports struct {
	Transcripts gobble.Handle
	Abundance   gobble.Handle
	Coverage    gobble.Handle
}

// Add records one validated reference-guided StringTie command.
func Add(parent modules.Parent, bam, gtf gobble.Handle, options Options) (Ports, error) {
	const unit = "stringtie"
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
		outDir = gobble.Dir("work/stringtie")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "sample"
	}
	transcripts := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".transcripts.gtf"}
	abundance := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".gene.abundance.txt"}
	coverage := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".coverage.gtf"}
	transcriptsPath, _ := transcripts.Render()
	abundancePath, _ := abundance.Render()
	coveragePath, _ := coverage.Render()
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 2, Memory: "2g"}
	}
	command := []string{"stringtie", bamPath, "-G", gtfPath, "-o", transcriptsPath, "-A", abundancePath, "-C", coveragePath, "-p", strconv.Itoa(modules.ThreadCount(resources.CPU))}
	switch options.Strandedness {
	case gobble.StrandednessForward:
		command = append(command, "--fr")
	case gobble.StrandednessReverse:
		command = append(command, "--rf")
	case "", "auto", gobble.StrandednessUnstranded:
	default:
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "strandedness must be unstranded, forward, reverse, or auto")
	}
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"-G", "-o", "-A", "-C", "-p", "--fr", "--rf"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bam", From: bam}, {Name: "gtf", From: gtf}}, Outputs: []gobble.Bind{{Name: "transcripts", Spec: transcripts}, {Name: "abundance", Spec: abundance}, {Name: "coverage", Spec: coverage}}})
	return Ports{Transcripts: task.Out("transcripts"), Abundance: task.Out("abundance"), Coverage: task.Out("coverage")}, nil
}

// Pipeline returns a standalone validated reference-guided StringTie module.
func Pipeline(bam, gtf gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("stringtie", []modules.Input{{Name: "bam", Spec: bam}, {Name: "gtf", Spec: gtf}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], options)
		return err
	})
}
