package assets

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
)

// multiqcImage is the biocontainers tag for the nf-core MultiQC 1.35 pin.
const multiqcImage = "quay.io/biocontainers/multiqc:1.35--pyhdfd78af_0"

const multiqcTaskName = "multiqc"

// MultiQCOptions are typed MultiQC settings. ExtraArgs are argv tokens
// appended after named flags.
//
// MultiQC has no thread flag. Resources.CPU is not copied into Command.
// --zip-data-dir publishes the data directory as a regular zip file.
type MultiQCOptions struct {
	ExtraArgs []string
	Resources gobble.Resources
	OutDir    gobble.Directory
}

// MultiQCPorts are the declared html and data regular-file outputs.
type MultiQCPorts struct {
	HTML gobble.Handle
	Data gobble.Handle
}

// AddMultiQC records one MultiQC task on parent. reports is a known-length
// list of declared report files, one regular-file bind each. A Group From
// cannot merge distinct single-file ports. The shared builder does not
// call AddInput.
func AddMultiQC(parent Parent, reports []gobble.Handle, opts MultiQCOptions) MultiQCPorts {
	return addMultiQC(parent, reports, opts)
}

// MultiQCPipeline returns a standalone MultiQC pipeline. It AddInputs
// each report, then calls the same builder as AddMultiQC.
func MultiQCPipeline(reports []gobble.PathSpec, opts MultiQCOptions) *gobble.Pipeline {
	inputs := make([]Input, len(reports))
	for i, spec := range reports {
		inputs[i] = Input{Name: multiqcReportBind(i), Spec: spec}
	}
	return Standalone("multiqc", inputs, func(parent Parent, hs []gobble.Handle) {
		addMultiQC(parent, hs, opts)
	})
}

func addMultiQC(parent Parent, reports []gobble.Handle, opts MultiQCOptions) MultiQCPorts {
	outDir := opts.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/multiqc")
	}
	htmlSpec := gobble.PathSpec{Dir: outDir, Base: "multiqc_report", Ext: ".html"}
	dataSpec := gobble.PathSpec{Dir: outDir, Base: "multiqc_data", Ext: ".zip"}

	cmd := []string{"multiqc", "--force", "--outdir", outDir.String(), "--zip-data-dir"}
	inputs := make([]gobble.Bind, 0, len(reports))
	for i, report := range reports {
		inputs = append(inputs, gobble.Bind{Name: multiqcReportBind(i), From: report})
		cmd = append(cmd, mustCommandPath(report.Spec()))
	}
	cmd = AppendExtraArgs(cmd, opts.ExtraArgs)

	task := AddTask(parent, gobble.TaskSpec{
		Name:    multiqcTaskName,
		Command: cmd,
		Image:   multiqcImage,
		Inputs:  inputs,
		Outputs: []gobble.Bind{
			{Name: "html", Spec: htmlSpec},
			{Name: "data", Spec: dataSpec},
		},
		Resources: opts.Resources,
	})
	return MultiQCPorts{HTML: task.Out("html"), Data: task.Out("data")}
}

func multiqcReportBind(i int) string {
	return "report_" + strconv.Itoa(i)
}
