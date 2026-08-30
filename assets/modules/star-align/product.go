package staralign

import (
	"strconv"
	"strings"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// DefaultImage is the nf-core/rnaseq 3.26.0 STAR image resolved for
// linux/amd64.
const DefaultImage modules.Image = "community.wave.seqera.io/library/htslib_samtools_star_gawk:ae438e9a604351a4@sha256:4a468118dbd7491a69bf9813c68233afa8558d1f3380fd8cab03e0e3d3135190"

// Options controls the selected STAR genome and transcriptome alignment.
type Options struct {
	modules.Options
	OutDir    gobble.Directory
	Sample    string
	ReadGroup string
	Platform  string
	Center    string
}

// Ports are the selected STAR alignment and report artifacts.
type Ports struct {
	GenomeBAM     gobble.Handle
	TranscriptBAM gobble.Handle
	Junctions     gobble.Handle
	LogFinal      gobble.Handle
}

// Add records one validated STAR alignment command. A zero read2 is
// single-end; otherwise the command is paired-end.
func Add(parent modules.Parent, index, gtf, read1, read2 gobble.Handle, options Options) (Ports, error) {
	const unit = "star_align"
	read1Path, err := modules.HandlePath(unit, read1)
	if err != nil {
		return Ports{}, err
	}
	gtfPath, err := modules.HandlePath(unit, gtf)
	if err != nil {
		return Ports{}, err
	}
	reads := []string{read1Path}
	if !read2.IsZero() {
		read2Path, pathErr := modules.HandlePath(unit, read2)
		if pathErr != nil {
			return Ports{}, pathErr
		}
		reads = append(reads, read2Path)
	}
	outDir := options.OutDir
	if outDir.IsZero() {
		outDir = defaultSTARAlignDir()
	}
	prefix := starAlignPrefix(outDir)
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = gobble.Resources{CPU: 4, Memory: "8g"}
	}
	command := []string{"STAR", "--genomeDir", index.Tree().Dir.String(), "--readFilesIn"}
	command = append(command, reads...)
	if strings.HasSuffix(strings.ToLower(read1Path), ".gz") {
		command = append(command, "--readFilesCommand", "zcat")
	}
	command = append(command, "--sjdbGTFfile", gtfPath, "--outFileNamePrefix", prefix, "--outSAMtype", "BAM", "Unsorted", "--quantMode", "TranscriptomeSAM", "GeneCounts")
	if n := modules.ThreadCount(resources.CPU); n > 0 {
		command = append(command, "--runThreadN", strconv.Itoa(n))
	}
	if options.ReadGroup != "" {
		command = append(command, "--outSAMattrRGline", "ID:"+options.ReadGroup, "SM:"+options.Sample)
		if options.Platform != "" {
			command = append(command, "PL:"+options.Platform)
		}
		if options.Center != "" {
			command = append(command, "CN:"+options.Center)
		}
	}
	base := options.Options
	base.Resources = resources
	command, image, resources, err := modules.ResolveOptions(unit, base, DefaultImage, resources, command, []string{"--genomeDir", "--readFilesIn", "--readFilesCommand", "--sjdbGTFfile", "--outFileNamePrefix", "--outSAMtype", "--quantMode", "--runThreadN", "--outSAMattrRGline"})
	if err != nil {
		return Ports{}, err
	}
	inputs := []gobble.Bind{{Name: "index", From: index, Tree: gobble.DeclareTree(outDir)}, {Name: "gtf", From: gtf}, {Name: "read1", From: read1}}
	// Restage the index at its own declared directory, not the alignment output.
	inputs[0].Tree = gobble.DeclareTree(index.Tree().Dir)
	if !read2.IsZero() {
		inputs = append(inputs, gobble.Bind{Name: "read2", From: read2})
	}
	genome := gobble.PathSpec{Dir: outDir, Base: "Aligned", Ext: ".out.bam"}
	transcript := gobble.PathSpec{Dir: outDir, Base: "Aligned.toTranscriptome", Ext: ".out.bam"}
	junctions := gobble.PathSpec{Dir: outDir, Base: "SJ", Ext: ".out.tab"}
	log := gobble.PathSpec{Dir: outDir, Base: "Log", Ext: ".final.out"}
	task := parent.AddTask(gobble.TaskSpec{
		Name: unit, Command: command, Image: image, Resources: resources, Inputs: inputs,
		Outputs: []gobble.Bind{{Name: "genome_bam", Spec: genome}, {Name: "transcript_bam", Spec: transcript}, {Name: "junctions", Spec: junctions}, {Name: "log_final", Spec: log}},
	})
	return Ports{GenomeBAM: task.Out("genome_bam"), TranscriptBAM: task.Out("transcript_bam"), Junctions: task.Out("junctions"), LogFinal: task.Out("log_final")}, nil
}

// Pipeline returns a standalone validated selected STAR alignment module.
func Pipeline(index gobble.Tree, gtf, read1, read2 gobble.PathSpec, options Options) *gobble.Pipeline {
	inputs := []modules.Input{{Name: "index", Tree: index}, {Name: "gtf", Spec: gtf}, {Name: "read1", Spec: read1}}
	if !productPathSpecUnset(read2) {
		inputs = append(inputs, modules.Input{Name: "read2", Spec: read2})
	}
	return modules.StandaloneChecked("star-align", inputs, func(parent modules.Parent, handles []gobble.Handle) error {
		var mate gobble.Handle
		if len(handles) > 3 {
			mate = handles[3]
		}
		_, err := Add(parent, handles[0], handles[1], handles[2], mate, options)
		return err
	})
}

func productPathSpecUnset(spec gobble.PathSpec) bool {
	return spec.Dir.IsZero() && spec.Prefix == "" && spec.Base == "" && len(spec.Suffixes) == 0 && spec.Ext == ""
}
