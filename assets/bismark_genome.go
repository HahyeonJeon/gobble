package assets

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
)

// bismarkImage is the biocontainers tag for the nf-core Bismark 3.1.0 pin
// (modules/nf-core/bismark/genomepreparation environment.yml). Current
// nf-core modules use a Seqera community image; this pin keeps the same
// quay.io/biocontainers form as the other first-party assets.
const bismarkImage = "quay.io/biocontainers/bismark:3.1.0--hfa8f182_0"

const bismarkGenomeTaskName = "bismark_genome"

type bismarkGenomeMember struct {
	name string
	sub  []string
	file string
	ext  string
}

var bismarkGenomeMembers = []bismarkGenomeMember{
	{name: "CT_1", sub: []string{"Bisulfite_Genome", "CT_conversion"}, file: "BS_CT", ext: ".1.bt2"},
	{name: "CT_2", sub: []string{"Bisulfite_Genome", "CT_conversion"}, file: "BS_CT", ext: ".2.bt2"},
	{name: "CT_3", sub: []string{"Bisulfite_Genome", "CT_conversion"}, file: "BS_CT", ext: ".3.bt2"},
	{name: "CT_4", sub: []string{"Bisulfite_Genome", "CT_conversion"}, file: "BS_CT", ext: ".4.bt2"},
	{name: "CT_rev1", sub: []string{"Bisulfite_Genome", "CT_conversion"}, file: "BS_CT.rev", ext: ".1.bt2"},
	{name: "CT_rev2", sub: []string{"Bisulfite_Genome", "CT_conversion"}, file: "BS_CT.rev", ext: ".2.bt2"},
	{name: "CT_mfa", sub: []string{"Bisulfite_Genome", "CT_conversion"}, file: "genome_mfa", ext: ".CT_conversion.fa"},
	{name: "GA_1", sub: []string{"Bisulfite_Genome", "GA_conversion"}, file: "BS_GA", ext: ".1.bt2"},
	{name: "GA_2", sub: []string{"Bisulfite_Genome", "GA_conversion"}, file: "BS_GA", ext: ".2.bt2"},
	{name: "GA_3", sub: []string{"Bisulfite_Genome", "GA_conversion"}, file: "BS_GA", ext: ".3.bt2"},
	{name: "GA_4", sub: []string{"Bisulfite_Genome", "GA_conversion"}, file: "BS_GA", ext: ".4.bt2"},
	{name: "GA_rev1", sub: []string{"Bisulfite_Genome", "GA_conversion"}, file: "BS_GA.rev", ext: ".1.bt2"},
	{name: "GA_rev2", sub: []string{"Bisulfite_Genome", "GA_conversion"}, file: "BS_GA.rev", ext: ".2.bt2"},
	{name: "GA_mfa", sub: []string{"Bisulfite_Genome", "GA_conversion"}, file: "genome_mfa", ext: ".GA_conversion.fa"},
}

func bismarkGenomeFolder(spec gobble.PathSpec) gobble.Directory {
	return spec.Dir
}

func bismarkGenomeFolderToken(dir gobble.Directory) string {
	if dir.IsZero() {
		return "."
	}
	return dir.String()
}

// BismarkGenomeOptions are typed bismark genome_preparation settings.
// ExtraArgs are argv tokens appended after named flags and before the
// genome folder.
//
// --parallel copies Resources.CPU when CPU is at least 2, as an integer.
// Bismark requires --parallel >= 2, so CPU 1 omits the flag. The genome
// folder is the FASTA PathSpec.Dir, the parent of Bisulfite_Genome, not
// a directory port. Isolate restage of an input bind does not copy the
// FASTA, so the index is written next to it.
type BismarkGenomeOptions struct {
	ExtraArgs []string
	Resources gobble.Resources
}

// BismarkGenomePorts are the declared Bismark index Group output.
type BismarkGenomePorts struct {
	Index gobble.Handle
}

// AddBismarkGenome records one bismark_genome_preparation task on parent.
// Index siblings are a Group of regular files under Bisulfite_Genome.
// The parent folder is PathSpec.Dir, not a directory port. The shared
// builder does not call AddInput.
func AddBismarkGenome(parent Parent, fasta gobble.Handle, opts BismarkGenomeOptions) BismarkGenomePorts {
	return addBismarkGenome(parent, fasta, opts)
}

// BismarkGenomePipeline returns a standalone bismark genome pipeline.
// It AddInputs fasta, then calls the same builder as AddBismarkGenome.
func BismarkGenomePipeline(fasta gobble.PathSpec, opts BismarkGenomeOptions) *gobble.Pipeline {
	return Standalone("bismark-genome", []Input{{Name: "fasta", Spec: fasta}}, func(parent Parent, hs []gobble.Handle) {
		addBismarkGenome(parent, hs[0], opts)
	})
}

func addBismarkGenome(parent Parent, fasta gobble.Handle, opts BismarkGenomeOptions) BismarkGenomePorts {
	folder := bismarkGenomeFolder(fasta.Spec())

	cmd := []string{"bismark_genome_preparation", "--bowtie2"}
	if n := threadCount(opts.Resources.CPU); n >= 2 {
		cmd = append(cmd, "--parallel", strconv.Itoa(n))
	}
	cmd = AppendExtraArgs(cmd, opts.ExtraArgs)
	cmd = append(cmd, bismarkGenomeFolderToken(folder))

	task := AddTask(parent, gobble.TaskSpec{
		Name:      bismarkGenomeTaskName,
		Command:   cmd,
		Image:     bismarkImage,
		Inputs:    []gobble.Bind{{Name: "fasta", From: fasta}},
		Outputs:   []gobble.Bind{{Name: "index", Group: bismarkGenomeGroup(folder)}},
		Resources: opts.Resources,
	})
	return BismarkGenomePorts{Index: task.Out("index")}
}

func bismarkGenomeGroup(dir gobble.Directory) gobble.Group {
	g := make(gobble.Group, 0, len(bismarkGenomeMembers))
	for _, m := range bismarkGenomeMembers {
		g = append(g, gobble.Member{
			Name: m.name,
			Spec: gobble.PathSpec{Dir: dir.Join(m.sub...), Name: m.file, Ext: m.ext},
		})
	}
	return g
}

func bismarkGenomeGroupFrom() gobble.Group {
	g := make(gobble.Group, 0, len(bismarkGenomeMembers))
	for _, m := range bismarkGenomeMembers {
		g = append(g, gobble.Member{Name: m.name})
	}
	return g
}
