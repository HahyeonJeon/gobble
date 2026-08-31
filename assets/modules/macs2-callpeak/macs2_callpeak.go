// Package macs2callpeak owns one MACS2 callpeak command.
package macs2callpeak

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/atacseq 2.1.2 MACS2 2.2.7.1 image resolved for linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/macs2:2.2.7.1--py38h4a8c8d9_3@sha256:b8506329f67a88c4c9c0e433028c25ba5c29b199a8936f629a02c40a4ecf1942"

// Mode selects broad or narrow output from the same command path.
type Mode string

const (
	Broad  Mode = "broad"
	Narrow Mode = "narrow"
)

// Options controls one callpeak command.
type Options struct {
	modules.Options
	OutDir              gobble.Directory
	Prefix              string
	Mode                Mode
	Paired              bool
	EffectiveGenomeSize string
	QValue              float64
}

// Ports contains the selected peak set, summit positions, and MACS2 call table.
type Ports struct {
	Peaks   gobble.Handle
	Summits gobble.Handle
	XLS     gobble.Handle
}

// Add records one control-aware callpeak command. A zero control is an explicit no-control call.
func Add(parent modules.Parent, treatment, control gobble.Handle, options Options) (Ports, error) {
	const unit = "macs2_callpeak"
	treatmentPath, err := modules.HandlePath(unit, treatment)
	if err != nil {
		return Ports{}, err
	}
	if options.Mode != Broad && options.Mode != Narrow {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "peak mode must be broad or narrow")
	}
	if options.EffectiveGenomeSize == "" || options.QValue < 0 || options.QValue > 1 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "effective genome size and q-value are invalid")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/macs2")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "peaks"
	}
	format := "BAM"
	if options.Paired {
		format = "BAMPE"
	}
	command := []string{"macs2", "callpeak", "--gsize", options.EffectiveGenomeSize, "--format", format, "--name", prefix, "--treatment", treatmentPath, "--outdir", outDir.String(), "--qvalue", formatFloat(options.QValue), "--call-summits"}
	inputs := []gobble.Bind{{Name: "treatment", From: treatment}}
	if !control.IsZero() {
		controlPath, pathErr := modules.HandlePath(unit, control)
		if pathErr != nil {
			return Ports{}, pathErr
		}
		command = append(command, "--control", controlPath)
		inputs = append(inputs, gobble.Bind{Name: "control", From: control})
	}
	peakExt := ".broadPeak"
	if options.Mode == Broad {
		command = append(command, "--broad")
	} else {
		peakExt = ".narrowPeak"
	}
	protected := []string{"--gsize", "-g", "--format", "-f", "--name", "-n", "--treatment", "-t", "--control", "-c", "--outdir", "--qvalue", "-q", "--broad", "--call-summits"}
	if err := modules.RejectExtraArgs(unit, options.ExtraArgs, protected); err != nil {
		return Ports{}, err
	}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 2, Memory: "4g"}, command, protected)
	if err != nil {
		return Ports{}, err
	}
	peaks := gobble.PathSpec{Dir: outDir, Base: prefix + "_peaks", Ext: peakExt}
	summits := gobble.PathSpec{Dir: outDir, Base: prefix + "_summits", Ext: ".bed"}
	xls := gobble.PathSpec{Dir: outDir, Base: prefix + "_peaks", Ext: ".xls"}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{{Name: "peaks", Spec: peaks}, {Name: "summits", Spec: summits}, {Name: "xls", Spec: xls}}})
	return Ports{Peaks: task.Out("peaks"), Summits: task.Out("summits"), XLS: task.Out("xls")}, nil
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}
