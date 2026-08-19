package assets

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
)

// bismarkImage is the nf-core/methylseq 4.2.0 Bismark 0.25.1 Seqera
// community image (modules/nf-core/bismark/* environment.yml).
const bismarkImage = "community.wave.seqera.io/library/bismark:0.25.1--1f50935de5d79c47"

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

func bismarkGenomeDir() gobble.Directory {
	return gobble.Dir("work/bismark-genome")
}

func bismarkRestageFASTA() gobble.PathSpec {
	return gobble.PathSpec{Dir: bismarkGenomeDir()}
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
// folder is work/bismark-genome, the restaged FASTA dest and parent of
// Bisulfite_Genome, not a directory port. Isolate copies the FASTA From
// path onto that dest.
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
// The parent folder is the restaged FASTA dest, not a directory port.
// The shared builder does not call AddInput.
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
	folder := bismarkGenomeDir()

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
		Inputs:    []gobble.Bind{{Name: "fasta", From: fasta, Spec: bismarkRestageFASTA()}},
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
