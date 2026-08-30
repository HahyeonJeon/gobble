// Package tximport owns one tximeta/tximport cohort merge command.
package tximport

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

const tximportR = `
args <- commandArgs(trailingOnly=TRUE)
gtf <- args[[1]]
outdir <- args[[2]]
pairs <- args[-c(1,2)]
sample_names <- pairs[seq(1, length(pairs), 2)]
files <- pairs[seq(2, length(pairs), 2)]
names(files) <- sample_names
suppressPackageStartupMessages(library(tximport))
lines <- readLines(gtf, warn=FALSE)
lines <- lines[!grepl("^#", lines) & grepl("transcript_id", lines) & grepl("gene_id", lines)]
attr <- sub("^[^\t]*\t[^\t]*\t[^\t]*\t[^\t]*\t[^\t]*\t[^\t]*\t[^\t]*\t[^\t]*\t", "", lines)
extract <- function(x, key) sub(paste0(".*", key, "[ =\\\"]+([^;\\\"]+).*"), "\\1", x)
tx2gene <- unique(data.frame(transcript_id=extract(attr, "transcript_id"), gene_id=extract(attr, "gene_id"), stringsAsFactors=FALSE))
txi <- tximport(files, type="salmon", txOut=TRUE)
gene <- summarizeToGene(txi, tx2gene=tx2gene)
gene_ls <- summarizeToGene(txi, tx2gene=tx2gene, countsFromAbundance="lengthScaledTPM")
gene_s <- summarizeToGene(txi, tx2gene=tx2gene, countsFromAbundance="scaledTPM")
write_matrix <- function(x, file) write.table(data.frame(id=rownames(x), x, check.names=FALSE), file=file.path(outdir, file), sep="\t", quote=FALSE, row.names=FALSE)
dir.create(outdir, recursive=TRUE, showWarnings=FALSE)
write_matrix(gene$counts, "gene_counts.tsv")
write_matrix(gene$abundance, "gene_tpm.tsv")
write_matrix(gene_ls$counts, "gene_counts_length_scaled.tsv")
write_matrix(gene_s$counts, "gene_counts_scaled.tsv")
write_matrix(gene$length, "gene_lengths.tsv")
write_matrix(txi$counts, "transcript_counts.tsv")
write_matrix(txi$abundance, "transcript_tpm.tsv")
write_matrix(txi$length, "transcript_lengths.tsv")
write.table(tx2gene, file=file.path(outdir, "tx2gene_augmented.tsv"), sep="\t", quote=FALSE, row.names=FALSE)
saveRDS(list(transcript=txi, gene=gene, gene_length_scaled=gene_ls, gene_scaled=gene_s), file.path(outdir, "tximport.rds"))
`

// DefaultImage is the nf-core/rnaseq 3.26.0 tximeta image resolved for
// linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/bioconductor-tximeta:1.20.1--r43hdfd78af_0@sha256:3dfeb5c838ed192efaab23476078775ef214bd6b4472fb5b966bfc119e0b77c0"

// Options controls one complete required-cohort tximport merge. The inline R
// program uses all trailing operands for sample/file pairs, so ExtraArgs are
// rejected.
type Options struct {
	modules.Options
	OutDir gobble.Directory
}

// Ports are the selected gene, transcript, length, and R-object outputs.
type Ports struct {
	GeneCounts        gobble.Handle
	GeneTPM           gobble.Handle
	GeneLengthScaled  gobble.Handle
	GeneScaled        gobble.Handle
	GeneLengths       gobble.Handle
	TranscriptCounts  gobble.Handle
	TranscriptTPM     gobble.Handle
	TranscriptLengths gobble.Handle
	Tx2Gene           gobble.Handle
	RObject           gobble.Handle
}

// Add records one validated tximport merge over every declared quant.sf.
func Add(parent modules.Parent, quants []gobble.Handle, sampleNames []string, gtf gobble.Handle, options Options) (Ports, error) {
	const unit = "tximport"
	if len(options.ExtraArgs) != 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "the tximport R program does not accept ExtraArgs")
	}
	if len(quants) == 0 || len(quants) != len(sampleNames) {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "quant files and sample names must be non-empty and equal length")
	}
	gtfPath, err := modules.HandlePath(unit, gtf)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/salmon")
	}
	command := []string{"Rscript", "-e", tximportR, gtfPath, outDir.String()}
	inputs := []gobble.Bind{{Name: "gtf", From: gtf}}
	for i, quant := range quants {
		path, pathErr := modules.HandlePath(unit, quant)
		if pathErr != nil {
			return Ports{}, pathErr
		}
		command = append(command, sampleNames[i], path)
		inputs = append(inputs, gobble.Bind{Name: "quant_" + strconv.Itoa(i), From: quant})
	}
	base := options.Options
	base.ExtraArgs = nil
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, gobble.Resources{CPU: 2, Memory: "4g"}, command, []string{"-e"})
	if err != nil {
		return Ports{}, err
	}
	file := func(base, ext string) gobble.PathSpec { return gobble.PathSpec{Dir: outDir, Base: base, Ext: ext} }
	outputs := []gobble.Bind{
		{Name: "gene_counts", Spec: file("gene_counts", ".tsv")},
		{Name: "gene_tpm", Spec: file("gene_tpm", ".tsv")},
		{Name: "gene_length_scaled", Spec: file("gene_counts_length_scaled", ".tsv")},
		{Name: "gene_scaled", Spec: file("gene_counts_scaled", ".tsv")},
		{Name: "gene_lengths", Spec: file("gene_lengths", ".tsv")},
		{Name: "transcript_counts", Spec: file("transcript_counts", ".tsv")},
		{Name: "transcript_tpm", Spec: file("transcript_tpm", ".tsv")},
		{Name: "transcript_lengths", Spec: file("transcript_lengths", ".tsv")},
		{Name: "tx2gene", Spec: file("tx2gene_augmented", ".tsv")},
		{Name: "r_object", Spec: file("tximport", ".rds")},
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs, Outputs: outputs})
	return Ports{
		GeneCounts: task.Out("gene_counts"), GeneTPM: task.Out("gene_tpm"), GeneLengthScaled: task.Out("gene_length_scaled"), GeneScaled: task.Out("gene_scaled"), GeneLengths: task.Out("gene_lengths"),
		TranscriptCounts: task.Out("transcript_counts"), TranscriptTPM: task.Out("transcript_tpm"), TranscriptLengths: task.Out("transcript_lengths"), Tx2Gene: task.Out("tx2gene"), RObject: task.Out("r_object"),
	}, nil
}

// Pipeline returns a standalone validated tximport cohort merge.
func Pipeline(quants []gobble.PathSpec, sampleNames []string, gtf gobble.PathSpec, options Options) *gobble.Pipeline {
	inputs := []modules.Input{{Name: "gtf", Spec: gtf}}
	for i, quant := range quants {
		inputs = append(inputs, modules.Input{Name: "quant_" + strconv.Itoa(i), Spec: quant})
	}
	return modules.StandaloneChecked("tximport", inputs, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[1:], sampleNames, handles[0], options)
		return err
	})
}
