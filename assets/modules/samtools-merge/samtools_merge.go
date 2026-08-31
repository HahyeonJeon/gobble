// Package samtoolsmerge owns one samtools merge command.
package samtoolsmerge

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the Sarek 3.10.0 samtools image resolved for linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/htslib_samtools:1.24--d697cfb9dce007cd@sha256:e994bf4eb3731150511a14f5706b7bdfd64df1b6d40898fff334286c027e0859"

// Options controls one samtools merge command. InputPaths is required only for
// a Gather handle whose member paths are resolved at runtime.
type Options struct {
	modules.Options
	OutDir     gobble.Directory
	Prefix     string
	InputPaths []gobble.PathSpec
}

// Ports contains the merged BAM.
type Ports struct{ BAM gobble.Handle }

// Add records one validated merge over explicit indexed BAM handles.
func Add(parent modules.Parent, bams, indexes []gobble.Handle, options Options) (Ports, error) {
	if len(bams) < 2 || len(indexes) != len(bams) {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, "samtools_merge", "at least two BAM inputs are required")
	}
	paths := make([]string, len(bams))
	binds := make([]gobble.Bind, 0, 2*len(bams))
	for i, bam := range bams {
		path, err := modules.HandlePath("samtools_merge", bam)
		if err != nil {
			return Ports{}, err
		}
		paths[i] = path
		if _, err := modules.HandlePath("samtools_merge", indexes[i]); err != nil {
			return Ports{}, err
		}
		binds = append(binds, gobble.Bind{Name: "bam_" + strconv.Itoa(i), From: bam}, gobble.Bind{Name: "index_" + strconv.Itoa(i), From: indexes[i]})
	}
	return add(parent, binds, paths, options)
}

// AddGather records one validated merge over every member of a Gather handle.
func AddGather(parent modules.Parent, parts gobble.Handle, options Options) (Ports, error) {
	if len(options.InputPaths) == 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, "samtools_merge", "gather input paths are empty")
	}
	paths := make([]string, len(options.InputPaths))
	for i, spec := range options.InputPaths {
		path, err := spec.Render()
		if err != nil {
			return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, "samtools_merge", "gather BAM path is invalid")
		}
		paths[i] = path
	}
	return add(parent, []gobble.Bind{{Name: "parts", From: parts}}, paths, options)
}

func add(parent modules.Parent, inputs []gobble.Bind, paths []string, options Options) (Ports, error) {
	const unit = "samtools_merge"
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/samtools-merge")
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
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 2, Memory: "2g"}
	}
	command := []string{"samtools", "merge", "--force", "--output", outputPath}
	if n := modules.ThreadCount(resources.CPU); n > 1 {
		command = append(command, "--threads", strconv.Itoa(n-1))
	}
	protected := []string{"--force", "--output", "--threads", "--write-index"}
	if err := modules.RejectExtraArgPrefixes(unit, options.ExtraArgs, protected); err != nil {
		return Ports{}, err
	}
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, protected)
	if err != nil {
		return Ports{}, err
	}
	command = append(command, paths...)
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{{Name: "bam", Spec: output}}})
	return Ports{BAM: task.Out("bam")}, nil
}

// Pipeline returns a standalone validated samtools merge module.
func Pipeline(bams, indexes []gobble.PathSpec, options Options) *gobble.Pipeline {
	inputs := make([]modules.Input, 0, len(bams)+len(indexes))
	for i, bam := range bams {
		inputs = append(inputs, modules.Input{Name: "bam_" + strconv.Itoa(i), Spec: bam})
	}
	for i, index := range indexes {
		inputs = append(inputs, modules.Input{Name: "index_" + strconv.Itoa(i), Spec: index})
	}
	return modules.StandaloneChecked("samtools-merge", inputs, func(parent modules.Parent, handles []gobble.Handle) error {
		if len(handles) != len(bams)+len(indexes) {
			return modules.ComposeDefect(gobble.DefectInvalidValue, "samtools_merge", "BAM and index input counts differ")
		}
		_, err := Add(parent, handles[:len(bams)], handles[len(bams):], options)
		return err
	})
}
