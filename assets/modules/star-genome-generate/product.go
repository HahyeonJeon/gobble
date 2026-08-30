package stargenomegenerate

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 STAR image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/htslib_samtools_star_gawk:ae438e9a604351a4@sha256:4a468118dbd7491a69bf9813c68233afa8558d1f3380fd8cab03e0e3d3135190"

// Options controls one lifted STAR genomeGenerate command.
type Options struct {
	modules.Options
	OutDir       gobble.Directory
	SJDBOverhang int
	GenomeSAIndexNBases int
}

// Ports contains the complete STAR index Tree.
type Ports struct{ Index gobble.Handle }

// Add records one validated STAR genomeGenerate command.
func Add(parent modules.Parent, fasta, gtf gobble.Handle, options Options) (Ports, error) {
	const unit = "star_genome_generate"
	fastaPath, err := modules.HandlePath(unit, fasta)
	if err != nil {
		return Ports{}, err
	}
	gtfPath, err := modules.HandlePath(unit, gtf)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = defaultSTARGenomeDir()
	}
	if options.SJDBOverhang < 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "SJDBOverhang must not be negative")
	}
	if options.GenomeSAIndexNBases < 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "GenomeSAIndexNBases must not be negative")
	}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 2, Memory: "6g"}
	}
	command := []string{"STAR", "--runMode", "genomeGenerate", "--genomeDir", outDir.String(), "--genomeFastaFiles", fastaPath, "--sjdbGTFfile", gtfPath}
	if n := modules.ThreadCount(resources.CPU); n > 0 {
		command = append(command, "--runThreadN", strconv.Itoa(n))
	}
	if options.SJDBOverhang > 0 {
		command = append(command, "--sjdbOverhang", strconv.Itoa(options.SJDBOverhang))
	}
	if options.GenomeSAIndexNBases > 0 {
		command = append(command, "--genomeSAindexNbases", strconv.Itoa(options.GenomeSAIndexNBases))
	}
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"--runMode", "--genomeDir", "--genomeFastaFiles", "--sjdbGTFfile", "--runThreadN", "--sjdbOverhang", "--genomeSAindexNbases"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Command: command, Image: image, Resources: resources,
		Inputs:  []gobble.Bind{{Name: "fasta", From: fasta}, {Name: "gtf", From: gtf}},
		Outputs: []gobble.Bind{{Name: "index", Tree: gobble.DeclareTree(outDir)}},
	})
	return Ports{Index: task.Out("index")}, nil
}

// Pipeline returns a standalone validated STAR genomeGenerate module.
func Pipeline(fasta, gtf gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("star-genome-generate", []modules.Input{{Name: "fasta", Spec: fasta}, {Name: "gtf", Spec: gtf}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], options)
		return err
	})
}
