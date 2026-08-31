// Package bismarksummary owns one Bismark bismark2summary command.
package bismarksummary

import (
	"path"
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/methylseq 4.2.0 Bismark image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/bismark:0.25.1--1f50935de5d79c47@sha256:7b49e02b15de6fd59643224db5defb229433de4aebee982d6a03b612077755a0"

// SampleReports names one sample's alignment and report inputs. BAM is the
// original Bismark alignment BAM whose basename relates the reports.
type SampleReports struct {
	BAM                 gobble.Handle
	AlignmentReport     gobble.Handle
	DeduplicationReport gobble.Handle
	SplittingReport     gobble.Handle
	MBiasReport         gobble.Handle
}

// SampleInputs names one sample's standalone regular-file inputs.
type SampleInputs struct {
	BAM                 gobble.PathSpec
	AlignmentReport     gobble.PathSpec
	DeduplicationReport gobble.PathSpec
	SplittingReport     gobble.PathSpec
	MBiasReport         gobble.PathSpec
}

// Options controls one run-level Bismark summary command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the Bismark run summary text and HTML reports.
type Ports struct {
	HTML gobble.Handle
	Text gobble.Handle
}

// Add records one bismark2summary command. Inputs are restaged together so the
// command's documented basename discovery sees every related report.
func Add(parent modules.Parent, samples []SampleReports, options Options) (Ports, error) {
	const unit = "bismark_summary"
	if len(samples) == 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "sample reports must not be empty")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/methylseq/summary")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "bismark_summary_report"
	}
	html := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".html"}
	text := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".txt"}
	if _, err := html.Render(); err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "Bismark summary output path is invalid")
	}
	basename := outDir.String() + "/" + prefix
	if err := modules.RejectExtraArgPrefixes(unit, options.ExtraArgs, []string{"--version", "--help", "-o", "--basename"}); err != nil {
		return Ports{}, err
	}
	command := []string{"bismark2summary", "--basename", basename}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "2g"}, command, []string{"-o", "--basename"})
	if err != nil {
		return Ports{}, err
	}
	stageDir := gobble.Dir("work/bismark-summary-inputs")
	inputs := make([]gobble.Bind, 0, len(samples)*5)
	for i, sample := range samples {
		index := strconv.Itoa(i + 1)
		entries := []struct {
			name   string
			handle gobble.Handle
		}{
			{name: "bam_" + index, handle: sample.BAM},
			{name: "alignment_report_" + index, handle: sample.AlignmentReport},
			{name: "deduplication_report_" + index, handle: sample.DeduplicationReport},
			{name: "splitting_report_" + index, handle: sample.SplittingReport},
			{name: "mbias_report_" + index, handle: sample.MBiasReport},
		}
		for _, entry := range entries {
			inputPath, pathErr := modules.HandlePath(unit, entry.handle)
			if pathErr != nil {
				return Ports{}, pathErr
			}
			staged := gobble.Literal(path.Base(inputPath)).WithDir(stageDir)
			inputs = append(inputs, gobble.Bind{Name: entry.name, From: entry.handle, Spec: staged})
			if entry.name == "bam_"+index {
				stagedPath, _ := staged.Render()
				command = append(command, stagedPath)
			}
		}
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{{Name: "html", Spec: html}, {Name: "text", Spec: text}}})
	return Ports{HTML: task.Out("html"), Text: task.Out("text")}, nil
}

// Pipeline returns a standalone validated Bismark run-summary module.
func Pipeline(samples []SampleInputs, options Options) *gobble.Pipeline {
	inputs := make([]modules.Input, 0, len(samples)*5)
	for i, sample := range samples {
		index := strconv.Itoa(i + 1)
		inputs = append(inputs,
			modules.Input{Name: "bam_" + index, Spec: sample.BAM},
			modules.Input{Name: "alignment_report_" + index, Spec: sample.AlignmentReport},
			modules.Input{Name: "deduplication_report_" + index, Spec: sample.DeduplicationReport},
			modules.Input{Name: "splitting_report_" + index, Spec: sample.SplittingReport},
			modules.Input{Name: "mbias_report_" + index, Spec: sample.MBiasReport},
		)
	}
	return modules.StandaloneChecked("bismark-summary", inputs, func(parent modules.Parent, handles []gobble.Handle) error {
		reports := make([]SampleReports, len(samples))
		for i := range samples {
			offset := i * 5
			reports[i] = SampleReports{BAM: handles[offset], AlignmentReport: handles[offset+1], DeduplicationReport: handles[offset+2], SplittingReport: handles[offset+3], MBiasReport: handles[offset+4]}
		}
		_, err := Add(parent, reports, options)
		return err
	})
}
