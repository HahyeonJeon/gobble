// Package mergecounts owns the graph-stable merge-counts command module.
package mergecounts

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

const deseq2Image = "quay.io/biocontainers/bioconductor-deseq2:1.50.2--r45ha27e39d_0"

const mergeCountsTaskName = "merge_counts"

const mergeCountsR = `args <- commandArgs(trailingOnly=TRUE)
if (length(args) >= 1L && identical(args[[1]], "--")) {
  args <- args[-1L]
}
out <- args[[1]]
n <- as.integer(args[[2]])
mat <- NULL
i <- 1L
while (i <= n) {
  name <- args[[2L + 2L * i - 1L]]
  path <- args[[2L + 2L * i]]
  fc <- read.delim(path, comment.char="#", check.names=FALSE)
  part <- fc[, c(1L, ncol(fc))]
  colnames(part) <- c("gene_id", name)
  part$gene_id <- as.character(part$gene_id)
  if (is.null(mat)) {
    mat <- part
  } else {
    mat <- merge(mat, part, by="gene_id", all=FALSE)
  }
  i <- i + 1L
}
write.csv(mat, out, row.names=FALSE)`

// MergeCountsOptions are typed merge-counts settings. ExtraArgs are
// argv tokens appended after named flags and the count-file operands.
//
// SampleNames are the count-table column names in handle-list order.
// An empty or short SampleNames list fills remaining columns as
// sample_0, sample_1, …. Default dest is work/deseq2/counts.csv.
type MergeCountsOptions struct {
	ExtraArgs   []string
	Resources   gobble.Resources
	OutDir      gobble.Directory
	SampleNames []string
}

// MergeCountsPorts are the declared gene-by-sample count table.
type MergeCountsPorts struct {
	Counts gobble.Handle
}

// AddMergeCounts records one merge-counts task on parent. counts is a
// known-length list of per-sample featureCounts tables, one regular-file
// bind each. A Group From cannot merge distinct single-file ports. The
// shared builder does not call AddInput.
func AddMergeCounts(parent modules.Parent, counts []gobble.Handle, opts MergeCountsOptions) MergeCountsPorts {
	return addMergeCounts(parent, counts, opts)
}

// MergeCountsPipeline returns a standalone merge-counts pipeline. It
// AddInputs each counts file, then calls the same builder as AddMergeCounts.
func MergeCountsPipeline(counts []gobble.PathSpec, opts MergeCountsOptions) *gobble.Pipeline {
	inputs := make([]modules.Input, len(counts))
	for i, spec := range counts {
		inputs[i] = modules.Input{Name: mergeCountsBind(i), Spec: spec}
	}
	return modules.Standalone("merge-counts", inputs, func(parent modules.Parent, hs []gobble.Handle) {
		addMergeCounts(parent, hs, opts)
	})
}

func addMergeCounts(parent modules.Parent, counts []gobble.Handle, opts MergeCountsOptions) MergeCountsPorts {
	outDir := opts.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/deseq2")
	}
	outSpec := gobble.PathSpec{Dir: outDir, Base: "counts", Ext: ".csv"}

	cmd := []string{"Rscript", "-e", mergeCountsR, "--", modules.MustCommandPath(outSpec), strconv.Itoa(len(counts))}
	inputs := make([]gobble.Bind, 0, len(counts))
	for i, h := range counts {
		inputs = append(inputs, gobble.Bind{Name: mergeCountsBind(i), From: h})
		cmd = append(cmd, mergeSampleName(opts.SampleNames, i), modules.MustCommandPath(h.Spec()))
	}
	cmd = modules.AppendLegacyExtraArgs(cmd, opts.ExtraArgs)

	task := parent.AddTask(gobble.TaskSpec{
		Name:      mergeCountsTaskName,
		Command:   cmd,
		Image:     deseq2Image,
		Inputs:    inputs,
		Outputs:   []gobble.Bind{{Name: "counts", Spec: outSpec}},
		Resources: opts.Resources,
	})
	return MergeCountsPorts{Counts: task.Out("counts")}
}

func mergeCountsBind(i int) string {
	return "counts_" + strconv.Itoa(i)
}

func mergeSampleName(names []string, i int) string {
	if i < len(names) && names[i] != "" {
		return names[i]
	}
	return "sample_" + strconv.Itoa(i)
}
