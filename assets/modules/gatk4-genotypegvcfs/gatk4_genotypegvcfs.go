// Package gatk4genotypegvcfs owns one GATK GenotypeGVCFs command.
package gatk4genotypegvcfs

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// Database maps one stable interval member name to its GenomicsDB Tree.
type Database struct {
	Interval string
	Tree     gobble.Handle
}

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

// Add records one validated interval-scattered GenotypeGVCFs command.
func Add(parent modules.Parent, databases []Database, interval, fasta, fai, dict, dbsnp, dbsnpTBI gobble.Handle, options Options) (Ports, error) {
	const unit = "gatk4_genotypegvcfs"
	if len(databases) == 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "GenomicsDB set is empty")
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
	inputs := []gobble.Bind{{Name: "interval", From: interval}, {Name: "fasta", From: fasta}, {Name: "fai", From: fai}, {Name: "dict", From: dict}, {Name: "dbsnp", From: dbsnp}, {Name: "dbsnp_tbi", From: dbsnpTBI}}
	caseScript := "case \"$stem\" in\n"
	for i, database := range databases {
		if database.Interval == "" || database.Tree.Tree().IsZero() {
			return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "GenomicsDB interval mapping is invalid")
		}
		caseScript += "  " + modules.ShellQuote(database.Interval) + ") database=" + modules.ShellQuote(database.Tree.Tree().Dir.String()) + " ;;\n"
		inputs = append(inputs, gobble.Bind{Name: "database_" + strconv.Itoa(i), From: database.Tree, Tree: gobble.DeclareTree(gobble.Directory{})})
	}
	caseScript += "  *) echo \"no GenomicsDB for interval $stem\" >&2; exit 2 ;;\nesac\n"
	protected := []string{"--variant", "--output", "--reference", "--dbsnp", "--intervals", "--tmp-dir"}
	extra, image, resources, err := modules.ResolveGATK4Options(unit, options.Options, gobble.Resources{CPU: 1, Memory: "4g"}, protected)
	if err != nil {
		return Ports{}, err
	}
	command := []string{"gatk", "GenotypeGVCFs", "--reference", paths[0], "--dbsnp", paths[3], "--tmp-dir", "."}
	script := prelude + caseScript + "output=" + modules.ShellQuote(outDir.String()) + "/$stem.vcf.gz\n" + modules.ShellCommand(command) + " --variant \"gendb://$database\" --intervals \"$interval\" --output \"$output\""
	if len(extra) > 0 {
		script += " " + modules.ShellCommand(extra)
	}
	vcf := gobble.Bind{Name: "vcf", From: interval, Spec: gobble.PathSpec{Dir: outDir, Ext: ".vcf.gz"}, Rule: gobble.DeriveReplaceExt}
	tbi := gobble.Bind{Name: "tbi", From: interval, Spec: gobble.PathSpec{Dir: outDir, Ext: ".vcf.gz.tbi"}, Rule: gobble.DeriveReplaceExt}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: script, Image: image, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{vcf, tbi}})
	return Ports{VCF: task.Out("vcf"), TBI: task.Out("tbi")}, nil
}

// Pipeline returns a standalone validated one-interval GenotypeGVCFs module.
func Pipeline(database gobble.Tree, interval, fasta, fai, dict, dbsnp, dbsnpTBI gobble.PathSpec, options Options) *gobble.Pipeline {
	inputs := []modules.Input{{Name: "database", Tree: database}, {Name: "intervals", Group: gobble.Group{{Name: interval.Base, Spec: interval}}}, {Name: "fasta", Spec: fasta}, {Name: "fai", Spec: fai}, {Name: "dict", Spec: dict}, {Name: "dbsnp", Spec: dbsnp}, {Name: "dbsnp_tbi", Spec: dbsnpTBI}}
	options.IntervalDir = interval.Dir
	return modules.StandaloneScatterChecked("gatk4-genotypegvcfs", "intervals", inputs, 1, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, []Database{{Interval: interval.Base, Tree: handles[0]}}, handles[1], handles[2], handles[3], handles[4], handles[5], handles[6], options)
		return err
	})
}
