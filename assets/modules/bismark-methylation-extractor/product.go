package bismarkmethylationextractor

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/methylseq 4.2.0 Bismark image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/bismark:0.25.1--1f50935de5d79c47@sha256:7b49e02b15de6fd59643224db5defb229433de4aebee982d6a03b612077755a0"

// Options controls one comprehensive Bismark methylation-extraction command.
// Comprehensive context output gives the command a complete, fixed set of
// CpG, CHG, and CHH call files.
type Options struct {
	modules.Options
	OutDir         gobble.Directory
	ExcludeOverlap bool
	IgnoreR1       int
	IgnoreR2       int
	Ignore3PrimeR1 int
	Ignore3PrimeR2 int
	CoverageCutoff int
}

// Ports are all regular files produced by the selected extractor command.
type Ports struct {
	CpG      gobble.Handle
	CHG      gobble.Handle
	CHH      gobble.Handle
	BedGraph gobble.Handle
	Coverage gobble.Handle
	Report   gobble.Handle
	MBias    gobble.Handle
}

// Add records one validated bismark_methylation_extractor command.
func Add(parent modules.Parent, bam gobble.Handle, paired bool, options Options) (Ports, error) {
	const unit = "bismark_methylation_extractor"
	bamPath, err := modules.HandlePath(unit, bam)
	if err != nil {
		return Ports{}, err
	}
	if options.IgnoreR1 < 0 || options.IgnoreR2 < 0 || options.Ignore3PrimeR1 < 0 || options.Ignore3PrimeR2 < 0 || options.CoverageCutoff < 0 {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "extractor ignore and cutoff values must not be negative")
	}
	if !paired && (options.IgnoreR2 != 0 || options.Ignore3PrimeR2 != 0) {
		return Ports{}, modules.ComposeDefect(gobble.DefectInvalidValue, unit, "read-2 ignore values require paired-end reads")
	}
	protected := []string{
		"--CX", "--CX_context", "--cytosine_report", "--yacht", "--merge_non_CpG", "--zero_based", "--ucsc", "--mbias_only", "--mbias_off", "--sam", "--version", "--help",
		"-s", "--single-end", "-p", "--paired-end", "--bedGraph", "--counts", "--gzip", "--report", "--comprehensive", "-o", "--output_dir", "--no_overlap", "--include_overlap", "--ignore", "--ignore_r2", "--ignore_3prime", "--ignore_3prime_r2", "--cutoff", "--parallel", "--multicore",
	}
	if err := modules.RejectExtraArgPrefixes(unit, options.ExtraArgs, protected); err != nil {
		return Ports{}, err
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/bismark-methylation-extractor")
	}
	stem := bismarkExtractorStem(bam.Spec())
	outputs := map[string]gobble.PathSpec{
		"cpg":      {Dir: outDir, Base: "CpG_context_" + stem, Ext: ".txt.gz"},
		"chg":      {Dir: outDir, Base: "CHG_context_" + stem, Ext: ".txt.gz"},
		"chh":      {Dir: outDir, Base: "CHH_context_" + stem, Ext: ".txt.gz"},
		"bedgraph": {Dir: outDir, Base: stem, Ext: ".bedGraph.gz"},
		"coverage": {Dir: outDir, Base: stem, Ext: ".bismark.cov.gz"},
		"report":   {Dir: outDir, Base: stem + "_splitting_report", Ext: ".txt"},
		"mbias":    {Dir: outDir, Base: stem, Ext: ".M-bias.txt"},
	}
	mode := "--single-end"
	if paired {
		mode = "--paired-end"
	}
	command := []string{"bismark_methylation_extractor", bamPath, "--bedGraph", "--counts", "--gzip", "--report", "--comprehensive", mode, "--output_dir", outDir.String()}
	if paired {
		if options.ExcludeOverlap {
			command = append(command, "--no_overlap")
		} else {
			command = append(command, "--include_overlap")
		}
	}
	if options.IgnoreR1 > 0 {
		command = append(command, "--ignore", strconv.Itoa(options.IgnoreR1))
	}
	if options.IgnoreR2 > 0 {
		command = append(command, "--ignore_r2", strconv.Itoa(options.IgnoreR2))
	}
	if options.Ignore3PrimeR1 > 0 {
		command = append(command, "--ignore_3prime", strconv.Itoa(options.Ignore3PrimeR1))
	}
	if options.Ignore3PrimeR2 > 0 {
		command = append(command, "--ignore_3prime_r2", strconv.Itoa(options.Ignore3PrimeR2))
	}
	if options.CoverageCutoff > 0 {
		command = append(command, "--cutoff", strconv.Itoa(options.CoverageCutoff))
	}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 6, Memory: "15g"}
	}
	if n := modules.ThreadCount(resources.CPU) / 3; n >= 2 {
		command = append(command, "--multicore", strconv.Itoa(n))
	}
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"-s", "--single-end", "-p", "--paired-end", "--bedGraph", "--counts", "--gzip", "--report", "--comprehensive", "-o", "--output_dir", "--no_overlap", "--include_overlap", "--ignore", "--ignore_r2", "--ignore_3prime", "--ignore_3prime_r2", "--cutoff", "--parallel", "--multicore"})
	if err != nil {
		return Ports{}, err
	}
	binds := make([]gobble.Bind, 0, 7)
	for _, name := range []string{"cpg", "chg", "chh", "bedgraph", "coverage", "report", "mbias"} {
		binds = append(binds, gobble.Bind{Name: name, Spec: outputs[name]})
	}
	task := parent.AddTask(gobble.TaskSpec{Name: unit, Command: command, Image: image, Resources: resources, Inputs: []gobble.Bind{{Name: "bam", From: bam}}, Outputs: binds})
	return Ports{CpG: task.Out("cpg"), CHG: task.Out("chg"), CHH: task.Out("chh"), BedGraph: task.Out("bedgraph"), Coverage: task.Out("coverage"), Report: task.Out("report"), MBias: task.Out("mbias")}, nil
}

// Pipeline returns a standalone validated methylation-extractor module.
func Pipeline(bam gobble.PathSpec, paired bool, options Options) *gobble.Pipeline {
	return modules.StandaloneChecked("bismark-methylation-extractor", []modules.Input{{Name: "bam", Spec: bam}}, func(parent modules.Parent, handles []gobble.Handle) error {
		_, err := Add(parent, handles[0], paired, options)
		return err
	})
}
