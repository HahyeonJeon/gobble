// Package mosdepth owns one mosdepth command.
package mosdepth

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the Sarek 3.10.0 mosdepth image resolved for linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/htslib_mosdepth_gzip:4108dd38be84e40a@sha256:426ef2ec8c93bc292862c2902a91f193ae336bdb84750581637d35f59017d2bc"

// Options controls one mosdepth command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains required whole-alignment coverage reports.
type Ports struct {
	Summary gobble.Handle
	Global  gobble.Handle
}

// Add records one validated mosdepth command.
func Add(parent modules.Parent, bam, bai, fasta gobble.Handle, options Options) (Ports, error) {
	const unit = "mosdepth"
	bamPath, err := modules.HandlePath(unit, bam)
	if err != nil {
		return Ports{}, err
	}
	if _, err = modules.HandlePath(unit, bai); err != nil {
		return Ports{}, err
	}
	fastaPath, err := modules.HandlePath(unit, fasta)
	if err != nil {
		return Ports{}, err
	}
	outDir, prefix := options.OutDir, options.Prefix
	if outDir.IsZero() {
		outDir = gobble.Dir("work/mosdepth")
	}
	if prefix == "" {
		prefix = "coverage"
	}
	prefixSpec := gobble.PathSpec{Dir: outDir, Base: prefix}
	prefixPath, err := prefixSpec.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "mosdepth prefix is invalid")
	}
	summary := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".mosdepth.summary.txt"}
	global := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".mosdepth.global.dist.txt"}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 2, Memory: "2g"}
	}
	command := []string{"mosdepth", "--fasta", fastaPath}
	if n := modules.ThreadCount(resources.CPU); n > 0 {
		command = append(command, "--threads", strconv.Itoa(n))
	}
	protected := []string{"--fasta", "--threads", "--by"}
	if err := modules.RejectExtraArgPrefixes(unit, options.ExtraArgs, protected); err != nil {
		return Ports{}, err
	}
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, protected)
	if err != nil {
		return Ports{}, err
	}
	command = append(command, prefixPath, bamPath)
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bam", From: bam}, {Name: "bai", From: bai}, {Name: "fasta", From: fasta}}, Outputs: []gobble.Bind{{Name: "summary", Spec: summary}, {Name: "global", Spec: global}}})
	return Ports{Summary: task.Out("summary"), Global: task.Out("global")}, nil
}

// Pipeline returns a standalone validated mosdepth module.
func Pipeline(bam, bai, fasta gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("mosdepth", []modules.Input{{Name: "bam", Spec: bam}, {Name: "bai", Spec: bai}, {Name: "fasta", Spec: fasta}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], handles[2], options)
		return err
	})
}
