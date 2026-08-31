// Package samtoolsviewcount owns one samtools view -c command.
package samtoolsviewcount

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/atacseq 2.1.2 samtools 1.17 image resolved for linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/samtools:1.17--h00cdaf9_0@sha256:6f88956b747a67b2a39a3ff72c4de30e665239ee11db610624dd4298e30db1bf"

// Options controls one count command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains one count file.
type Ports struct{ Count gobble.Handle }

// Add records one validated samtools view count command.
func Add(parent modules.Parent, bam gobble.Handle, options Options) (Ports, error) {
	const unit = "samtools_view_count"
	bamPath, err := modules.HandlePath(unit, bam)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/samtools-view-count")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "reads"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".count.txt"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "count output path is invalid")
	}
	command := []string{"samtools", "view", "-c", bamPath}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "1g"}, command, []string{"-c", "-o", "-f", "-F", "-q"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: modules.ShellRedirect(command, outputPath), Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bam", From: bam}}, Outputs: []gobble.Bind{{Name: "count", Spec: output}}})
	return Ports{Count: task.Out("count")}, nil
}
