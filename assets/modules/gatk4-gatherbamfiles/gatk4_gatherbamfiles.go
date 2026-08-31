// Package gatk4gatherbamfiles owns one GATK GatherBamFiles command.
package gatk4gatherbamfiles

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// Options controls one GatherBamFiles command.
type Options struct {
	modules.Options
	InputPaths []gobble.PathSpec
	OutDir     gobble.Directory
	Prefix     string
}

// Ports contains the gathered sample BAM.
type Ports struct{ BAM gobble.Handle }

// Add records one validated GatherBamFiles command over a Gather handle.
func Add(parent modules.Parent, parts []gobble.Handle, options Options) (Ports, error) {
	const unit = "gatk4_gatherbamfiles"
	if len(options.InputPaths) == 0 || len(parts) == 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "recalibrated BAM paths are empty")
	}
	outDir, prefix := options.OutDir, options.Prefix
	if outDir.IsZero() {
		outDir = gobble.Dir("work/gatk4-gatherbamfiles")
	}
	if prefix == "" {
		prefix = "recalibrated"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".bam"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "gathered BAM path is invalid")
	}
	protected := []string{"--INPUT", "--OUTPUT", "--TMP_DIR"}
	extra, image, resources, err := modules.ResolveGATK4Options(unit, options.Options, gobble.Resources{CPU: 1, Memory: "4g"}, protected)
	if err != nil {
		return Ports{}, err
	}
	command := []string{"gatk", "GatherBamFiles"}
	for _, spec := range options.InputPaths {
		path, renderErr := spec.Render()
		if renderErr != nil {
			return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "recalibrated BAM input path is invalid")
		}
		command = append(command, "--INPUT", path)
	}
	command = append(command, "--OUTPUT", outputPath, "--TMP_DIR", ".")
	command = append(command, extra...)
	inputs := make([]gobble.Bind, len(parts))
	for i, part := range parts {
		inputs[i] = gobble.Bind{Name: "bams_" + strconv.Itoa(i), From: part}
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{{Name: "bam", Spec: output}}})
	return Ports{BAM: task.Out("bam")}, nil
}

// Pipeline returns a standalone validated GatherBamFiles module.
func Pipeline(bams []gobble.PathSpec, options Options) *gobble.Pipeline {
	inputs := make([]modules.Input, len(bams))
	for i, bam := range bams {
		inputs[i] = modules.Input{Name: "bam_" + strconv.Itoa(i), Spec: bam}
	}
	options.InputPaths = append([]gobble.PathSpec(nil), bams...)
	return modules.StandaloneChecked("gatk4-gatherbamfiles", inputs, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles, options)
		return err
	})
}
