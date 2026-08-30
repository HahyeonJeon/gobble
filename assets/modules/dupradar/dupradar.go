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
	matrix := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: "_dupMatrix.txt"}
	multiQC := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: "_dup_intercept_mqc.txt"}
	matrixPath, _ := matrix.Render()
	multiQCPath, _ := multiQC.Render()
	strand := "0"
	if options.Strandedness == gobble.StrandednessForward {
		strand = "1"
	} else if options.Strandedness == gobble.StrandednessReverse {
		strand = "2"
	}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 2, Memory: "4g"}
	}
	paired := "false"
	if options.Paired {
		paired = "true"
	}
	command := []string{"Rscript", "-e", dupRadarR, "--", bamPath, gtfPath, strand, paired, strconv.Itoa(modules.ThreadCount(resources.CPU)), matrixPath, multiQCPath, prefix}
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"-e", "--"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bam", From: bam}, {Name: "gtf", From: gtf}}, Outputs: []gobble.Bind{{Name: "matrix", Spec: matrix}, {Name: "multiqc", Spec: multiQC}}})
	return Ports{Matrix: task.Out("matrix"), MultiQC: task.Out("multiqc")}, nil
}

// Pipeline returns a standalone validated dupRadar module.
func Pipeline(bam, gtf gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("dupradar", []modules.Input{{Name: "bam", Spec: bam}, {Name: "gtf", Spec: gtf}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], options)
		return err
	})
}
