package multiqc

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 MultiQC image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/multiqc:1.33--ee7739d47738383b@sha256:abd5751768f8dadb626cd9698d3d11be8f1b6458074757df57c08a0b909dac93"

// Options controls one lifted MultiQC command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
}

// Ports are the aggregate HTML report and data Tree.
type Ports struct {
	HTML gobble.Handle
	Data gobble.Handle
}

// Add records one validated MultiQC command over declared report files.
func Add(parent modules.Parent, reports []gobble.Handle, options Options) (Ports, error) {
	const unit = "multiqc"
	if len(reports) == 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "reports must not be empty")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/multiqc")
	}
	command := []string{"multiqc", "--force", "--outdir", outDir.String(), "."}
	inputs := make([]gobble.Bind, len(reports))
	for i, report := range reports {
		if _, err := modules.HandlePath(unit, report); err != nil {
			return Ports{}, err
		}
		inputs[i] = gobble.Bind{Name: "report_" + strconv.Itoa(i), From: report}
	}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 1, Memory: "2g"}, command, []string{"--force", "--outdir"})
	if err != nil {
		return Ports{}, err
	}
	html := gobble.PathSpec{Dir: outDir, Base: "multiqc_report", Ext: ".html"}
	dataDir := gobble.Dir(outDir.String() + "/multiqc_data")
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{{Name: "html", Spec: html}, {Name: "data", Tree: gobble.DeclareTree(dataDir)}}})
	return Ports{HTML: task.Out("html"), Data: task.Out("data")}, nil
}

// ProductPipeline returns a standalone validated lifted MultiQC module.
func ProductPipeline(reports []gobble.PathSpec, options Options) *gobble.Pipeline {
	inputs := make([]modules.Input, len(reports))
	for i, report := range reports {
		inputs[i] = modules.Input{Name: "report_" + strconv.Itoa(i), Spec: report}
	}
	return modules.StandaloneChecked("multiqc", inputs, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles, options)
		return err
	})
}
