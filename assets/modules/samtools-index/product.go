package samtoolsindex

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 samtools image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/htslib_samtools_star_gawk:ae438e9a604351a4@sha256:4a468118dbd7491a69bf9813c68233afa8558d1f3380fd8cab03e0e3d3135190"

// Options controls one lifted samtools index command.
type Options struct{ modules.Options }

// Ports contains the BAI companion file.
type Ports struct{ BAI gobble.Handle }

// Add records one validated samtools index command.
func Add(parent modules.Parent, bam gobble.Handle, options Options) (Ports, error) {
	const unit = "samtools_index"
	bamPath, err := modules.HandlePath(unit, bam)
	if err != nil {
		return Ports{}, err
	}
	bai := bam.Spec().AppendExt(".bai")
	baiPath, err := bai.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "index output path is invalid")
	}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 1, Memory: "1g"}
	}
	command := []string{"samtools", "index"}
	if n := modules.ThreadCount(resources.CPU); n > 0 {
		command = append(command, "-@", strconv.Itoa(n))
	}
	command = append(command, bamPath, baiPath)
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"-@"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bam", From: bam}}, Outputs: []gobble.Bind{{Name: "bai", Spec: bai}}})
	return Ports{BAI: task.Out("bai")}, nil
}

// ProductPipeline returns a standalone validated lifted samtools index module.
func ProductPipeline(bam gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("samtools-index", []modules.Input{{Name: "bam", Spec: bam}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], options)
		return err
	})
}
