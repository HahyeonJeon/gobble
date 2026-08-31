// Package bismarkdeduplicate owns one Bismark deduplicate_bismark command.
package bismarkdeduplicate

import (
	"strings"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/methylseq 4.2.0 Bismark image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/bismark:0.25.1--1f50935de5d79c47@sha256:7b49e02b15de6fd59643224db5defb229433de4aebee982d6a03b612077755a0"

// Options controls one Bismark deduplication command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports are the deduplicated BAM and Bismark deduplication report.
type Ports struct {
	BAM    gobble.Handle
	Report gobble.Handle
}

// Add records one validated deduplicate_bismark command.
func Add(parent modules.Parent, bam gobble.Handle, paired bool, options Options) (Ports, error) {
	const unit = "bismark_deduplicate"
	bamPath, err := modules.HandlePath(unit, bam)
	if err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/bismark-deduplicate")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = alignmentStem(bamPath)
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix + ".deduplicated", Ext: ".bam"}
	report := gobble.PathSpec{Dir: outDir, Base: prefix + ".deduplication_report", Ext: ".txt"}
	if _, err := output.Render(); err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "Bismark deduplication output path is invalid")
	}
	protected := []string{
		"--sam", "--multiple", "--barcode", "--umi", "--version", "--help",
		"-s", "--single", "-p", "--paired", "--bam", "--output_dir", "-o", "--outfile",
	}
	if err := modules.RejectExtraArgPrefixes(unit, options.ExtraArgs, protected); err != nil {
		return Ports{}, err
	}
	mode := "--single"
	if paired {
		mode = "--paired"
	}
	command := []string{"deduplicate_bismark", mode, "--bam", "--output_dir", outDir.String(), "--outfile", prefix}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "4g"}, command, []string{"-s", "--single", "-p", "--paired", "--bam", "--output_dir", "-o", "--outfile"})
	if err != nil {
		return Ports{}, err
	}
	command = append(command, bamPath)
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bam", From: bam}}, Outputs: []gobble.Bind{{Name: "deduplicated_bam", Spec: output}, {Name: "report", Spec: report}}})
	return Ports{BAM: task.Out("deduplicated_bam"), Report: task.Out("report")}, nil
}

// Pipeline returns a standalone validated Bismark deduplication module.
func Pipeline(bam gobble.PathSpec, paired bool, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("bismark-deduplicate", []modules.Input{{Name: "bam", Spec: bam}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], paired, options)
		return err
	})
}

func alignmentStem(value string) string {
	base := value
	if split := strings.LastIndexByte(base, '/'); split >= 0 {
		base = base[split+1:]
	}
	for _, suffix := range []string{".bam", ".sam", ".cram"} {
		if strings.HasSuffix(strings.ToLower(base), suffix) {
			return base[:len(base)-len(suffix)]
		}
	}
	return base
}
