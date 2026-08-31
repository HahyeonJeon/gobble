// Package samtoolsflagstat owns one samtools flagstat command.
package samtoolsflagstat

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the Sarek 3.10.0 samtools image resolved for linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/htslib_samtools:1.24--d697cfb9dce007cd@sha256:e994bf4eb3731150511a14f5706b7bdfd64df1b6d40898fff334286c027e0859"

// Options controls one samtools flagstat command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the flag statistics report.
type Ports struct{ Report gobble.Handle }

// Add records one validated samtools flagstat command.
func Add(parent modules.Parent, bam, bai gobble.Handle, options Options) (Ports, error) {
	const unit = "samtools_flagstat"
	bamPath, err := modules.HandlePath(unit, bam)
	if err != nil {
		return Ports{}, err
	}
	if _, err = modules.HandlePath(unit, bai); err != nil {
		return Ports{}, err
	}
	outDir, prefix := options.OutDir, options.Prefix
	if outDir.IsZero() {
		outDir = gobble.Dir("work/samtools-flagstat")
	}
	if prefix == "" {
		prefix = "alignment"
	}
	report := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".flagstat.txt"}
	reportPath, err := report.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "flagstat report path is invalid")
	}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 1, Memory: "1g"}
	}
	command := []string{"samtools", "flagstat"}
	if n := modules.ThreadCount(resources.CPU); n > 0 {
		command = append(command, "--threads", strconv.Itoa(n))
	}
	protected := []string{"--threads", "--output-fmt"}
	if err := modules.RejectExtraArgPrefixes(unit, options.ExtraArgs, protected); err != nil {
		return Ports{}, err
	}
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, protected)
	if err != nil {
		return Ports{}, err
	}
	command = append(command, bamPath)
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Script: modules.ShellRedirect(command, reportPath), Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bam", From: bam}, {Name: "bai", From: bai}}, Outputs: []gobble.Bind{{Name: "report", Spec: report}}})
	return Ports{Report: task.Out("report")}, nil
}

// Pipeline returns a standalone validated samtools flagstat module.
func Pipeline(bam, bai gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("samtools-flagstat", []modules.Input{{Name: "bam", Spec: bam}, {Name: "bai", Spec: bai}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], options)
		return err
	})
}
