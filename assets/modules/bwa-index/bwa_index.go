// Package bwaindex owns the graph-stable bwa index command module.
package bwaindex

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// bwaImage is the graph-stable WGS checkpoint image.
const bwaImage = "quay.io/biocontainers/bwa:0.7.18--h577a1d6_2"

const bwaIndexTaskName = "bwa_index"

var bwaIndexMemberNames = []string{"amb", "ann", "bwt", "pac", "sa"}

// BWAIndexOptions are typed bwa index settings. ExtraArgs are argv
// tokens appended after named flags and before the FASTA path.
//
// bwa index has no thread flag. Resources.CPU is not copied into Command.
type BWAIndexOptions struct {
	ExtraArgs []string
	Resources gobble.Resources
}

// BWAIndexPorts are the declared bwa index Group output.
type BWAIndexPorts struct {
	Index gobble.Handle
}

// AddBWAIndex records one bwa index task on parent. Index siblings are
// a Group. The parent folder is PathSpec.Dir, not a directory port.
// The shared builder does not call AddInput.
func AddBWAIndex(parent modules.Parent, fasta gobble.Handle, opts BWAIndexOptions) BWAIndexPorts {
	return addBWAIndex(parent, fasta, opts)
}

// BWAIndexPipeline returns a standalone bwa index pipeline. It AddInputs
// fasta, then calls the same builder as AddBWAIndex.
func BWAIndexPipeline(fasta gobble.PathSpec, opts BWAIndexOptions) *gobble.Pipeline {
	return modules.Standalone("bwa-index", []modules.Input{{Name: "fasta", Spec: fasta}}, func(parent modules.Parent, hs []gobble.Handle) {
		addBWAIndex(parent, hs[0], opts)
	})
}

func addBWAIndex(parent modules.Parent, fasta gobble.Handle, opts BWAIndexOptions) BWAIndexPorts {
	cmd := []string{"bwa", "index"}
	cmd = modules.AppendLegacyExtraArgs(cmd, opts.ExtraArgs)
	cmd = append(cmd, modules.MustCommandPath(fasta.Spec()))

	task := parent.AddTask(gobble.TaskSpec{
		Name:      bwaIndexTaskName,
		Command:   cmd,
		Image:     bwaImage,
		Inputs:    []gobble.Bind{{Name: "fasta", From: fasta}},
		Outputs:   []gobble.Bind{{Name: "index", Group: indexGroup(fasta.Spec())}},
		Resources: opts.Resources,
	})
	return BWAIndexPorts{Index: task.Out("index")}
}

func indexGroup(fasta gobble.PathSpec) gobble.Group {
	g := make(gobble.Group, 0, len(bwaIndexMemberNames))
	for _, name := range bwaIndexMemberNames {
		g = append(g, gobble.Member{Name: name, Spec: fasta.AppendExt("." + name)})
	}
	return g
}
