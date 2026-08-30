package design

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	"github.com/HahyeonJeon/gobble/assets/modules/multiqc"
)

// LinkedQC returns an independent FastQC-plus-MultiQC design fixture. Its input
// paths preserve the old proof graph, but it owns no assay pin or product.
func LinkedQC() *gobble.Pipeline {
	p := gobble.NewPipeline("linked-qc")
	rna := p.AddInput("rna", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "SRR6357072_1", Ext: ".fastq.gz"})
	methyl := p.AddInput("methyl", gobble.PathSpec{Dir: gobble.Dir("in"), Base: "Ecoli_10K_methylated_R1", Ext: ".fastq.gz"})
	qcRNA := fastqc.AddFastQC(p.AddModule("rna"), rna, fastqc.FastQCOptions{OutDir: gobble.Dir("work/rna/fastqc")})
	qcMethyl := fastqc.AddFastQC(p.AddModule("methyl"), methyl, fastqc.FastQCOptions{OutDir: gobble.Dir("work/methyl/fastqc")})
	multiqc.AddMultiQC(p, []gobble.Handle{qcRNA.HTML, qcRNA.Zip, qcMethyl.HTML, qcMethyl.Zip}, multiqc.MultiQCOptions{})
	return p
}
