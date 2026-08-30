// Package trimgalore owns one Trim Galore command.
package trimgalore

import (
	"path"
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 Trim Galore image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/trim-galore:2.1.0--27e6376b8f6c1872@sha256:9d747504e44dbf5dfa8a2d66cbbd3bd80f897cc2e17ebe406821b4809b34a3a4"

// Options controls one single- or paired-end Trim Galore command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
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
	command := []string{"trim_galore", "--cores", strconv.Itoa(cores), "--gzip", "--output_dir", outDir.String(), "--basename", prefix}
	if !read2.IsZero() {
		command = append(command, "--paired")
	}
	command = append(command, read1Path)
	if read2Path != "" {
		command = append(command, read2Path)
	}
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"--cores", "--gzip", "--output_dir", "--basename", "--paired"})
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
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs, Outputs: outputs})
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
