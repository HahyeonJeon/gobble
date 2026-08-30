// Package salmonquant owns one Salmon quant command.
package salmonquant

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 Salmon image resolved for
// linux/amd64.
const DefaultImage modules.Image = "quay.io/biocontainers/salmon:1.10.3--h6dccd9a_2@sha256:f83ebb158845ee8138d793347f83b92c75e83c58dd8f4600c6fea2a2453ef08e"

// Options controls one Salmon quant command.
type Options struct {
	modules.Options
	OutDir gobble.Directory
	Prefix string
	LibType string
}

// Ports are the selected Salmon quantification and report files.
type Ports struct {
	Quant           gobble.Handle
	MetaInfo        gobble.Handle
	LibFormatCounts gobble.Handle
	Log             gobble.Handle
}

// AddAlignment records alignment-mode Salmon quant over STAR's transcriptome
// BAM. transcriptome and gtf own -t and --geneMap.
func AddAlignment(parent modules.Parent, bam, transcriptome, gtf gobble.Handle, options Options) (Ports, error) {
	const unit = "salmon_quant"
	bamPath, err := modules.HandlePath(unit, bam)
	if err != nil {
		return Ports{}, err
	}
	transcriptomePath, err := modules.HandlePath(unit, transcriptome)
	if err != nil {
		return Ports{}, err
	}
	gtfPath, err := modules.HandlePath(unit, gtf)
	if err != nil {
		return Ports{}, err
	}
	libType := options.LibType
	if libType == "" {
		libType = "A"
	}
	command := []string{"salmon", "quant", "--geneMap", gtfPath, "--libType", libType, "-t", transcriptomePath, "-a", bamPath}
	return add(parent, unit, command, []gobble.Bind{{Name: "bam", From: bam}, {Name: "transcriptome", From: transcriptome}, {Name: "gtf", From: gtf}}, options)
}

// AddInference records read-mode Salmon quant with automatic library-type
// inference. A zero read2 selects single-end operation.
func AddInference(parent modules.Parent, index, read1, read2 gobble.Handle, options Options) (Ports, error) {
	const unit = "salmon_strandedness"
	read1Path, err := modules.HandlePath(unit, read1)
	if err != nil {
		return Ports{}, err
	}
	command := []string{"salmon", "quant", "--libType", "A", "-i", index.Tree().Dir.String()}
	inputs := []gobble.Bind{{Name: "index", From: index, Tree: gobble.DeclareTree(index.Tree().Dir)}, {Name: "read1", From: read1}}
	if read2.IsZero() {
		command = append(command, "-r", read1Path)
	} else {
		read2Path, pathErr := modules.HandlePath(unit, read2)
		if pathErr != nil {
			return Ports{}, pathErr
		}
		command = append(command, "-1", read1Path, "-2", read2Path)
		inputs = append(inputs, gobble.Bind{Name: "read2", From: read2})
	}
	return add(parent, unit, command, inputs, options)
}

// AlignmentPipeline returns a standalone alignment-mode Salmon module.
func AlignmentPipeline(bam, transcriptome, gtf gobble.PathSpec, options Options) *gobble.Pipeline {
	inputs := []modules.Input{{Name: "bam", Spec: bam}, {Name: "transcriptome", Spec: transcriptome}, {Name: "gtf", Spec: gtf}}
	return modules.StandaloneChecked("salmon-quant", inputs, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := AddAlignment(parent, handles[0], handles[1], handles[2], options)
		return err
	})
}

func add(parent modules.Parent, unit string, command []string, inputs []gobble.Bind, options Options) (Ports, error) {
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/salmon-quant")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "quant"
	}
	resultDir := gobble.Dir(outDir.String() + "/" + prefix)
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 2, Memory: "2g"}
	}
	command = append(command, "--threads", strconv.Itoa(modules.ThreadCount(resources.CPU)), "-o", resultDir.String())
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"--geneMap", "--libType", "-t", "-a", "-i", "-r", "-1", "-2", "--threads", "-o"})
	if err != nil {
		return Ports{}, err
	}
	quant := gobble.PathSpec{Dir: resultDir, Base: "quant", Ext: ".sf"}
	meta := gobble.PathSpec{Dir: resultDir.Join("aux_info"), Base: "meta_info", Ext: ".json"}
	lib := gobble.PathSpec{Dir: resultDir, Base: "lib_format_counts", Ext: ".json"}
	log := gobble.PathSpec{Dir: resultDir.Join("logs"), Base: "salmon_quant", Ext: ".log"}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{{Name: "quant", Spec: quant}, {Name: "meta_info", Spec: meta}, {Name: "lib_format_counts", Spec: lib}, {Name: "log", Spec: log}}})
	return Ports{Quant: task.Out("quant"), MetaInfo: task.Out("meta_info"), LibFormatCounts: task.Out("lib_format_counts"), Log: task.Out("log")}, nil
}
