// Package simpleafquant owns one Simpleaf mapping and quantification command.
package simpleafquant

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/scrnaseq 4.2.0 SIMPLEAF_QUANT image resolved for
// linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/simpleaf:0.19.5--ha6fb395_0@sha256:3e2971957942246f54d8fe55b43a6dfae242a641114805f20aca147a687d73f9"

// Options controls the typed Simpleaf chemistry, UMI resolution, output root,
// and command policy. Cell filtering is fixed to an unfiltered permit list so
// QCatch remains the selected filter owner.
type Options struct {
	modules.Options
	Chemistry  string
	Resolution string
	OutDir     gobble.Directory
}

// Ports contains both complete command-produced directory authorities.
type Ports struct {
	Map   gobble.Handle
	Quant gobble.Handle
}

// Add records one Simpleaf quant command over a complete index Tree and paired
// consolidated reads.
func Add(parent modules.Parent, index, t2g, whitelist, read1, read2 gobble.Handle, options Options) (Ports, error) {
	const unit = "simpleaf_quant"
	if index.IsZero() || index.Tree().IsZero() || index.Tree().Dir.IsZero() {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "complete Simpleaf index Tree is required")
	}
	t2gPath, err := modules.HandlePath(unit, t2g)
	if err != nil {
		return Ports{}, err
	}
	whitelistPath, err := modules.HandlePath(unit, whitelist)
	if err != nil {
		return Ports{}, err
	}
	read1Path, err := modules.HandlePath(unit, read1)
	if err != nil {
		return Ports{}, err
	}
	read2Path, err := modules.HandlePath(unit, read2)
	if err != nil {
		return Ports{}, err
	}
	if options.Chemistry == "" || options.Resolution == "" {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "typed chemistry and UMI resolution are required")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/scrnaseq/simpleaf")
	}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 4, Memory: "8g"}
	}
	protected := []string{"--map-dir", "--index", "--t2g-map", "--chemistry", "--reads1", "--reads2", "--resolution", "--output", "-o", "--threads", "--anndata-out", "--knee", "--forced-cells", "--expect-cells", "--explicit-pl", "--unfiltered-pl", "--no-piscem", "--use-selective-alignment"}
	if err := modules.RejectExtraArgPrefixes(unit, options.ExtraArgs, protected); err != nil {
		return Ports{}, err
	}
	command := []string{
		"simpleaf", "quant",
		"--index", index.Tree().Dir.String(),
		"--t2g-map", t2gPath,
		"--chemistry", options.Chemistry,
		"--reads1", read1Path,
		"--reads2", read2Path,
		"--resolution", options.Resolution,
		"--output", outDir.String(),
		"--threads", strconv.Itoa(modules.ThreadCount(resources.CPU)),
		"--anndata-out",
		"--unfiltered-pl", whitelistPath,
	}
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, protected)
	if err != nil {
		return Ports{}, err
	}
	script := "'simpleaf' 'set-paths'\n" + modules.ShellCommand(command)
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Script: script, Image: image, Resources: resources,
		Env: map[string]string{"ALEVIN_FRY_HOME": "."},
		Inputs: []gobble.Bind{
			{Name: "index", From: index, Tree: gobble.DeclareTree(index.Tree().Dir)},
			{Name: "t2g", From: t2g}, {Name: "whitelist", From: whitelist},
			{Name: "read1", From: read1}, {Name: "read2", From: read2},
		},
		Outputs: []gobble.Bind{
			{Name: "map", Tree: gobble.DeclareTree(outDir.Join("af_map"))},
			{Name: "quant", Tree: gobble.DeclareTree(outDir.Join("af_quant"))},
		},
	})
	return Ports{Map: task.Out("map"), Quant: task.Out("quant")}, nil
}

// Pipeline returns a standalone validated Simpleaf quant module.
func Pipeline(index gobble.Tree, t2g, whitelist, read1, read2 gobble.PathSpec, options Options) *gobble.Pipeline {
	inputs := []modules.Input{{Name: "index", Tree: index}, {Name: "t2g", Spec: t2g}, {Name: "whitelist", Spec: whitelist}, {Name: "read1", Spec: read1}, {Name: "read2", Spec: read2}}
	return modules.StandaloneChecked("simpleaf-quant", inputs, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], handles[1], handles[2], handles[3], handles[4], options)
		return err
	})
}
