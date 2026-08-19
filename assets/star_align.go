package assets

import (
	"strconv"
	"strings"

	"github.com/HahyeonJeon/gobble"
)

const starAlignTaskName = "star_align"

func defaultSTARAlignDir() gobble.Directory {
	return gobble.Dir("work/star-align")
}

func starAlignDir(dir gobble.Directory) gobble.Directory {
	if dir.IsZero() {
		return defaultSTARAlignDir()
	}
	return dir
}

func starAlignPrefix(dir gobble.Directory) string {
	s := dir.String()
	if s == "" || strings.HasSuffix(s, "/") {
		return s
	}
	return s + "/"
}

// STARAlignOptions are typed STAR align settings. ExtraArgs are argv
// tokens appended after named flags.
//
// --runThreadN copies Resources.CPU when CPU is at least 1, as an integer.
// GenomeDir is --genomeDir and must match the STAR genomeGenerate OutDir.
// OutDir is the parent folder of Aligned.out.bam and Log.final.out.
type STARAlignOptions struct {
	ExtraArgs []string
	Resources gobble.Resources
	OutDir    gobble.Directory
	GenomeDir gobble.Directory
}

// STARAlignPorts are the declared BAM and Log.final.out outputs.
type STARAlignPorts struct {
	BAM         gobble.Handle
	LogFinalOut gobble.Handle
}

// AddSTARAlign records one STAR align task on parent. index is the Tree
// handle from AddSTARGenomeGenerate. --genomeDir argv is the rendered
// Tree directory. The command emits BAM and Log.final.out and does not
// call samtools. The shared builder does not call AddInput.
func AddSTARAlign(parent Parent, index, r1, r2 gobble.Handle, opts STARAlignOptions) STARAlignPorts {
	return addSTARAlign(parent, index, r1, r2, opts)
}

// STARAlignPipeline returns a standalone STAR align pipeline. Index is
// a Tree pipeline input under GenomeDir, not a live STAR genomeGenerate
// run. Compose and BuildPlan do not need the members staged.
func STARAlignPipeline(r1, r2 gobble.PathSpec, opts STARAlignOptions) *gobble.Pipeline {
	return Standalone("star-align", []Input{
		{Name: "index", Tree: gobble.DeclareTree(starGenomeDir(opts.GenomeDir))},
		{Name: "r1", Spec: r1},
		{Name: "r2", Spec: r2},
	}, func(parent Parent, hs []gobble.Handle) {
		addSTARAlign(parent, hs[0], hs[1], hs[2], opts)
	})
}

func addSTARAlign(parent Parent, index, r1, r2 gobble.Handle, opts STARAlignOptions) STARAlignPorts {
	outDir := starAlignDir(opts.OutDir)
	genomeDir := starGenomeDir(opts.GenomeDir)
	bamSpec := gobble.PathSpec{Dir: outDir, Base: "Aligned", Ext: ".out.bam"}
	logSpec := gobble.PathSpec{Dir: outDir, Base: "Log", Ext: ".final.out"}

	r1path := mustCommandPath(r1.Spec())
	r2path := mustCommandPath(r2.Spec())
	cmd := []string{"STAR", "--genomeDir", genomeDir.String(), "--readFilesIn", r1path, r2path}
	if strings.HasSuffix(strings.ToLower(r1path), ".gz") {
		cmd = append(cmd, "--readFilesCommand", "zcat")
	}
	if n := threadCount(opts.Resources.CPU); n > 0 {
		cmd = append(cmd, "--runThreadN", strconv.Itoa(n))
	}
	cmd = append(cmd, "--outFileNamePrefix", starAlignPrefix(outDir), "--outSAMtype", "BAM", "Unsorted")
	cmd = AppendExtraArgs(cmd, opts.ExtraArgs)

	task := AddTask(parent, gobble.TaskSpec{
		Name:    starAlignTaskName,
		Command: cmd,
		Image:   starImage,
		Inputs: []gobble.Bind{
			{Name: "index", From: index, Tree: gobble.DeclareTree(genomeDir)},
			{Name: "r1", From: r1},
			{Name: "r2", From: r2},
		},
		Outputs: []gobble.Bind{
			{Name: "bam", Spec: bamSpec},
			{Name: "log_final", Spec: logSpec},
		},
		Resources: opts.Resources,
	})
	return STARAlignPorts{BAM: task.Out("bam"), LogFinalOut: task.Out("log_final")}
}
