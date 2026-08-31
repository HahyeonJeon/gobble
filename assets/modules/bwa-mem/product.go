package bwamem

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the Sarek 3.10.0 BWA image resolved for linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/bwa_htslib_samtools:83b50ff84ead50d0@sha256:d7e24dc1e4d93ca4d3a76a78d4c834a7be3985b0e1e56fddd61662e047863a8a"

// Options controls one lifted read-grouped bwa mem command.
type Options struct {
	modules.Options
	IndexPrefix gobble.PathSpec
	ReadGroup   string
	OutDir      gobble.Directory
	Prefix      string
}

// Ports contains the lane SAM emitted by bwa mem.
type Ports struct{ SAM gobble.Handle }

// Add records one validated read-grouped bwa mem command. A zero read2 selects
// the command's supported single-end form.
func Add(parent modules.Parent, fasta, index, read1, read2 gobble.Handle, options Options) (Ports, error) {
	const unit = "bwa_mem"
	if _, err := modules.HandlePath(unit, fasta); err != nil {
		return Ports{}, err
	}
	read1Path, err := modules.HandlePath(unit, read1)
	if err != nil {
		return Ports{}, err
	}
	var read2Path string
	if !read2.IsZero() {
		read2Path, err = modules.HandlePath(unit, read2)
		if err != nil {
			return Ports{}, err
		}
	}
	indexPath, err := options.IndexPrefix.Render()
	if err != nil || indexPath == "" {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "BWA index prefix is invalid")
	}
	if options.ReadGroup == "" {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "read group is empty")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/bwa-mem")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "aligned"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".sam"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "BWA MEM output is invalid")
	}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 4, Memory: "4g"}
	}
	command := []string{"bwa", "mem", "-R", options.ReadGroup}
	if n := modules.ThreadCount(resources.CPU); n > 0 {
		command = append(command, "-t", strconv.Itoa(n))
	}
	if err := modules.RejectExtraArgs(unit, options.ExtraArgs, []string{"-R", "-t", "-o"}); err != nil {
		return Ports{}, err
	}
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"-R", "-t"})
	if err != nil {
		return Ports{}, err
	}
	command = append(command, indexPath, read1Path)
	if read2Path != "" {
		command = append(command, read2Path)
	}
	inputs := []gobble.Bind{
		{Name: "fasta", From: fasta},
		{Name: "index", From: index, Group: inputIndexGroup()},
		{Name: "read1", From: read1},
	}
	if !read2.IsZero() {
		inputs = append(inputs, gobble.Bind{Name: "read2", From: read2})
	}
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Script: modules.ShellRedirect(command, outputPath), Image: image, Resources: resources,
		Inputs:  inputs,
		Outputs: []gobble.Bind{{Name: "sam", Spec: output}},
	})
	return Ports{SAM: task.Out("sam")}, nil
}

// ProductPipeline returns a standalone validated lifted bwa mem module.
func ProductPipeline(fasta gobble.PathSpec, index gobble.Group, read1, read2 gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("bwa-mem", []modules.Input{{Name: "fasta", Spec: fasta}, {Name: "index", Group: index}, {Name: "read1", Spec: read1}, {Name: "read2", Spec: read2}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], handles[2], handles[3], options)
		return err
	})
}
