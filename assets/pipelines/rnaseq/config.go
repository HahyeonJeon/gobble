package rnaseq

import (
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
	bedtoolsgenomecov "github.com/HahyeonJeon/gobble/assets/modules/bedtools-genomecov"
	catfastq "github.com/HahyeonJeon/gobble/assets/modules/cat-fastq"
	cutchromsizes "github.com/HahyeonJeon/gobble/assets/modules/cut-chrom-sizes"
	deseq2qc "github.com/HahyeonJeon/gobble/assets/modules/deseq2-qc"
	"github.com/HahyeonJeon/gobble/assets/modules/dupradar"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	"github.com/HahyeonJeon/gobble/assets/modules/featurecounts"
	fqlint "github.com/HahyeonJeon/gobble/assets/modules/fq-lint"
	"github.com/HahyeonJeon/gobble/assets/modules/gffread"
	gtffilter "github.com/HahyeonJeon/gobble/assets/modules/gtf-filter"
	"github.com/HahyeonJeon/gobble/assets/modules/gunzip"
	"github.com/HahyeonJeon/gobble/assets/modules/multiqc"
	picardmarkduplicates "github.com/HahyeonJeon/gobble/assets/modules/picard-markduplicates"
	qualimapbamqc "github.com/HahyeonJeon/gobble/assets/modules/qualimap-bamqc"
	rseqcinferexperiment "github.com/HahyeonJeon/gobble/assets/modules/rseqc-inferexperiment"
	salmonindex "github.com/HahyeonJeon/gobble/assets/modules/salmon-index"
	salmonquant "github.com/HahyeonJeon/gobble/assets/modules/salmon-quant"
	sampleretentionmapped "github.com/HahyeonJeon/gobble/assets/modules/sample-retention-mapped"
	sampleretentiontrimmed "github.com/HahyeonJeon/gobble/assets/modules/sample-retention-trimmed"
	samtoolsfaidx "github.com/HahyeonJeon/gobble/assets/modules/samtools-faidx"
	samtoolsindex "github.com/HahyeonJeon/gobble/assets/modules/samtools-index"
	samtoolssort "github.com/HahyeonJeon/gobble/assets/modules/samtools-sort"
	samtoolsstats "github.com/HahyeonJeon/gobble/assets/modules/samtools-stats"
	staralign "github.com/HahyeonJeon/gobble/assets/modules/star-align"
	stargenomegenerate "github.com/HahyeonJeon/gobble/assets/modules/star-genome-generate"
	"github.com/HahyeonJeon/gobble/assets/modules/stringtie"
	trimgalore "github.com/HahyeonJeon/gobble/assets/modules/trim-galore"
	"github.com/HahyeonJeon/gobble/assets/modules/tximport"
	ucscbedclip "github.com/HahyeonJeon/gobble/assets/modules/ucsc-bedclip"
	ucscbedgraphtobigwig "github.com/HahyeonJeon/gobble/assets/modules/ucsc-bedgraphtobigwig"
)

// DefaultConfig returns a fresh nf-core/rnaseq 3.26.0 STAR-Salmon config. It
// names caller-staged workspace inputs and performs no I/O or download.
func DefaultConfig() Config {
	return Config{
		Reference: ReferenceConfig{
			FASTA:         gobble.PathSpec{Dir: gobble.Dir("in/reference"), Base: "genome", Ext: ".fasta"},
			GTF:           gobble.PathSpec{Dir: gobble.Dir("in/reference"), Base: "genes_with_empty_tid", Ext: ".gtf.gz"},
			GTFCompressed: true,
		},
		Results: gobble.Dir("results/rnaseq"),
		SampleRemoval: SampleRemovalThresholds{
			MinTrimmedReads:  10000,
			MinMappedPercent: 5,
		},
		StrandednessInference: StrandednessInferenceThresholds{
			StrandedFraction:     0.8,
			UnstrandedDifference: 0.1,
		},
		Publication: PublicationPolicy{
			FinalBAMs:      true,
			Quantification: true,
			Matrices:       true,
			CoverageTracks: true,
			Reports:        true,
		},
		GTFFilter:        gtfFilterDefault(),
		GFFRead:          gffreadDefault(),
		Gunzip:           gunzipDefault(),
		STARGenome:       starGenomeDefault(),
		SalmonIndex:      salmonIndexDefault(),
		FAIDX:            faidxDefault(),
		ChromSizes:       cutChromSizesDefault(),
		CatFASTQ:         catDefault(),
		FQLint:           fqLintDefault(),
		FastQC:           fastQCDefault(),
		TrimGalore:       trimDefault(),
		TrimmedRetention: trimmedRetentionDefault(),
		STAR:             starDefault(),
		MappedRetention:  mappedRetentionDefault(),
		Salmon:           salmonDefault(),
		Sort:             sortDefault(),
		MarkDuplicates:   markDuplicatesDefault(),
		Index:            indexDefault(),
		Stats:            statsDefault(),
		StringTie:        stringTieDefault(),
		GenomeCov:        genomeCovDefault(),
		BedClip:          bedClipDefault(),
		BedGraphToBigWig: bigWigDefault(),
		RSeQC:            rseqcDefault(),
		Qualimap:         qualimapDefault(),
		DupRadar:         dupRadarDefault(),
		BiotypeQC:        biotypeDefault(),
		TxImport:         tximportDefault(),
		DESeq2QC:         deseq2Default(),
		MultiQC:          multiQCDefault(),
	}
}

func base(cpu float64, memory string) modules.Options {
	return modules.Options{Resources: gobble.Resources{CPU: cpu, Memory: memory}}
}

func gtfFilterDefault() gtffilter.Options { return gtffilter.Options{Options: base(1, "1g")} }
func gffreadDefault() gffread.Options     { return gffread.Options{Options: base(1, "1g")} }
func gunzipDefault() gunzip.Options       { return gunzip.Options{Options: base(1, "256m")} }
func starGenomeDefault() stargenomegenerate.Options {
	return stargenomegenerate.Options{Options: base(2, "6g"), SJDBOverhang: 100, GenomeSAIndexNBases: 7}
}
func salmonIndexDefault() salmonindex.Options { return salmonindex.Options{Options: base(2, "2g")} }
func faidxDefault() samtoolsfaidx.Options     { return samtoolsfaidx.Options{Options: base(1, "1g")} }
func cutChromSizesDefault() cutchromsizes.Options {
	return cutchromsizes.Options{Options: base(1, "256m")}
}
func catDefault() catfastq.Options    { return catfastq.Options{Options: base(1, "512m")} }
func fqLintDefault() fqlint.Options   { return fqlint.Options{Options: base(1, "512m")} }
func fastQCDefault() fastqc.Options   { return fastqc.Options{Options: base(2, "1g")} }
func trimDefault() trimgalore.Options { return trimgalore.Options{Options: base(4, "2g")} }
func trimmedRetentionDefault() sampleretentiontrimmed.Options {
	return sampleretentiontrimmed.Options{Options: base(1, "256m")}
}
func starDefault() staralign.Options { return staralign.Options{Options: base(4, "8g")} }
func mappedRetentionDefault() sampleretentionmapped.Options {
	return sampleretentionmapped.Options{Options: base(1, "256m")}
}
func salmonDefault() salmonquant.Options { return salmonquant.Options{Options: base(2, "2g")} }
func sortDefault() samtoolssort.Options  { return samtoolssort.Options{Options: base(2, "2g")} }
func markDuplicatesDefault() picardmarkduplicates.Options {
	return picardmarkduplicates.Options{Options: base(2, "4g")}
}
func indexDefault() samtoolsindex.Options { return samtoolsindex.Options{Options: base(1, "1g")} }
func statsDefault() samtoolsstats.Options { return samtoolsstats.Options{Options: base(1, "1g")} }
func stringTieDefault() stringtie.Options { return stringtie.Options{Options: base(2, "2g")} }
func genomeCovDefault() bedtoolsgenomecov.Options {
	return bedtoolsgenomecov.Options{Options: base(1, "1g")}
}
func bedClipDefault() ucscbedclip.Options { return ucscbedclip.Options{Options: base(1, "1g")} }
func bigWigDefault() ucscbedgraphtobigwig.Options {
	return ucscbedgraphtobigwig.Options{Options: base(1, "1g")}
}
func rseqcDefault() rseqcinferexperiment.Options {
	return rseqcinferexperiment.Options{Options: base(1, "2g")}
}
func qualimapDefault() qualimapbamqc.Options { return qualimapbamqc.Options{Options: base(2, "4g")} }
func dupRadarDefault() dupradar.Options      { return dupradar.Options{Options: base(2, "4g")} }
func biotypeDefault() featurecounts.BiotypeOptions {
	return featurecounts.BiotypeOptions{Options: base(2, "2g")}
}
func tximportDefault() tximport.Options { return tximport.Options{Options: base(2, "4g")} }
func deseq2Default() deseq2qc.Options   { return deseq2qc.Options{Options: base(2, "4g")} }
func multiQCDefault() multiqc.Options   { return multiqc.Options{Options: base(1, "2g")} }
