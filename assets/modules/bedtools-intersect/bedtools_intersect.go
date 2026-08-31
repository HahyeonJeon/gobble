// Package bedtoolsintersect owns one bedtools intersect command.
package bedtoolsintersect

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/atacseq 2.1.2 bedtools 2.30.0 image resolved for linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/bedtools:2.30.0--hc088bd4_0@sha256:b0018bd0a10853e19ee92f6d46d8d12f1c41e516845105e1f02c91b4d7b961b1"

// Options controls one BAM/interval intersection.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
	Invert bool
}

// Ports contains the selected BAM records.
type Ports struct{ BAM gobble.Handle }

// Add records one validated bedtools intersect command.
func Add(parent modules.Parent, bam, intervals gobble.Handle, options Options) (Ports, error) {
	const unit = "bedtools_intersect"
	bamPath, err := modules.HandlePath(unit, bam)
	if err != nil {
		return Ports{}, err
	}
	intervalPath, err := modules.HandlePath(unit, intervals)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/bedtools-intersect")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "intersect"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".bam"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "intersection BAM path is invalid")
	}
	command := []string{"bedtools", "intersect", "-abam", bamPath, "-b", intervalPath}
	if options.Invert {
		command = append(command, "-v")
	} else {
		command = append(command, "-u")
	}
	protected := []string{"-a", "-abam", "-b", "-v", "-u", "-wa", "-wb"}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "2g"}, command, protected)
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: modules.ShellRedirect(command, outputPath), Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bam", From: bam}, {Name: "intervals", From: intervals}}, Outputs: []gobble.Bind{{Name: "selected_bam", Spec: output}}})
	return Ports{BAM: task.Out("selected_bam")}, nil
}
