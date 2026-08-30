// Package samtoolsfaidx owns one samtools faidx command.
package samtoolsfaidx

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 samtools image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/htslib_samtools_star_gawk:ae438e9a604351a4@sha256:4a468118dbd7491a69bf9813c68233afa8558d1f3380fd8cab03e0e3d3135190"

// Options controls one samtools faidx command.
type Options struct{ modules.Options }

// Ports contains the FASTA index used as chromosome sizes.
type Ports struct{ FAI gobble.Handle }

// Add records one validated samtools faidx command.
func Add(parent modules.Parent, fasta gobble.Handle, options Options) (Ports, error) {
	const unit = "samtools_faidx"
	fastaPath, err := modules.HandlePath(unit, fasta)
	if err != nil {
		return Ports{}, err
	}
	command := []string{"samtools", "faidx", fastaPath}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "1g"}, command, nil)
	if err != nil {
		return Ports{}, err
	}
	fai := fasta.Spec().AppendExt(".fai")
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "fasta", From: fasta}}, Outputs: []gobble.Bind{{Name: "fai", Spec: fai}}})
	return Ports{FAI: task.Out("fai")}, nil
}

// Pipeline returns a standalone validated samtools faidx module.
func Pipeline(fasta gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("samtools-faidx", []modules.Input{{Name: "fasta", Spec: fasta}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], options)
		return err
	})
}
