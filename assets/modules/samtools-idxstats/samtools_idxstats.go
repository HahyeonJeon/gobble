// Package samtoolsidxstats owns one samtools idxstats command.
package samtoolsidxstats

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the Sarek 3.10.0 samtools image resolved for linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/htslib_samtools:1.24--d697cfb9dce007cd@sha256:a55ddea590e567a91df592300a960aa534cfc1bd16e7623e3938ec21f4f3df15"

// Options controls one samtools idxstats command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the index statistics report.
type Ports struct{ Report gobble.Handle }

// Add records one validated samtools idxstats command.
func Add(parent modules.Parent, bam, bai gobble.Handle, options Options) (Ports, error) {
	const unit = "samtools_idxstats"
	bamPath, err := modules.HandlePath(unit, bam)
	if err != nil {
		return Ports{}, err
	}
	if _, err = modules.HandlePath(unit, bai); err != nil {
		return Ports{}, err
	}
	outDir, prefix := options.OutDir, options.Prefix
	if outDir.IsZero() {
		outDir = gobble.Dir("work/samtools-idxstats")
	}
	if prefix == "" {
		prefix = "alignment"
	}
	report := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".idxstats.txt"}
	reportPath, err := report.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "idxstats report path is invalid")
	}
	if err := modules.RejectExtraArgs(unit, options.ExtraArgs, []string{"-X"}); err != nil {
		return Ports{}, err
	}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "1g"}, []string{"samtools", "idxstats"}, nil)
	if err != nil {
		return Ports{}, err
	}
	command = append(command, bamPath)
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: modules.ShellRedirect(command, reportPath), Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bam", From: bam}, {Name: "bai", From: bai}}, Outputs: []gobble.Bind{{Name: "report", Spec: report}}})
	return Ports{Report: task.Out("report")}, nil
}

// Pipeline returns a standalone validated samtools idxstats module.
func Pipeline(bam, bai gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("samtools-idxstats", []modules.Input{{Name: "bam", Spec: bam}, {Name: "bai", Spec: bai}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], options)
		return err
	})
}
