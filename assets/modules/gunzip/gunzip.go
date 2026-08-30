// Package gunzip owns one gzip decompression command.
package gunzip

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 gzip image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/coreutils_grep_gzip_lbzip2_pruned:838ba80435a629f8@sha256:63c2c6b22e83b2f656e88fbb1553e595da4e9e58794e3bfcb98b20b3837f328a"

// Options controls one gzip decompression command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the decompressed regular file.
type Ports struct{ File gobble.Handle }

// Add records one validated gzip -cd command.
func Add(parent modules.Parent, compressed gobble.Handle, options Options) (Ports, error) {
	const unit = "gunzip"
	input, err := modules.HandlePath(unit, compressed)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/gunzip")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "decompressed"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".gtf"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "decompressed output path is invalid")
	}
	command := []string{"gzip", "-cd", input}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "256m"}, command, []string{"-c", "-d", "-cd"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: modules.ShellRedirect(command, outputPath), Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "compressed", From: compressed}}, Outputs: []gobble.Bind{{Name: "file", Spec: output}}})
	return Ports{File: task.Out("file")}, nil
}

// Pipeline returns a standalone validated gzip decompression module.
func Pipeline(compressed gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("gunzip", []modules.Input{{Name: "compressed", Spec: compressed}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], options)
		return err
	})
}
