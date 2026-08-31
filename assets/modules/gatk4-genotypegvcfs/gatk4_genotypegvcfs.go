// Package gatk4genotypegvcfs owns one GATK GenotypeGVCFs command.
package gatk4genotypegvcfs

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// Options controls one interval-scattered GenotypeGVCFs command.
type Options struct {
	modules.Options
	IntervalDir gobble.Directory
	OutDir      gobble.Directory
}

// Ports contains one indexed unfiltered interval joint VCF.
type Ports struct {
	VCF gobble.Handle
	TBI gobble.Handle
}

// Add records one validated interval-scattered GenotypeGVCFs command. Database
// is the matching per-member Tree from the same interval Scatter.
func Add(parent modules.Parent, database, interval, fasta, fai, dict, dbsnp, dbsnpTBI gobble.Handle, options Options) (Ports, error) {
	const unit = "gatk4_genotypegvcfs"
	if database.Tree().IsZero() || database.Tree().Dir.IsZero() {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "GenomicsDB Tree is empty")
	}
	paths := make([]string, 0, 5)
	for _, handle := range []gobble.Handle{fasta, fai, dict, dbsnp, dbsnpTBI} {
		path, err := modules.HandlePath(unit, handle)
		if err != nil {
			return Ports{}, err
		}
		paths = append(paths, path)
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/gatk4-genotypegvcfs")
	}
	prelude, err := modules.ScatterFilePrelude(unit, options.IntervalDir)
	if err != nil {
		return Ports{}, err
	}
	inputs := []gobble.Bind{
		{Name: "database", From: database, Tree: gobble.DeclareTree(gobble.Directory{})},
		{Name: "interval", From: interval}, {Name: "fasta", From: fasta}, {Name: "fai", From: fai},
		{Name: "dict", From: dict}, {Name: "dbsnp", From: dbsnp}, {Name: "dbsnp_tbi", From: dbsnpTBI},
	}
	protected := []string{"--variant", "--output", "--reference", "--dbsnp", "--intervals", "--tmp-dir", "--create-output-variant-index"}
	extra, image, resources, err := modules.ResolveGATK4Options(unit, options.Options, gobble.Resources{CPU: 1, Memory: "4g"}, protected)
	if err != nil {
		return Ports{}, err
	}
	command := []string{"gatk", "GenotypeGVCFs", "--reference", paths[0], "--dbsnp", paths[3], "--tmp-dir", "."}
	script := prelude +
		"database=" + modules.ShellQuote(database.Tree().Dir.String()) + "/$stem\n" +
		"output=" + modules.ShellQuote(outDir.String()) + "/$stem.vcf.gz\n" +
		modules.ShellCommand(command) + " --variant \"gendb://$database\" --intervals \"$interval\" --output \"$output\""
	if len(extra) > 0 {
		script += " " + modules.ShellCommand(extra)
	}
	vcf := gobble.Bind{Name: "vcf", From: interval, Spec: gobble.PathSpec{Dir: outDir, Ext: ".vcf.gz"}, Rule: gobble.DeriveReplaceExt}
	tbi := gobble.Bind{Name: "tbi", From: interval, Spec: gobble.PathSpec{Dir: outDir, Ext: ".vcf.gz.tbi"}, Rule: gobble.DeriveReplaceExt}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: script, Image: image, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{vcf, tbi}})
	return Ports{VCF: task.Out("vcf"), TBI: task.Out("tbi")}, nil
}

// Pipeline returns a standalone validated one-interval GenotypeGVCFs module.
// Database is a root Tree with one child directory named interval.Base.
func Pipeline(database gobble.Tree, interval, fasta, fai, dict, dbsnp, dbsnpTBI gobble.PathSpec, options Options) *gobble.Pipeline {
	inputs := []modules.Input{{Name: "database", Tree: database}, {Name: "intervals", Group: gobble.Group{{Name: interval.Base, Spec: interval}}}, {Name: "fasta", Spec: fasta}, {Name: "fai", Spec: fai}, {Name: "dict", Spec: dict}, {Name: "dbsnp", Spec: dbsnp}, {Name: "dbsnp_tbi", Spec: dbsnpTBI}}
	options.IntervalDir = interval.Dir
	return modules.StandaloneScatterChecked("gatk4-genotypegvcfs", "intervals", inputs, 1, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], handles[2], handles[3], handles[4], handles[5], handles[6], options)
		return err
	})
}
