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
// OutDir is the parent folder of Aligned.out.bam.
type STARAlignOptions struct {
	ExtraArgs []string
	Resources gobble.Resources
	OutDir    gobble.Directory
	GenomeDir gobble.Directory
}

// STARAlignPorts are the declared BAM output.
type STARAlignPorts struct {
	BAM gobble.Handle
}

// AddSTARAlign records one STAR align task on parent. index is the
// Group handle from AddSTARGenomeGenerate. The command emits BAM and
// does not call samtools. The shared builder does not call AddInput.
func AddSTARAlign(parent Parent, index, r1, r2 gobble.Handle, opts STARAlignOptions) STARAlignPorts {
	return addSTARAlign(parent, index, r1, r2, opts)
}

// STARAlignPipeline returns a standalone STAR align pipeline. Index
// siblings are PathSpec-authored Group members under GenomeDir, not a
// live STAR genomeGenerate run. Pipeline inputs cannot be a Group, so
// the wrapper records a Group fixture task for AddSTARAlign to From.
func STARAlignPipeline(r1, r2 gobble.PathSpec, opts STARAlignOptions) *gobble.Pipeline {
	return Standalone("star-align", []Input{
		{Name: "r1", Spec: r1},
		{Name: "r2", Spec: r2},
	}, func(parent Parent, hs []gobble.Handle) {
		fixture := AddTask(parent, gobble.TaskSpec{
			Name:    "index_files",
			Command: []string{"true"},
			Outputs: []gobble.Bind{{Name: "index", Group: starGenomeGroup(starGenomeDir(opts.GenomeDir))}},
		})
		addSTARAlign(parent, fixture.Out("index"), hs[0], hs[1], opts)
	})
}

func addSTARAlign(parent Parent, index, r1, r2 gobble.Handle, opts STARAlignOptions) STARAlignPorts {
	outDir := starAlignDir(opts.OutDir)
	genomeDir := starGenomeDir(opts.GenomeDir)
	bamSpec := gobble.PathSpec{Dir: outDir, Name: "Aligned", Ext: ".out.bam"}

	cmd := []string{"STAR", "--genomeDir", genomeDir.String()}
	if path, err := CommandPath(r1.Spec()); err == nil {
		cmd = append(cmd, "--readFilesIn", path)
		if r2path, err := CommandPath(r2.Spec()); err == nil {
			cmd = append(cmd, r2path)
		}
		if strings.HasSuffix(strings.ToLower(path), ".gz") {
			cmd = append(cmd, "--readFilesCommand", "zcat")
		}
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
			{Name: "index", From: index, Group: starGenomeGroupFrom()},
			{Name: "r1", From: r1},
			{Name: "r2", From: r2},
		},
		Outputs:   []gobble.Bind{{Name: "bam", Spec: bamSpec}},
		Resources: opts.Resources,
	})
	return STARAlignPorts{BAM: task.Out("bam")}
}
