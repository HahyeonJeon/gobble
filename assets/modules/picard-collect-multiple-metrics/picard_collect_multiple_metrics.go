// Package picardcollectmultiplemetrics owns one Picard CollectMultipleMetrics command.
package picardcollectmultiplemetrics

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/atacseq 2.1.2 Picard 3.0.0 image resolved for linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/picard:3.0.0--hdfd78af_1@sha256:1807618ee8ac1af18a2a4656dd8b4d4a0a6f679b6a1e554a6603ac7a6d732d95"

// Options controls one CollectMultipleMetrics command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the complete directory of Picard metric and plot artifacts.
type Ports struct{ Metrics gobble.Handle }

// Add records one validated CollectMultipleMetrics command.
func Add(parent modules.Parent, bam, bai, fasta, fai gobble.Handle, options Options) (Ports, error) {
	const unit = "picard_collect_multiple_metrics"
	bamPath, err := modules.HandlePath(unit, bam)
	if err != nil {
		return Ports{}, err
	}
	for _, handle := range []gobble.Handle{bai, fai} {
		if _, err = modules.HandlePath(unit, handle); err != nil {
			return Ports{}, err
		}
	}
	fastaPath, err := modules.HandlePath(unit, fasta)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/picard-collect-multiple-metrics")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "alignment"
	}
	outputPrefix := outDir.String() + "/" + prefix + ".CollectMultipleMetrics"
	command := []string{"picard", "CollectMultipleMetrics", "--INPUT", bamPath, "--OUTPUT", outputPrefix, "--REFERENCE_SEQUENCE", fastaPath}
	protected := []string{"--INPUT", "--OUTPUT", "--REFERENCE_SEQUENCE", "INPUT", "OUTPUT", "REFERENCE_SEQUENCE", "I", "O", "R"}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "4g"}, command, protected)
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Command: command, Image: image, Resources: resources,
		Inputs:  []gobble.Bind{{Name: "bam", From: bam}, {Name: "bai", From: bai}, {Name: "fasta", From: fasta}, {Name: "fai", From: fai}},
		Outputs: []gobble.Bind{{Name: "metrics", Tree: gobble.DeclareTree(outDir)}},
	})
	return Ports{Metrics: task.Out("metrics")}, nil
}
