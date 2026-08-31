// Package picardmergesamfiles owns one Picard MergeSamFiles command.
package picardmergesamfiles

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/atacseq 2.1.2 Picard 3.0.0 image resolved for linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/picard:3.0.0--hdfd78af_1@sha256:1807618ee8ac1af18a2a4656dd8b4d4a0a6f679b6a1e554a6603ac7a6d732d95"

// Options controls one MergeSamFiles command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the merged BAM.
type Ports struct{ BAM gobble.Handle }

// Add records one strict MergeSamFiles fan-in. Every BAM is a required input.
func Add(parent modules.Parent, bams []gobble.Handle, options Options) (Ports, error) {
	const unit = "picard_merge_sam_files"
	if len(bams) == 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "BAM membership must not be empty")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/picard-merge-sam-files")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "merged"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".bam"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "merged BAM path is invalid")
	}
	command := []string{"picard", "MergeSamFiles"}
	inputs := make([]gobble.Bind, len(bams))
	for i, bam := range bams {
		bamPath, pathErr := modules.HandlePath(unit, bam)
		if pathErr != nil {
			return Ports{}, pathErr
		}
		command = append(command, "--INPUT", bamPath)
		inputs[i] = gobble.Bind{Name: "bam_" + strconv.Itoa(i), From: bam}
	}
	command = append(command, "--OUTPUT", outputPath)
	protected := []string{"--INPUT", "--OUTPUT", "INPUT", "OUTPUT", "I", "O"}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 2, Memory: "4g"}, command, protected)
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{{Name: "bam", Spec: output}}})
	return Ports{BAM: task.Out("bam")}, nil
}
