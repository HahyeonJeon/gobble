// Package gtftot2g owns one transcript-to-gene relation command.
package gtftot2g

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

const relationScript = `
import re
import sys

gtf_path, output_path = sys.argv[1:]
seen = set()
written = 0
with open(gtf_path, encoding="utf-8") as source, open(output_path, "w", encoding="utf-8") as output:
    for line_number, line in enumerate(source, start=1):
        if line.startswith("#"):
            continue
        fields = line.rstrip("\n").split("\t")
        if len(fields) != 9:
            raise ValueError(f"invalid GTF line {line_number}: expected 9 columns")
        if fields[2] != "transcript":
            continue
        attrs = fields[8]
        transcript = re.search(r'transcript_id "([^"]+)"', attrs)
        gene = re.search(r'gene_id "([^"]+)"', attrs)
        name = re.search(r'gene_name "([^"]+)"', attrs)
        if not transcript or not gene or not name or transcript.group(1) in seen:
            continue
        output.write(f"{transcript.group(1)}\t{gene.group(1)}\t{name.group(1)}\n")
        seen.add(transcript.group(1))
        written += 1
if written == 0:
    raise ValueError("GTF contains no complete transcript-to-gene relations")
`

// DefaultImage is the nf-core/scrnaseq 4.2.0 reference Python image resolved
// for linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/python:3.9--1@sha256:d97d2b329b4e44d2e07a9737ba348b185d6a47f34fba0ef301d44d11669cac60"

// Options controls one three-column transcript-to-gene relation output.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the transcript, gene, and gene-name relation.
type Ports struct{ T2G gobble.Handle }

// Add records one exact GTF relation projection.
func Add(parent modules.Parent, gtf gobble.Handle, options Options) (Ports, error) {
	const unit = "gtf_to_t2g"
	if len(options.ExtraArgs) != 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "relation operands are typed and ExtraArgs are unsupported")
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
		prefix = "transcript_to_gene"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".tsv"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "transcript-to-gene output path is invalid")
	}
	base := options.Options
	base.ExtraArgs = nil
	command := []string{"python", "-c", relationScript, gtfPath, outputPath}
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, gobble.Resources{CPU: 1, Memory: "1g"}, command, []string{"-c"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "gtf", From: gtf}}, Outputs: []gobble.Bind{{Name: "t2g", Spec: output}}})
	return Ports{T2G: task.Out("t2g")}, nil
}

// Pipeline returns a standalone validated relation module.
func Pipeline(gtf gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("gtf-to-t2g", []modules.Input{{Name: "gtf", Spec: gtf}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], options)
		return err
	})
}
