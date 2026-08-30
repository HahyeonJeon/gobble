// Package picardmarkduplicates owns one Picard MarkDuplicates command.
package picardmarkduplicates

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 Picard image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/picard:3.4.0--e9963040df0a9bf6@sha256:e269216786463d44f9d83a0d6e877b34bca2c7b4d35211b4b369fe98e39ef1a5"

// Options controls one MarkDuplicates command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports are the duplicate-marked BAM and metrics.
type Ports struct {
	BAM     gobble.Handle
	Metrics gobble.Handle
}

// Add records one validated Picard MarkDuplicates command.
func Add(parent modules.Parent, bam gobble.Handle, options Options) (Ports, error) {
	const unit = "picard_markduplicates"
	bamPath, err := modules.HandlePath(unit, bam)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/picard-markduplicates")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "marked"
	}
	marked := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".bam"}
	metrics := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".metrics.txt"}
	markedPath, err := marked.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "marked BAM path is invalid")
	}
	metricsPath, _ := metrics.Render()
	command := []string{"picard", "MarkDuplicates", "--INPUT", bamPath, "--OUTPUT", markedPath, "--METRICS_FILE", metricsPath}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 2, Memory: "4g"}, command, []string{"--INPUT", "--OUTPUT", "--METRICS_FILE"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bam", From: bam}}, Outputs: []gobble.Bind{{Name: "marked_bam", Spec: marked}, {Name: "metrics", Spec: metrics}}})
	return Ports{BAM: task.Out("marked_bam"), Metrics: task.Out("metrics")}, nil
}

// Pipeline returns a standalone validated MarkDuplicates module.
func Pipeline(bam gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("picard-markduplicates", []modules.Input{{Name: "bam", Spec: bam}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], options)
		return err
	})
}
