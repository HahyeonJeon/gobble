// Package gtffilter owns one reference GTF filtering command.
package gtffilter

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

const filterScript = `
import re
import sys

gtf_path, fasta_path, output_path = sys.argv[1:]

with open(fasta_path, encoding="utf-8") as fasta:
    sequence_names = {
        line[1:].split(None, 1)[0]
        for line in fasta
        if line.startswith(">")
    }

if not sequence_names:
    raise ValueError("FASTA contains no sequence names")

kept = 0
with open(gtf_path, encoding="utf-8") as source, open(output_path, "w", encoding="utf-8") as output:
    for line_number, line in enumerate(source, start=1):
        if line.startswith("#"):
            continue
        fields = line.rstrip("\n").split("\t")
        if len(fields) != 9:
            raise ValueError(f"invalid GTF line {line_number}: expected 9 tab-separated columns")
        if fields[0] not in sequence_names:
            continue
        if not re.search(r'transcript_id "([^"]+)"', fields[8]):
            continue
        output.write(line)
        kept += 1

if kept == 0:
    raise ValueError("all GTF lines removed by sequence and transcript_id filters")
`

// DefaultImage is the nf-core/rnaseq 3.26.0 custom/gtffilter Python image
// resolved for linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/python:3.9--1@sha256:d97d2b329b4e44d2e07a9737ba348b185d6a47f34fba0ef301d44d11669cac60"

// Options controls one required sequence and transcript-id filtering command.
// The selected RNA product does not permit bypass arguments.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the normalized annotation used by every reference and
// sample-level GTF consumer.
type Ports struct{ GTF gobble.Handle }

// Add records one GTF filter over the declared annotation and reference FASTA.
func Add(parent modules.Parent, gtf, fasta gobble.Handle, options Options) (Ports, error) {
	const unit = "gtf_filter"
	if len(options.ExtraArgs) != 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "the required GTF filter does not accept ExtraArgs")
	}
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
		prefix = "genes.filtered"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".gtf"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "filtered GTF output path is invalid")
	}
	base := options.Options
	base.ExtraArgs = nil
	command := []string{"python", "-c", filterScript, gtfPath, fastaPath, outputPath}
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, gobble.Resources{CPU: 1, Memory: "1g"}, command, []string{"-c"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Command: command, Image: image, Resources: resources,
		Inputs:  []gobble.Bind{{Name: "gtf", From: gtf}, {Name: "fasta", From: fasta}},
		Outputs: []gobble.Bind{{Name: "filtered_gtf", Spec: output}},
	})
	return Ports{GTF: task.Out("filtered_gtf")}, nil
}

// Pipeline returns a standalone validated GTF filter module.
func Pipeline(gtf, fasta gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("gtf-filter", []modules.Input{{Name: "gtf", Spec: gtf}, {Name: "fasta", Spec: fasta}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], options)
		return err
	})
}
