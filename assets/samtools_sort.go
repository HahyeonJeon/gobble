package assets

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
)

// samtoolsImage is copied from tests/wgs-e2e/wgs_e2e_thin_test.go.
// Do not import that package.
const samtoolsImage = "quay.io/biocontainers/samtools:1.24--h9dcdb79_1"

const samtoolsSortTaskName = "samtools_sort"

// SamtoolsSortOptions are typed samtools sort settings. ExtraArgs are
// argv tokens appended after named flags and before the SAM path.
//
// -@ copies Resources.CPU when CPU is at least 1, as an integer.
type SamtoolsSortOptions struct {
	ExtraArgs []string
	Resources gobble.Resources
	OutDir    gobble.Directory
}

// SamtoolsSortPorts are the declared BAM output.
type SamtoolsSortPorts struct {
	BAM gobble.Handle
}

// AddSamtoolsSort records one samtools sort task on parent. SAM is
// sorted to BAM. The shared builder does not call AddInput.
func AddSamtoolsSort(parent Parent, sam gobble.Handle, opts SamtoolsSortOptions) SamtoolsSortPorts {
	return addSamtoolsSort(parent, sam, opts)
}

// SamtoolsSortPipeline returns a standalone samtools sort pipeline. It
// AddInputs sam, then calls the same builder as AddSamtoolsSort.
func SamtoolsSortPipeline(sam gobble.PathSpec, opts SamtoolsSortOptions) *gobble.Pipeline {
	return Standalone("samtools-sort", []Input{{Name: "sam", Spec: sam}}, func(parent Parent, hs []gobble.Handle) {
		addSamtoolsSort(parent, hs[0], opts)
	})
}

func addSamtoolsSort(parent Parent, sam gobble.Handle, opts SamtoolsSortOptions) SamtoolsSortPorts {
	outDir := opts.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/samtools-sort")
	}
	bamSpec := sam.Spec().ReplaceExtension(".bam").WithDir(outDir)

	cmd := []string{"samtools", "sort"}
	if n := threadCount(opts.Resources.CPU); n > 0 {
		cmd = append(cmd, "-@", strconv.Itoa(n))
	}
	cmd = append(cmd, "-o", mustCommandPath(bamSpec))
	cmd = AppendExtraArgs(cmd, opts.ExtraArgs)
	cmd = append(cmd, mustCommandPath(sam.Spec()))

	task := AddTask(parent, gobble.TaskSpec{
		Name:      samtoolsSortTaskName,
		Command:   cmd,
		Image:     samtoolsImage,
		Inputs:    []gobble.Bind{{Name: "sam", From: sam}},
		Outputs:   []gobble.Bind{{Name: "bam", Spec: bamSpec}},
		Resources: opts.Resources,
	})
	return SamtoolsSortPorts{BAM: task.Out("bam")}
}
