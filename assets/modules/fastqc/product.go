package fastqc

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 FastQC image resolved for
// linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/fastqc:0.12.1--hdfd78af_0@sha256:e194048df39c3145d9b4e0a14f4da20b59d59250465b6f2a9cb698445fd45900"

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
	html := gobble.PathSpec{Dir: outDir, Base: fastqcStem(reads.Spec()), Ext: ".html"}
	zip := gobble.PathSpec{Dir: outDir, Base: fastqcStem(reads.Spec()), Ext: ".zip"}
	command := []string{"fastqc", "--outdir", outDir.String(), "--noextract"}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 2, Memory: "1g"}
	}
	if n := modules.ThreadCount(resources.CPU); n > 0 {
		command = append(command, "--threads", strconv.Itoa(n))
	}
	command = append(command, readPath)
	base := options.Options
	base.Resources = resources
	if err := modules.RejectExtraArgs(unit, options.ExtraArgs, []string{"--extract", "--version", "--help"}); err != nil {
		return Ports{}, err
	}
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"--outdir", "--noextract", "--threads"})
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
