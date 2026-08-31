// Package gatk4haplotypecaller owns one GATK HaplotypeCaller command.
package gatk4haplotypecaller

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// Options controls one interval-scattered HaplotypeCaller command.
type Options struct {
	modules.Options
	IntervalDir gobble.Directory
	OutDir      gobble.Directory
}

// Ports contains one indexed interval gVCF.
type Ports struct {
	GVCF gobble.Handle
	TBI  gobble.Handle
}

// Add records one validated gVCF-mode HaplotypeCaller command.
func Add(parent modules.Parent, bam, bai, fasta, fai, dict, dbsnp, dbsnpTBI, interval gobble.Handle, options Options) (Ports, error) {
	const unit = "gatk4_haplotypecaller"
	paths := make([]string, 0, 7)
	for _, handle := range []gobble.Handle{bam, bai, fasta, fai, dict, dbsnp, dbsnpTBI} {
		path, err := modules.HandlePath(unit, handle)
		if err != nil {
			return Ports{}, err
		}
		paths = append(paths, path)
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/gatk4-haplotypecaller")
	}
	prelude, err := modules.ScatterFilePrelude(unit, options.IntervalDir)
	if err != nil {
		return Ports{}, err
	}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 2, Memory: "4g"}
	}
	protected := []string{"--input", "--output", "--reference", "--dbsnp", "--intervals", "--emit-ref-confidence", "--native-pair-hmm-threads", "--tmp-dir"}
	base := options.Options
	base.Resources = resources
	extra, image, resources, err := modules.ResolveGATK4Options(unit, base, resources, protected)
	if err != nil {
		return Ports{}, err
	}
	command := []string{"gatk", "HaplotypeCaller", "--input", paths[0], "--reference", paths[2], "--dbsnp", paths[5], "--emit-ref-confidence", "GVCF", "--native-pair-hmm-threads", strconv.Itoa(modules.ThreadCount(resources.CPU)), "--tmp-dir", "."}
	script := prelude + "output=" + modules.ShellQuote(outDir.String()) + "/$stem.g.vcf.gz\n" + modules.ShellCommand(command) + " --intervals \"$interval\" --output \"$output\""
	if len(extra) > 0 {
		script += " " + modules.ShellCommand(extra)
	}
	gvcf := gobble.Bind{Name: "gvcf", From: interval, Spec: gobble.PathSpec{Dir: outDir, Ext: ".g.vcf.gz"}, Rule: gobble.DeriveReplaceExt}
	tbi := gobble.Bind{Name: "tbi", From: interval, Spec: gobble.PathSpec{Dir: outDir, Ext: ".g.vcf.gz.tbi"}, Rule: gobble.DeriveReplaceExt}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: script, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bam", From: bam}, {Name: "bai", From: bai}, {Name: "fasta", From: fasta}, {Name: "fai", From: fai}, {Name: "dict", From: dict}, {Name: "dbsnp", From: dbsnp}, {Name: "dbsnp_tbi", From: dbsnpTBI}, {Name: "interval", From: interval}}, Outputs: []gobble.Bind{gvcf, tbi}})
	return Ports{GVCF: task.Out("gvcf"), TBI: task.Out("tbi")}, nil
}

// Pipeline returns a standalone validated one-interval HaplotypeCaller module.
func Pipeline(bam, bai, fasta, fai, dict, dbsnp, dbsnpTBI, interval gobble.PathSpec, options Options) *gobble.Pipeline {
	inputs := []modules.Input{{Name: "bam", Spec: bam}, {Name: "bai", Spec: bai}, {Name: "fasta", Spec: fasta}, {Name: "fai", Spec: fai}, {Name: "dict", Spec: dict}, {Name: "dbsnp", Spec: dbsnp}, {Name: "dbsnp_tbi", Spec: dbsnpTBI}, {Name: "intervals", Group: gobble.Group{{Name: interval.Base, Spec: interval}}}}
	options.IntervalDir = interval.Dir
	return modules.StandaloneScatterChecked("gatk4-haplotypecaller", "intervals", inputs, 7, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], handles[2], handles[3], handles[4], handles[5], handles[6], handles[7], options)
		return err
	})
}
