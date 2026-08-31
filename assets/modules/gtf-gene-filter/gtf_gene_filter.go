// Package gtfgenefilter owns the scrnaseq GTF-to-reference sequence filter.
package gtfgenefilter

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

const filterScript = `
import sys

fasta_path, gtf_path, output_path = sys.argv[1:]
with open(fasta_path, encoding="utf-8") as fasta:
    names = {line[1:].split(None, 1)[0] for line in fasta if line.startswith(">")}
if not names:
    raise ValueError("FASTA contains no sequence names")
kept = 0
with open(gtf_path, encoding="utf-8") as source, open(output_path, "w", encoding="utf-8") as output:
    for line_number, line in enumerate(source, start=1):
        if line.startswith("#"):
            continue
        fields = line.rstrip("\n").split("\t")
        if len(fields) != 9:
            raise ValueError(f"invalid GTF line {line_number}: expected 9 columns")
        if fields[0] in names:
            output.write(line)
            kept += 1
if kept == 0:
    raise ValueError("GTF has no annotations on reference sequences")
`

// DefaultImage is the nf-core/scrnaseq 4.2.0 GTF_GENE_FILTER Python image for
// linux/amd64. docker.io is the explicit registry form of nf-core's
// biocontainers/python reference.
const DefaultImage modules.Image = "docker.io/biocontainers/python:3.9--1@sha256:d97d2b329b4e44d2e07a9737ba348b185d6a47f34fba0ef301d44d11669cac60"

// Options controls one required GTF sequence filter.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the filtered annotation.
type Ports struct{ GTF gobble.Handle }

// Add records one exact sequence-membership filter. ExtraArgs are rejected
// because all operands and the output binding are typed.
func Add(parent modules.Parent, fasta, gtf gobble.Handle, options Options) (Ports, error) {
	const unit = "gtf_gene_filter"
	if len(options.ExtraArgs) != 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "GTF filter operands are typed and ExtraArgs are unsupported")
	}
	fastaPath, err := modules.HandlePath(unit, fasta)
	if err != nil {
		return Ports{}, err
	}
	gtfPath, err := modules.HandlePath(unit, gtf)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/scrnaseq/reference")
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
	command := []string{"python", "-c", filterScript, fastaPath, gtfPath, outputPath}
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, gobble.Resources{CPU: 1, Memory: "1g"}, command, []string{"-c"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Command: command, Image: image, Resources: resources,
		Inputs:  []gobble.Bind{{Name: "fasta", From: fasta}, {Name: "gtf", From: gtf}},
		Outputs: []gobble.Bind{{Name: "filtered_gtf", Spec: output}},
	})
	return Ports{GTF: task.Out("filtered_gtf")}, nil
}

// Pipeline returns a standalone validated GTF sequence filter.
func Pipeline(fasta, gtf gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("gtf-gene-filter", []modules.Input{{Name: "fasta", Spec: fasta}, {Name: "gtf", Spec: gtf}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], options)
		return err
	})
}
