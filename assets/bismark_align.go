package assets

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
)

const bismarkAlignTaskName = "bismark_align"

const bismarkAlignBasename = "aligned"

func defaultBismarkAlignDir() gobble.Directory {
	return gobble.Dir("work/bismark-align")
}

func bismarkAlignDir(dir gobble.Directory) gobble.Directory {
	if dir.IsZero() {
		return defaultBismarkAlignDir()
	}
	return dir
}

// BismarkAlignOptions are typed bismark align settings. ExtraArgs are
// argv tokens appended after named flags.
//
// -p copies Resources.CPU when CPU is at least 2, as an integer.
// Bismark requires -p >= 2, so CPU 1 omits the flag. --genome is the
// FASTA PathSpec.Dir and must match AddBismarkGenome. OutDir is
// --output_dir, the parent folder of aligned_pe.bam.
type BismarkAlignOptions struct {
	ExtraArgs []string
	Resources gobble.Resources
	OutDir    gobble.Directory
}

// BismarkAlignPorts are the declared BAM output.
type BismarkAlignPorts struct {
	BAM gobble.Handle
}

// AddBismarkAlign records one bismark align task on parent. fasta is
// the same FASTA AddBismarkGenome indexed. index is the Group handle
// from AddBismarkGenome. The command emits BAM and does not call
// samtools. The shared builder does not call AddInput.
func AddBismarkAlign(parent Parent, fasta, index, r1, r2 gobble.Handle, opts BismarkAlignOptions) BismarkAlignPorts {
	return addBismarkAlign(parent, fasta, index, r1, r2, opts)
}

// BismarkAlignPipeline returns a standalone bismark align pipeline.
// Index siblings are PathSpec-authored Group members next to fasta, not
// a live bismark genome run. Pipeline inputs cannot be a Group, so the
// wrapper records a Group fixture task for AddBismarkAlign to From.
func BismarkAlignPipeline(fasta, r1, r2 gobble.PathSpec, opts BismarkAlignOptions) *gobble.Pipeline {
	return Standalone("bismark-align", []Input{
		{Name: "fasta", Spec: fasta},
		{Name: "r1", Spec: r1},
		{Name: "r2", Spec: r2},
	}, func(parent Parent, hs []gobble.Handle) {
		fixture := AddTask(parent, gobble.TaskSpec{
			Name:    "index_files",
			Command: []string{"true"},
			Outputs: []gobble.Bind{{Name: "index", Group: bismarkGenomeGroup(bismarkGenomeFolder(fasta))}},
		})
		addBismarkAlign(parent, hs[0], fixture.Out("index"), hs[1], hs[2], opts)
	})
}

func addBismarkAlign(parent Parent, fasta, index, r1, r2 gobble.Handle, opts BismarkAlignOptions) BismarkAlignPorts {
	outDir := bismarkAlignDir(opts.OutDir)
	genomeDir := bismarkGenomeFolder(fasta.Spec())
	bamSpec := gobble.PathSpec{Dir: outDir, Name: bismarkAlignBasename + "_pe", Ext: ".bam"}

	cmd := []string{"bismark", "--genome", bismarkGenomeFolderToken(genomeDir), "--bam", "--output_dir", outDir.String(), "--basename", bismarkAlignBasename}
	if n := threadCount(opts.Resources.CPU); n >= 2 {
		cmd = append(cmd, "-p", strconv.Itoa(n))
	}
	if path, err := CommandPath(r1.Spec()); err == nil {
		cmd = append(cmd, "-1", path)
	}
	if path, err := CommandPath(r2.Spec()); err == nil {
		cmd = append(cmd, "-2", path)
	}
	cmd = AppendExtraArgs(cmd, opts.ExtraArgs)

	task := AddTask(parent, gobble.TaskSpec{
		Name:    bismarkAlignTaskName,
		Command: cmd,
		Image:   bismarkImage,
		Inputs: []gobble.Bind{
			{Name: "fasta", From: fasta},
			{Name: "index", From: index, Group: bismarkGenomeGroupFrom()},
			{Name: "r1", From: r1},
			{Name: "r2", From: r2},
		},
		Outputs:   []gobble.Bind{{Name: "bam", Spec: bamSpec}},
		Resources: opts.Resources,
	})
	return BismarkAlignPorts{BAM: task.Out("bam")}
}
