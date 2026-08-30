// Package ucscbedclip owns one UCSC bedClip command.
package ucscbedclip

import (
	"fmt"

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

// OptionalPorts contains status.txt and an optional clipped.bedGraph in one
// Tree.
type OptionalPorts struct{ Artifacts gobble.Handle }

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

// AddOptional clips a bedGraph only when the inferred coverage Tree contains
// one. Unstranded inference propagates a not-applicable status Tree.
func AddOptional(parent modules.Parent, bedgraphTree, sizes gobble.Handle, options Options) (OptionalPorts, error) {
	const unit = "ucsc_bedclip_inferred"
	sizesPath, err := modules.HandlePath(unit, sizes)
	if err != nil {
		return OptionalPorts{}, err
	}
	inputDir := bedgraphTree.Tree().Dir
	if inputDir.IsZero() {
		return OptionalPorts{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "bedGraph artifacts must be a Tree")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/coverage")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "coverage.clipped"
	}
	treeDir := outDir.Join(prefix)
	inputPath := inputDir.String() + "/coverage.bedGraph"
	outputPath := treeDir.String() + "/clipped.bedGraph"
	command := []string{"bedClip", inputPath, sizesPath, outputPath}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "1g"}, command, nil)
	if err != nil {
		return OptionalPorts{}, err
	}
	script := fmt.Sprintf(`mkdir -p %s
cp %s %s
if test -f %s; then
  %s
fi`, modules.ShellQuote(treeDir.String()), modules.ShellQuote(inputDir.String()+"/status.txt"), modules.ShellQuote(treeDir.String()+"/status.txt"), modules.ShellQuote(inputPath), modules.ShellCommand(command))
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Script: script, Image: image, Resources: resources,
		Inputs:  []gobble.Bind{{Name: "bedgraph_artifacts", From: bedgraphTree, Tree: gobble.DeclareTree(inputDir)}, {Name: "sizes", From: sizes}},
		Outputs: []gobble.Bind{{Name: "artifacts", Tree: gobble.DeclareTree(treeDir)}},
	})
	return OptionalPorts{Artifacts: task.Out("artifacts")}, nil
}

// Pipeline returns a standalone validated bedClip module.
func Pipeline(bedgraph, sizes gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("ucsc-bedclip", []modules.Input{{Name: "bedgraph", Spec: bedgraph}, {Name: "sizes", Spec: sizes}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], options)
		return err
	})
}
