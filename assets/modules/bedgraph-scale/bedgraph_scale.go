// Package bedgraphscale owns one awk command that RPM-normalizes a bedGraph.
package bedgraphscale

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/atacseq 2.1.2 bedtools image used by its normalized bedGraph stage.
const DefaultImage modules.Image = "quay.io/biocontainers/bedtools:2.30.0--hc088bd4_0@sha256:b0018bd0a10853e19ee92f6d46d8d12f1c41e516845105e1f02c91b4d7b961b1"

// Options controls one scale command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the normalized bedGraph.
type Ports struct{ BedGraph gobble.Handle }

// Add records one awk command. The first non-primary mapped line in flagstat owns the RPM denominator.
func Add(parent modules.Parent, bedgraph, flagstat gobble.Handle, options Options) (Ports, error) {
	const unit = "bedgraph_scale"
	bedgraphPath, err := modules.HandlePath(unit, bedgraph)
	if err != nil {
		return Ports{}, err
	}
	flagstatPath, err := modules.HandlePath(unit, flagstat)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/bedgraph-scale")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "normalized"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".bedGraph"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "normalized bedGraph path is invalid")
	}
	program := `FNR==NR { if ($0 ~ / mapped \(/ && $0 !~ /primary/) mapped=$1; next } { if (mapped <= 0) exit 2; $4=$4*1000000/mapped; print $1,$2,$3,$4 }`
	command := []string{"awk", "-v", "OFS=\t", program, flagstatPath, bedgraphPath}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "512m"}, command, []string{"-v", "-f"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: modules.ShellRedirect(command, outputPath), Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bedgraph", From: bedgraph}, {Name: "flagstat", From: flagstat}}, Outputs: []gobble.Bind{{Name: "scaled_bedgraph", Spec: output}}})
	return Ports{BedGraph: task.Out("scaled_bedgraph")}, nil
}
