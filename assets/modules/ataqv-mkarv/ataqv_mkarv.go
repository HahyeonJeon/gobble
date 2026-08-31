// Package ataqvmkarv owns one mkarv command.
package ataqvmkarv

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/atacseq 2.1.2 ataqv 1.3.1 image resolved for linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/ataqv:1.3.1--py310ha155cf9_1@sha256:cb5ff55de7d503a959eea0bc219067012ad9ac8fdb43c7e563b3909a28a5159a"

// Options controls one mkarv command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
}

// Ports contains the complete browser-report Tree.
type Ports struct{ HTML gobble.Handle }

// Add records one strict mkarv fan-in over every ataqv JSON report.
func Add(parent modules.Parent, reports []gobble.Handle, options Options) (Ports, error) {
	const unit = "ataqv_mkarv"
	if len(reports) == 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "ataqv report membership must not be empty")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/ataqv/html")
	}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 2, Memory: "4g"}
	}
	threads := modules.ThreadCount(resources.CPU)
	if threads < 1 {
		threads = 1
	}
	command := []string{"mkarv", "--concurrency", strconv.Itoa(threads), "--force", outDir.String()}
	inputs := make([]gobble.Bind, len(reports))
	for i, report := range reports {
		path, err := modules.HandlePath(unit, report)
		if err != nil {
			return Ports{}, err
		}
		command = append(command, path)
		inputs[i] = gobble.Bind{Name: "json_" + strconv.Itoa(i), From: report}
	}
	protected := []string{"--concurrency", "--force"}
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, protected)
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{{Name: "html", Tree: gobble.DeclareTree(outDir)}}})
	return Ports{HTML: task.Out("html")}, nil
}
