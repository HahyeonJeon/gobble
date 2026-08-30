package samtoolssort

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 samtools image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/htslib_samtools_star_gawk:ae438e9a604351a4@sha256:4a468118dbd7491a69bf9813c68233afa8558d1f3380fd8cab03e0e3d3135190"

// Options controls one lifted samtools sort command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the coordinate-sorted BAM.
type Ports struct{ BAM gobble.Handle }

// Add records one validated samtools sort command.
func Add(parent modules.Parent, alignment gobble.Handle, options Options) (Ports, error) {
	return add(parent, alignment, gobble.Handle{}, options)
}

// AddAfter records samtools sort after a sample policy gate.
func AddAfter(parent modules.Parent, alignment, after gobble.Handle, options Options) (Ports, error) {
	return add(parent, alignment, after, options)
}

func add(parent modules.Parent, alignment, after gobble.Handle, options Options) (Ports, error) {
	const unit = "samtools_sort"
	inputPath, err := modules.HandlePath(unit, alignment)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/samtools-sort")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "aligned"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".bam"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "sort output path is invalid")
	}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 2, Memory: "2g"}
	}
	command := []string{"samtools", "sort", "-o", outputPath}
	if n := modules.ThreadCount(resources.CPU); n > 0 {
		command = append(command, "-@", strconv.Itoa(n))
	}
	command = append(command, inputPath)
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"-o", "-@"})
	if err != nil {
		return Ports{}, err
	}
	inputs := []gobble.Bind{{Name: "alignment", From: alignment}}
	if !after.IsZero() {
		inputs = append(inputs, gobble.Bind{Name: "sample_accepted", From: after})
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{{Name: "bam", Spec: output}}})
	return Ports{BAM: task.Out("bam")}, nil
}

// ProductPipeline returns a standalone validated lifted samtools sort module.
func ProductPipeline(alignment gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("samtools-sort", []modules.Input{{Name: "alignment", Spec: alignment}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], options)
		return err
	})
}
