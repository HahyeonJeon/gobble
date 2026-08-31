// Package deeptoolsplotfingerprint owns one deepTools plotFingerprint command.
package deeptoolsplotfingerprint

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/atacseq 2.1.2 deepTools 3.5.1 image resolved for linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/deeptools:3.5.1--py_0@sha256:5d16e7a95afb816a455df599646ef25335c624b0a0142f4f159d6275a09aa8dc"

// Options controls one cohort fingerprint command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the fingerprint plot, raw counts, and quality metrics.
type Ports struct {
	PDF     gobble.Handle
	Raw     gobble.Handle
	Metrics gobble.Handle
}

// Add records one strict fingerprint fan-in over indexed BAMs.
func Add(parent modules.Parent, bams, bais []gobble.Handle, options Options) (Ports, error) {
	const unit = "deeptools_plot_fingerprint"
	if len(bams) == 0 || len(bams) != len(bais) {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "BAM and index membership must be non-empty and equal")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/deeptools")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "cohort"
	}
	pdf := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".plotFingerprint.pdf"}
	raw := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".plotFingerprint.raw.tsv"}
	metrics := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".plotFingerprint.qcmetrics.tsv"}
	pdfPath, err := pdf.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "fingerprint output path is invalid")
	}
	rawPath, _ := raw.Render()
	metricsPath, _ := metrics.Render()
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 2, Memory: "4g"}
	}
	threads := modules.ThreadCount(resources.CPU)
	if threads < 1 {
		threads = 1
	}
	command := []string{"plotFingerprint", "--bamfiles"}
	inputs := make([]gobble.Bind, 0, len(bams)*2)
	for i, bam := range bams {
		bamPath, pathErr := modules.HandlePath(unit, bam)
		if pathErr != nil {
			return Ports{}, pathErr
		}
		if _, pathErr = modules.HandlePath(unit, bais[i]); pathErr != nil {
			return Ports{}, pathErr
		}
		command = append(command, bamPath)
		inputs = append(inputs, gobble.Bind{Name: "bam_" + strconv.Itoa(i), From: bam}, gobble.Bind{Name: "bai_" + strconv.Itoa(i), From: bais[i]})
	}
	command = append(command, "--plotFile", pdfPath, "--outRawCounts", rawPath, "--outQualityMetrics", metricsPath, "--numberOfProcessors", strconv.Itoa(threads))
	protected := []string{"--bamfiles", "-b", "--plotFile", "-plot", "--outRawCounts", "--outQualityMetrics", "--numberOfProcessors", "-p"}
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, protected)
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{{Name: "pdf", Spec: pdf}, {Name: "raw", Spec: raw}, {Name: "metrics", Spec: metrics}}})
	return Ports{PDF: task.Out("pdf"), Raw: task.Out("raw"), Metrics: task.Out("metrics")}, nil
}
