// Package bedtoolsgenomecov owns one bedtools genomecov command.
package bedtoolsgenomecov

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 bedtools image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/bedtools_coreutils:a623c13f66d5262b@sha256:87d2a7cad1272517e4b6e7e2f846a3763c012b46978f1ff0dc5c345f32dbeeac"

// Options controls one BAM-to-bedGraph coverage command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
	Strand string
}

// Ports contains one sorted bedGraph.
type Ports struct{ BedGraph gobble.Handle }

// Add records one validated bedtools genomecov command.
func Add(parent modules.Parent, bam gobble.Handle, options Options) (Ports, error) {
	const unit = "bedtools_genomecov"
	bamPath, err := modules.HandlePath(unit, bam)
	if err != nil {
		return Ports{}, err
	}
	if options.Strand != "" && options.Strand != "+" && options.Strand != "-" {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "strand must be empty, +, or -")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/coverage")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "coverage"
	}
	bedgraph := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".bedGraph"}
	bedgraphPath, _ := bedgraph.Render()
	command := []string{"bedtools", "genomecov", "-bg", "-split", "-ibam", bamPath}
	if options.Strand != "" {
		command = append(command, "-strand", options.Strand)
	}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "1g"}, command, []string{"-bg", "-split", "-ibam", "-strand"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: modules.ShellRedirect(command, bedgraphPath), Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bam", From: bam}}, Outputs: []gobble.Bind{{Name: "bedgraph", Spec: bedgraph}}})
	return Ports{BedGraph: task.Out("bedgraph")}, nil
}

// Pipeline returns a standalone validated bedtools genomecov module.
func Pipeline(bam gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("bedtools-genomecov", []modules.Input{{Name: "bam", Spec: bam}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], options)
		return err
	})
}
