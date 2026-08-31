// Package deeptoolsplotprofile owns one deepTools plotProfile command.
package deeptoolsplotprofile

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/atacseq 2.1.2 deepTools 3.5.1 image resolved for linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/deeptools:3.5.1--py_0@sha256:5d16e7a95afb816a455df599646ef25335c624b0a0142f4f159d6275a09aa8dc"

// Options controls one coverage profile.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the plot and tabular profile.
type Ports struct {
	PDF   gobble.Handle
	Table gobble.Handle
}

// Add records one validated plotProfile command.
func Add(parent modules.Parent, matrix gobble.Handle, options Options) (Ports, error) {
	const unit = "deeptools_plot_profile"
	matrixPath, err := modules.HandlePath(unit, matrix)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/deeptools")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "coverage"
	}
	pdf := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".plotProfile.pdf"}
	table := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".plotProfile.tsv"}
	pdfPath, err := pdf.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "profile output path is invalid")
	}
	tablePath, _ := table.Render()
	command := []string{"plotProfile", "--matrixFile", matrixPath, "--outFileName", pdfPath, "--outFileNameData", tablePath}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "2g"}, command, []string{"--matrixFile", "-m", "--outFileName", "-out", "--outFileNameData"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "matrix", From: matrix}}, Outputs: []gobble.Bind{{Name: "pdf", Spec: pdf}, {Name: "table", Spec: table}}})
	return Ports{PDF: task.Out("pdf"), Table: task.Out("table")}, nil
}
