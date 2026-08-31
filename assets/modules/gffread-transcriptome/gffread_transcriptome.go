// Package gffreadtranscriptome owns one gffread transcript extraction command.
package gffreadtranscriptome

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/scrnaseq 4.2.0 GFFREAD_TRANSCRIPTOME image for
// linux/amd64. docker.io is the explicit registry form of nf-core's
// biocontainers/gffread reference.
const DefaultImage modules.Image = "docker.io/biocontainers/gffread:0.12.7--hd03093a_1@sha256:f46049f79cc002aaa23c31eb30b4ee7037c76c1429217a15792b242e0dbf365d"

// Options controls one transcript FASTA extraction.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the extracted transcript FASTA.
type Ports struct{ FASTA gobble.Handle }

// Add records gffread -F with typed GTF, genome, and output operands.
func Add(parent modules.Parent, gtf, fasta gobble.Handle, options Options) (Ports, error) {
	const unit = "gffread_transcriptome"
	gtfPath, err := modules.HandlePath(unit, gtf)
	if err != nil {
		return Ports{}, err
	}
	fastaPath, err := modules.HandlePath(unit, fasta)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/scrnaseq/reference")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "transcripts"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".fasta"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "transcript FASTA output path is invalid")
	}
	if err := modules.RejectExtraArgPrefixes(unit, options.ExtraArgs, []string{"-F", "-w", "-g", "-o"}); err != nil {
		return Ports{}, err
	}
	command := []string{"gffread", "-F", gtfPath, "-w", outputPath, "-g", fastaPath}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "1g"}, command, []string{"-F", "-w", "-g"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Command: command, Image: image, Resources: resources,
		Inputs:  []gobble.Bind{{Name: "gtf", From: gtf}, {Name: "fasta", From: fasta}},
		Outputs: []gobble.Bind{{Name: "transcript_fasta", Spec: output}},
	})
	return Ports{FASTA: task.Out("transcript_fasta")}, nil
}

// Pipeline returns a standalone validated transcript extraction module.
func Pipeline(gtf, fasta gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("gffread-transcriptome", []modules.Input{{Name: "gtf", Spec: gtf}, {Name: "fasta", Spec: fasta}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], options)
		return err
	})
}
