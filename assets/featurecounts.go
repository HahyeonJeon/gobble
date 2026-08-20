package assets

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
)

// featurecountsImage is the biocontainers subread pin for featureCounts.
const featurecountsImage = "quay.io/biocontainers/subread:2.1.1--h577a1d6_0"

const featurecountsTaskName = "featurecounts"

// FeatureCountsOptions are typed featureCounts settings. ExtraArgs are
// argv tokens appended after named flags and before the BAM path.
//
// -T copies Resources.CPU when CPU is at least 1, as an integer.
// -s copies Strandedness: unstranded 0, forward 1, reverse 2. Empty
// Strandedness uses [gobble.DefaultRNAStrandedness] (reverse). The
// command is paired-end and always passes -p. It does not pass -t or -g.
type FeatureCountsOptions struct {
	ExtraArgs    []string
	Resources    gobble.Resources
	OutDir       gobble.Directory
	Strandedness string
}

// FeatureCountsPorts are the declared gene-count output.
type FeatureCountsPorts struct {
	Counts gobble.Handle
}

// AddFeatureCounts records one featureCounts task on parent. bam is the
// sorted BAM. gtf is the annotation. The shared builder does not call
// AddInput.
func AddFeatureCounts(parent Parent, bam, gtf gobble.Handle, opts FeatureCountsOptions) FeatureCountsPorts {
	return addFeatureCounts(parent, bam, gtf, opts)
}

// FeatureCountsPipeline returns a standalone featureCounts pipeline. It
// AddInputs bam and gtf, then calls the same builder as AddFeatureCounts.
func FeatureCountsPipeline(bam, gtf gobble.PathSpec, opts FeatureCountsOptions) *gobble.Pipeline {
	return Standalone("featurecounts", []Input{
		{Name: "bam", Spec: bam},
		{Name: "gtf", Spec: gtf},
	}, func(parent Parent, hs []gobble.Handle) {
		addFeatureCounts(parent, hs[0], hs[1], opts)
	})
}

func addFeatureCounts(parent Parent, bam, gtf gobble.Handle, opts FeatureCountsOptions) FeatureCountsPorts {
	outDir := opts.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/featurecounts")
	}
	countsSpec := gobble.PathSpec{Dir: outDir, Base: "counts", Ext: ".txt"}

	cmd := []string{
		"featureCounts",
		"-a", mustCommandPath(gtf.Spec()),
		"-o", mustCommandPath(countsSpec),
		"-p",
		"-s", featureCountsStrand(opts.Strandedness),
	}
	if n := threadCount(opts.Resources.CPU); n > 0 {
		cmd = append(cmd, "-T", strconv.Itoa(n))
	}
	cmd = AppendExtraArgs(cmd, opts.ExtraArgs)
	cmd = append(cmd, mustCommandPath(bam.Spec()))

	task := AddTask(parent, gobble.TaskSpec{
		Name:    featurecountsTaskName,
		Command: cmd,
		Image:   featurecountsImage,
		Inputs: []gobble.Bind{
			{Name: "bam", From: bam},
			{Name: "gtf", From: gtf},
		},
		Outputs:   []gobble.Bind{{Name: "counts", Spec: countsSpec}},
		Resources: opts.Resources,
	})
	return FeatureCountsPorts{Counts: task.Out("counts")}
}

func featureCountsStrand(s string) string {
	if s == "" {
		s = gobble.DefaultRNAStrandedness
	}
	switch s {
	case gobble.StrandednessUnstranded:
		return "0"
	case gobble.StrandednessForward:
		return "1"
	case gobble.StrandednessReverse:
		return "2"
	default:
		return "0"
	}
}
