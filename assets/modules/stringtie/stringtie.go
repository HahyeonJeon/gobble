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
	return add(parent, bam, gtf, gobble.Handle{}, options)
}

// AddInferred records StringTie with --fr/--rf selected from a completed
// strandedness inference task.
func AddInferred(parent modules.Parent, bam, gtf, strandedness gobble.Handle, options Options) (Ports, error) {
	return add(parent, bam, gtf, strandedness, options)
}

func add(parent modules.Parent, bam, gtf, strandedness gobble.Handle, options Options) (Ports, error) {
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
	commandFor := func(strand string) ([]string, string, gobble.Resources, error) {
		command := []string{"stringtie", bamPath, "-G", gtfPath, "-o", transcriptsPath, "-A", abundancePath, "-C", coveragePath, "-p", strconv.Itoa(modules.ThreadCount(resources.CPU))}
		switch strand {
		case gobble.StrandednessForward:
			command = append(command, "--fr")
		case gobble.StrandednessReverse:
			command = append(command, "--rf")
		case "", gobble.StrandednessUnstranded:
		default:
			return nil, "", gobble.Resources{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "strandedness must be unstranded, forward, or reverse")
		}
		base := options.Options
		base.Resources = resources
		return modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"-G", "-o", "-A", "-C", "-p", "--fr", "--rf"})
	}
	inputs := []gobble.Bind{{Name: "bam", From: bam}, {Name: "gtf", From: gtf}}
	spec := gobble.TaskSpec{Name: unit, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{{Name: "transcripts", Spec: transcripts}, {Name: "abundance", Spec: abundance}, {Name: "coverage", Spec: coverage}}}
	if strandedness.IsZero() {
		command, image, resolvedResources, commandErr := commandFor(options.Strandedness)
		if commandErr != nil {
			return Ports{}, commandErr
		}
		spec.Command, spec.Image, spec.Resources = command, image, resolvedResources
	} else {
		strandPath, pathErr := modules.HandlePath(unit, strandedness)
		if pathErr != nil {
			return Ports{}, pathErr
		}
		commands := make(map[string][]string, 3)
		for _, strand := range []string{gobble.StrandednessUnstranded, gobble.StrandednessForward, gobble.StrandednessReverse} {
			command, image, resolvedResources, commandErr := commandFor(strand)
			if commandErr != nil {
				return Ports{}, commandErr
			}
			commands[strand] = command
			spec.Image, spec.Resources = image, resolvedResources
		}
		spec.Script = modules.StrandedCommand(strandPath, commands[gobble.StrandednessUnstranded], commands[gobble.StrandednessForward], commands[gobble.StrandednessReverse])
		spec.Inputs = append(spec.Inputs, gobble.Bind{Name: "strandedness", From: strandedness})
	}
	task := parent.AddTask(spec)
	return Ports{Transcripts: task.Out("transcripts"), Abundance: task.Out("abundance"), Coverage: task.Out("coverage")}, nil
}

// Pipeline returns a standalone validated reference-guided StringTie module.
func Pipeline(bam, gtf gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("stringtie", []modules.Input{{Name: "bam", Spec: bam}, {Name: "gtf", Spec: gtf}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], options)
		return err
	})
}
