// Package gatk4markduplicates owns one GATK MarkDuplicates command.
package gatk4markduplicates

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the Sarek 3.10.0 MarkDuplicates image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/gatk4_gcnvkernel_htslib_samtools:d3becb6465454c35@sha256:e3d753d93f57969fe76b8628a8dfcd23ef44bccd08c4ced7089c1f94bf47c89f"

// Options controls one GATK MarkDuplicates command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the duplicate-marked BAM, BAI, and metrics.
type Ports struct {
	BAM     gobble.Handle
	BAI     gobble.Handle
	Metrics gobble.Handle
}

// Add records one validated GATK MarkDuplicates command.
func Add(parent modules.Parent, bam, inputBAI gobble.Handle, options Options) (Ports, error) {
	const unit = "gatk4_markduplicates"
	bamPath, err := modules.HandlePath(unit, bam)
	if err != nil {
		return Ports{}, err
	}
	if _, err = modules.HandlePath(unit, inputBAI); err != nil {
		return Ports{}, err
	}
	outDir, prefix := options.OutDir, options.Prefix
	if outDir.IsZero() {
		outDir = gobble.Dir("work/gatk4-markduplicates")
	}
	if prefix == "" {
		prefix = "marked"
	}
	marked := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".bam"}
	bai := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".bai"}
	metrics := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".markduplicates.metrics"}
	markedPath, err := marked.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "marked BAM path is invalid")
	}
	metricsPath, err := metrics.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "duplicate metrics path is invalid")
	}
	protected := []string{"--INPUT", "--OUTPUT", "--METRICS_FILE", "--CREATE_INDEX", "--TMP_DIR"}
	base := options.Options
	if base.Image == "" {
		base.Image = DefaultImage
	}
	extra, image, resources, err := modules.ResolveGATK4Options(unit, base, gobble.Resources{CPU: 2, Memory: "6g"}, protected)
	if err != nil {
		return Ports{}, err
	}
	command := []string{"gatk", "MarkDuplicates", "--INPUT", bamPath, "--OUTPUT", markedPath, "--METRICS_FILE", metricsPath, "--CREATE_INDEX", "true", "--TMP_DIR", "."}
	command = append(command, extra...)
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bam", From: bam}, {Name: "input_bai", From: inputBAI}}, Outputs: []gobble.Bind{{Name: "marked_bam", Spec: marked}, {Name: "bai", Spec: bai}, {Name: "metrics", Spec: metrics}}})
	return Ports{BAM: task.Out("marked_bam"), BAI: task.Out("bai"), Metrics: task.Out("metrics")}, nil
}

// Pipeline returns a standalone validated MarkDuplicates module.
func Pipeline(bam, bai gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("gatk4-markduplicates", []modules.Input{{Name: "bam", Spec: bam}, {Name: "bai", Spec: bai}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], options)
		return err
	})
}
