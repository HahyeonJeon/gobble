// Package homerannotatepeaks owns one HOMER annotatePeaks.pl command.
package homerannotatepeaks

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/atacseq 2.1.2 HOMER 4.11 image resolved for linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/homer:4.11--pl526hc9558a2_3@sha256:69628c89cf46a36838a0bdc2b6c76fba1587d994d07437440636f26d0320b447"

// Options controls one annotatePeaks.pl command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the peak annotation table.
type Ports struct{ Annotation gobble.Handle }

// Add records one validated HOMER annotation command.
func Add(parent modules.Parent, peaks, fasta, gtf gobble.Handle, options Options) (Ports, error) {
	const unit = "homer_annotate_peaks"
	peakPath, err := modules.HandlePath(unit, peaks)
	if err != nil {
		return Ports{}, err
	}
	fastaPath, err := modules.HandlePath(unit, fasta)
	if err != nil {
		return Ports{}, err
	}
	gtfPath, err := modules.HandlePath(unit, gtf)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/peak-annotation")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "peaks"
	}
	annotation := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".annotatePeaks.txt"}
	annotationPath, err := annotation.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "annotation output path is invalid")
	}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 2, Memory: "4g"}
	}
	threads := modules.ThreadCount(resources.CPU)
	if threads < 1 {
		threads = 1
	}
	command := []string{"annotatePeaks.pl", peakPath, fastaPath, "-gtf", gtfPath, "-cpu", strconv.Itoa(threads)}
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"-gtf", "-cpu"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: modules.ShellRedirect(command, annotationPath), Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "peaks", From: peaks}, {Name: "fasta", From: fasta}, {Name: "gtf", From: gtf}}, Outputs: []gobble.Bind{{Name: "annotation", Spec: annotation}}})
	return Ports{Annotation: task.Out("annotation")}, nil
}
