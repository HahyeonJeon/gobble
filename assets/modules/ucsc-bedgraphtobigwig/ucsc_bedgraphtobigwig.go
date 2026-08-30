// Package ucscbedgraphtobigwig owns one UCSC bedGraphToBigWig command.
package ucscbedgraphtobigwig

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 bedGraphToBigWig image resolved
// for linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/ucsc-bedgraphtobigwig:469--h9b8f530_0@sha256:8b789d0f8b293eff4e54946df880f43bd2e33fa025acf9e2ccd407dd8e988c1d"

// Options controls one bedGraphToBigWig command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the BigWig track.
type Ports struct{ BigWig gobble.Handle }

// Add records one validated bedGraphToBigWig command.
func Add(parent modules.Parent, bedgraph, sizes gobble.Handle, options Options) (Ports, error) {
	const unit = "ucsc_bedgraphtobigwig"
	bedgraphPath, err := modules.HandlePath(unit, bedgraph)
	if err != nil {
		return Ports{}, err
	}
	sizesPath, err := modules.HandlePath(unit, sizes)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/coverage")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "coverage"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".bigWig"}
	outputPath, _ := output.Render()
	command := []string{"bedGraphToBigWig", bedgraphPath, sizesPath, outputPath}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "1g"}, command, nil)
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bedgraph", From: bedgraph}, {Name: "sizes", From: sizes}}, Outputs: []gobble.Bind{{Name: "bigwig", Spec: output}}})
	return Ports{BigWig: task.Out("bigwig")}, nil
}

// Pipeline returns a standalone validated bedGraphToBigWig module.
func Pipeline(bedgraph, sizes gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("ucsc-bedgraphtobigwig", []modules.Input{{Name: "bedgraph", Spec: bedgraph}, {Name: "sizes", Spec: sizes}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], options)
		return err
	})
}
