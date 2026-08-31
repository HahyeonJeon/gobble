package bwaindex

import (
	"strings"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the Sarek 3.10.0 BWA image resolved for linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/bwa_htslib_samtools:83b50ff84ead50d0@sha256:d7e24dc1e4d93ca4d3a76a78d4c834a7be3985b0e1e56fddd61662e047863a8a"

// Options controls one lifted bwa index command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the fixed BWA sidecar Group and its command prefix.
type Ports struct {
	Index  gobble.Handle
	Prefix gobble.PathSpec
}

// Add records one validated bwa index command.
func Add(parent modules.Parent, fasta gobble.Handle, options Options) (Ports, error) {
	const unit = "bwa_index"
	fastaPath, err := modules.HandlePath(unit, fasta)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/reference/bwa")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "genome"
	}
	prefixSpec := gobble.PathSpec{Dir: outDir, Base: prefix}
	prefixPath, err := prefixSpec.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "BWA index prefix is invalid")
	}
	command := []string{"bwa", "index", "-p", prefixPath}
	if err := modules.RejectExtraArgs(unit, options.ExtraArgs, []string{"-p", "-a"}); err != nil {
		return Ports{}, err
	}
	if err := validateExtraArgs(unit, options.ExtraArgs); err != nil {
		return Ports{}, err
	}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "6g"}, command, []string{"-p"})
	if err != nil {
		return Ports{}, err
	}
	command = append(command, fastaPath)
	members := make(gobble.Group, 0, len(bwaIndexMemberNames))
	for _, name := range bwaIndexMemberNames {
		members = append(members, gobble.Member{Name: name, Spec: prefixSpec.AppendExt("." + name)})
	}
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Command: command, Image: image, Resources: resources,
		Inputs:  []gobble.Bind{{Name: "fasta", From: fasta}},
		Outputs: []gobble.Bind{{Name: "index", Group: members}},
	})
	return Ports{Index: task.Out("index"), Prefix: prefixSpec}, nil
}

func validateExtraArgs(unit string, extraArgs []string) error {
	for _, arg := range extraArgs {
		if arg == "" || arg == "-" || arg == "--" || !strings.HasPrefix(arg, "-") {
			return modules.ComposeDefect(gobble.DefectInvalidValue, unit, "ExtraArgs cannot add positional operands before the typed FASTA input")
		}
	}
	return nil
}

// ProductPipeline returns a standalone validated lifted bwa index module.
func ProductPipeline(fasta gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("bwa-index", []modules.Input{{Name: "fasta", Spec: fasta}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], options)
		return err
	})
}
