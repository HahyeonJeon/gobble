// Package sampleretentiontrimmed owns one trimmed-read retention gate.
package sampleretentiontrimmed

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

const trimmedRetentionPython = `
import gzip
import sys

input_path, minimum_text, output_path = sys.argv[1:]
minimum = int(minimum_text)
line_count = 0
with gzip.open(input_path, "rt", encoding="ascii", newline="") as source:
    for line_count, _ in enumerate(source, start=1):
        pass

if line_count % 4 != 0:
    raise ValueError("FASTQ does not contain complete four-line records")

count = line_count // 4
if count < minimum:
    raise ValueError(f"trimmed read count {count} is below minimum {minimum}")

with open(output_path, "w", encoding="ascii") as output:
    output.write(f"{count}\n")
`

// DefaultImage is the nf-core/rnaseq 3.26.0 custom/gtffilter Python image
// resolved for linux/amd64. Its standard library reads gzip without a helper
// process, so the retention task remains one executable command.
const DefaultImage modules.Image = "quay.io/biocontainers/python:3.9--1@sha256:d97d2b329b4e44d2e07a9737ba348b185d6a47f34fba0ef301d44d11669cac60"

// Options controls one trimmed-read retention command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the accepted fragment count prerequisite.
type Ports struct{ Accepted gobble.Handle }

// Add records one gate that accepts a FASTQ when its first mate contains at
// least minimum complete records. One first-mate record represents one
// single-end read or one paired-end fragment.
func Add(parent modules.Parent, read1 gobble.Handle, minimum int64, options Options) (Ports, error) {
	const unit = "sample_retention_trimmed"
	if minimum < 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "minimum trimmed reads must be non-negative")
	}
	if len(options.ExtraArgs) != 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "the trimmed-read retention gate does not accept ExtraArgs")
	}
	read1Path, err := modules.HandlePath(unit, read1)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/sample-retention-trimmed")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "trimmed_reads"
	}
	accepted := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".accepted.txt"}
	acceptedPath, err := accepted.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "trimmed-read retention output path is invalid")
	}
	base := options.Options
	base.ExtraArgs = nil
	command := []string{"python", "-c", trimmedRetentionPython, read1Path, strconv.FormatInt(minimum, 10), acceptedPath}
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, gobble.Resources{CPU: 1, Memory: "256m"}, command, []string{"-c"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Command: command, Image: image, Resources: resources,
		Inputs:  []gobble.Bind{{Name: "read1", From: read1}},
		Outputs: []gobble.Bind{{Name: "accepted", Spec: accepted}},
	})
	return Ports{Accepted: task.Out("accepted")}, nil
}

// Pipeline returns a standalone validated trimmed-read retention module.
func Pipeline(read1 gobble.PathSpec, minimum int64, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("sample-retention-trimmed", []modules.Input{{Name: "read1", Spec: read1}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], minimum, options)
		return err
	})
}
