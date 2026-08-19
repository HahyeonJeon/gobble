package assets

import "github.com/HahyeonJeon/gobble"

// LinkedQC returns an independent FastQC-plus-MultiQC pipeline. It
// AddInputs one official RNA FASTQ and one official Methyl FASTQ
// (pinnedRNAFASTQ1 and pinnedMethylFASTQ1). It does not call WGS,
// RNASeq, or MethylSeq.
func LinkedQC() *gobble.Pipeline {
	p := gobble.NewPipeline("linked-qc")
	rna := p.AddInput("rna", pinnedRNAFASTQ1())
	methyl := p.AddInput("methyl", pinnedMethylFASTQ1())
	qcRNA := AddFastQC(AddModule(p, "rna"), rna, FastQCOptions{OutDir: gobble.Dir("work/rna/fastqc")})
	qcMethyl := AddFastQC(AddModule(p, "methyl"), methyl, FastQCOptions{OutDir: gobble.Dir("work/methyl/fastqc")})
	AddMultiQC(p, []gobble.Handle{qcRNA.HTML, qcRNA.Zip, qcMethyl.HTML, qcMethyl.Zip}, MultiQCOptions{})
	return p
}
