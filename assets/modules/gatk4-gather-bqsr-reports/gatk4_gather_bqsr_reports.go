// Package gatk4gatherbqsrreports owns one GATK GatherBQSRReports command.
package gatk4gatherbqsrreports

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// Options controls one GatherBQSRReports command.
type Options struct {
	modules.Options
	InputPaths []gobble.PathSpec
	OutDir     gobble.Directory
	Prefix     string
}

// Ports contains the complete sample recalibration table.
type Ports struct{ Table gobble.Handle }

// Add records one validated GatherBQSRReports command over a Gather handle.
func Add(parent modules.Parent, parts []gobble.Handle, options Options) (Ports, error) {
	const unit = "gatk4_gather_bqsr_reports"
	if len(options.InputPaths) == 0 || len(parts) == 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "recalibration table paths are empty")
	}
	outDir, prefix := options.OutDir, options.Prefix
	if outDir.IsZero() {
		outDir = gobble.Dir("work/gatk4-gather-bqsr-reports")
	}
	if prefix == "" {
		prefix = "recalibration"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".table"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "gathered recalibration table path is invalid")
	}
	protected := []string{"--input", "--output", "--tmp-dir"}
	extra, image, resources, err := modules.ResolveGATK4Options(unit, options.Options, gobble.Resources{CPU: 1, Memory: "3g"}, protected)
	if err != nil {
		return Ports{}, err
	}
	command := []string{"gatk", "GatherBQSRReports"}
	for _, spec := range options.InputPaths {
		path, renderErr := spec.Render()
		if renderErr != nil {
			return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "recalibration table input path is invalid")
		}
		command = append(command, "--input", path)
	}
	command = append(command, "--output", outputPath, "--tmp-dir", ".")
	command = append(command, extra...)
	inputs := make([]gobble.Bind, len(parts))
	for i, part := range parts {
		inputs[i] = gobble.Bind{Name: "tables_" + strconv.Itoa(i), From: part}
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{{Name: "table", Spec: output}}})
	return Ports{Table: task.Out("table")}, nil
}

// Pipeline returns a standalone validated GatherBQSRReports module.
func Pipeline(tables []gobble.PathSpec, options Options) *gobble.Pipeline {
	inputs := make([]modules.Input, len(tables))
	for i, table := range tables {
		inputs[i] = modules.Input{Name: "table_" + strconv.Itoa(i), Spec: table}
	}
	options.InputPaths = append([]gobble.PathSpec(nil), tables...)
	return modules.StandaloneChecked("gatk4-gather-bqsr-reports", inputs, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles, options)
		return err
	})
}
