// Package sampleretentionmapped owns one mapped-read retention gate.
package sampleretentionmapped

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

const mappedRetentionPython = `
import math
import sys

input_path, minimum_text, output_path = sys.argv[1:]
minimum = float(minimum_text)
mapped_text = None

with open(input_path, encoding="utf-8") as source:
    for line in source:
        label, separator, value = line.partition("|")
        if separator and label.strip() == "Uniquely mapped reads %":
            value = value.strip()
            if not value.endswith("%"):
                raise ValueError("STAR uniquely mapped reads value has no percent suffix")
            mapped_text = value[:-1].strip()
            break

if mapped_text is None:
    raise ValueError("STAR log has no uniquely mapped reads percentage")

mapped = float(mapped_text)
if not math.isfinite(mapped) or mapped < 0 or mapped > 100:
    raise ValueError("STAR uniquely mapped reads percentage is outside 0..100")
if mapped < minimum:
    raise ValueError(f"mapped read percentage {mapped} is below minimum {minimum}")

with open(output_path, "w", encoding="ascii") as output:
    output.write(f"{mapped_text}\n")
`

// DefaultImage is the nf-core/rnaseq 3.26.0 custom/gtffilter Python image
// resolved for linux/amd64. One Python process parses, checks, and records the
// STAR metric.
const DefaultImage modules.Image = "quay.io/biocontainers/python:3.9--1@sha256:d97d2b329b4e44d2e07a9737ba348b185d6a47f34fba0ef301d44d11669cac60"

// Options controls one mapped-read retention command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the accepted mapping percentage prerequisite.
type Ports struct{ Accepted gobble.Handle }

// Add records one gate over STAR's uniquely mapped read percentage.
func Add(parent modules.Parent, starLog, prerequisite gobble.Handle, minimum float64, options Options) (Ports, error) {
	const unit = "sample_retention_mapped"
	if minimum < 0 || minimum > 100 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "minimum mapped percent must be between 0 and 100")
	}
	if len(options.ExtraArgs) != 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "the mapped-read retention gate does not accept ExtraArgs")
	}
	logPath, err := modules.HandlePath(unit, starLog)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/sample-retention-mapped")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "mapped_reads"
	}
	accepted := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".accepted.txt"}
	acceptedPath, err := accepted.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "mapped-read retention output path is invalid")
	}
	base := options.Options
	base.ExtraArgs = nil
	command := []string{"python", "-c", mappedRetentionPython, logPath, strconv.FormatFloat(minimum, 'g', -1, 64), acceptedPath}
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, gobble.Resources{CPU: 1, Memory: "256m"}, command, []string{"-c"})
	if err != nil {
		return Ports{}, err
	}
	inputs := []gobble.Bind{{Name: "star_log", From: starLog}}
	if !prerequisite.IsZero() {
		inputs = append(inputs, gobble.Bind{Name: "prerequisite", From: prerequisite})
	}
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Command: command, Image: image, Resources: resources,
		Inputs: inputs, Outputs: []gobble.Bind{{Name: "accepted", Spec: accepted}},
	})
	return Ports{Accepted: task.Out("accepted")}, nil
}

// Pipeline returns a standalone validated mapped-read retention module.
func Pipeline(starLog gobble.PathSpec, minimum float64, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("sample-retention-mapped", []modules.Input{{Name: "star_log", Spec: starLog}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], gobble.Handle{}, minimum, options)
		return err
	})
}
