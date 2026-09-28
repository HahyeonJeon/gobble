// Package fastqc owns the validated FastQC command module.
package fastqc

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	"github.com/HahyeonJeon/gobble/internal/modulecommand"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 FastQC image resolved for
// linux/amd64.
const DefaultImage modules.Image = modulecommand.FastQCImage

var protectedExtraArgs = []string{
	"--outdir", "-o", "--threads", "--noextract", "--extract",
	"--adapters", "-a", "--contaminants", "-c", "--limits", "-l", "--dir", "-d",
	"--version", "--help",
}

// Options controls one lifted FastQC command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
}

// Ports are the FastQC report files.
type Ports struct {
	HTML gobble.Handle
	Zip  gobble.Handle
}

// ProtectedExtraArg returns the first FastQC option that competes with the
// module-owned input, outputs, resources, or command behavior.
func ProtectedExtraArg(extraArgs []string) string {
	return modules.MatchProtectedExtraArg(extraArgs, protectedExtraArgs)
}

// Add records one validated FastQC command.
func Add(parent modules.Parent, reads gobble.Handle, options Options) (Ports, error) {
	const unit = "fastqc"
	readPath, err := modules.HandlePath(unit, reads)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/fastqc")
	}
	stem := fastqcStem(readPath)
	html := gobble.PathSpec{Dir: outDir, Base: stem, Ext: ".html"}
	zip := gobble.PathSpec{Dir: outDir, Base: stem, Ext: ".zip"}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 2, Memory: "1g"}
	}
	command := modulecommand.FastQC(readPath, outDir.String(), modules.ThreadCount(resources.CPU))
	base := options.Options
	base.Resources = resources
	if err := modules.RejectExtraArgPrefixes(unit, options.ExtraArgs, protectedExtraArgs); err != nil {
		return Ports{}, err
	}
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, protectedExtraArgs)
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Command: command, Image: image, Resources: resources,
		Inputs:  []gobble.Bind{{Name: "reads", From: reads}},
		Outputs: []gobble.Bind{{Name: "html", Spec: html}, {Name: "zip", Spec: zip}},
	})
	return Ports{HTML: task.Out("html"), Zip: task.Out("zip")}, nil
}

// Pipeline returns a standalone validated FastQC module.
func Pipeline(reads gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("fastqc", []modules.Input{{Name: "reads", Spec: reads}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], options)
		return err
	})
}

func fastqcStem(readPath string) string { return modulecommand.FastQCStem(readPath) }
