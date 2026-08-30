package bismarkgenome

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/methylseq 4.2.0 Bismark image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/bismark:0.25.1--1f50935de5d79c47@sha256:7b49e02b15de6fd59643224db5defb229433de4aebee982d6a03b612077755a0"

// Options controls one directional Bowtie2 Bismark genome-preparation command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
}

// Ports contains the complete consumed Bismark genome directory as a Tree.
// The Tree includes the staged FASTA and every file Bismark creates below it.
type Ports struct{ Index gobble.Handle }

// Add records one validated bismark_genome_preparation command.
func Add(parent modules.Parent, fasta gobble.Handle, options Options) (Ports, error) {
	const unit = "bismark_genome_preparation"
	if _, err := modules.HandlePath(unit, fasta); err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/bismark-index")
	}
	if err := modules.RejectExtraArgs(unit, options.ExtraArgs, []string{"--hisat2", "--slam", "--version", "--help"}); err != nil {
		return Ports{}, err
	}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 4, Memory: "15g"}
	}
	command := []string{"bismark_genome_preparation", "--bowtie2"}
	if n := modules.ThreadCount(resources.CPU); n >= 2 {
		command = append(command, "--parallel", strconv.Itoa(n))
	}
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"--bowtie2", "--parallel"})
	if err != nil {
		return Ports{}, err
	}
	command = append(command, outDir.String())
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Command: command, Image: image, Resources: resources,
		Inputs:  []gobble.Bind{{Name: "fasta", From: fasta, Spec: gobble.PathSpec{Dir: outDir}}},
		Outputs: []gobble.Bind{{Name: "index", Tree: gobble.DeclareTree(outDir)}},
	})
	return Ports{Index: task.Out("index")}, nil
}

// Pipeline returns a standalone validated Bismark genome-preparation module.
func Pipeline(fasta gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("bismark-genome-preparation", []modules.Input{{Name: "fasta", Spec: fasta}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], options)
		return err
	})
}
