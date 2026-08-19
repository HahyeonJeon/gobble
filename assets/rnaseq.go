package assets

import "github.com/HahyeonJeon/gobble"

// RNASeq returns an RNA-seq proof pipeline. It calls first-party adders
// only and wires Handles. Reads, FASTA, and GTF are the official RNA
// pins, not the WGS stand-ins.
func RNASeq() *gobble.Pipeline {
	p := gobble.NewPipeline("rnaseq")
	fasta := p.AddInput("fasta", pinnedRNAFASTA())
	gtf := p.AddInput("gtf", pinnedRNAGTF())
	r1 := p.AddInput("r1", pinnedRNAFASTQ1())
	r2 := p.AddInput("r2", pinnedRNAFASTQ2())

	rawQC := AddFastQC(AddModule(p, "raw"), r1, FastQCOptions{OutDir: gobble.Dir("work/raw/fastqc")})
	fastp := AddFastp(p, r1, r2, FastpOptions{})
	cleanQC := AddFastQC(AddModule(p, "clean"), fastp.CleanR1, FastQCOptions{OutDir: gobble.Dir("work/clean/fastqc")})
	index := AddSTARGenomeGenerate(p, fasta, gtf, STARGenomeGenerateOptions{
		ExtraArgs: []string{"--genomeSAindexNbases", "7", "--sjdbOverhang", "100"},
		Resources: gobble.Resources{CPU: 2},
	})
	align := AddSTARAlign(p, index.Index, fastp.CleanR1, fastp.CleanR2, STARAlignOptions{
		SJDB:      true,
		Resources: gobble.Resources{CPU: 2},
	})
	sorted := AddSamtoolsSort(p, align.BAM, SamtoolsSortOptions{})
	AddSamtoolsIndex(p, sorted.BAM, SamtoolsIndexOptions{})
	AddMultiQC(p, []gobble.Handle{
		rawQC.HTML, rawQC.Zip, cleanQC.HTML, cleanQC.Zip,
		fastp.JSON, fastp.HTML,
		align.LogFinalOut,
	}, MultiQCOptions{})
	return p
}
