package assets

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
)

// starImage is the biocontainers tag for the nf-core STAR 2.7.11b pin
// (modules/nf-core/star/genomegenerate environment.yml). Current nf-core
// modules use a Seqera community image; this pin keeps the same
// quay.io/biocontainers form as the other first-party assets.
const starImage = "quay.io/biocontainers/star:2.7.11b--h5ca1c30_5"

const starGenomeTaskName = "star_genome_generate"

var starGenomeMembers = []struct {
	name string
	ext  string
}{
	{name: "Genome"},
	{name: "SA"},
	{name: "SAindex"},
	{name: "chrLength", ext: ".txt"},
	{name: "chrName", ext: ".txt"},
	{name: "chrNameLength", ext: ".txt"},
	{name: "chrStart", ext: ".txt"},
	{name: "genomeParameters", ext: ".txt"},
}

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
// OutDir is --genomeDir, the parent folder of the index Group, not a
// directory port.
type STARGenomeGenerateOptions struct {
	ExtraArgs []string
	Resources gobble.Resources
	OutDir    gobble.Directory
}

// STARGenomeGeneratePorts are the declared STAR index Group output.
type STARGenomeGeneratePorts struct {
	Index gobble.Handle
}

// AddSTARGenomeGenerate records one STAR genomeGenerate task on parent.
// Index siblings are a Group of regular files. The parent folder is
// PathSpec.Dir, not a directory port. The shared builder does not call
// AddInput.
func AddSTARGenomeGenerate(parent Parent, fasta gobble.Handle, opts STARGenomeGenerateOptions) STARGenomeGeneratePorts {
	return addSTARGenomeGenerate(parent, fasta, opts)
}

// STARGenomeGeneratePipeline returns a standalone STAR genomeGenerate
// pipeline. It AddInputs fasta, then calls the same builder as
// AddSTARGenomeGenerate.
func STARGenomeGeneratePipeline(fasta gobble.PathSpec, opts STARGenomeGenerateOptions) *gobble.Pipeline {
	return Standalone("star-genome-generate", []Input{{Name: "fasta", Spec: fasta}}, func(parent Parent, hs []gobble.Handle) {
		addSTARGenomeGenerate(parent, hs[0], opts)
	})
}

func addSTARGenomeGenerate(parent Parent, fasta gobble.Handle, opts STARGenomeGenerateOptions) STARGenomeGeneratePorts {
	outDir := starGenomeDir(opts.OutDir)

	cmd := []string{"STAR", "--runMode", "genomeGenerate", "--genomeDir", outDir.String()}
	if path, err := CommandPath(fasta.Spec()); err == nil {
		cmd = append(cmd, "--genomeFastaFiles", path)
	}
	if n := threadCount(opts.Resources.CPU); n > 0 {
		cmd = append(cmd, "--runThreadN", strconv.Itoa(n))
	}
	cmd = AppendExtraArgs(cmd, opts.ExtraArgs)

	task := AddTask(parent, gobble.TaskSpec{
		Name:      starGenomeTaskName,
		Command:   cmd,
		Image:     starImage,
		Inputs:    []gobble.Bind{{Name: "fasta", From: fasta}},
		Outputs:   []gobble.Bind{{Name: "index", Group: starGenomeGroup(outDir)}},
		Resources: opts.Resources,
	})
	return STARGenomeGeneratePorts{Index: task.Out("index")}
}

func starGenomeGroup(dir gobble.Directory) gobble.Group {
	g := make(gobble.Group, 0, len(starGenomeMembers))
	for _, m := range starGenomeMembers {
		g = append(g, gobble.Member{Name: m.name, Spec: gobble.PathSpec{Dir: dir, Name: m.name, Ext: m.ext}})
	}
	return g
}

func starGenomeGroupFrom() gobble.Group {
	g := make(gobble.Group, 0, len(starGenomeMembers))
	for _, m := range starGenomeMembers {
		g = append(g, gobble.Member{Name: m.name})
	}
	return g
}
