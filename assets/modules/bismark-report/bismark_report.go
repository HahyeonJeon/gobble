// Package bismarkreport owns one Bismark bismark2report command.
package bismarkreport

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/methylseq 4.2.0 Bismark image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/bismark:0.25.1--1f50935de5d79c47@sha256:7b49e02b15de6fd59643224db5defb229433de4aebee982d6a03b612077755a0"

// Options controls one Bismark per-sample HTML report.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the generated Bismark HTML report.
type Ports struct{ HTML gobble.Handle }

// Add records one bismark2report command with every report input explicit.
func Add(parent modules.Parent, alignment, deduplication, splitting, mbias gobble.Handle, options Options) (Ports, error) {
	const unit = "bismark_report"
	paths := make([]string, 4)
	for i, handle := range []gobble.Handle{alignment, deduplication, splitting, mbias} {
		path, err := modules.HandlePath(unit, handle)
		if err != nil {
			return Ports{}, err
		}
		paths[i] = path
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/methylseq/reports")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "sample"
	}
	filename := prefix + ".bismark_report.html"
	output := gobble.Literal(filename).WithDir(outDir)
	if _, err := output.Render(); err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "Bismark report output path is invalid")
	}
	protected := []string{
		"--nucleotide_report", "--version", "--help",
		"--alignment_report", "--dedup_report", "--splitting_report", "--mbias_report", "--dir", "-o", "--output",
	}
	if err := modules.RejectExtraArgPrefixes(unit, options.ExtraArgs, protected); err != nil {
		return Ports{}, err
	}
	command := []string{"bismark2report", "--alignment_report", paths[0], "--dedup_report", paths[1], "--splitting_report", paths[2], "--mbias_report", paths[3], "--dir", outDir.String(), "--output", filename}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "2g"}, command, []string{"--alignment_report", "--dedup_report", "--splitting_report", "--mbias_report", "--nucleotide_report", "--dir", "-o", "--output"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "alignment_report", From: alignment}, {Name: "deduplication_report", From: deduplication}, {Name: "splitting_report", From: splitting}, {Name: "mbias_report", From: mbias}}, Outputs: []gobble.Bind{{Name: "html", Spec: output}}})
	return Ports{HTML: task.Out("html")}, nil
}

// Pipeline returns a standalone validated Bismark report module.
func Pipeline(alignment, deduplication, splitting, mbias gobble.PathSpec, options Options) *gobble.Pipeline {
	inputs := []modules.Input{{Name: "alignment_report", Spec: alignment}, {Name: "deduplication_report", Spec: deduplication}, {Name: "splitting_report", Spec: splitting}, {Name: "mbias_report", Spec: mbias}}
	return modules.StandaloneChecked("bismark-report", inputs, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], handles[2], handles[3], options)
		return err
	})
}
