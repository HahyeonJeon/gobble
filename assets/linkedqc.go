package assets

import "github.com/HahyeonJeon/gobble"

// LinkedQC returns an independent FastQC-plus-MultiQC pipeline. It
// AddInputs the same pinned FASTQ PathSpecs the assay constructors use.
// It does not call WGS, RNASeq, or MethylSeq.
func LinkedQC() *gobble.Pipeline {
	p := gobble.NewPipeline("linked-qc")
	r1 := p.AddInput("r1", pinnedWGSFASTQ1())
	r2 := p.AddInput("r2", pinnedWGSFASTQ2())
	qc1 := AddFastQC(AddModule(p, "r1"), r1, FastQCOptions{OutDir: gobble.Dir("work/r1/fastqc")})
	qc2 := AddFastQC(AddModule(p, "r2"), r2, FastQCOptions{OutDir: gobble.Dir("work/r2/fastqc")})
	AddMultiQC(p, []gobble.Handle{qc1.HTML, qc1.Zip, qc2.HTML, qc2.Zip}, MultiQCOptions{})
	return p
}
