package assets

import "github.com/HahyeonJeon/gobble"

// MethylSeq returns a Methyl-seq proof pipeline. It calls first-party
// adders only and wires Handles. Reads and FASTA are the official
// Methyl-seq pin PathSpecs.
func MethylSeq() *gobble.Pipeline {
	p := gobble.NewPipeline("methylseq")
	fasta := p.AddInput("fasta", pinnedMethylFASTA())
	r1 := p.AddInput("r1", pinnedMethylFASTQ1())
	r2 := p.AddInput("r2", pinnedMethylFASTQ2())

	rawQC := AddFastQC(AddModule(p, "raw"), r1, FastQCOptions{OutDir: gobble.Dir("work/raw/fastqc")})
	fastp := AddFastp(p, r1, r2, FastpOptions{})
	cleanQC := AddFastQC(AddModule(p, "clean"), fastp.CleanR1, FastQCOptions{OutDir: gobble.Dir("work/clean/fastqc")})
	index := AddBismarkGenome(p, fasta, BismarkGenomeOptions{})
	align := AddBismarkAlign(p, fasta, index.Index, fastp.CleanR1, fastp.CleanR2, BismarkAlignOptions{})
	extractor := AddBismarkMethylationExtractor(p, align.BAM, BismarkMethylationExtractorOptions{})
	AddMultiQC(p, []gobble.Handle{
		rawQC.HTML, rawQC.Zip, cleanQC.HTML, cleanQC.Zip, fastp.JSON, fastp.HTML,
		align.Report, extractor.Report, extractor.Mbias, extractor.Coverage,
	}, MultiQCOptions{})
	return p
}
