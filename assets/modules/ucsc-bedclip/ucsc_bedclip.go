// Package ucscbedclip owns one UCSC bedClip command.
package ucscbedclip

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 bedClip image resolved for
// linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/ucsc-bedclip:377--h0b8a92a_2@sha256:d848a443bc2ee59504de4a2389abc196f48b3f1886938eb7d3c1cbf4a260b285"

// Options controls one bedClip command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the clipped bedGraph.
type Ports struct{ BedGraph gobble.Handle }

// Add records one validated bedClip command.
func Add(parent modules.Parent, bedgraph, sizes gobble.Handle, options Options) (Ports, error) {
	const unit = "ucsc_bedclip"
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
		outDir = gobble.Dir("work/coverage")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "coverage.clipped"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".bedGraph"}
	outputPath, _ := output.Render()
	command := []string{"bedClip", bedgraphPath, sizesPath, outputPath}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "1g"}, command, nil)
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bedgraph", From: bedgraph}, {Name: "sizes", From: sizes}}, Outputs: []gobble.Bind{{Name: "clipped", Spec: output}}})
	return Ports{BedGraph: task.Out("clipped")}, nil
}

// Pipeline returns a standalone validated bedClip module.
func Pipeline(bedgraph, sizes gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("ucsc-bedclip", []modules.Input{{Name: "bedgraph", Spec: bedgraph}, {Name: "sizes", Spec: sizes}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], options)
		return err
	})
}
