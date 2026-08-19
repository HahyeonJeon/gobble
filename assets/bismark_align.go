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
// Bismark requires -p >= 2, so CPU 1 omits the flag. --genome is
// work/bismark-genome, the restaged FASTA dest, and must match
// AddBismarkGenome. OutDir is --output_dir, the parent folder of
// aligned_pe.bam and aligned_PE_report.txt.
type BismarkAlignOptions struct {
	ExtraArgs []string
	Resources gobble.Resources
	OutDir    gobble.Directory
}

// BismarkAlignPorts are the declared BAM and PE report outputs.
type BismarkAlignPorts struct {
	BAM    gobble.Handle
	Report gobble.Handle
}

// AddBismarkAlign records one bismark align task on parent. fasta is
// the same FASTA AddBismarkGenome indexed. index is the Group handle
// from AddBismarkGenome. The command emits BAM and does not call
// samtools. The shared builder does not call AddInput.
func AddBismarkAlign(parent Parent, fasta, index, r1, r2 gobble.Handle, opts BismarkAlignOptions) BismarkAlignPorts {
	return addBismarkAlign(parent, fasta, index, r1, r2, opts)
}

// BismarkAlignPipeline returns a standalone bismark align pipeline.
// Index siblings are PathSpec-authored Group members under the restaged
// genome dest, not a live bismark genome run. The wrapper records the
// index with AddInputGroup and does not record a fixture task.
func BismarkAlignPipeline(fasta, r1, r2 gobble.PathSpec, opts BismarkAlignOptions) *gobble.Pipeline {
	return Standalone("bismark-align", []Input{
		{Name: "fasta", Spec: fasta},
		{Name: "index", Group: bismarkGenomeGroup(bismarkGenomeDir())},
		{Name: "r1", Spec: r1},
		{Name: "r2", Spec: r2},
	}, func(parent Parent, hs []gobble.Handle) {
		addBismarkAlign(parent, hs[0], hs[1], hs[2], hs[3], opts)
	})
}

func addBismarkAlign(parent Parent, fasta, index, r1, r2 gobble.Handle, opts BismarkAlignOptions) BismarkAlignPorts {
	outDir := bismarkAlignDir(opts.OutDir)
	genomeDir := bismarkGenomeDir()
	bamSpec := gobble.PathSpec{Dir: outDir, Base: bismarkAlignBasename + "_pe", Ext: ".bam"}
	reportSpec := gobble.PathSpec{Dir: outDir, Base: bismarkAlignBasename + "_PE_report", Ext: ".txt"}

	cmd := []string{"bismark", "--genome", bismarkGenomeFolderToken(genomeDir), "--bam", "--output_dir", outDir.String(), "--basename", bismarkAlignBasename}
	if n := threadCount(opts.Resources.CPU); n >= 2 {
		cmd = append(cmd, "-p", strconv.Itoa(n))
	}
	cmd = append(cmd, "-1", mustCommandPath(r1.Spec()), "-2", mustCommandPath(r2.Spec()))
	cmd = AppendExtraArgs(cmd, opts.ExtraArgs)

	task := AddTask(parent, gobble.TaskSpec{
		Name:    bismarkAlignTaskName,
		Command: cmd,
		Image:   bismarkImage,
		Inputs: []gobble.Bind{
			{Name: "fasta", From: fasta, Spec: bismarkRestageFASTA()},
			{Name: "index", From: index, Group: bismarkGenomeGroupFrom()},
			{Name: "r1", From: r1},
			{Name: "r2", From: r2},
		},
		Outputs: []gobble.Bind{
			{Name: "bam", Spec: bamSpec},
			{Name: "report", Spec: reportSpec},
		},
		Resources: opts.Resources,
	})
	return BismarkAlignPorts{BAM: task.Out("bam"), Report: task.Out("report")}
}
