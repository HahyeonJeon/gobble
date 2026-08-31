// Package samtoolsview owns one samtools view filtering command.
package samtoolsview

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/atacseq 2.1.2 samtools 1.17 image resolved for linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/samtools:1.17--h00cdaf9_0@sha256:6f88956b747a67b2a39a3ff72c4de30e665239ee11db610624dd4298e30db1bf"

// Options controls mapping, duplicate, orphan, and mitochondrial filtering.
type Options struct {
	modules.Options
	OutDir       gobble.Directory
	Prefix       string
	MinimumMAPQ  int
	Paired       bool
	RemoveOrphan bool
	RemoveMito   bool
	MitoName     string
}

// Ports contains the filtered BAM.
type Ports struct{ BAM gobble.Handle }

// Add records one validated samtools view command.
func Add(parent modules.Parent, bam gobble.Handle, options Options) (Ports, error) {
	const unit = "samtools_view"
	bamPath, err := modules.HandlePath(unit, bam)
	if err != nil {
		return Ports{}, err
	}
	if options.MinimumMAPQ < 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "minimum MAPQ must not be negative")
	}
	if options.RemoveMito && options.MitoName == "" {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "mitochondrial contig is required when mitochondrial filtering is enabled")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/samtools-view")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "filtered"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".bam"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "filtered BAM path is invalid")
	}
	// Exclude unmapped, secondary, QC-failed, duplicate, and supplementary reads.
	command := []string{"samtools", "view", "-b", "-F", "3844", "-q", strconv.Itoa(options.MinimumMAPQ), "-o", outputPath}
	if options.Paired && options.RemoveOrphan {
		command = append(command, "-f", "2")
	}
	if options.RemoveMito {
		command = append(command, "-e", "rname != "+strconv.Quote(options.MitoName))
	}
	command = append(command, bamPath)
	protected := []string{"-b", "-F", "-q", "-o", "-f", "-e", "-h", "-H", "-c"}
	if err := modules.RejectExtraArgs(unit, options.ExtraArgs, protected); err != nil {
		return Ports{}, err
	}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 2, Memory: "2g"}, command, protected)
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bam", From: bam}}, Outputs: []gobble.Bind{{Name: "filtered_bam", Spec: output}}})
	return Ports{BAM: task.Out("filtered_bam")}, nil
}
