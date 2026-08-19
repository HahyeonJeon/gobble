package assets

import "github.com/HahyeonJeon/gobble"

// RNASeq returns an RNA-seq proof pipeline. It calls first-party adders
// only and wires Handles. Reads and FASTA are the WGS pin PathSpecs.
func RNASeq() *gobble.Pipeline {
	p := gobble.NewPipeline("rnaseq")
	fasta := p.AddInput("fasta", pinnedWGSFASTA())
	r1 := p.AddInput("r1", pinnedWGSFASTQ1())
	r2 := p.AddInput("r2", pinnedWGSFASTQ2())

	rawQC := AddFastQC(AddModule(p, "raw"), r1, FastQCOptions{OutDir: gobble.Dir("work/raw/fastqc")})
	fastp := AddFastp(p, r1, r2, FastpOptions{})
	cleanQC := AddFastQC(AddModule(p, "clean"), fastp.CleanR1, FastQCOptions{OutDir: gobble.Dir("work/clean/fastqc")})
	index := AddSTARGenomeGenerate(p, fasta, STARGenomeGenerateOptions{
		ExtraArgs: []string{"--genomeSAindexNbases", "7"},
	})
	align := AddSTARAlign(p, index.Index, fastp.CleanR1, fastp.CleanR2, STARAlignOptions{})
	sorted := AddSamtoolsSort(p, align.BAM, SamtoolsSortOptions{})
	AddSamtoolsIndex(p, sorted.BAM, SamtoolsIndexOptions{})
	AddMultiQC(p, []gobble.Handle{rawQC.HTML, rawQC.Zip, cleanQC.HTML, cleanQC.Zip, fastp.JSON, fastp.HTML}, MultiQCOptions{})
	return p
}
