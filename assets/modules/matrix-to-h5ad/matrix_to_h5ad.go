// Package matrixtoh5ad owns one raw Simpleaf matrix-to-h5ad conversion.
package matrixtoh5ad

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

const conversionScript = `
import json
import sys
import scanpy as sc

source, output, sample, expected_cells, seq_center = sys.argv[1:]
adata = sc.read_h5ad(source + "/alevin/quants.h5ad")
adata.obs_names = adata.obs["barcodes"].values
adata.var_names = adata.var["gene_id"].values
adata.obs["sample"] = sample
if expected_cells:
    adata.obs["expected_cells"] = int(expected_cells)
if seq_center:
    adata.obs["seq_center"] = seq_center
adata = adata[adata.obs_names.sort_values(), adata.var_names.sort_values()].copy()

adata.var["gene_versions"] = adata.var["gene_id"]
adata.var.index = adata.var["gene_versions"].str.split(".").str[0].values
adata.var_names_make_unique()

adata = adata[adata.obs_names.sort_values(), adata.var_names.sort_values()].copy()

simpleaf_map_info = json.loads(adata.uns["simpleaf_map_info"])
simpleaf_map_info.pop("runtime_seconds")
adata.uns["simpleaf_map_info"] = json.dumps(simpleaf_map_info, sort_keys=True)

adata.write_h5ad(output)
`

// DefaultImage is the nf-core/scrnaseq 4.2.0 MTX_TO_H5AD image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/scanpy:1.10.2--e83da2205b92a538@sha256:fbd40d3d00751ac0df11564b3697006ecf8604af48960833910d32755033575f"

// Options controls one sample's raw-matrix conversion and typed metadata.
type Options struct {
	modules.Options
	Sample        string
	ExpectedCells int
	SeqCenter     string
	OutDir        gobble.Directory
	Prefix        string
}

// Ports contains the provenance-named raw h5ad file.
type Ports struct{ H5AD gobble.Handle }

// Add records one direct conversion of the declared quants.h5ad member.
func Add(parent modules.Parent, quant gobble.Handle, options Options) (Ports, error) {
	const unit = "matrix_to_h5ad"
	if quant.IsZero() || quant.Tree().IsZero() || quant.Tree().Dir.IsZero() {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "complete Simpleaf quantification Tree is required")
	}
	if options.Sample == "" || len(options.ExtraArgs) != 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "sample and typed conversion operands are required; ExtraArgs are unsupported")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/scrnaseq/matrices")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = options.Sample + "_raw_matrix"
	}
	output := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".h5ad"}
	outputPath, err := output.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "raw h5ad output path is invalid")
	}
	expected := ""
	if options.ExpectedCells > 0 {
		expected = strconv.Itoa(options.ExpectedCells)
	}
	base := options.Options
	base.ExtraArgs = nil
	command := []string{"python", "-c", conversionScript, quant.Tree().Dir.String(), outputPath, options.Sample, expected, options.SeqCenter}
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, gobble.Resources{CPU: 2, Memory: "6g"}, command, []string{"-c"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Command: command, Image: image, Resources: resources,
		Env:     map[string]string{"NUMBA_CACHE_DIR": "."},
		Inputs:  []gobble.Bind{{Name: "quant", From: quant, Tree: gobble.DeclareTree(quant.Tree().Dir)}},
		Outputs: []gobble.Bind{{Name: "h5ad", Spec: output}},
	})
	return Ports{H5AD: task.Out("h5ad")}, nil
}

// Pipeline returns a standalone validated raw matrix conversion module.
func Pipeline(quant gobble.Tree, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("matrix-to-h5ad", []modules.Input{{Name: "quant", Tree: quant}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], options)
		return err
	})
}
