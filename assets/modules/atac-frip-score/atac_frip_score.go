// Package atacfripscore owns one awk command that reports FRiP from declared counts.
package atacfripscore

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/atacseq 2.1.2 FRiP bedtools/samtools image resolved for linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/mulled-v2-8186960447c5cb2faa697666dc1e6d919ad23f3e:3127fcae6b6bdaf8181e21a26ae61231030a9fcb-0@sha256:fb819d1b6cd6de9b710be3928a7f6e1840bc850b250cb3c33ecd06d456eab85e"

// Options controls one FRiP report.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the engineering FRiP report.
type Ports struct{ Report gobble.Handle }

// Add records one arithmetic command over strict total and in-peak count inputs.
func Add(parent modules.Parent, total, inPeaks gobble.Handle, options Options) (Ports, error) {
	const unit = "atac_frip_score"
	totalPath, err := modules.HandlePath(unit, total)
	if err != nil {
		return Ports{}, err
	}
	inPeaksPath, err := modules.HandlePath(unit, inPeaks)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/peak-qc")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "peaks"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".FRiP.tsv"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "FRiP output path is invalid")
	}
	program := `NR==FNR { total=$1; next } { inpeaks=$1 } END { if (total <= 0) exit 2; print "sample\treads_in_peaks\ttotal_reads\tfrip"; print sample "\t" inpeaks "\t" total "\t" inpeaks/total }`
	command := []string{"awk", "-v", "sample=" + prefix, program, totalPath, inPeaksPath}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "256m"}, command, []string{"-v", "-f"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: modules.ShellRedirect(command, outputPath), Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "total", From: total}, {Name: "in_peaks", From: inPeaks}}, Outputs: []gobble.Bind{{Name: "report", Spec: output}}})
	return Ports{Report: task.Out("report")}, nil
}
