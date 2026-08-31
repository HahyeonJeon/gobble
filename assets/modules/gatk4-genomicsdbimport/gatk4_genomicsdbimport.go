// Package gatk4genomicsdbimport owns one GATK GenomicsDBImport command.
package gatk4genomicsdbimport

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// Variant is one indexed sample gVCF in a complete cohort.
type Variant struct {
	GVCF gobble.Handle
	TBI  gobble.Handle
}

// Options controls one interval GenomicsDBImport command.
type Options struct {
	modules.Options
	IntervalDir gobble.Directory
	OutDir      gobble.Directory
}

// Ports contains the complete GenomicsDB directory Tree.
type Ports struct{ Database gobble.Handle }

// Add records one validated interval-scattered GenomicsDBImport command.
func Add(parent modules.Parent, variants []Variant, interval gobble.Handle, options Options) (Ports, error) {
	const unit = "gatk4_genomicsdbimport"
	if len(variants) < 2 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "at least two cohort gVCFs are required")
	}
	prelude, err := modules.ScatterFilePrelude(unit, options.IntervalDir)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/gatk4-genomicsdbimport")
	}
	protected := []string{"--variant", "--genomicsdb-workspace-path", "--intervals", "--tmp-dir", "--sample-name-map", "--genomicsdb-update-workspace-path"}
	extra, image, resources, err := modules.ResolveGATK4Options(unit, options.Options, gobble.Resources{CPU: 2, Memory: "6g"}, protected)
	if err != nil {
		return Ports{}, err
	}
	command := []string{"gatk", "GenomicsDBImport"}
	inputs := []gobble.Bind{{Name: "interval", From: interval}}
	for i, variant := range variants {
		gvcfPath, pathErr := modules.HandlePath(unit, variant.GVCF)
		if pathErr != nil {
			return Ports{}, pathErr
		}
		if _, pathErr = modules.HandlePath(unit, variant.TBI); pathErr != nil {
			return Ports{}, pathErr
		}
		command = append(command, "--variant", gvcfPath)
		inputs = append(inputs, gobble.Bind{Name: "gvcf_" + strconv.Itoa(i), From: variant.GVCF}, gobble.Bind{Name: "tbi_" + strconv.Itoa(i), From: variant.TBI})
	}
	command = append(command, "--tmp-dir", ".")
	script := prelude +
		"workspace=" + modules.ShellQuote(outDir.String()) + "/$stem\n" +
		modules.ShellCommand(command) + " --genomicsdb-workspace-path \"$workspace\" --intervals \"$interval\""
	if len(extra) > 0 {
		script += " " + modules.ShellCommand(extra)
	}
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Script: script, Image: image, Resources: resources, Inputs: inputs,
		Outputs: []gobble.Bind{{Name: "database", From: interval, Tree: gobble.DeclareTree(outDir)}},
	})
	return Ports{Database: task.Out("database")}, nil
}

// Pipeline returns a standalone validated GenomicsDBImport module.
func Pipeline(gvcfs, indexes []gobble.PathSpec, interval gobble.PathSpec, options Options) *gobble.Pipeline {
	inputs := []modules.Input{{Name: "intervals", Group: gobble.Group{{Name: interval.Base, Spec: interval}}}}
	for i := range gvcfs {
		inputs = append(inputs, modules.Input{Name: "gvcf_" + strconv.Itoa(i), Spec: gvcfs[i]})
		if i < len(indexes) {
			inputs = append(inputs, modules.Input{Name: "index_" + strconv.Itoa(i), Spec: indexes[i]})
		}
	}
	options.IntervalDir = interval.Dir
	return modules.StandaloneScatterChecked("gatk4-genomicsdbimport", "intervals", inputs, 0, func(parent modules.Parent, handles []gobble.Handle) error {
		if len(handles) != 1+2*len(gvcfs) {
			return modules.ComposeDefect(gobble.DefectInvalidValue, "gatk4_genomicsdbimport", "gVCF and index input counts differ")
		}
		variants := make([]Variant, len(gvcfs))
		for i := range variants {
			variants[i] = Variant{GVCF: handles[1+2*i], TBI: handles[2+2*i]}
		}
		_, err := Add(parent, variants, handles[0], options)
		return err
	})
}
