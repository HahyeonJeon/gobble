// Package samtoolsindex owns the graph-stable samtools index command module.
package samtoolsindex

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

const samtoolsImage = "quay.io/biocontainers/samtools:1.24--h9dcdb79_1"

const samtoolsIndexTaskName = "samtools_index"

// SamtoolsIndexOptions are typed samtools index settings. ExtraArgs are
// argv tokens appended after named flags and before the BAM and BAI
// paths.
//
// -@ copies Resources.CPU when CPU is at least 1, as an integer.
// BAI is a second-task related-file From of the BAM. Same-task From is
// a cycle.
type SamtoolsIndexOptions struct {
	ExtraArgs []string
	Resources gobble.Resources
}

// SamtoolsIndexPorts are the declared BAI companion output.
type SamtoolsIndexPorts struct {
	BAI gobble.Handle
}

// AddSamtoolsIndex records one samtools index task on parent. BAI is a
// related file of bam. The shared builder does not call AddInput.
func AddSamtoolsIndex(parent modules.Parent, bam gobble.Handle, opts SamtoolsIndexOptions) SamtoolsIndexPorts {
	return addSamtoolsIndex(parent, bam, opts)
}

// SamtoolsIndexPipeline returns a standalone samtools index pipeline.
// It AddInputs bam, then calls the same builder as AddSamtoolsIndex.
func SamtoolsIndexPipeline(bam gobble.PathSpec, opts SamtoolsIndexOptions) *gobble.Pipeline {
	return modules.Standalone("samtools-index", []modules.Input{{Name: "bam", Spec: bam}}, func(parent modules.Parent, hs []gobble.Handle) {
		addSamtoolsIndex(parent, hs[0], opts)
	})
}

func addSamtoolsIndex(parent modules.Parent, bam gobble.Handle, opts SamtoolsIndexOptions) SamtoolsIndexPorts {
	baiSpec := bam.Spec().AppendExt(".bai")

	cmd := []string{"samtools", "index"}
	if n := modules.ThreadCount(opts.Resources.CPU); n > 0 {
		cmd = append(cmd, "-@", strconv.Itoa(n))
	}
	cmd = modules.AppendLegacyExtraArgs(cmd, opts.ExtraArgs)
	cmd = append(cmd, modules.MustCommandPath(bam.Spec()), modules.MustCommandPath(baiSpec))

	task := parent.AddTask(gobble.TaskSpec{
		Name:    samtoolsIndexTaskName,
		Command: cmd,
		Image:   samtoolsImage,
		Inputs:  []gobble.Bind{{Name: "bam", From: bam}},
		Outputs: []gobble.Bind{{
			Name: "bai",
			Spec: gobble.PathSpec{Ext: ".bai"},
			From: bam,
		}},
		Resources: opts.Resources,
	})
	return SamtoolsIndexPorts{BAI: task.Out("bai")}
}
