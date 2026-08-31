// Package plotmacs2qc owns one R command that plots MACS2 peak quality control.
package plotmacs2qc

import (
	"strconv"
	"strings"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/atacseq 2.1.2 peak-QC R image resolved for linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/mulled-v2-ad9dd5f398966bf899ae05f8e7c54d0fb10cdfa7:05678da05b8e5a7a5130e90a9f9a6c585b965afa-0@sha256:7965e3c784e5ea0399ddaab5fab08dd916d13e0e5f0a7b47675b4c719149ee7d"

const plotR = `
args <- commandArgs(trailingOnly=TRUE)
if (length(args) < 4 || length(args) %% 2 != 0) stop("MACS2 QC needs output paths and matched sample/peak pairs")
summary_path <- args[[1]]
plot_path <- args[[2]]
labels <- args[seq(3, length(args), by=2)]
paths <- args[seq(4, length(args), by=2)]
if (any(labels == "") || anyDuplicated(labels)) stop("MACS2 QC sample labels must be non-empty and unique")
peaks <- lapply(paths, function(path) {
  value <- read.delim(path, header=FALSE, comment.char="#", stringsAsFactors=FALSE)
  if (ncol(value) < 9 || nrow(value) == 0) stop(paste("invalid or empty MACS2 peak file:", path))
  data.frame(length=pmax(1, value[[3]] - value[[2]]), fold=value[[7]], pvalue=value[[8]], qvalue=value[[9]])
})
names(peaks) <- labels
summaries <- do.call(rbind, lapply(seq_along(peaks), function(index) {
  do.call(rbind, lapply(names(peaks[[index]]), function(measure) {
    values <- peaks[[index]][[measure]]
    data.frame(sample=labels[[index]], measure=measure, minimum=min(values), first_quartile=unname(quantile(values, 0.25)), median=median(values), mean=mean(values), third_quartile=unname(quantile(values, 0.75)), maximum=max(values), num_peaks=length(values), check.names=FALSE)
  }))
}))
write.table(summaries, summary_path, sep="\t", quote=FALSE, row.names=FALSE)
pdf(plot_path, width=max(6, 3 * length(labels)), height=9)
par(mfrow=c(3, 2), mar=c(8, 4, 3, 1))
barplot(setNames(vapply(peaks, nrow, integer(1)), labels), las=2, ylab="Number of peaks", main="Peak count")
boxplot(lapply(peaks, function(value) log10(value$length)), names=labels, las=2, ylab="log10 peak length", main="Peak length distribution")
boxplot(lapply(peaks, function(value) log2(pmax(value$fold, .Machine$double.eps))), names=labels, las=2, ylab="log2 fold-enrichment", main="Fold-change distribution")
boxplot(lapply(peaks, function(value) value$qvalue), names=labels, las=2, ylab="-log10 qvalue", main="FDR distribution")
boxplot(lapply(peaks, function(value) value$pvalue), names=labels, las=2, ylab="-log10 pvalue", main="Pvalue distribution")
plot.new()
dev.off()
`

// Options controls one strict MACS2 peak-QC plot command. The inline program
// models every accepted operand, so ExtraArgs are unsupported.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the MACS2 summary table and plot document.
type Ports struct {
	Summary gobble.Handle
	PDF     gobble.Handle
}

// Add records one strict plot command over matched peak sets and sample labels.
func Add(parent modules.Parent, peaks []gobble.Handle, labels []string, options Options) (Ports, error) {
	const unit = "plot_macs2_qc"
	if message := membershipError(peaks, labels); message != "" {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, message)
	}
	if len(options.ExtraArgs) != 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "MACS2 QC plot operands are typed and ExtraArgs are unsupported")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/peak-qc")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "macs2_peak"
	}
	summary := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".summary.txt"}
	pdf := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".plots.pdf"}
	summaryPath, err := summary.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "MACS2 QC output path is invalid")
	}
	pdfPath, err := pdf.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "MACS2 QC output path is invalid")
	}
	command := []string{"Rscript", "-e", plotR, summaryPath, pdfPath}
	inputs := make([]gobble.Bind, len(peaks))
	for i, peak := range peaks {
		path, pathErr := modules.HandlePath(unit, peak)
		if pathErr != nil {
			return Ports{}, pathErr
		}
		command = append(command, labels[i], path)
		inputs[i] = gobble.Bind{Name: "peaks_" + strconv.Itoa(i), From: peak}
	}
	base := options.Options
	base.ExtraArgs = nil
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, gobble.Resources{CPU: 2, Memory: "4g"}, command, []string{"-e"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{{Name: "summary", Spec: summary}, {Name: "pdf", Spec: pdf}}})
	return Ports{Summary: task.Out("summary"), PDF: task.Out("pdf")}, nil
}

// Pipeline returns a standalone validated MACS2 peak-QC plot module.
func Pipeline(peaks []gobble.PathSpec, labels []string, options Options) *gobble.Pipeline {
	inputs := make([]modules.Input, len(peaks))
	for i, peak := range peaks {
		inputs[i] = modules.Input{Name: "peaks_" + strconv.Itoa(i), Spec: peak}
	}
	return modules.StandaloneChecked("plot-macs2-qc", inputs, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles, append([]string(nil), labels...), options)
		return err
	})
}

func membershipError(peaks []gobble.Handle, labels []string) string {
	if len(peaks) == 0 || len(peaks) != len(labels) {
		return "peak and sample-label membership must be non-empty and equal"
	}
	seen := make(map[string]bool, len(labels))
	for _, label := range labels {
		if strings.TrimSpace(label) == "" || strings.ContainsAny(label, "\r\n\t") || seen[label] {
			return "sample labels must be non-empty and unique"
		}
		seen[label] = true
	}
	return ""
}
