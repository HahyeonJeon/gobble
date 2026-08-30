// Package deseq2 owns the graph-stable DESeq2 command module.
package deseq2

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// deseq2Image is the biocontainers pin for merge counts and DESeq2.
const deseq2Image = "quay.io/biocontainers/bioconductor-deseq2:1.50.2--r45ha27e39d_0"

const deseq2TaskName = "deseq2"

const deseq2R = `args <- commandArgs(trailingOnly=TRUE)
if (length(args) >= 1L && identical(args[[1]], "--")) {
  args <- args[-1L]
}
counts_path <- args[[1]]
out <- args[[2]]
n <- as.integer(args[[3]])
groups <- args[seq_len(n) + 3L]
counts <- read.csv(counts_path, row.names=1, check.names=FALSE)
if (ncol(counts) != n) {
  stop("group count must match sample columns")
}
samples <- colnames(counts)
col <- data.frame(group=groups, row.names=samples, check.names=FALSE)
lev <- sort(unique(groups))
col$group <- factor(col$group, levels=lev)
suppressPackageStartupMessages(library(DESeq2))
dds <- DESeqDataSetFromMatrix(countData=round(as.matrix(counts)), colData=col, design=~ group)
dds <- DESeq(dds)
res <- results(dds, contrast=c("group", as.character(lev[[2]]), as.character(lev[[1]])))
tab <- as.data.frame(res)
tab$gene_id <- rownames(tab)
tab <- tab[, c("gene_id", "baseMean", "log2FoldChange", "lfcSE", "stat", "pvalue", "padj")]
write.csv(tab, out, row.names=FALSE)`

// DESeq2Options are typed DESeq2 settings. ExtraArgs are argv tokens
// appended after named flags and the counts, dest, and group operands.
//
// Default dest is work/deseq2/results.csv. results.csv columns are
// gene_id, baseMean, log2FoldChange, lfcSE, stat, pvalue, and padj.
type DESeq2Options struct {
	ExtraArgs []string
	Resources gobble.Resources
	OutDir    gobble.Directory
}

// DESeq2Ports are the declared two-group contrast table.
type DESeq2Ports struct {
	Results gobble.Handle
}

// AddDESeq2 records one DESeq2 task on parent. counts is the merged
// gene-by-sample table. groups are the two-group labels in sample-column
// order. Design is ~ group. The contrast is the lexicographically first
// group (reference) against the other group (target). The shared builder
// does not call AddInput.
func AddDESeq2(parent modules.Parent, counts gobble.Handle, groups []string, opts DESeq2Options) DESeq2Ports {
	return addDESeq2(parent, counts, groups, opts)
}

// DESeq2Pipeline returns a standalone DESeq2 pipeline. It AddInputs
// counts, then calls the same builder as AddDESeq2.
func DESeq2Pipeline(counts gobble.PathSpec, groups []string, opts DESeq2Options) *gobble.Pipeline {
	return modules.Standalone("deseq2", []modules.Input{{Name: "counts", Spec: counts}}, func(parent modules.Parent, hs []gobble.Handle) {
		addDESeq2(parent, hs[0], groups, opts)
	})
}

func addDESeq2(parent modules.Parent, counts gobble.Handle, groups []string, opts DESeq2Options) DESeq2Ports {
	outDir := opts.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/deseq2")
	}
	outSpec := gobble.PathSpec{Dir: outDir, Base: "results", Ext: ".csv"}

	cmd := []string{
		"Rscript", "-e", deseq2R, "--",
		modules.MustCommandPath(counts.Spec()),
		modules.MustCommandPath(outSpec),
		strconv.Itoa(len(groups)),
	}
	cmd = append(cmd, groups...)
	cmd = modules.AppendLegacyExtraArgs(cmd, opts.ExtraArgs)

	task := parent.AddTask(gobble.TaskSpec{
		Name:      deseq2TaskName,
		Command:   cmd,
		Image:     deseq2Image,
		Inputs:    []gobble.Bind{{Name: "counts", From: counts}},
		Outputs:   []gobble.Bind{{Name: "results", Spec: outSpec}},
		Resources: opts.Resources,
	})
	return DESeq2Ports{Results: task.Out("results")}
}
