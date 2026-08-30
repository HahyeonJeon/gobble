// Package salmonquant owns one Salmon quant command.
package salmonquant

import (
	"fmt"
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
	OutDir  gobble.Directory
	Prefix  string
	LibType string
}

// InferenceThresholds are the nf-core/rnaseq confidence boundaries used to
// turn Salmon's observed strand bias into one typed library orientation.
type InferenceThresholds struct {
	StrandedFraction     float64
	UnstrandedDifference float64
}

// Ports are the selected Salmon quantification and report files.
type Ports struct {
	Quant           gobble.Handle
	MetaInfo        gobble.Handle
	LibFormatCounts gobble.Handle
	Log             gobble.Handle
	Strandedness    gobble.Handle
}

// AddAlignment records alignment-mode Salmon quant over STAR's transcriptome
// BAM. transcriptome and gtf own -t and --geneMap.
func AddAlignment(parent modules.Parent, bam, transcriptome, gtf gobble.Handle, options Options) (Ports, error) {
	return addAlignment(parent, bam, transcriptome, gtf, gobble.Handle{}, options)
}

// AddAlignmentAfter records alignment-mode Salmon after a sample policy gate.
func AddAlignmentAfter(parent modules.Parent, bam, transcriptome, gtf, accepted gobble.Handle, options Options) (Ports, error) {
	return addAlignment(parent, bam, transcriptome, gtf, accepted, options)
}

func addAlignment(parent modules.Parent, bam, transcriptome, gtf, accepted gobble.Handle, options Options) (Ports, error) {
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
	inputs := []gobble.Bind{{Name: "bam", From: bam}, {Name: "transcriptome", From: transcriptome}, {Name: "gtf", From: gtf}}
	if !accepted.IsZero() {
		inputs = append(inputs, gobble.Bind{Name: "sample_accepted", From: accepted})
	}
	return add(parent, unit, command, inputs, options)
}

// AddAlignmentInferred records alignment-mode Salmon quant after a successful
// sample policy gate. Its runtime --libType is selected from the typed
// strandedness file produced by AddInference.
func AddAlignmentInferred(parent modules.Parent, bam, transcriptome, gtf, strandedness, accepted gobble.Handle, paired bool, options Options) (Ports, error) {
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
	strandPath, err := modules.HandlePath(unit, strandedness)
	if err != nil {
		return Ports{}, err
	}
	resultDir, resources := outputPolicy(options)
	commandFor := func(libType string) ([]string, string, gobble.Resources, error) {
		command := []string{"salmon", "quant", "--geneMap", gtfPath, "--libType", libType, "-t", transcriptomePath, "-a", bamPath, "--threads", strconv.Itoa(modules.ThreadCount(resources.CPU)), "-o", resultDir.String()}
		base := options.Options
		base.Resources = resources
		return modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"--geneMap", "--libType", "-t", "-a", "-i", "-r", "-1", "-2", "--threads", "-o"})
	}
	libTypes := map[string]string{"unstranded": "U", "forward": "SF", "reverse": "SR"}
	if paired {
		libTypes = map[string]string{"unstranded": "IU", "forward": "ISF", "reverse": "ISR"}
	}
	commands := make(map[string][]string, len(libTypes))
	var image string
	for _, strand := range []string{"unstranded", "forward", "reverse"} {
		command, resolvedImage, resolvedResources, commandErr := commandFor(libTypes[strand])
		if commandErr != nil {
			return Ports{}, commandErr
		}
		commands[strand] = command
		image = resolvedImage
		resources = resolvedResources
	}
	quant, meta, _, log := outputSpecs(resultDir)
	inputs := []gobble.Bind{{Name: "bam", From: bam}, {Name: "transcriptome", From: transcriptome}, {Name: "gtf", From: gtf}, {Name: "strandedness", From: strandedness}}
	if !accepted.IsZero() {
		inputs = append(inputs, gobble.Bind{Name: "sample_accepted", From: accepted})
	}
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Script: modules.StrandedCommand(strandPath, commands["unstranded"], commands["forward"], commands["reverse"]), Image: image, Resources: resources, Inputs: inputs,
		Outputs: []gobble.Bind{{Name: "quant", Spec: quant}, {Name: "meta_info", Spec: meta}, {Name: "log", Spec: log}},
	})
	return Ports{Quant: task.Out("quant"), MetaInfo: task.Out("meta_info"), Log: task.Out("log")}, nil
}

// AddInference records read-mode Salmon quant with automatic library-type
// inference. A zero read2 selects single-end operation. after is an optional
// sample-retention gate that must complete before inference.
func AddInference(parent modules.Parent, index, read1, read2, after gobble.Handle, thresholds InferenceThresholds, options Options) (Ports, error) {
	const unit = "salmon_strandedness"
	if thresholds.StrandedFraction < 0.5 || thresholds.StrandedFraction > 1 || thresholds.UnstrandedDifference < 0 || thresholds.UnstrandedDifference > 1 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "inference thresholds must satisfy stranded fraction 0.5..1 and unstranded difference 0..1")
	}
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
	if !after.IsZero() {
		inputs = append(inputs, gobble.Bind{Name: "sample_retained", From: after})
	}
	resultDir, resources := outputPolicy(options)
	command = append(command, "--threads", strconv.Itoa(modules.ThreadCount(resources.CPU)), "-o", resultDir.String())
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"--geneMap", "--libType", "-t", "-a", "-i", "-r", "-1", "-2", "--threads", "-o"})
	if err != nil {
		return Ports{}, err
	}
	quant, meta, lib, log := outputSpecs(resultDir)
	strand := gobble.PathSpec{Dir: resultDir, Base: "strandedness", Ext: ".txt"}
	libPath, _ := lib.Render()
	strandPath, _ := strand.Render()
	script := modules.ShellCommand(command) + "\n" + inferenceScript(libPath, strandPath, thresholds)
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Script: script, Image: image, Resources: resources, Inputs: inputs,
		Outputs: []gobble.Bind{{Name: "quant", Spec: quant}, {Name: "meta_info", Spec: meta}, {Name: "lib_format_counts", Spec: lib}, {Name: "log", Spec: log}, {Name: "strandedness", Spec: strand}},
	})
	return Ports{Quant: task.Out("quant"), MetaInfo: task.Out("meta_info"), LibFormatCounts: task.Out("lib_format_counts"), Log: task.Out("log"), Strandedness: task.Out("strandedness")}, nil
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
	resultDir, resources := outputPolicy(options)
	command = append(command, "--threads", strconv.Itoa(modules.ThreadCount(resources.CPU)), "-o", resultDir.String())
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"--geneMap", "--libType", "-t", "-a", "-i", "-r", "-1", "-2", "--threads", "-o"})
	if err != nil {
		return Ports{}, err
	}
	quant, meta, _, log := outputSpecs(resultDir)
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{{Name: "quant", Spec: quant}, {Name: "meta_info", Spec: meta}, {Name: "log", Spec: log}}})
	return Ports{Quant: task.Out("quant"), MetaInfo: task.Out("meta_info"), Log: task.Out("log")}, nil
}

func outputPolicy(options Options) (gobble.Directory, gobble.Resources) {
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/salmon-quant")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "quant"
	}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 2, Memory: "2g"}
	}
	return gobble.Dir(outDir.String() + "/" + prefix), resources
}

func outputSpecs(resultDir gobble.Directory) (gobble.PathSpec, gobble.PathSpec, gobble.PathSpec, gobble.PathSpec) {
	return gobble.PathSpec{Dir: resultDir, Base: "quant", Ext: ".sf"},
		gobble.PathSpec{Dir: resultDir.Join("aux_info"), Base: "meta_info", Ext: ".json"},
		gobble.PathSpec{Dir: resultDir, Base: "lib_format_counts", Ext: ".json"},
		gobble.PathSpec{Dir: resultDir.Join("logs"), Base: "salmon_quant", Ext: ".log"}
}

func inferenceScript(libPath, strandPath string, thresholds InferenceThresholds) string {
	return fmt.Sprintf(`json=%s
format=$(awk -F'"' '/"expected_format"/ {print $4; exit}' "$json")
bias=$(awk -F: '/"strand_mapping_bias"/ {v=$NF; gsub(/[^0-9.eE+-]/, "", v); print v; exit}' "$json")
test -n "$format" && test -n "$bias"
case "$format" in
  U|IU)
    awk -v bias="$bias" -v limit=%s 'BEGIN {d=(2*bias)-1; if (d<0) d=-d; exit !(d<=limit)}'
    value=unstranded
    ;;
  SF|ISF)
    awk -v bias="$bias" -v limit=%s 'BEGIN {exit !(bias>=limit)}'
    value=forward
    ;;
  SR|ISR)
    awk -v bias="$bias" -v limit=%s 'BEGIN {exit !((1-bias)>=limit)}'
    value=reverse
    ;;
  *)
    echo "unsupported inferred Salmon library format: $format" >&2
    exit 2
    ;;
esac
printf '%%s\n' "$value" > %s`, modules.ShellQuote(libPath),
		strconv.FormatFloat(thresholds.UnstrandedDifference, 'g', -1, 64),
		strconv.FormatFloat(thresholds.StrandedFraction, 'g', -1, 64),
		strconv.FormatFloat(thresholds.StrandedFraction, 'g', -1, 64),
		modules.ShellQuote(strandPath))
}
