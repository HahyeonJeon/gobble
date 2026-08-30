// Package salmonindex owns one Salmon index command.
package salmonindex

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 Salmon image resolved for
// linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/salmon:1.10.3--h6dccd9a_2@sha256:f83ebb158845ee8138d793347f83b92c75e83c58dd8f4600c6fea2a2453ef08e"

// Options controls one Salmon index command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
}

// Ports contains the complete Salmon index Tree.
type Ports struct{ Index gobble.Handle }

// Add records one validated Salmon index command.
func Add(parent modules.Parent, transcriptome gobble.Handle, options Options) (Ports, error) {
	const unit = "salmon_index"
	transcriptomePath, err := modules.HandlePath(unit, transcriptome)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/reference/salmon-index")
	}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 2, Memory: "2g"}
	}
	command := []string{"salmon", "index", "-t", transcriptomePath, "-i", outDir.String(), "-p", strconv.Itoa(modules.ThreadCount(resources.CPU))}
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"-t", "-i", "-p"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "transcriptome", From: transcriptome}}, Outputs: []gobble.Bind{{Name: "index", Tree: gobble.DeclareTree(outDir)}}})
	return Ports{Index: task.Out("index")}, nil
}

// Pipeline returns a standalone validated Salmon index module.
func Pipeline(transcriptome gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("salmon-index", []modules.Input{{Name: "transcriptome", Spec: transcriptome}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], options)
		return err
	})
}
