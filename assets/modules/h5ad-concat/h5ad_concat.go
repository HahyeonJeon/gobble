// Package h5adconcat owns one cohort raw-h5ad concatenation command.
package h5adconcat

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

const concatScript = `
import sys
import anndata as ad
import scanpy as sc

output = sys.argv[1]
pairs = sys.argv[2:]
if not pairs or len(pairs) % 2:
    raise ValueError("sample and h5ad arguments must be complete")
labels = pairs[0::2]
paths = pairs[1::2]
objects = [sc.read_h5ad(path) for path in paths]
combined = ad.concat(objects, keys=labels, label="sample", join="outer", merge="unique", index_unique="_")
combined.write_h5ad(output)
`

// DefaultImage is the nf-core/scrnaseq 4.2.0 CONCAT_H5AD image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/scanpy:1.10.2--e83da2205b92a538@sha256:fbd40d3d00751ac0df11564b3697006ecf8604af48960833910d32755033575f"

// Options controls one complete cohort fan-in.
type Options struct {
	modules.Options
	Labels []string
	OutDir gobble.Directory
	Prefix string
}

// Ports contains the combined raw h5ad matrix.
type Ports struct{ H5AD gobble.Handle }

// Add records one concatenation over every declared sample matrix.
func Add(parent modules.Parent, h5ads []gobble.Handle, options Options) (Ports, error) {
	const unit = "h5ad_concat"
	if len(h5ads) == 0 || len(h5ads) != len(options.Labels) {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "complete sample labels and h5ad membership are required")
	}
	if len(options.ExtraArgs) != 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "cohort matrix operands are typed and ExtraArgs are unsupported")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/scrnaseq/matrices")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "combined_raw_matrix"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".h5ad"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "combined h5ad output path is invalid")
	}
	command := []string{"python", "-c", concatScript, outputPath}
	inputs := make([]gobble.Bind, len(h5ads))
	seen := make(map[string]bool, len(options.Labels))
	for i, h5ad := range h5ads {
		if options.Labels[i] == "" || seen[options.Labels[i]] {
			return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "sample labels must be non-empty and unique")
		}
		seen[options.Labels[i]] = true
		path, pathErr := modules.HandlePath(unit, h5ad)
		if pathErr != nil {
			return Ports{}, pathErr
		}
		command = append(command, options.Labels[i], path)
		inputs[i] = gobble.Bind{Name: "h5ad_" + strconv.Itoa(i+1), From: h5ad}
	}
	base := options.Options
	base.ExtraArgs = nil
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, gobble.Resources{CPU: 2, Memory: "6g"}, command, []string{"-c"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Env: map[string]string{"NUMBA_CACHE_DIR": "."}, Inputs: inputs, Outputs: []gobble.Bind{{Name: "h5ad", Spec: output}}})
	return Ports{H5AD: task.Out("h5ad")}, nil
}

// Pipeline returns a standalone validated cohort concatenation module.
func Pipeline(h5ads []gobble.PathSpec, options Options) *gobble.Pipeline {
	inputs := make([]modules.Input, len(h5ads))
	for i, h5ad := range h5ads {
		inputs[i] = modules.Input{Name: "h5ad_" + strconv.Itoa(i+1), Spec: h5ad}
	}
	return modules.StandaloneChecked("h5ad-concat", inputs, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles, options)
		return err
	})
}
