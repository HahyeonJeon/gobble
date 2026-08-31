// Package gatk4applybqsr owns one GATK ApplyBQSR command.
package gatk4applybqsr

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// Options controls one interval-scattered ApplyBQSR command.
type Options struct {
	modules.Options
	IntervalDir gobble.Directory
	TableDir    gobble.Directory
	OutDir      gobble.Directory
}

// Ports contains one interval recalibrated BAM.
type Ports struct{ BAM gobble.Handle }

// Add records one validated interval-scattered ApplyBQSR command.
func Add(parent modules.Parent, bam, bai, fasta, fai, dict, table, interval gobble.Handle, options Options) (Ports, error) {
	const unit = "gatk4_applybqsr"
	paths := make([]string, 0, 5)
	for _, handle := range []gobble.Handle{bam, bai, fasta, fai, dict} {
		path, err := modules.HandlePath(unit, handle)
		if err != nil {
			return Ports{}, err
		}
		paths = append(paths, path)
	}
	if options.TableDir.IsZero() {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "recalibration table directory is empty")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/gatk4-applybqsr")
	}
	prelude, err := modules.ScatterFilePrelude(unit, options.IntervalDir)
	if err != nil {
		return Ports{}, err
	}
	protected := []string{"--input", "--output", "--reference", "--bqsr-recal-file", "--intervals", "--tmp-dir"}
	extra, image, resources, err := modules.ResolveGATK4Options(unit, options.Options, gobble.Resources{CPU: 2, Memory: "4g"}, protected)
	if err != nil {
		return Ports{}, err
	}
	command := []string{"gatk", "ApplyBQSR", "--input", paths[0], "--reference", paths[2], "--tmp-dir", "."}
	script := prelude +
		"table=" + modules.ShellQuote(options.TableDir.String()) + "/$stem.table\n" +
		"output=" + modules.ShellQuote(outDir.String()) + "/$stem.bam\n" +
		modules.ShellCommand(command) + " --bqsr-recal-file \"$table\" --intervals \"$interval\" --output \"$output\""
	if len(extra) > 0 {
		script += " " + modules.ShellCommand(extra)
	}
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Script: script, Image: image, Resources: resources,
		Inputs:  []gobble.Bind{{Name: "bam", From: bam}, {Name: "bai", From: bai}, {Name: "fasta", From: fasta}, {Name: "fai", From: fai}, {Name: "dict", From: dict}, {Name: "table", From: table}, {Name: "interval", From: interval}},
		Outputs: []gobble.Bind{{Name: "recalibrated_bam", From: interval, Spec: gobble.PathSpec{Dir: outDir, Ext: ".bam"}, Rule: gobble.DeriveReplaceExt}},
	})
	return Ports{BAM: task.Out("recalibrated_bam")}, nil
}

// Pipeline returns a standalone validated one-interval ApplyBQSR module.
func Pipeline(bam, bai, fasta, fai, dict, table, interval gobble.PathSpec, options Options) *gobble.Pipeline {
	inputs := []modules.Input{{Name: "bam", Spec: bam}, {Name: "bai", Spec: bai}, {Name: "fasta", Spec: fasta}, {Name: "fai", Spec: fai}, {Name: "dict", Spec: dict}, {Name: "table", Spec: table}, {Name: "intervals", Group: gobble.Group{{Name: interval.Base, Spec: interval}}}}
	options.IntervalDir = interval.Dir
	options.TableDir = table.Dir
	return modules.StandaloneScatterChecked("gatk4-applybqsr", "intervals", inputs, 6, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], handles[2], handles[3], handles[4], handles[5], handles[6], options)
		return err
	})
}
