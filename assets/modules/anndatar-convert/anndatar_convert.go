// Package anndatarconvert owns one AnnDataR h5ad-to-RDS conversion command.
package anndatarconvert

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

const conversionScript = `
args <- commandArgs(trailingOnly=TRUE)
suppressPackageStartupMessages(library(anndataR))
suppressPackageStartupMessages(library(SeuratObject))
suppressPackageStartupMessages(library(SingleCellExperiment))
adata <- read_h5ad(args[[1]])
saveRDS(adata$as_Seurat(), file=args[[2]])
saveRDS(adata$as_SingleCellExperiment(), file=args[[3]])
`

// DefaultImage is the nf-core/scrnaseq 4.2.0 ANNDATAR_CONVERT image resolved
// for linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/bioconductor-anndatar_bioconductor-rhdf5_bioconductor-singlecellexperiment_r-seurat:a0f51df063bb9b2a@sha256:42b75cf9f00d0bd96a5c6b528291a285fd6e33c729c05452dfd73cbbc4bc7a66"

// Options controls one pair of raw-matrix RDS outputs.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
}

// Ports contains Seurat and SingleCellExperiment RDS files.
type Ports struct {
	Seurat gobble.Handle
	SCE    gobble.Handle
}

// Add records one exact AnnDataR conversion command.
func Add(parent modules.Parent, h5ad gobble.Handle, options Options) (Ports, error) {
	const unit = "anndatar_convert"
	h5adPath, err := modules.HandlePath(unit, h5ad)
	if err != nil {
		return Ports{}, err
	}
	if len(options.ExtraArgs) != 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "RDS conversion operands are typed and ExtraArgs are unsupported")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/scrnaseq/matrices")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "raw_matrix"
	}
	seurat := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".seurat.rds"}
	sce := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".sce.rds"}
	seuratPath, err := seurat.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "Seurat RDS output path is invalid")
	}
	scePath, err := sce.Render()
	if err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "SingleCellExperiment RDS output path is invalid")
	}
	base := options.Options
	base.ExtraArgs = nil
	command := []string{"Rscript", "-e", conversionScript, h5adPath, seuratPath, scePath}
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, gobble.Resources{CPU: 2, Memory: "6g"}, command, []string{"-e"})
	if err != nil {
		return Ports{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "h5ad", From: h5ad}}, Outputs: []gobble.Bind{{Name: "seurat_rds", Spec: seurat}, {Name: "sce_rds", Spec: sce}}})
	return Ports{Seurat: task.Out("seurat_rds"), SCE: task.Out("sce_rds")}, nil
}

// Pipeline returns a standalone validated AnnDataR conversion module.
func Pipeline(h5ad gobble.PathSpec, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("anndatar-convert", []modules.Input{{Name: "h5ad", Spec: h5ad}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], options)
		return err
	})
}
