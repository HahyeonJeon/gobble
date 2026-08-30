// Package gffread owns one gffread annotation command.
package gffread

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 gffread image resolved for
// linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/gffread:0.12.7--hdcf5f25_4@sha256:88df8382561fbe6b8ad43279c649d0139fbee022127ebdff4608da845d703bab"

// Options controls one gffread output.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains one reference-derived regular file.
type Ports struct{ Output gobble.Handle }

// AddTranscriptome records gffread transcript extraction with -w.
func AddTranscriptome(parent modules.Parent, gtf, fasta gobble.Handle, options Options) (Ports, error) {
	return add(parent, "gffread_transcriptome", gtf, fasta, options, ".transcripts.fasta", []string{"-w"}, []string{"-w"})
}

// AddBED records gffread BED conversion for reference-guided QC.
func AddBED(parent modules.Parent, gtf, fasta gobble.Handle, options Options) (Ports, error) {
	return add(parent, "gffread_bed", gtf, fasta, options, ".genes.bed", []string{"--bed", "-o"}, []string{"--bed", "-o"})
}

// TranscriptomePipeline returns a standalone transcript-extraction module.
func TranscriptomePipeline(gtf, fasta gobble.PathSpec, options Options) *gobble.Pipeline {
	return standalone("gffread-transcriptome", gtf, fasta, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := AddTranscriptome(parent, handles[0], handles[1], options)
		return err
	})
}

// BEDPipeline returns a standalone annotation-to-BED module.
func BEDPipeline(gtf, fasta gobble.PathSpec, options Options) *gobble.Pipeline {
	return standalone("gffread-bed", gtf, fasta, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := AddBED(parent, handles[0], handles[1], options)
		return err
	})
}

func standalone(name string, gtf, fasta gobble.PathSpec, build func(modules.Parent, []gobble.Handle) error) *gobble.Pipeline {
	return modules.StandaloneChecked(name, []modules.Input{{Name: "gtf", Spec: gtf}, {Name: "fasta", Spec: fasta}}, build)
}

func add(parent modules.Parent, unit string, gtf, fasta gobble.Handle, options Options, ext string, outputArgs, namedFlags []string) (Ports, error) {
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
		outDir = gobble.Dir("work/reference")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "reference"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ext}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "gffread output path is invalid")
	}
	command := []string{"gffread", gtfPath, "-g", fastaPath}
	command = append(command, outputArgs...)
	command = append(command, outputPath)
	namedFlags = append([]string{"-g"}, namedFlags...)
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "1g"}, command, namedFlags)
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "gtf", From: gtf}, {Name: "fasta", From: fasta}}, Outputs: []gobble.Bind{{Name: "output", Spec: output}}})
	return Ports{Output: task.Out("output")}, nil
}
