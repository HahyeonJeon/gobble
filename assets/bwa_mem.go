package assets

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
)

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
func AddBWAMem(parent Parent, fasta, index, r1, r2 gobble.Handle, opts BWAMemOptions) BWAMemPorts {
	return addBWAMem(parent, fasta, index, r1, r2, opts)
}

// BWAMemPipeline returns a standalone bwa mem pipeline. Index siblings
// are PathSpec-authored Group members next to fasta, not a live bwa
// index run. Pipeline inputs cannot be a Group, so the wrapper records
// a Group fixture task for AddBWAMem to From.
func BWAMemPipeline(fasta, r1, r2 gobble.PathSpec, opts BWAMemOptions) *gobble.Pipeline {
	return Standalone("bwa-mem", []Input{
		{Name: "fasta", Spec: fasta},
		{Name: "r1", Spec: r1},
		{Name: "r2", Spec: r2},
	}, func(parent Parent, hs []gobble.Handle) {
		fixture := AddTask(parent, gobble.TaskSpec{
			Name:    "index_files",
			Command: []string{"true"},
			Outputs: []gobble.Bind{{Name: "index", Group: bwaIndexGroup(fasta)}},
		})
		addBWAMem(parent, hs[0], fixture.Out("index"), hs[1], hs[2], opts)
	})
}

func addBWAMem(parent Parent, fasta, index, r1, r2 gobble.Handle, opts BWAMemOptions) BWAMemPorts {
	outDir := opts.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/bwa-mem")
	}
	samSpec := gobble.PathSpec{Dir: outDir, Name: "aligned", Ext: ".sam"}

	cmd := []string{"bwa", "mem"}
	if n := threadCount(opts.Resources.CPU); n > 0 {
		cmd = append(cmd, "-t", strconv.Itoa(n))
	}
	if path, err := CommandPath(samSpec); err == nil {
		cmd = append(cmd, "-o", path)
	}
	cmd = AppendExtraArgs(cmd, opts.ExtraArgs)
	if path, err := CommandPath(fasta.Spec()); err == nil {
		cmd = append(cmd, path)
	}
	if path, err := CommandPath(r1.Spec()); err == nil {
		cmd = append(cmd, path)
	}
	if path, err := CommandPath(r2.Spec()); err == nil {
		cmd = append(cmd, path)
	}

	task := AddTask(parent, gobble.TaskSpec{
		Name:    bwaMemTaskName,
		Command: cmd,
		Image:   bwaImage,
		Inputs: []gobble.Bind{
			{Name: "fasta", From: fasta},
			{Name: "index", From: index, Group: bwaIndexGroupFrom()},
			{Name: "r1", From: r1},
			{Name: "r2", From: r2},
		},
		Outputs:   []gobble.Bind{{Name: "sam", Spec: samSpec}},
		Resources: opts.Resources,
	})
	return BWAMemPorts{SAM: task.Out("sam")}
}
