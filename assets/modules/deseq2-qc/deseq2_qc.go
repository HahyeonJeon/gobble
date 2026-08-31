// Package deseq2qc owns DESeq2 PCA and sample-distance quality control only.
package deseq2qc

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

const deseq2QCR = `
args <- commandArgs(trailingOnly=TRUE)
counts <- read.delim(args[[1]], check.names=FALSE, row.names=1)
counts <- round(as.matrix(counts))
suppressPackageStartupMessages(library(DESeq2))
coldata <- data.frame(sample=colnames(counts), row.names=colnames(counts))
dds <- DESeqDataSetFromMatrix(countData=counts, colData=coldata, design=~1)
v <- varianceStabilizingTransformation(dds, blind=TRUE)
mat <- assay(v)
write.table(data.frame(id=rownames(mat), mat, check.names=FALSE), args[[4]], sep="\t", quote=FALSE, row.names=FALSE)
pdf(args[[2]])
plotPCA(v, intgroup="sample")
dev.off()
pdf(args[[3]])
heatmap(as.matrix(dist(t(mat))), symm=TRUE)
dev.off()
`

const atacQCR = `
args <- commandArgs(trailingOnly=TRUE)
table <- read.delim(args[[1]], comment.char="#", check.names=FALSE)
if (ncol(table) < 8) stop("ATAC featureCounts table needs at least two count columns")
counts <- round(as.matrix(table[, 7:ncol(table), drop=FALSE]))
rownames(counts) <- table[[1]]
suppressPackageStartupMessages(library(DESeq2))
coldata <- data.frame(sample=colnames(counts), row.names=colnames(counts))
dds <- DESeqDataSetFromMatrix(countData=counts, colData=coldata, design=~1)
dds <- estimateSizeFactors(dds)
write.table(data.frame(sample=names(sizeFactors(dds)), size_factor=sizeFactors(dds)), args[[5]], sep="\t", quote=FALSE, row.names=FALSE)
v <- varianceStabilizingTransformation(dds, blind=TRUE)
mat <- assay(v)
write.table(data.frame(id=rownames(mat), mat, check.names=FALSE), args[[4]], sep="\t", quote=FALSE, row.names=FALSE)
pdf(args[[2]])
plotPCA(v, intgroup="sample")
dev.off()
pdf(args[[3]])
heatmap(as.matrix(dist(t(mat))), symm=TRUE)
dev.off()
`

// DefaultImage is the nf-core/rnaseq 3.26.0 local DESeq2-QC image resolved for
// linux/amd64. It contains R 4.4.2 and DESeq2 1.46.0. This module does not
// expose a design formula or contrast.
const DefaultImage modules.Image = "community.wave.seqera.io/library/r-base_r-optparse_r-ggplot2_r-rcolorbrewer_pruned:9e75394d0bc21987@sha256:afd00df7ce26f38ecb2a063f65d441fc20c0803e5c7319ee5cbe3a23732a30dd"

// Options controls cohort expression QC. There is intentionally no contrast.
// The inline R program has no optional argv region, so ExtraArgs are rejected.
type Options struct {
	modules.Options
	OutDir gobble.Directory
}

// Ports are PCA and sample-distance quality-control artifacts.
type Ports struct {
	PCA         gobble.Handle
	Distance    gobble.Handle
	Matrix      gobble.Handle
	SizeFactors gobble.Handle
}

// AddATAC records DESeq2 size-factor, PCA, and distance QC for one consensus
// count matrix. It accepts no design formula or contrast.
func AddATAC(parent modules.Parent, counts gobble.Handle, options Options) (Ports, error) {
	const unit = "deseq2_qc"
	if len(options.ExtraArgs) != 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "ATAC DESeq2-QC does not accept ExtraArgs, designs, or contrasts")
	}
	countsPath, err := modules.HandlePath(unit, counts)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/deseq2-qc")
	}
	pca := gobble.PathSpec{Dir: outDir, Base: "pca", Ext: ".pdf"}
	distance := gobble.PathSpec{Dir: outDir, Base: "sample_distance", Ext: ".pdf"}
	normalized := gobble.PathSpec{Dir: outDir, Base: "vst_matrix", Ext: ".tsv"}
	sizeFactors := gobble.PathSpec{Dir: outDir, Base: "size_factors", Ext: ".tsv"}
	pcaPath, err := pca.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "ATAC DESeq2-QC output path is invalid")
	}
	distancePath, _ := distance.Render()
	normalizedPath, _ := normalized.Render()
	sizeFactorsPath, _ := sizeFactors.Render()
	base := options.Options
	base.ExtraArgs = nil
	command := []string{"Rscript", "-e", atacQCR, countsPath, pcaPath, distancePath, normalizedPath, sizeFactorsPath}
	command, image, resources, err := modules.ResolveOptions(unit, base, "quay.io/biocontainers/mulled-v2-8849acf39a43cdd6c839a369a74c0adc823e2f91:ab110436faf952a33575c64dd74615a84011450b-0@sha256:d2db38b68b3ad9a4fd817d3c3cb67792ddcefff87049af5d851fe93af9a56893", gobble.Resources{CPU: 2, Memory: "4g"}, command, []string{"-e", "--design", "--contrast"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "counts", From: counts}}, Outputs: []gobble.Bind{{Name: "pca", Spec: pca}, {Name: "distance", Spec: distance}, {Name: "matrix", Spec: normalized}, {Name: "size_factors", Spec: sizeFactors}}})
	return Ports{PCA: task.Out("pca"), Distance: task.Out("distance"), Matrix: task.Out("matrix"), SizeFactors: task.Out("size_factors")}, nil
}

// Add records one validated DESeq2 quality-control command.
func Add(parent modules.Parent, lengthScaled gobble.Handle, options Options) (Ports, error) {
	const unit = "deseq2_qc"
	if len(options.ExtraArgs) != 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "the DESeq2-QC R program does not accept ExtraArgs")
	}
	matrixPath, err := modules.HandlePath(unit, lengthScaled)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/deseq2-qc")
	}
	pca := gobble.PathSpec{Dir: outDir, Base: "pca", Ext: ".pdf"}
	distance := gobble.PathSpec{Dir: outDir, Base: "sample_distance", Ext: ".pdf"}
	normalized := gobble.PathSpec{Dir: outDir, Base: "vst_matrix", Ext: ".tsv"}
	pcaPath, _ := pca.Render()
	distancePath, _ := distance.Render()
	normalizedPath, _ := normalized.Render()
	command := []string{"Rscript", "-e", deseq2QCR, matrixPath, pcaPath, distancePath, normalizedPath}
	base := options.Options
	base.ExtraArgs = nil
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, gobble.Resources{CPU: 2, Memory: "4g"}, command, []string{"-e", "--design", "--contrast"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "length_scaled", From: lengthScaled}}, Outputs: []gobble.Bind{{Name: "pca", Spec: pca}, {Name: "distance", Spec: distance}, {Name: "matrix", Spec: normalized}}})
	return Ports{PCA: task.Out("pca"), Distance: task.Out("distance"), Matrix: task.Out("matrix")}, nil
}

// Pipeline returns a standalone validated DESeq2 cohort-QC module.
func Pipeline(lengthScaled gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("deseq2-qc", []modules.Input{{Name: "length_scaled", Spec: lengthScaled}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], options)
		return err
	})
}
