// Package ucscbedgraphtobigwig owns one UCSC bedGraphToBigWig command.
package ucscbedgraphtobigwig

import (
	"fmt"

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

// OptionalPorts contains status.txt and, for inferred stranded libraries, a
// directional coverage.bigWig in one Tree.
type OptionalPorts struct{ Artifacts gobble.Handle }

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

// AddOptional converts a clipped bedGraph only when the inferred coverage
// Tree contains one. Unstranded inference remains an explicit status artifact.
func AddOptional(parent modules.Parent, bedgraphTree, sizes gobble.Handle, options Options) (OptionalPorts, error) {
	const unit = "ucsc_bedgraphtobigwig_inferred"
	sizesPath, err := modules.HandlePath(unit, sizes)
	if err != nil {
		return OptionalPorts{}, err
	}
	inputDir := bedgraphTree.Tree().Dir
	if inputDir.IsZero() {
		return OptionalPorts{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "clipped bedGraph artifacts must be a Tree")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/coverage")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "coverage"
	}
	treeDir := outDir.Join(prefix)
	inputPath := inputDir.String() + "/clipped.bedGraph"
	outputPath := treeDir.String() + "/coverage.bigWig"
	command := []string{"bedGraphToBigWig", inputPath, sizesPath, outputPath}
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

// Pipeline returns a standalone validated bedGraphToBigWig module.
func Pipeline(bedgraph, sizes gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("ucsc-bedgraphtobigwig", []modules.Input{{Name: "bedgraph", Spec: bedgraph}, {Name: "sizes", Spec: sizes}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], options)
		return err
	})
}
