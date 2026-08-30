// Package samtoolsstats owns one samtools stats command.
package samtoolsstats

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 samtools image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/htslib_samtools_star_gawk:ae438e9a604351a4@sha256:4a468118dbd7491a69bf9813c68233afa8558d1f3380fd8cab03e0e3d3135190"

// Options controls one samtools stats command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the retained alignment statistics.
type Ports struct{ Stats gobble.Handle }

// Add records one validated samtools stats command.
func Add(parent modules.Parent, bam gobble.Handle, options Options) (Ports, error) {
	const unit = "samtools_stats"
	bamPath, err := modules.HandlePath(unit, bam)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/samtools-stats")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "alignment"
	}
	stats := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".stats.txt"}
	statsPath, _ := stats.Render()
	command := []string{"samtools", "stats", bamPath}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "1g"}, command, nil)
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: modules.ShellRedirect(command, statsPath), Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bam", From: bam}}, Outputs: []gobble.Bind{{Name: "stats", Spec: stats}}})
	return Ports{Stats: task.Out("stats")}, nil
}

// Pipeline returns a standalone validated samtools stats module.
func Pipeline(bam gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("samtools-stats", []modules.Input{{Name: "bam", Spec: bam}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], options)
		return err
	})
}
