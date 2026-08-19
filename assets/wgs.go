package assets

import "github.com/HahyeonJeon/gobble"

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

// WGS returns a two-sample WGS proof pipeline. It calls first-party
// adders only and wires Handles. The two samples share the pinned FASTQ
// PathSpecs under distinct input names. The pinned FAI is an AddInput
// because samtools faidx is omitted.
func WGS() *gobble.Pipeline {
	p := gobble.NewPipeline("wgs")
	fasta := p.AddInput("fasta", pinnedWGSFASTA())
	_ = p.AddInput("fai", pinnedWGSFAI())
	index := AddBWAIndex(p, fasta, BWAIndexOptions{})

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
		mod := AddModule(p, s.name)
		fastp := AddFastp(mod, r1, r2, FastpOptions{OutDir: gobble.Dir("work/" + s.name + "/fastp")})
		mem := AddBWAMem(mod, fasta, index.Index, fastp.CleanR1, fastp.CleanR2, BWAMemOptions{
			OutDir: gobble.Dir("work/" + s.name + "/bwa-mem"),
		})
		sorted := AddSamtoolsSort(mod, mem.SAM, SamtoolsSortOptions{
			OutDir: gobble.Dir("work/" + s.name + "/samtools-sort"),
		})
		AddSamtoolsIndex(mod, sorted.BAM, SamtoolsIndexOptions{})
		reports = append(reports, fastp.JSON, fastp.HTML)
		if i == 0 {
			rawReads = r1
			cleanReads = fastp.CleanR1
		}
	}

	rawQC := AddFastQC(AddModule(p, "raw"), rawReads, FastQCOptions{OutDir: gobble.Dir("work/raw/fastqc")})
	cleanQC := AddFastQC(AddModule(p, "clean"), cleanReads, FastQCOptions{OutDir: gobble.Dir("work/clean/fastqc")})
	AddMultiQC(p, append([]gobble.Handle{rawQC.HTML, rawQC.Zip, cleanQC.HTML, cleanQC.Zip}, reports...), MultiQCOptions{})
	return p
}
