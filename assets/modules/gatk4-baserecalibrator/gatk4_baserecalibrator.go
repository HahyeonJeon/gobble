// Package gatk4baserecalibrator owns one GATK BaseRecalibrator command.
package gatk4baserecalibrator

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// KnownSite is one indexed known-variant resource.
type KnownSite struct {
	VCF gobble.Handle
	TBI gobble.Handle
}

// Options controls one interval-scattered BaseRecalibrator command.
type Options struct {
	modules.Options
	IntervalDir gobble.Directory
	OutDir      gobble.Directory
}

// Ports contains one interval recalibration table.
type Ports struct{ Table gobble.Handle }

// Add records one validated interval-scattered BaseRecalibrator command.
func Add(parent modules.Parent, bam, bai, fasta, fai, dict, interval gobble.Handle, knownSites []KnownSite, options Options) (Ports, error) {
	const unit = "gatk4_baserecalibrator"
	if len(knownSites) == 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "known sites are empty")
	}
	paths := make([]string, 0, 5+2*len(knownSites))
	for _, handle := range []gobble.Handle{bam, bai, fasta, fai, dict} {
		path, err := modules.HandlePath(unit, handle)
		if err != nil {
			return Ports{}, err
		}
		paths = append(paths, path)
	}
	inputs := []gobble.Bind{{Name: "bam", From: bam}, {Name: "bai", From: bai}, {Name: "fasta", From: fasta}, {Name: "fai", From: fai}, {Name: "dict", From: dict}, {Name: "interval", From: interval}}
	knownPaths := make([]string, len(knownSites))
	for i, site := range knownSites {
		vcfPath, err := modules.HandlePath(unit, site.VCF)
		if err != nil {
			return Ports{}, err
		}
		if _, err = modules.HandlePath(unit, site.TBI); err != nil {
			return Ports{}, err
		}
		knownPaths[i] = vcfPath
		inputs = append(inputs, gobble.Bind{Name: "known_vcf_" + strconv.Itoa(i), From: site.VCF}, gobble.Bind{Name: "known_tbi_" + strconv.Itoa(i), From: site.TBI})
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/gatk4-baserecalibrator")
	}
	prelude, err := modules.ScatterFilePrelude(unit, options.IntervalDir)
	if err != nil {
		return Ports{}, err
	}
	protected := []string{"--input", "--output", "--reference", "--intervals", "--known-sites", "--tmp-dir"}
	extra, image, resources, err := modules.ResolveGATK4Options(unit, options.Options, gobble.Resources{CPU: 2, Memory: "4g"}, protected)
	if err != nil {
		return Ports{}, err
	}
	command := []string{"gatk", "BaseRecalibrator", "--input", paths[0], "--reference", paths[2]}
	for _, path := range knownPaths {
		command = append(command, "--known-sites", path)
	}
	command = append(command, "--tmp-dir", ".")
	script := prelude + "output=" + modules.ShellQuote(outDir.String()) + "/$stem.table\n" + modules.ShellCommand(command) + " --intervals \"$interval\" --output \"$output\""
	if len(extra) > 0 {
		script += " " + modules.ShellCommand(extra)
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: script, Image: image, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{{Name: "table", From: interval, Spec: gobble.PathSpec{Dir: outDir, Ext: ".table"}, Rule: gobble.DeriveReplaceExt}}})
	return Ports{Table: task.Out("table")}, nil
}

// Pipeline returns a standalone validated one-interval BaseRecalibrator module.
func Pipeline(bam, bai, fasta, fai, dict, interval gobble.PathSpec, knownVCFs, knownIndexes []gobble.PathSpec, options Options) *gobble.Pipeline {
	member := interval.Base
	inputs := []modules.Input{{Name: "bam", Spec: bam}, {Name: "bai", Spec: bai}, {Name: "fasta", Spec: fasta}, {Name: "fai", Spec: fai}, {Name: "dict", Spec: dict}, {Name: "intervals", Group: gobble.Group{{Name: member, Spec: interval}}}}
	for i := range knownVCFs {
		inputs = append(inputs, modules.Input{Name: "known_vcf_" + strconv.Itoa(i), Spec: knownVCFs[i]})
		if i < len(knownIndexes) {
			inputs = append(inputs, modules.Input{Name: "known_index_" + strconv.Itoa(i), Spec: knownIndexes[i]})
		}
	}
	options.IntervalDir = interval.Dir
	return modules.StandaloneScatterChecked("gatk4-baserecalibrator", "intervals", inputs, 5, func(parent modules.Parent, handles []gobble.Handle) error {
		if len(handles) != 6+2*len(knownVCFs) {
			return modules.ComposeDefect(gobble.DefectInvalidValue, "gatk4_baserecalibrator", "known-site VCF and index counts differ")
		}
		sites := make([]KnownSite, len(knownVCFs))
		for i := range sites {
			sites[i] = KnownSite{VCF: handles[6+2*i], TBI: handles[7+2*i]}
		}
		_, err := Add(parent, handles[0], handles[1], handles[2], handles[3], handles[4], handles[5], sites, options)
		return err
	})
}
