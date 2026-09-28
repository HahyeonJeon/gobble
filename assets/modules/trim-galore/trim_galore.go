// Package trimgalore owns one Trim Galore command.
package trimgalore

import (
	"path"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	"github.com/HahyeonJeon/gobble/internal/modulecommand"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 Trim Galore image resolved for
// linux/amd64.
const DefaultImage modules.Image = modulecommand.TrimImage

// Options controls one single- or paired-end Trim Galore command.
type Options struct {
	modules.Options
	OutDir           gobble.Directory
	Prefix           string
	ClipR1           int
	ClipR2           int
	ThreePrimeClipR1 int
	ThreePrimeClipR2 int
	Quality          int
	Length           int
	Adapter          string
	Adapter2         string
}

// Ports are trimmed reads and input-derived Trim Galore reports. Report2 is
// zero for single-end input.
type Ports struct {
	Read1   gobble.Handle
	Read2   gobble.Handle
	Report1 gobble.Handle
	Report2 gobble.Handle
}

// Add records one validated Trim Galore command. A zero read2 selects
// single-end operation.
func Add(parent modules.Parent, read1, read2 gobble.Handle, options Options) (Ports, error) {
	const unit = "trim_galore"
	read1Path, err := modules.HandlePath(unit, read1)
	if err != nil {
		return Ports{}, err
	}
	var read2Path string
	if !read2.IsZero() {
		read2Path, err = modules.HandlePath(unit, read2)
		if err != nil {
			return Ports{}, err
		}
	}
	if options.ClipR1 < 0 || options.ClipR2 < 0 || options.ThreePrimeClipR1 < 0 || options.ThreePrimeClipR2 < 0 || options.Quality < 0 || options.Length < 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "Trim Galore clipping, quality, and length values must not be negative")
	}
	if read2.IsZero() && (options.ClipR2 != 0 || options.ThreePrimeClipR2 != 0 || options.Adapter2 != "") {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "read-2 Trim Galore options require paired-end reads")
	}
	if err := modules.RejectExtraArgs(unit, options.ExtraArgs, []string{"--fastqc", "--rrbs", "--dont_gzip", "--retain_unpaired", "--hardtrim3", "--hardtrim5", "--version", "--help"}); err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/trim-galore")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "sample"
	}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 4, Memory: "2g"}
	}
	cores := modules.ThreadCount(resources.CPU) - 3
	if !read2.IsZero() {
		cores--
	}
	if cores < 1 {
		cores = 1
	}
	if cores > 8 {
		cores = 8
	}
	command := modulecommand.Trim(modulecommand.TrimOptions{Read1: read1Path, Read2: read2Path, OutDir: outDir.String(), Prefix: prefix, Cores: cores, ClipR1: options.ClipR1, ClipR2: options.ClipR2, ThreePrimeClipR1: options.ThreePrimeClipR1, ThreePrimeClipR2: options.ThreePrimeClipR2, Quality: options.Quality, Length: options.Length, Adapter: options.Adapter, Adapter2: options.Adapter2})
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"--cores", "--gzip", "--output_dir", "--basename", "--paired", "--clip_R1", "--clip_R2", "--three_prime_clip_R1", "--three_prime_clip_R2", "--quality", "--length", "--adapter", "--adapter2"})
	if err != nil {
		return Ports{}, err
	}
	read1Out := gobble.PathSpec{Dir: outDir, Base: prefix + "_val_1", Ext: ".fq.gz"}
	if read2.IsZero() {
		read1Out = gobble.Literal(prefix + "_trimmed.fq.gz").WithDir(outDir)
	}
	read2Out := gobble.PathSpec{Dir: outDir, Base: prefix + "_val_2", Ext: ".fq.gz"}
	report1 := gobble.Literal(path.Base(read1Path) + "_trimming_report.txt").WithDir(outDir)
	inputs := []gobble.Bind{{Name: "read1", From: read1}}
	outputs := []gobble.Bind{{Name: "trimmed_read1", Spec: read1Out}, {Name: "report1", Spec: report1}}
	if !read2.IsZero() {
		inputs = append(inputs, gobble.Bind{Name: "read2", From: read2})
		report2 := gobble.Literal(path.Base(read2Path) + "_trimming_report.txt").WithDir(outDir)
		outputs = append(outputs, gobble.Bind{Name: "trimmed_read2", Spec: read2Out}, gobble.Bind{Name: "report2", Spec: report2})
	}
	settings := []gobble.IntegerSetting{integerSetting("quality", "Quality threshold", "Phred", options.Quality), integerSetting("length", "Minimum length", "bp", options.Length)}
	task := parent.AddTask(gobble.TaskSpec{InspectionSettings: settings, Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs, Outputs: outputs})
	ports := Ports{Read1: task.Out("trimmed_read1"), Report1: task.Out("report1")}
	if !read2.IsZero() {
		ports.Read2 = task.Out("trimmed_read2")
		ports.Report2 = task.Out("report2")
	}
	return ports, nil
}

// Pipeline returns a standalone validated Trim Galore module.
func Pipeline(read1, read2 gobble.PathSpec, options Options) *gobble.Pipeline {
	inputs := []modules.Input{{Name: "read1", Spec: read1}}
	if !pathSpecUnset(read2) {
		inputs = append(inputs, modules.Input{Name: "read2", Spec: read2})
	}
	return modules.StandaloneChecked("trim-galore", inputs, func(parent modules.Parent, handles []gobble.Handle) error {
		var mate gobble.Handle
		if len(handles) > 1 {
			mate = handles[1]
		}
		_, err := Add(parent, handles[0], mate, options)
		return err
	})
}

func pathSpecUnset(spec gobble.PathSpec) bool {
	return spec.Dir.IsZero() && spec.Prefix == "" && spec.Base == "" && len(spec.Suffixes) == 0 && spec.Ext == ""
}

// Zero is omitted by command construction, so preserve the unknown tool default.
func integerSetting(key, label, unit string, value int) gobble.IntegerSetting {
	setting := gobble.IntegerSetting{Key: key, Label: label, Unit: unit}
	if value > 0 {
		setting.Value = &value
	}
	return setting
}
