// Package bcftoolsstats owns one bcftools stats command.
package bcftoolsstats

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the Sarek 3.10.0 bcftools image resolved for linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/bcftools_htslib:1.23.1--9f08ec665533d64a@sha256:0b4d52ca9a56d07be3f78a12af654e5116f5112908dba277e6796fd9dfb83fe5"

// Options controls one bcftools stats command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the callset statistics report.
type Ports struct{ Stats gobble.Handle }

// Add records one validated bcftools stats command.
func Add(parent modules.Parent, vcf, tbi, fasta gobble.Handle, options Options) (Ports, error) {
	const unit = "bcftools_stats"
	vcfPath, err := modules.HandlePath(unit, vcf)
	if err != nil {
		return Ports{}, err
	}
	if _, err = modules.HandlePath(unit, tbi); err != nil {
		return Ports{}, err
	}
	fastaPath, err := modules.HandlePath(unit, fasta)
	if err != nil {
		return Ports{}, err
	}
	outDir, prefix := options.OutDir, options.Prefix
	if outDir.IsZero() {
		outDir = gobble.Dir("results/wgs/qc/callset")
	}
	if prefix == "" {
		prefix = "joint_germline"
	}
	stats := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".bcftools_stats.txt"}
	statsPath, err := stats.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "callset statistics path is invalid")
	}
	protected := []string{"--fasta-ref", "--regions-file", "--targets-file", "--samples-file", "--exons"}
	if err := modules.RejectExtraArgPrefixes(unit, options.ExtraArgs, protected); err != nil {
		return Ports{}, err
	}
	command := []string{"bcftools", "stats", "--fasta-ref", fastaPath}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "2g"}, command, protected)
	if err != nil {
		return Ports{}, err
	}
	command = append(command, vcfPath)
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: modules.ShellRedirect(command, statsPath), Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "vcf", From: vcf}, {Name: "tbi", From: tbi}, {Name: "fasta", From: fasta}}, Outputs: []gobble.Bind{{Name: "stats", Spec: stats}}})
	return Ports{Stats: task.Out("stats")}, nil
}

// Pipeline returns a standalone validated bcftools stats module.
func Pipeline(vcf, tbi, fasta gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("bcftools-stats", []modules.Input{{Name: "vcf", Spec: vcf}, {Name: "tbi", Spec: tbi}, {Name: "fasta", Spec: fasta}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], handles[2], options)
		return err
	})
}
