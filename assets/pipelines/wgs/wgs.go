// Package wgs owns the graph-stable WGS migration checkpoint.
package wgs

import (
	"github.com/HahyeonJeon/gobble"
	bwaindex "github.com/HahyeonJeon/gobble/assets/modules/bwa-index"
	bwamem "github.com/HahyeonJeon/gobble/assets/modules/bwa-mem"
	"github.com/HahyeonJeon/gobble/assets/modules/fastp"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	"github.com/HahyeonJeon/gobble/assets/modules/multiqc"
	samtoolsindex "github.com/HahyeonJeon/gobble/assets/modules/samtools-index"
	samtoolssort "github.com/HahyeonJeon/gobble/assets/modules/samtools-sort"
)

// pinnedWGSFASTQ1 is the workspace PathSpec for PinWGSTest1FASTQ.
func pinnedWGSFASTQ1() gobble.PathSpec {
	return gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_1", Ext: ".fastq.gz"}
}

// pinnedWGSFASTQ2 is the workspace PathSpec for PinWGSTest2FASTQ.
func pinnedWGSFASTQ2() gobble.PathSpec {
	return gobble.PathSpec{Dir: gobble.Dir("in"), Base: "test_2", Ext: ".fastq.gz"}
}

// pinnedWGSFASTA is the workspace PathSpec for PinWGSGenomeFASTA.
func pinnedWGSFASTA() gobble.PathSpec {
	return gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta"}
}

// pinnedWGSFAI is the workspace PathSpec for PinWGSGenomeFAI.
func pinnedWGSFAI() gobble.PathSpec {
	return gobble.PathSpec{Dir: gobble.Dir("in"), Base: "genome", Ext: ".fasta.fai"}
}

// Pipeline returns the graph-stable two-sample WGS checkpoint. It calls first-party
// adders only and wires Handles. The two samples share the pinned FASTQ
// PathSpecs under distinct input names. The pinned FAI is an AddInput
// because samtools faidx is omitted.
func Pipeline() *gobble.Pipeline {
	p := gobble.NewPipeline("wgs")
	fasta := p.AddInput("fasta", pinnedWGSFASTA())
	_ = p.AddInput("fai", pinnedWGSFAI())
	index := bwaindex.AddBWAIndex(p, fasta, bwaindex.BWAIndexOptions{})

	type sample struct {
		name string
		r1   string
		r2   string
	}
	samples := []sample{
		{name: "sample1", r1: "sample1_r1", r2: "sample1_r2"},
		{name: "sample2", r1: "sample2_r1", r2: "sample2_r2"},
	}

	var reports []gobble.Handle
	var rawReads gobble.Handle
	var cleanReads gobble.Handle
	for i, s := range samples {
		r1 := p.AddInput(s.r1, pinnedWGSFASTQ1())
		r2 := p.AddInput(s.r2, pinnedWGSFASTQ2())
		mod := p.AddModule(s.name)
		trimmed := fastp.AddFastp(mod, r1, r2, fastp.FastpOptions{OutDir: gobble.Dir("work/" + s.name + "/fastp")})
		mem := bwamem.AddBWAMem(mod, fasta, index.Index, trimmed.CleanR1, trimmed.CleanR2, bwamem.BWAMemOptions{
			OutDir: gobble.Dir("work/" + s.name + "/bwa-mem"),
		})
		sorted := samtoolssort.AddSamtoolsSort(mod, mem.SAM, samtoolssort.SamtoolsSortOptions{
			OutDir: gobble.Dir("work/" + s.name + "/samtools-sort"),
		})
		samtoolsindex.AddSamtoolsIndex(mod, sorted.BAM, samtoolsindex.SamtoolsIndexOptions{})
		reports = append(reports, trimmed.JSON, trimmed.HTML)
		if i == 0 {
			rawReads = r1
			cleanReads = trimmed.CleanR1
		}
	}

	rawQC := fastqc.AddFastQC(p.AddModule("raw"), rawReads, fastqc.FastQCOptions{OutDir: gobble.Dir("work/raw/fastqc")})
	cleanQC := fastqc.AddFastQC(p.AddModule("clean"), cleanReads, fastqc.FastQCOptions{OutDir: gobble.Dir("work/clean/fastqc")})
	multiqc.AddMultiQC(p, append([]gobble.Handle{rawQC.HTML, rawQC.Zip, cleanQC.HTML, cleanQC.Zip}, reports...), multiqc.MultiQCOptions{})
	return p
}
