// Package gatk4mergevcfs owns one GATK MergeVcfs command.
package gatk4mergevcfs

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// Options controls one MergeVcfs command over a complete Gather membership.
type Options struct {
	modules.Options
	InputPaths []gobble.PathSpec
	OutDir     gobble.Directory
	Prefix     string
}

// Ports contains one indexed merged VCF or gVCF.
type Ports struct {
	VCF gobble.Handle
	TBI gobble.Handle
}

// Add records one validated MergeVcfs command over a Gather handle.
func Add(parent modules.Parent, parts, indexes []gobble.Handle, dict gobble.Handle, options Options) (Ports, error) {
	const unit = "gatk4_mergevcfs"
	if len(options.InputPaths) == 0 || len(parts) == 0 || len(indexes) != len(parts) {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "VCF input paths are empty")
	}
	dictPath, err := modules.HandlePath(unit, dict)
	if err != nil {
		return Ports{}, err
	}
	outDir, prefix := options.OutDir, options.Prefix
	if outDir.IsZero() {
		outDir = gobble.Dir("work/gatk4-mergevcfs")
	}
	if prefix == "" {
		prefix = "merged"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".vcf.gz"}
	tbi := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".vcf.gz.tbi"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "merged VCF path is invalid")
	}
	protected := []string{"--INPUT", "--OUTPUT", "--SEQUENCE_DICTIONARY", "--TMP_DIR", "--CREATE_INDEX"}
	extra, image, resources, err := modules.ResolveGATK4Options(unit, options.Options, gobble.Resources{CPU: 1, Memory: "4g"}, protected)
	if err != nil {
		return Ports{}, err
	}
	command := []string{"gatk", "MergeVcfs"}
	for _, spec := range options.InputPaths {
		path, renderErr := spec.Render()
		if renderErr != nil {
			return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "VCF input path is invalid")
		}
		command = append(command, "--INPUT", path)
	}
	command = append(command, "--OUTPUT", outputPath, "--SEQUENCE_DICTIONARY", dictPath, "--CREATE_INDEX", "true", "--TMP_DIR", ".")
	command = append(command, extra...)
	inputs := []gobble.Bind{{Name: "dict", From: dict}}
	for i := range parts {
		inputs = append(inputs, gobble.Bind{Name: "vcfs_" + strconv.Itoa(i), From: parts[i]}, gobble.Bind{Name: "indexes_" + strconv.Itoa(i), From: indexes[i]})
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{{Name: "vcf", Spec: output}, {Name: "tbi", Spec: tbi}}})
	return Ports{VCF: task.Out("vcf"), TBI: task.Out("tbi")}, nil
}

// Pipeline returns a standalone validated MergeVcfs module.
func Pipeline(vcfs, indexes []gobble.PathSpec, dict gobble.PathSpec, options Options) *gobble.Pipeline {
	inputs := []modules.Input{{Name: "dict", Spec: dict}}
	for i := range vcfs {
		inputs = append(inputs, modules.Input{Name: "vcf_" + strconv.Itoa(i), Spec: vcfs[i]})
		if i < len(indexes) {
			inputs = append(inputs, modules.Input{Name: "index_" + strconv.Itoa(i), Spec: indexes[i]})
		}
	}
	options.InputPaths = append([]gobble.PathSpec(nil), vcfs...)
	return modules.StandaloneChecked("gatk4-mergevcfs", inputs, func(parent modules.Parent, handles []gobble.Handle) error {
		if len(handles) != 1+2*len(vcfs) {
			return modules.ComposeDefect(gobble.DefectInvalidValue, "gatk4_mergevcfs", "VCF and index input counts differ")
		}
		vcfHandles := make([]gobble.Handle, len(vcfs))
		indexHandles := make([]gobble.Handle, len(vcfs))
		for i := range vcfs {
			vcfHandles[i], indexHandles[i] = handles[1+2*i], handles[2+2*i]
		}
		_, err := Add(parent, vcfHandles, indexHandles, handles[0], options)
		return err
	})
}
