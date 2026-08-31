// Package qcatch owns one QCatch cell-filtering and QC command.
package qcatch

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/scrnaseq 4.2.0 QCATCH image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/pip_qcatch:03b88593a5cca75b@sha256:2687142908996acd8d74700e0536802ed35c5fa4167ba007d5408c608d2401e8"

// Options controls one QCatch filter and report. Chemistry and NPartitions are
// mutually exclusive. VisualizeDoublets requires RemoveDoublets.
type Options struct {
	modules.Options
	Chemistry         string
	NPartitions       int
	RemoveDoublets    bool
	VisualizeDoublets bool
	SkipUMAPTSNE      bool
	OutDir            gobble.Directory
}

// Ports contains QCatch's report, metrics, and separate filtered matrix.
type Ports struct {
	Report       gobble.Handle
	Metrics      gobble.Handle
	FilteredH5AD gobble.Handle
}

// Add records QCatch over one complete Simpleaf quantification Tree.
func Add(parent modules.Parent, quant gobble.Handle, options Options) (Ports, error) {
	const unit = "qcatch"
	if quant.IsZero() || quant.Tree().IsZero() || quant.Tree().Dir.IsZero() {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "complete Simpleaf quantification Tree is required")
	}
	if (options.Chemistry == "") == (options.NPartitions == 0) || options.NPartitions < 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "exactly one supported chemistry or positive partition count is required")
	}
	if options.VisualizeDoublets && !options.RemoveDoublets {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "doublet visualization requires doublet removal")
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("results/scrnaseq/qcatch")
	}
	protected := []string{"--input", "-i", "--output", "-o", "--chemistry", "-c", "--n_partitions", "-n", "--save_filtered_h5ad", "-s", "--export_summary_table", "-x", "--remove_doublets", "-d", "--visualize_doublets", "-vd", "--skip_umap_tsne", "-u", "--gene_id2name_file", "-g", "--valid_cell_list", "-l"}
	if err := modules.RejectExtraArgPrefixes(unit, options.ExtraArgs, protected); err != nil {
		return Ports{}, err
	}
	command := []string{"qcatch", "--input", quant.Tree().Dir.String(), "--output", outDir.String()}
	if options.Chemistry != "" {
		command = append(command, "--chemistry", options.Chemistry)
	} else {
		command = append(command, "--n_partitions", strconv.Itoa(options.NPartitions))
	}
	command = append(command, "--save_filtered_h5ad", "--export_summary_table")
	if options.RemoveDoublets {
		command = append(command, "--remove_doublets")
	}
	if options.VisualizeDoublets {
		command = append(command, "--visualize_doublets")
	}
	if options.SkipUMAPTSNE {
		command = append(command, "--skip_umap_tsne")
	}
	command, image, resources, err := modules.ResolveOptions(unit, options.Options, DefaultImage, gobble.Resources{CPU: 2, Memory: "6g"}, command, protected)
	if err != nil {
		return Ports{}, err
	}
	report := gobble.PathSpec{Dir: outDir, Base: "QCatch_report", Ext: ".html"}
	metrics := gobble.PathSpec{Dir: outDir, Base: "summary_table", Ext: ".csv"}
	filtered := gobble.PathSpec{Dir: outDir, Base: "filtered_quants", Ext: ".h5ad"}
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Command: command, Image: image, Resources: resources,
		Inputs:  []gobble.Bind{{Name: "quant", From: quant, Tree: gobble.DeclareTree(quant.Tree().Dir)}},
		Outputs: []gobble.Bind{{Name: "report", Spec: report}, {Name: "metrics", Spec: metrics}, {Name: "filtered_h5ad", Spec: filtered}},
	})
	return Ports{Report: task.Out("report"), Metrics: task.Out("metrics"), FilteredH5AD: task.Out("filtered_h5ad")}, nil
}

// Pipeline returns a standalone validated QCatch module.
func Pipeline(quant gobble.Tree, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("qcatch", []modules.Input{{Name: "quant", Tree: quant}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], options)
		return err
	})
}
