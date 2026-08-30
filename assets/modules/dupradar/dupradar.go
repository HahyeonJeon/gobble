// Package dupradar owns one dupRadar R command.
package dupradar

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

const dupRadarR = `
args <- commandArgs(trailingOnly=TRUE)
suppressPackageStartupMessages(library(dupRadar))
dm <- analyzeDuprates(args[[1]], args[[2]], as.integer(args[[3]]), as.logical(args[[4]]), as.integer(args[[5]]), verbose=TRUE)
write.table(dm, file=args[[6]], quote=FALSE, row.names=FALSE, sep="\t")
fit <- duprateExpFit(DupMat=dm)
lines <- c(
  "#id: DupInt",
  "#plot_type: 'generalstats'",
  "Sample dupRadar_intercept",
  paste(args[[8]], fit$intercept)
)
writeLines(lines, args[[7]])
`

// DefaultImage is the nf-core/rnaseq 3.26.0 dupRadar image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/bioconductor-dupradar:1.38.0--831da16eb40a64ab@sha256:9509e959f6a9fe5ebc3cc16b907ba0fc0423fe03d2c02d6ac2e05c5fb50114fb"

// Options controls one dupRadar command.
type Options struct {
	modules.Options
	OutDir       gobble.Directory
	Prefix       string
	Strandedness string
	Paired       bool
}

// Ports are selected dupRadar QC outputs.
type Ports struct {
	Matrix  gobble.Handle
	MultiQC gobble.Handle
}

// Add records one validated dupRadar command.
func Add(parent modules.Parent, bam, gtf gobble.Handle, options Options) (Ports, error) {
	return add(parent, bam, gtf, gobble.Handle{}, options)
}

// AddInferred records dupRadar with its strand code selected from a completed
// strandedness inference task.
func AddInferred(parent modules.Parent, bam, gtf, strandedness gobble.Handle, options Options) (Ports, error) {
	return add(parent, bam, gtf, strandedness, options)
}

func add(parent modules.Parent, bam, gtf, strandedness gobble.Handle, options Options) (Ports, error) {
	const unit = "dupradar"
	bamPath, err := modules.HandlePath(unit, bam)
	if err != nil {
		return Ports{}, err
	}
	gtfPath, err := modules.HandlePath(unit, gtf)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/dupradar")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "sample"
	}
	matrix := gobble.Literal(prefix + "_dupMatrix.txt").WithDir(outDir)
	multiQC := gobble.Literal(prefix + "_dup_intercept_mqc.txt").WithDir(outDir)
	matrixPath, _ := matrix.Render()
	multiQCPath, _ := multiQC.Render()
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 2, Memory: "4g"}
	}
	paired := "false"
	if options.Paired {
		paired = "true"
	}
	commandFor := func(stranded string) ([]string, string, gobble.Resources, error) {
		strand := "0"
		switch stranded {
		case gobble.StrandednessForward:
			strand = "1"
		case gobble.StrandednessReverse:
			strand = "2"
		case "", gobble.StrandednessUnstranded:
		default:
			return nil, "", gobble.Resources{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "strandedness must be unstranded, forward, or reverse")
		}
		command := []string{"Rscript", "-e", dupRadarR, "--", bamPath, gtfPath, strand, paired, strconv.Itoa(modules.ThreadCount(resources.CPU)), matrixPath, multiQCPath, prefix}
		base := options.Options
		base.Resources = resources
		return modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"-e", "--"})
	}
	spec := gobble.TaskSpec{Name: unit, Inputs: []gobble.Bind{{Name: "bam", From: bam}, {Name: "gtf", From: gtf}}, Outputs: []gobble.Bind{{Name: "matrix", Spec: matrix}, {Name: "multiqc", Spec: multiQC}}}
	if strandedness.IsZero() {
		command, image, resolvedResources, commandErr := commandFor(options.Strandedness)
		if commandErr != nil {
			return Ports{}, commandErr
		}
		spec.Command, spec.Image, spec.Resources = command, image, resolvedResources
	} else {
		strandPath, pathErr := modules.HandlePath(unit, strandedness)
		if pathErr != nil {
			return Ports{}, pathErr
		}
		commands := make(map[string][]string, 3)
		for _, strand := range []string{gobble.StrandednessUnstranded, gobble.StrandednessForward, gobble.StrandednessReverse} {
			command, image, resolvedResources, commandErr := commandFor(strand)
			if commandErr != nil {
				return Ports{}, commandErr
			}
			commands[strand] = command
			spec.Image, spec.Resources = image, resolvedResources
		}
		spec.Script = modules.StrandedCommand(strandPath, commands[gobble.StrandednessUnstranded], commands[gobble.StrandednessForward], commands[gobble.StrandednessReverse])
		spec.Inputs = append(spec.Inputs, gobble.Bind{Name: "strandedness", From: strandedness})
	}
	task := parent.AddTask(spec)
	return Ports{Matrix: task.Out("matrix"), MultiQC: task.Out("multiqc")}, nil
}

// Pipeline returns a standalone validated dupRadar module.
func Pipeline(bam, gtf gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("dupradar", []modules.Input{{Name: "bam", Spec: bam}, {Name: "gtf", Spec: gtf}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], options)
		return err
	})
}
