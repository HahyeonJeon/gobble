// Package deeptoolscomputematrix owns one deepTools computeMatrix command.
package deeptoolscomputematrix

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/atacseq 2.1.2 deepTools 3.5.1 image resolved for linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/deeptools:3.5.1--py_0@sha256:5d16e7a95afb816a455df599646ef25335c624b0a0142f4f159d6275a09aa8dc"

// Options controls one reference-point coverage matrix.
type Options struct {
	modules.Options
	OutDir     gobble.Directory
	Prefix     string
	Upstream   int
	Downstream int
}

// Ports contains the compressed matrix and tabular values.
type Ports struct {
	Matrix gobble.Handle
	Table  gobble.Handle
}

// Add records one strict matrix fan-in over every track and region set.
func Add(parent modules.Parent, tracks, regions []gobble.Handle, options Options) (Ports, error) {
	const unit = "deeptools_compute_matrix"
	if len(tracks) == 0 || len(regions) == 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "track and region membership must not be empty")
	}
	if options.Upstream < 0 || options.Downstream < 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "matrix windows must not be negative")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/deeptools")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "coverage"
	}
	matrix := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".computeMatrix.mat.gz"}
	table := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".computeMatrix.values.tsv"}
	matrixPath, err := matrix.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "matrix output path is invalid")
	}
	tablePath, _ := table.Render()
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 2, Memory: "4g"}
	}
	threads := modules.ThreadCount(resources.CPU)
	if threads < 1 {
		threads = 1
	}
	command := []string{"computeMatrix", "reference-point", "--referencePoint", "TSS", "--beforeRegionStartLength", strconv.Itoa(options.Upstream), "--afterRegionStartLength", strconv.Itoa(options.Downstream), "--regionsFileName"}
	inputs := make([]gobble.Bind, 0, len(tracks)+len(regions))
	for i, region := range regions {
		regionPath, pathErr := modules.HandlePath(unit, region)
		if pathErr != nil {
			return Ports{}, pathErr
		}
		command = append(command, regionPath)
		inputs = append(inputs, gobble.Bind{Name: "region_" + strconv.Itoa(i), From: region})
	}
	command = append(command, "--scoreFileName")
	for i, track := range tracks {
		trackPath, pathErr := modules.HandlePath(unit, track)
		if pathErr != nil {
			return Ports{}, pathErr
		}
		command = append(command, trackPath)
		inputs = append(inputs, gobble.Bind{Name: "track_" + strconv.Itoa(i), From: track})
	}
	command = append(command, "--outFileName", matrixPath, "--outFileNameMatrix", tablePath, "--numberOfProcessors", strconv.Itoa(threads))
	protected := []string{"--regionsFileName", "-R", "--scoreFileName", "-S", "--outFileName", "-o", "--outFileNameMatrix", "--numberOfProcessors", "-p", "--referencePoint", "--beforeRegionStartLength", "--afterRegionStartLength"}
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, protected)
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{{Name: "matrix", Spec: matrix}, {Name: "table", Spec: table}}})
	return Ports{Matrix: task.Out("matrix"), Table: task.Out("table")}, nil
}
