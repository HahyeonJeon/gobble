package bismarkalign

import (
	"math"
	"strconv"
	"strings"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/methylseq 4.2.0 Bismark image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/bismark:0.25.1--1f50935de5d79c47@sha256:7b49e02b15de6fd59643224db5defb229433de4aebee982d6a03b612077755a0"

// Options controls one directional Bowtie2 Bismark alignment command.
type Options struct {
	modules.Options
	OutDir        gobble.Directory
	Prefix        string
	ScoreMinSlope float64
	Local         bool
	MinInsert     int
	MaxInsert     int
	Multicore     int
}

// Ports are the Bismark alignment BAM and text report.
type Ports struct {
	BAM    gobble.Handle
	Report gobble.Handle
}

// Add records one validated directional Bismark command. A zero read2 selects
// single-end operation.
func Add(parent modules.Parent, index, read1, read2 gobble.Handle, options Options) (Ports, error) {
	const unit = "bismark_align"
	if index.IsZero() || index.Tree().IsZero() || index.Tree().Dir.IsZero() {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "Bismark index Tree directory is required")
	}
	read1Path, err := modules.HandlePath(unit, read1)
	if err != nil {
		return Ports{}, err
	}
	var read2Path string
	if !read2.IsZero() {
		read2Path, err = modules.HandlePath(unit, read2)
		if err != nil {
			return Ports{}, err
		}
	}
	if options.ScoreMinSlope < 0 || options.ScoreMinSlope > 1 || math.IsNaN(options.ScoreMinSlope) || math.IsInf(options.ScoreMinSlope, 0) {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "ScoreMinSlope must be between 0 and 1")
	}
	if options.MinInsert < 0 || options.MaxInsert < 0 || options.MinInsert > 0 && options.MaxInsert > 0 && options.MinInsert > options.MaxInsert {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "insert-size bounds are invalid")
	}
	if read2.IsZero() && (options.MinInsert != 0 || options.MaxInsert != 0) {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "insert-size bounds require paired-end reads")
	}
	if options.Multicore < 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "Multicore must not be negative")
	}
	if err := rejectProtectedExtraArgs(unit, options.ExtraArgs); err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/bismark-align")
	}
	prefix := options.Prefix
	if prefix == "" {
		prefix = "sample"
	}
	paired := !read2.IsZero()
	bamBase, reportBase := prefix, prefix+"_SE_report"
	if paired {
		bamBase, reportBase = prefix+"_pe", prefix+"_PE_report"
	}
	bam := gobble.PathSpec{Dir: outDir, Base: bamBase, Ext: ".bam"}
	report := gobble.PathSpec{Dir: outDir, Base: reportBase, Ext: ".txt"}
	if _, err := bam.Render(); err != nil {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidPath, unit, "Bismark alignment output path is invalid")
	}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 4, Memory: "15g"}
	}
	command := []string{"bismark", "--bowtie2", "--genome", index.Tree().Dir.String(), "--bam", "--output_dir", outDir.String(), "--basename", prefix}
	multicore := options.Multicore
	if multicore == 0 && modules.ThreadCount(resources.CPU) >= 6 {
		multicore = modules.ThreadCount(resources.CPU) / 3
	}
	if multicore > 1 {
		command = append(command, "--multicore", strconv.Itoa(multicore))
	}
	if options.ScoreMinSlope > 0 {
		command = append(command, "--score_min", "L,0,-"+strconv.FormatFloat(options.ScoreMinSlope, 'f', -1, 64))
	}
	if options.Local {
		command = append(command, "--local")
	}
	if options.MinInsert > 0 {
		command = append(command, "--minins", strconv.Itoa(options.MinInsert))
	}
	if options.MaxInsert > 0 {
		command = append(command, "--maxins", strconv.Itoa(options.MaxInsert))
	}
	if paired {
		command = append(command, "-1", read1Path, "-2", read2Path)
	} else {
		command = append(command, read1Path)
	}
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"--bowtie2", "--genome", "--bam", "--output_dir", "--basename", "--multicore", "--score_min", "--local", "--minins", "--maxins", "-1", "-2"})
	if err != nil {
		return Ports{}, err
	}
	inputs := []gobble.Bind{{Name: "index", From: index, Tree: gobble.DeclareTree(index.Tree().Dir)}, {Name: "read1", From: read1}}
	if paired {
		inputs = append(inputs, gobble.Bind{Name: "read2", From: read2})
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs, Outputs: []gobble.Bind{{Name: "bam", Spec: bam}, {Name: "report", Spec: report}}})
	return Ports{BAM: task.Out("bam"), Report: task.Out("report")}, nil
}

func rejectProtectedExtraArgs(unit string, extraArgs []string) error {
	if option := UnsupportedRouteOption(extraArgs); option != "" {
		return modules.ComposeDefect(gobble.DefectInvalidValue, unit, "ExtraArgs contains unsupported option "+option)
	}
	protected := []string{
		"--multicore", "--parallel", "--score_min", "--local",
		"--minins", "-I", "--maxins", "-X", "-1", "-2",
		"--version", "--help",
	}
	if err := modules.RejectExtraArgs(unit, extraArgs, protected); err != nil {
		return err
	}
	return nil
}

// UnsupportedRouteOption returns the canonical Bismark option selected by
// ExtraArgs when it changes the directional Bowtie2 route, inputs, or outputs.
// It recognizes Bismark's unique PBAT, SLAM, and output-directory prefixes.
func UnsupportedRouteOption(extraArgs []string) string {
	unsupported := []string{
		"--hisat2", "--minimap2", "--mm2",
		"--non_directional", "--pbat", "--slam",
		"--se", "--single_end", "-f", "--fasta",
		"-un", "--unmapped", "--ambiguous", "--ambig_bam",
		"--sam", "--cram", "--nucleotide_coverage",
		"--bowtie2", "--genome", "--genome_folder", "--bam",
		"-o", "--output_dir", "-B", "--basename", "--prefix",
	}
	for _, arg := range extraArgs {
		for _, option := range unsupported {
			if extraArgSelectsOption(arg, option) {
				return option
			}
		}
		for _, option := range []struct {
			minimum   string
			canonical string
		}{
			{minimum: "--pba", canonical: "--pbat"},
			{minimum: "--outp", canonical: "--output_dir"},
			{minimum: "--sla", canonical: "--slam"},
		} {
			name, _, _ := strings.Cut(arg, "=")
			if len(name) >= len(option.minimum) && strings.HasPrefix(option.canonical, name) {
				return option.canonical
			}
		}
	}
	return ""
}

func extraArgSelectsOption(arg, option string) bool {
	if arg == option || strings.HasPrefix(arg, option+"=") {
		return true
	}
	return len(option) == 2 && option[0] == '-' && option[1] != '-' && len(arg) > len(option) && strings.HasPrefix(arg, option)
}

// Pipeline returns a standalone validated directional Bismark alignment module.
func Pipeline(index gobble.Tree, read1, read2 gobble.PathSpec, options Options) *gobble.Pipeline {
	inputs := []modules.Input{{Name: "index", Tree: index}, {Name: "read1", Spec: read1}}
	if !productPathSpecUnset(read2) {
		inputs = append(inputs, modules.Input{Name: "read2", Spec: read2})
	}
	return modules.StandaloneChecked("bismark-align", inputs, func(parent modules.Parent, handles []gobble.Handle) error {
		var mate gobble.Handle
		if len(handles) > 2 {
			mate = handles[2]
		}
		_, err := Add(parent, handles[0], handles[1], mate, options)
		return err
	})
}

func productPathSpecUnset(spec gobble.PathSpec) bool {
	return spec.Dir.IsZero() && spec.Prefix == "" && spec.Base == "" && len(spec.Suffixes) == 0 && spec.Ext == ""
}
