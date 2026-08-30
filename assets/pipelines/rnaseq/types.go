// Package rnaseq owns the supported nf-core/rnaseq 3.26.0 STAR-Salmon bulk
// RNA-seq product.
//
// Build is the reusable typed entry point. Pipeline is the process-exclusive
// CLI adapter. The product supports repeated single- or paired-end runs and
// unstranded, forward, reverse, or automatically inferred libraries. DESeq2
// produces PCA and sample-distance QC only; this package has no study contrast
// or differential-expression result.
package rnaseq

import (
	"io"

	"github.com/HahyeonJeon/gobble"
	bedtoolsgenomecov "github.com/HahyeonJeon/gobble/assets/modules/bedtools-genomecov"
	catfastq "github.com/HahyeonJeon/gobble/assets/modules/cat-fastq"
	cutchromsizes "github.com/HahyeonJeon/gobble/assets/modules/cut-chrom-sizes"
	deseq2qc "github.com/HahyeonJeon/gobble/assets/modules/deseq2-qc"
	"github.com/HahyeonJeon/gobble/assets/modules/dupradar"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	"github.com/HahyeonJeon/gobble/assets/modules/featurecounts"
	fqlint "github.com/HahyeonJeon/gobble/assets/modules/fq-lint"
	"github.com/HahyeonJeon/gobble/assets/modules/gffread"
	"github.com/HahyeonJeon/gobble/assets/modules/gunzip"
	"github.com/HahyeonJeon/gobble/assets/modules/multiqc"
	picardmarkduplicates "github.com/HahyeonJeon/gobble/assets/modules/picard-markduplicates"
	qualimapbamqc "github.com/HahyeonJeon/gobble/assets/modules/qualimap-bamqc"
	rseqcinferexperiment "github.com/HahyeonJeon/gobble/assets/modules/rseqc-inferexperiment"
	salmonindex "github.com/HahyeonJeon/gobble/assets/modules/salmon-index"
	salmonquant "github.com/HahyeonJeon/gobble/assets/modules/salmon-quant"
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
	"github.com/HahyeonJeon/gobble/assets/pipelines"
)

const (
	// BenchmarkRelease is the dated upstream behavior benchmark.
	BenchmarkRelease = "nf-core/rnaseq 3.26.0"
	// GraphGeneration names the first STAR-Salmon Gobble graph generation.
	GraphGeneration = "rnaseq-star-salmon-v1"
)

// Strandedness is the assay-owned library-orientation value.
type Strandedness string

const (
	StrandednessUnstranded Strandedness = "unstranded"
	StrandednessForward   Strandedness = "forward"
	StrandednessReverse   Strandedness = "reverse"
	StrandednessAuto      Strandedness = "auto"
)

// Run is one explicit sequencing run belonging to a logical sample. ID is a
// stable Gobble name. Paths are workspace-relative regular files. Fastq2 is
// empty for single-end data.
type Run struct {
	ID     string
	Fastq1 string
	Fastq2 string
}

// Sample is one logical RNA sample after repeated rows are consolidated.
// Runs retains samplesheet order and is copied by Build.
type Sample struct {
	Name         string
	Runs         []Run
	Strandedness Strandedness
	SeqPlatform  string
	SeqCenter    string
}

// ReferenceConfig names a caller-staged FASTA, GTF, and optional ready
// indexes. GTFCompressed selects one explicit gzip decompression task. Zero
// ready Trees make Build generate the corresponding index.
type ReferenceConfig struct {
	FASTA       gobble.PathSpec
	GTF         gobble.PathSpec
	GTFCompressed bool
	STARIndex   gobble.Tree
	SalmonIndex gobble.Tree
}

// Config is the complete selected STAR-Salmon analysis and task-build policy.
// It has no aligner selector, quantifier selector, contrast, or skip list.
type Config struct {
	Reference ReferenceConfig
	Results   gobble.Directory

	GFFRead          gffread.Options
	Gunzip           gunzip.Options
	STARGenome       stargenomegenerate.Options
	SalmonIndex      salmonindex.Options
	FAIDX            samtoolsfaidx.Options
	ChromSizes       cutchromsizes.Options
	CatFASTQ         catfastq.Options
	FQLint           fqlint.Options
	FastQC           fastqc.Options
	TrimGalore       trimgalore.Options
	STAR             staralign.Options
	Salmon           salmonquant.Options
	Sort             samtoolssort.Options
	MarkDuplicates   picardmarkduplicates.Options
	Index            samtoolsindex.Options
	Stats            samtoolsstats.Options
	StringTie        stringtie.Options
	GenomeCov        bedtoolsgenomecov.Options
	BedClip          ucscbedclip.Options
	BedGraphToBigWig ucscbedgraphtobigwig.Options
	RSeQC            rseqcinferexperiment.Options
	Qualimap         qualimapbamqc.Options
	DupRadar         dupradar.Options
	BiotypeQC        featurecounts.BiotypeOptions
	TxImport         tximport.Options
	DESeq2QC         deseq2qc.Options
	MultiQC          multiqc.Options
}

// Contract is the compile-time RNA pipeline surface.
var Contract = pipelines.Contract[Sample, Config]{
	Parse:         Parse,
	Load:          Load,
	DefaultConfig: DefaultConfig,
	Build:         Build,
	Pipeline:      Pipeline,
}

// Lifecycle declares RNA participation in every shared scenario owner. A
// pre-lift featureCounts/two-group workspace is a different generation.
var Lifecycle = pipelines.LifecycleParticipation{
	GraphGeneration: GraphGeneration,
	Design: true, Build: true, Customize: true, Run: true,
	Resume: true, Stop: true, Failure: true, PreLiftResumable: false,
}

var (
	_ func(io.Reader) ([]Sample, error) = Parse
	_ func(string) ([]Sample, error)    = Load
)
