// Package bwamem owns the graph-stable bwa mem command module.
package bwamem

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

const bwaImage = "quay.io/biocontainers/bwa:0.7.18--h577a1d6_2"

const bwaMemTaskName = "bwa_mem"

// BWAMemOptions are typed bwa mem settings. ExtraArgs are argv tokens
// appended after named flags and before positional idxbase and reads.
//
// -t copies Resources.CPU when CPU is at least 1, as an integer.
type BWAMemOptions struct {
	ExtraArgs []string
	Resources gobble.Resources
	OutDir    gobble.Directory
}

// BWAMemPorts are the declared SAM output.
type BWAMemPorts struct {
	SAM gobble.Handle
}

// AddBWAMem records one bwa mem task on parent. index is the Group
// handle from AddBWAIndex. The command emits SAM and does not call
// samtools. The shared builder does not call AddInput.
func AddBWAMem(parent modules.Parent, fasta, index, r1, r2 gobble.Handle, opts BWAMemOptions) BWAMemPorts {
	return addBWAMem(parent, fasta, index, r1, r2, opts)
}

// BWAMemPipeline returns a standalone bwa mem pipeline. Index siblings
// are PathSpec-authored Group members next to fasta, not a live bwa
// index run. The wrapper records them with AddInputGroup so AddBWAMem
// can Group From the input handle.
func BWAMemPipeline(fasta, r1, r2 gobble.PathSpec, opts BWAMemOptions) *gobble.Pipeline {
	return modules.Standalone("bwa-mem", []modules.Input{
		{Name: "fasta", Spec: fasta},
		{Name: "index", Group: indexGroup(fasta)},
		{Name: "r1", Spec: r1},
		{Name: "r2", Spec: r2},
	}, func(parent modules.Parent, hs []gobble.Handle) {
		addBWAMem(parent, hs[0], hs[1], hs[2], hs[3], opts)
	})
}

func addBWAMem(parent modules.Parent, fasta, index, r1, r2 gobble.Handle, opts BWAMemOptions) BWAMemPorts {
	outDir := opts.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/bwa-mem")
	}
	samSpec := gobble.PathSpec{Dir: outDir, Base: "aligned", Ext: ".sam"}

	cmd := []string{"bwa", "mem"}
	if n := modules.ThreadCount(opts.Resources.CPU); n > 0 {
		cmd = append(cmd, "-t", strconv.Itoa(n))
	}
	cmd = append(cmd, "-o", modules.MustCommandPath(samSpec))
	cmd = modules.AppendLegacyExtraArgs(cmd, opts.ExtraArgs)
	cmd = append(cmd, modules.MustCommandPath(fasta.Spec()), modules.MustCommandPath(r1.Spec()), modules.MustCommandPath(r2.Spec()))

	task := parent.AddTask(gobble.TaskSpec{
		Name:    bwaMemTaskName,
		Command: cmd,
		Image:   bwaImage,
		Inputs: []gobble.Bind{
			{Name: "fasta", From: fasta},
			{Name: "index", From: index, Group: inputIndexGroup()},
			{Name: "r1", From: r1},
			{Name: "r2", From: r2},
		},
		Outputs:   []gobble.Bind{{Name: "sam", Spec: samSpec}},
		Resources: opts.Resources,
	})
	return BWAMemPorts{SAM: task.Out("sam")}
}

var indexMemberNames = []string{"amb", "ann", "bwt", "pac", "sa"}

func indexGroup(fasta gobble.PathSpec) gobble.Group {
	group := make(gobble.Group, 0, len(indexMemberNames))
	for _, name := range indexMemberNames {
		group = append(group, gobble.Member{Name: name, Spec: fasta.AppendExt("." + name)})
	}
	return group
}

func inputIndexGroup() gobble.Group {
	group := make(gobble.Group, 0, len(indexMemberNames))
	for _, name := range indexMemberNames {
		group = append(group, gobble.Member{Name: name})
	}
	return group
}
