// Package bcftoolssort owns one bcftools sort command.
package bcftoolssort

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the Sarek 3.10.0 bcftools image resolved for linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/bcftools_htslib:1.23.1--9f08ec665533d64a@sha256:9c0dea3c6d34d771912b4a2b0297d3e796db69d3c06620cb93b8ab41c072c613"

// Options controls one interval-scattered bcftools sort command.
type Options struct {
	modules.Options
	IntervalDir gobble.Directory
	InputDir    gobble.Directory
	OutDir      gobble.Directory
}

// Ports contains one indexed sorted interval VCF.
type Ports struct {
	VCF gobble.Handle
	TBI gobble.Handle
}

// Add records one validated interval-scattered bcftools sort command.
func Add(parent modules.Parent, vcf, tbi, interval gobble.Handle, options Options) (Ports, error) {
	const unit = "bcftools_sort"
	if options.InputDir.IsZero() {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "input VCF directory is empty")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/bcftools-sort")
	}
	prelude, err := modules.ScatterFilePrelude(unit, options.IntervalDir)
	if err != nil {
		return Ports{}, err
	}
	protected := []string{"--output", "--output-type", "--write-index", "--temp-dir", "--max-mem"}
	if err := modules.RejectExtraArgPrefixes(unit, options.ExtraArgs, protected); err != nil {
		return Ports{}, err
	}
	extra, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "2g"}, nil, protected)
	if err != nil {
		return Ports{}, err
	}
	command := []string{"bcftools", "sort", "--output-type", "z", "--write-index=tbi", "--temp-dir", "."}
	script := prelude +
		"input=" + modules.ShellQuote(options.InputDir.String()) + "/$stem.vcf.gz\n" +
		"output=" + modules.ShellQuote(outDir.String()) + "/$stem.sorted.vcf.gz\n" +
		modules.ShellCommand(command) + " --output \"$output\""
	if len(extra) > 0 {
		script += " " + modules.ShellCommand(extra)
	}
	script += " \"$input\""
	output := gobble.Bind{Name: "sorted_vcf", From: interval, Spec: gobble.PathSpec{Dir: outDir, Ext: ".sorted.vcf.gz"}, Rule: gobble.DeriveReplaceExt}
	index := gobble.Bind{Name: "sorted_tbi", From: interval, Spec: gobble.PathSpec{Dir: outDir, Ext: ".sorted.vcf.gz.tbi"}, Rule: gobble.DeriveReplaceExt}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: script, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "vcf", From: vcf}, {Name: "tbi", From: tbi}, {Name: "interval", From: interval}}, Outputs: []gobble.Bind{output, index}})
	return Ports{VCF: task.Out("sorted_vcf"), TBI: task.Out("sorted_tbi")}, nil
}

// Pipeline returns a standalone validated one-interval bcftools sort module.
func Pipeline(vcf, tbi, interval gobble.PathSpec, options Options) *gobble.Pipeline {
	inputs := []modules.Input{{Name: "vcf", Spec: vcf}, {Name: "tbi", Spec: tbi}, {Name: "intervals", Group: gobble.Group{{Name: interval.Base, Spec: interval}}}}
	options.IntervalDir = interval.Dir
	options.InputDir = vcf.Dir
	return modules.StandaloneScatterChecked("bcftools-sort", "intervals", inputs, 2, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], handles[2], options)
		return err
	})
}
