package assets

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
)

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
func AddSamtoolsIndex(parent Parent, bam gobble.Handle, opts SamtoolsIndexOptions) SamtoolsIndexPorts {
	return addSamtoolsIndex(parent, bam, opts)
}

// SamtoolsIndexPipeline returns a standalone samtools index pipeline.
// It AddInputs bam, then calls the same builder as AddSamtoolsIndex.
func SamtoolsIndexPipeline(bam gobble.PathSpec, opts SamtoolsIndexOptions) *gobble.Pipeline {
	return Standalone("samtools-index", []Input{{Name: "bam", Spec: bam}}, func(parent Parent, hs []gobble.Handle) {
		addSamtoolsIndex(parent, hs[0], opts)
	})
}

func addSamtoolsIndex(parent Parent, bam gobble.Handle, opts SamtoolsIndexOptions) SamtoolsIndexPorts {
	baiSpec := bam.Spec().Append(".bai")

	cmd := []string{"samtools", "index"}
	if n := threadCount(opts.Resources.CPU); n > 0 {
		cmd = append(cmd, "-@", strconv.Itoa(n))
	}
	cmd = AppendExtraArgs(cmd, opts.ExtraArgs)
	if path, err := CommandPath(bam.Spec()); err == nil {
		cmd = append(cmd, path)
	}
	if path, err := CommandPath(baiSpec); err == nil {
		cmd = append(cmd, path)
	}

	task := AddTask(parent, gobble.TaskSpec{
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
