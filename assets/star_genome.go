package assets

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
)

// starImage is the nf-core/rnaseq 3.26.0 STAR 2.7.11b Seqera image from
// modules/nf-core/star/genomegenerate and star/align.
const starImage = "community.wave.seqera.io/library/htslib_samtools_star_gawk:ae438e9a604351a4"

const starGenomeTaskName = "star_genome_generate"

// starGenomeFile is one STAR --genomeDir regular file. member is the
// Group name and must stay injective under ^[a-zA-Z][a-zA-Z0-9_-]*$.
// name+ext is the on-disk filename and may contain dots.
type starGenomeFile struct {
	member string
	name   string
	ext    string
}

// starGenomeBaseFiles are the no-GTF genomeGenerate regular files.
var starGenomeBaseFiles = []starGenomeFile{
	{member: "Genome", name: "Genome"},
	{member: "SA", name: "SA"},
	{member: "SAindex", name: "SAindex"},
	{member: "chrLength", name: "chrLength", ext: ".txt"},
	{member: "chrName", name: "chrName", ext: ".txt"},
	{member: "chrNameLength", name: "chrNameLength", ext: ".txt"},
	{member: "chrStart", name: "chrStart", ext: ".txt"},
	{member: "genomeParameters", name: "genomeParameters", ext: ".txt"},
}

// starGenomeSJDBFiles are extra regular files STAR 2.7.11b wrote into
// --genomeDir on one live genomeGenerate with --sjdbGTFfile. Member
// names stay injective after dots are dropped from the filename.
var starGenomeSJDBFiles = []starGenomeFile{
	{member: "Log", name: "Log", ext: ".out"},
	{member: "exonGeTrInfo", name: "exonGeTrInfo", ext: ".tab"},
	{member: "exonInfo", name: "exonInfo", ext: ".tab"},
	{member: "geneInfo", name: "geneInfo", ext: ".tab"},
	{member: "sjdbInfo", name: "sjdbInfo", ext: ".txt"},
	{member: "sjdbListFromGTF", name: "sjdbList", ext: ".fromGTF.out.tab"},
	{member: "sjdbListOut", name: "sjdbList", ext: ".out.tab"},
	{member: "transcriptInfo", name: "transcriptInfo", ext: ".tab"},
}

func defaultSTARGenomeDir() gobble.Directory {
	return gobble.Dir("work/star-genome")
}

func starGenomeDir(dir gobble.Directory) gobble.Directory {
	if dir.IsZero() {
		return defaultSTARGenomeDir()
	}
	return dir
}

func starGenomeFiles(sjdb bool) []starGenomeFile {
	if !sjdb {
		return starGenomeBaseFiles
	}
	out := make([]starGenomeFile, 0, len(starGenomeBaseFiles)+len(starGenomeSJDBFiles))
	out = append(out, starGenomeBaseFiles...)
	out = append(out, starGenomeSJDBFiles...)
	return out
}

// STARGenomeGenerateOptions are typed STAR genomeGenerate settings.
// ExtraArgs are argv tokens appended after named flags.
//
// --runThreadN copies Resources.CPU when CPU is at least 1, as an integer.
// OutDir is --genomeDir, the parent folder of the index Group, not a
// directory port. GTF is the optional standalone --sjdbGTFfile PathSpec.
type STARGenomeGenerateOptions struct {
	ExtraArgs []string
	Resources gobble.Resources
	OutDir    gobble.Directory
	GTF       gobble.PathSpec
}

// STARGenomeGeneratePorts are the declared STAR index Group output.
type STARGenomeGeneratePorts struct {
	Index gobble.Handle
}

// AddSTARGenomeGenerate records one STAR genomeGenerate task on parent.
// A zero gtf keeps the no-GTF 8-member index Group. A non-zero gtf adds
// --sjdbGTFfile, binds the GTF, and expands the Group. Index siblings
// are regular files. The parent folder is PathSpec.Dir, not a directory
// port. The shared builder does not call AddInput.
func AddSTARGenomeGenerate(parent Parent, fasta, gtf gobble.Handle, opts STARGenomeGenerateOptions) STARGenomeGeneratePorts {
	return addSTARGenomeGenerate(parent, fasta, gtf, opts)
}

// STARGenomeGeneratePipeline returns a standalone STAR genomeGenerate
// pipeline. It AddInputs fasta and, when opts.GTF is set, gtf, then
// calls the same builder as AddSTARGenomeGenerate.
func STARGenomeGeneratePipeline(fasta gobble.PathSpec, opts STARGenomeGenerateOptions) *gobble.Pipeline {
	inputs := []Input{{Name: "fasta", Spec: fasta}}
	if !pathSpecUnset(opts.GTF) {
		inputs = append(inputs, Input{Name: "gtf", Spec: opts.GTF})
	}
	return Standalone("star-genome-generate", inputs, func(parent Parent, hs []gobble.Handle) {
		var gtf gobble.Handle
		if len(hs) > 1 {
			gtf = hs[1]
		}
		addSTARGenomeGenerate(parent, hs[0], gtf, opts)
	})
}

func addSTARGenomeGenerate(parent Parent, fasta, gtf gobble.Handle, opts STARGenomeGenerateOptions) STARGenomeGeneratePorts {
	outDir := starGenomeDir(opts.OutDir)
	sjdb := !gtf.IsZero()

	cmd := []string{"STAR", "--runMode", "genomeGenerate", "--genomeDir", outDir.String()}
	cmd = append(cmd, "--genomeFastaFiles", starCommandPath(fasta.Spec()))
	if sjdb {
		cmd = append(cmd, "--sjdbGTFfile", starCommandPath(gtf.Spec()))
	}
	if n := threadCount(opts.Resources.CPU); n > 0 {
		cmd = append(cmd, "--runThreadN", strconv.Itoa(n))
	}
	cmd = AppendExtraArgs(cmd, opts.ExtraArgs)

	inputs := []gobble.Bind{{Name: "fasta", From: fasta}}
	if sjdb {
		inputs = append(inputs, gobble.Bind{Name: "gtf", From: gtf})
	}

	task := AddTask(parent, gobble.TaskSpec{
		Name:      starGenomeTaskName,
		Command:   cmd,
		Image:     starImage,
		Inputs:    inputs,
		Outputs:   []gobble.Bind{{Name: "index", Group: starGenomeGroup(outDir, sjdb)}},
		Resources: opts.Resources,
	})
	return STARGenomeGeneratePorts{Index: task.Out("index")}
}

func starGenomeGroup(dir gobble.Directory, sjdb bool) gobble.Group {
	files := starGenomeFiles(sjdb)
	g := make(gobble.Group, 0, len(files))
	for _, f := range files {
		g = append(g, gobble.Member{Name: f.member, Spec: gobble.PathSpec{Dir: dir, Name: f.name, Ext: f.ext}})
	}
	return g
}

func starGenomeGroupFrom(sjdb bool) gobble.Group {
	files := starGenomeFiles(sjdb)
	g := make(gobble.Group, 0, len(files))
	for _, f := range files {
		g = append(g, gobble.Member{Name: f.member})
	}
	return g
}

// starCommandPath renders spec for a STAR argv token. A render error
// still returns an empty token so the flag is not silently omitted.
func starCommandPath(spec gobble.PathSpec) string {
	path, err := CommandPath(spec)
	if err != nil {
		return ""
	}
	return path
}

func pathSpecUnset(p gobble.PathSpec) bool {
	return p.Dir.IsZero() && p.Lead == "" && p.Name == "" && len(p.Steps) == 0 && p.Ext == ""
}
