// Package stargenomegenerate owns the graph-stable STAR genomeGenerate command module.
package stargenomegenerate

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// starImage is the nf-core/rnaseq 3.26.0 STAR 2.7.11b Seqera image from
// modules/nf-core/star/genomegenerate and star/align.
const starImage = "community.wave.seqera.io/library/htslib_samtools_star_gawk:ae438e9a604351a4"

const starGenomeTaskName = "star_genome_generate"

func defaultSTARGenomeDir() gobble.Directory {
	return gobble.Dir("work/star-genome")
}

func starGenomeDir(dir gobble.Directory) gobble.Directory {
	if dir.IsZero() {
		return defaultSTARGenomeDir()
	}
	return dir
}

// STARGenomeGenerateOptions are typed STAR genomeGenerate settings.
// ExtraArgs are argv tokens appended after named flags.
//
// --runThreadN copies Resources.CPU when CPU is at least 1, as an integer.
// OutDir is --genomeDir, a Tree dest, not a Group of enumerated files.
// GTF is the optional standalone --sjdbGTFfile PathSpec.
type STARGenomeGenerateOptions struct {
	ExtraArgs []string
	Resources gobble.Resources
	OutDir    gobble.Directory
	GTF       gobble.PathSpec
}

// STARGenomeGeneratePorts are the declared STAR index Tree output.
type STARGenomeGeneratePorts struct {
	Index gobble.Handle
}

// AddSTARGenomeGenerate records one STAR genomeGenerate task on parent.
// --genomeDir is a Tree dest via DeclareTree of OutDir. A non-zero gtf
// adds --sjdbGTFfile and binds the GTF. The shared builder does not
// call AddInput.
func AddSTARGenomeGenerate(parent modules.Parent, fasta, gtf gobble.Handle, opts STARGenomeGenerateOptions) STARGenomeGeneratePorts {
	return addSTARGenomeGenerate(parent, fasta, gtf, opts)
}

// STARGenomeGeneratePipeline returns a standalone STAR genomeGenerate
// pipeline. It AddInputs fasta and, when opts.GTF is set, gtf, then
// calls the same builder as AddSTARGenomeGenerate.
func STARGenomeGeneratePipeline(fasta gobble.PathSpec, opts STARGenomeGenerateOptions) *gobble.Pipeline {
	inputs := []modules.Input{{Name: "fasta", Spec: fasta}}
	if !pathSpecUnset(opts.GTF) {
		inputs = append(inputs, modules.Input{Name: "gtf", Spec: opts.GTF})
	}
	return modules.Standalone("star-genome-generate", inputs, func(parent modules.Parent, hs []gobble.Handle) {
		var gtf gobble.Handle
		if len(hs) > 1 {
			gtf = hs[1]
		}
		addSTARGenomeGenerate(parent, hs[0], gtf, opts)
	})
}

func addSTARGenomeGenerate(parent modules.Parent, fasta, gtf gobble.Handle, opts STARGenomeGenerateOptions) STARGenomeGeneratePorts {
	outDir := starGenomeDir(opts.OutDir)
	sjdb := !gtf.IsZero()

	cmd := []string{"STAR", "--runMode", "genomeGenerate", "--genomeDir", outDir.String()}
	cmd = append(cmd, "--genomeFastaFiles", modules.MustCommandPath(fasta.Spec()))
	if sjdb {
		cmd = append(cmd, "--sjdbGTFfile", modules.MustCommandPath(gtf.Spec()))
	}
	if n := modules.ThreadCount(opts.Resources.CPU); n > 0 {
		cmd = append(cmd, "--runThreadN", strconv.Itoa(n))
	}
	cmd = modules.AppendLegacyExtraArgs(cmd, opts.ExtraArgs)

	inputs := []gobble.Bind{{Name: "fasta", From: fasta}}
	if sjdb {
		inputs = append(inputs, gobble.Bind{Name: "gtf", From: gtf})
	}

	task := parent.AddTask(gobble.TaskSpec{
		Name:      starGenomeTaskName,
		Command:   cmd,
		Image:     starImage,
		Inputs:    inputs,
		Outputs:   []gobble.Bind{{Name: "index", Tree: gobble.DeclareTree(outDir)}},
		Resources: opts.Resources,
	})
	return STARGenomeGeneratePorts{Index: task.Out("index")}
}

func pathSpecUnset(p gobble.PathSpec) bool {
	return p.Dir.IsZero() && p.Prefix == "" && p.Base == "" && len(p.Suffixes) == 0 && p.Ext == ""
}
