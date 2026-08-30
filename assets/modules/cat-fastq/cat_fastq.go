// Package catfastq owns one GNU cat FASTQ-consolidation command.
package catfastq

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 cat/fastq image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/coreutils_grep_gzip_lbzip2_pruned:838ba80435a629f8@sha256:63c2c6b22e83b2f656e88fbb1553e595da4e9e58794e3bfcb98b20b3837f328a"

// Options controls one mate-specific FASTQ concatenation.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the consolidated FASTQ.
type Ports struct{ FASTQ gobble.Handle }

// Add records one validated cat command. Inputs are ordered run files for one
// mate and are never token-split or discovered at runtime.
func Add(parent modules.Parent, inputs []gobble.Handle, options Options) (Ports, error) {
	const unit = "cat_fastq"
	if len(inputs) < 2 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "at least two FASTQ inputs are required")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/cat-fastq")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "reads"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".merged.fastq.gz"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "consolidated FASTQ path is invalid")
	}
	command := []string{"cat"}
	binds := make([]gobble.Bind, len(inputs))
	for i, input := range inputs {
		path, pathErr := modules.HandlePath(unit, input)
		if pathErr != nil {
			return Ports{}, pathErr
		}
		command = append(command, path)
		binds[i] = gobble.Bind{Name: "run_" + strconv.Itoa(i+1), From: input}
	}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "512m"}, command, nil)
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: modules.ShellRedirect(command, outputPath), Image: image, Resources: resources, Inputs: binds, Outputs: []gobble.Bind{{Name: "fastq", Spec: output}}})
	return Ports{FASTQ: task.Out("fastq")}, nil
}

// Pipeline returns a standalone validated FASTQ concatenation module.
func Pipeline(inputs []gobble.PathSpec, options Options) *gobble.Pipeline {
	declared := make([]modules.Input, len(inputs))
	for i, input := range inputs {
		declared[i] = modules.Input{Name: "run_" + strconv.Itoa(i+1), Spec: input}
	}
	return modules.StandaloneChecked("cat-fastq", declared, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles, options)
		return err
	})
}
