// Package ataqv owns one ataqv command.
package ataqv

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/atacseq 2.1.2 ataqv 1.3.1 image resolved for linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/ataqv:1.3.1--py310ha155cf9_1@sha256:cb5ff55de7d503a959eea0bc219067012ad9ac8fdb43c7e563b3909a28a5159a"

// Options controls one ataqv command.
type Options struct {
	modules.Options
	OutDir   gobble.Directory
	Prefix   string
	Organism string
	MitoName string
}

// Ports contains one ataqv JSON report.
type Ports struct{ JSON gobble.Handle }

// Add records one validated ataqv command.
func Add(parent modules.Parent, bam, bai, peaks, tss, autosomes gobble.Handle, options Options) (Ports, error) {
	const unit = "ataqv"
	if options.Organism == "" || options.MitoName == "" {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "organism and mitochondrial reference name are required")
	}
	paths := make([]string, 5)
	for i, handle := range []gobble.Handle{bam, bai, peaks, tss, autosomes} {
		path, err := modules.HandlePath(unit, handle)
		if err != nil {
			return Ports{}, err
		}
		paths[i] = path
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/ataqv")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "sample"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".ataqv.json"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "ataqv output path is invalid")
	}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 2, Memory: "4g"}
	}
	threads := modules.ThreadCount(resources.CPU)
	if threads < 1 {
		threads = 1
	}
	command := []string{"ataqv", "--metrics-file", outputPath, "--threads", strconv.Itoa(threads), "--name", prefix, "--mitochondrial-reference-name", options.MitoName, "--peak-file", paths[2], "--tss-file", paths[3], "--autosomal-reference-file", paths[4], options.Organism, paths[0]}
	protected := []string{"--metrics-file", "--threads", "--name", "--mitochondrial-reference-name", "--peak-file", "--tss-file", "--autosomal-reference-file"}
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, protected)
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bam", From: bam}, {Name: "bai", From: bai}, {Name: "peaks", From: peaks}, {Name: "tss", From: tss}, {Name: "autosomes", From: autosomes}}, Outputs: []gobble.Bind{{Name: "json", Spec: output}}})
	return Ports{JSON: task.Out("json")}, nil
}
