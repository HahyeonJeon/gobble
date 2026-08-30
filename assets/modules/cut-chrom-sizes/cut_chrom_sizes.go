// Package cutchromsizes owns one cut command that projects FASTA-index
// chromosome names and lengths for UCSC coverage tools.
package cutchromsizes

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 coreutils image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/coreutils_grep_gzip_lbzip2_pruned:838ba80435a629f8@sha256:63c2c6b22e83b2f656e88fbb1553e595da4e9e58794e3bfcb98b20b3837f328a"

// Options controls one chromosome-size projection command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
}

// Ports contains the two-column chromosome-size file.
type Ports struct{ Sizes gobble.Handle }

// Add records one validated cut command.
func Add(parent modules.Parent, fai gobble.Handle, options Options) (Ports, error) {
	const unit = "cut_chrom_sizes"
	faiPath, err := modules.HandlePath(unit, fai)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/reference")
	}
	sizes := gobble.PathSpec{Dir: outDir, Base: "chrom", Ext: ".sizes"}
	sizesPath, err := sizes.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "chromosome-size output path is invalid")
	}
	command := []string{"cut", "-f1,2", faiPath}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "256m"}, command, []string{"-f"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: modules.ShellRedirect(command, sizesPath), Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "fai", From: fai}}, Outputs: []gobble.Bind{{Name: "sizes", Spec: sizes}}})
	return Ports{Sizes: task.Out("sizes")}, nil
}

// Pipeline returns a standalone validated chromosome-size projection module.
func Pipeline(fai gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("cut-chrom-sizes", []modules.Input{{Name: "fai", Spec: fai}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], options)
		return err
	})
}
