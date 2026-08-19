package assets

import (
	"strings"

	"github.com/HahyeonJeon/gobble"
)

const bismarkExtractorTaskName = "bismark_methylation_extractor"

func defaultBismarkExtractorDir() gobble.Directory {
	return gobble.Dir("work/bismark-extractor")
}

func bismarkExtractorDir(dir gobble.Directory) gobble.Directory {
	if dir.IsZero() {
		return defaultBismarkExtractorDir()
	}
	return dir
}

// BismarkMethylationExtractorOptions are typed bismark
// methylation_extractor settings. ExtraArgs are argv tokens appended
// after named flags and before the BAM path.
//
// --parallel (alias --multicore) is not copied from Resources.CPU. The
// 0.25.1 extractor defaults to 1 and allows --parallel >= 1. Callers
// pass --parallel via ExtraArgs. OutDir is --output_dir, the parent
// folder of the declared reports.
type BismarkMethylationExtractorOptions struct {
	ExtraArgs []string
	Resources gobble.Resources
	OutDir    gobble.Directory
}

// BismarkMethylationExtractorPorts are the declared report regular files.
type BismarkMethylationExtractorPorts struct {
	BedGraph gobble.Handle
	Coverage gobble.Handle
	Report   gobble.Handle
	Mbias    gobble.Handle
	CpG      gobble.Handle
}

// AddBismarkMethylationExtractor records one bismark_methylation_extractor
// task on parent. The command is paired-end and comprehensive. The shared
// builder does not call AddInput.
func AddBismarkMethylationExtractor(parent Parent, bam gobble.Handle, opts BismarkMethylationExtractorOptions) BismarkMethylationExtractorPorts {
	return addBismarkMethylationExtractor(parent, bam, opts)
}

// BismarkMethylationExtractorPipeline returns a standalone extractor
// pipeline. It AddInputs bam, then calls the same builder as
// AddBismarkMethylationExtractor.
func BismarkMethylationExtractorPipeline(bam gobble.PathSpec, opts BismarkMethylationExtractorOptions) *gobble.Pipeline {
	return Standalone("bismark-methylation-extractor", []Input{{Name: "bam", Spec: bam}}, func(parent Parent, hs []gobble.Handle) {
		addBismarkMethylationExtractor(parent, hs[0], opts)
	})
}

func addBismarkMethylationExtractor(parent Parent, bam gobble.Handle, opts BismarkMethylationExtractorOptions) BismarkMethylationExtractorPorts {
	outDir := bismarkExtractorDir(opts.OutDir)
	stem := bismarkExtractorStem(bam.Spec())
	bedGraph := gobble.PathSpec{Dir: outDir, Name: stem, Ext: ".bedGraph.gz"}
	coverage := gobble.PathSpec{Dir: outDir, Name: stem, Ext: ".bismark.cov.gz"}
	report := gobble.PathSpec{Dir: outDir, Name: stem + "_splitting_report", Ext: ".txt"}
	mbias := gobble.PathSpec{Dir: outDir, Name: stem, Ext: ".M-bias.txt"}
	cpg := gobble.PathSpec{Dir: outDir, Name: "CpG_context_" + stem, Ext: ".txt.gz"}

	cmd := []string{"bismark_methylation_extractor", "--bedGraph", "--counts", "--gzip", "--report", "--comprehensive", "-p", "--output_dir", outDir.String()}
	cmd = AppendExtraArgs(cmd, opts.ExtraArgs)
	cmd = append(cmd, mustCommandPath(bam.Spec()))

	task := AddTask(parent, gobble.TaskSpec{
		Name:    bismarkExtractorTaskName,
		Command: cmd,
		Image:   bismarkImage,
		Inputs:  []gobble.Bind{{Name: "bam", From: bam}},
		Outputs: []gobble.Bind{
			{Name: "bedgraph", Spec: bedGraph},
			{Name: "coverage", Spec: coverage},
			{Name: "report", Spec: report},
			{Name: "mbias", Spec: mbias},
			{Name: "cpg", Spec: cpg},
		},
		Resources: opts.Resources,
	})
	return BismarkMethylationExtractorPorts{
		BedGraph: task.Out("bedgraph"),
		Coverage: task.Out("coverage"),
		Report:   task.Out("report"),
		Mbias:    task.Out("mbias"),
		CpG:      task.Out("cpg"),
	}
}

func bismarkExtractorStem(spec gobble.PathSpec) string {
	path, err := CommandPath(spec)
	base := spec.Name
	if err == nil {
		if i := strings.LastIndex(path, "/"); i >= 0 {
			base = path[i+1:]
		} else {
			base = path
		}
	}
	if base == "" {
		base = "aligned_pe"
	}
	lower := strings.ToLower(base)
	for _, suf := range []string{".bam", ".sam"} {
		if strings.HasSuffix(lower, suf) {
			return base[:len(base)-len(suf)]
		}
	}
	return base
}
