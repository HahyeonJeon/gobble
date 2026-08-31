// Package wgs owns the supported Sarek 3.10.0-derived WGS joint-germline
// product.
//
// Build is the reusable typed entry point. Pipeline is the process-exclusive
// CLI adapter. The product accepts at least two germline samples with paired
// lanes, performs BWA/GATK preprocessing, scatters interval work, and ends at
// an indexed unfiltered joint VCF. It has no somatic, VQSR, or alternate-caller
// route.
package wgs

import (
	"io"

	"github.com/HahyeonJeon/gobble"
	bcftoolssort "github.com/HahyeonJeon/gobble/assets/modules/bcftools-sort"
	bcftoolsstats "github.com/HahyeonJeon/gobble/assets/modules/bcftools-stats"
	bwaindex "github.com/HahyeonJeon/gobble/assets/modules/bwa-index"
	bwamem "github.com/HahyeonJeon/gobble/assets/modules/bwa-mem"
	"github.com/HahyeonJeon/gobble/assets/modules/fastp"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	gatk4applybqsr "github.com/HahyeonJeon/gobble/assets/modules/gatk4-applybqsr"
	gatk4baserecalibrator "github.com/HahyeonJeon/gobble/assets/modules/gatk4-baserecalibrator"
	gatk4gatherbqsrreports "github.com/HahyeonJeon/gobble/assets/modules/gatk4-gather-bqsr-reports"
	gatk4genomicsdbimport "github.com/HahyeonJeon/gobble/assets/modules/gatk4-genomicsdbimport"
	gatk4genotypegvcfs "github.com/HahyeonJeon/gobble/assets/modules/gatk4-genotypegvcfs"
	gatk4haplotypecaller "github.com/HahyeonJeon/gobble/assets/modules/gatk4-haplotypecaller"
	gatk4markduplicates "github.com/HahyeonJeon/gobble/assets/modules/gatk4-markduplicates"
	gatk4mergevcfs "github.com/HahyeonJeon/gobble/assets/modules/gatk4-mergevcfs"
	"github.com/HahyeonJeon/gobble/assets/modules/mosdepth"
	"github.com/HahyeonJeon/gobble/assets/modules/multiqc"
	samtoolsflagstat "github.com/HahyeonJeon/gobble/assets/modules/samtools-flagstat"
	samtoolsidxstats "github.com/HahyeonJeon/gobble/assets/modules/samtools-idxstats"
	samtoolsindex "github.com/HahyeonJeon/gobble/assets/modules/samtools-index"
	samtoolsmerge "github.com/HahyeonJeon/gobble/assets/modules/samtools-merge"
	samtoolssort "github.com/HahyeonJeon/gobble/assets/modules/samtools-sort"
	samtoolsstats "github.com/HahyeonJeon/gobble/assets/modules/samtools-stats"
	"github.com/HahyeonJeon/gobble/assets/pipelines"
)

const (
	// BenchmarkRelease is the dated upstream behavior benchmark.
	BenchmarkRelease = "nf-core/sarek 3.10.0"
	// GraphGeneration names the first joint-germline product generation.
	GraphGeneration = "wgs-joint-germline-v1"
)

// Lane is one paired sequencing lane belonging to a germline sample.
type Lane struct {
	ID     string
	Fastq1 string
	Fastq2 string
}

// Sample is one germline sample. Patient plus Name is its stable sample key.
// Lanes retains samplesheet order and is copied by Build. When present, Sex is
// experiment-data identity in the sample branch and downstream cohort joins;
// it is not caller policy.
type Sample struct {
	Patient string
	Name    string
	Sex     string
	Lanes   []Lane
}

// KnownSite is one indexed BQSR resource.
type KnownSite struct {
	Name  string
	VCF   gobble.PathSpec
	Index gobble.PathSpec
}

// ReadyBWAIndex is an optional caller-staged BWA prefix and complete fixed
// sidecar Group. The zero value asks Build to produce the index.
type ReadyBWAIndex struct {
	Prefix  gobble.PathSpec
	Members gobble.Group
}

// ReferenceConfig is the complete GATK/BWA reference and interval identity.
// Intervals is the non-empty stable runtime membership. DBSNP names one member
// of KnownSites for HaplotypeCaller and GenotypeGVCFs.
type ReferenceConfig struct {
	FASTA      gobble.PathSpec
	FAI        gobble.PathSpec
	Dictionary gobble.PathSpec
	BWAIndex   ReadyBWAIndex
	KnownSites []KnownSite
	DBSNP      string
	Intervals  gobble.Group
}

// OutputFormat is the selected recalibrated alignment representation.
type OutputFormat string

const (
	// OutputBAM selects indexed BAM outputs.
	OutputBAM OutputFormat = "bam"
)

// PublicationPolicy identifies required results and optional intermediates.
// Required categories cannot be disabled without changing this product.
type PublicationPolicy struct {
	RecalibratedAlignments bool
	SampleGVCFs            bool
	JointCallset           bool
	Reports                bool
	PreparedReads          bool
	GeneratedBWAIndex      bool
}

// Config is the complete selected BWA/GATK analysis and task-build policy. It
// has no caller selector, VQSR resource, somatic status, or skip list.
type Config struct {
	Reference   ReferenceConfig
	Results     gobble.Directory
	Format      OutputFormat
	Publication PublicationPolicy

	BWAIndex          bwaindex.Options
	FastQC            fastqc.Options
	FastP             fastp.Options
	BWAMem            bwamem.Options
	SamtoolsSort      samtoolssort.Options
	SamtoolsMerge     samtoolsmerge.Options
	MarkDuplicates    gatk4markduplicates.Options
	BaseRecalibrator  gatk4baserecalibrator.Options
	GatherBQSRReports gatk4gatherbqsrreports.Options
	ApplyBQSR         gatk4applybqsr.Options
	GatherBAM         samtoolsmerge.Options
	SamtoolsIndex     samtoolsindex.Options
	SamtoolsStats     samtoolsstats.Options
	SamtoolsFlagstat  samtoolsflagstat.Options
	SamtoolsIdxstats  samtoolsidxstats.Options
	Mosdepth          mosdepth.Options
	HaplotypeCaller   gatk4haplotypecaller.Options
	MergeGVCFs        gatk4mergevcfs.Options
	GenomicsDBImport  gatk4genomicsdbimport.Options
	GenotypeGVCFs     gatk4genotypegvcfs.Options
	BCFToolsSort      bcftoolssort.Options
	MergeJointVCFs    gatk4mergevcfs.Options
	BCFToolsStats     bcftoolsstats.Options
	MultiQC           multiqc.Options
}

// Contract is the compile-time WGS pipeline surface.
var Contract = pipelines.Contract[Sample, Config]{
	Parse: Parse, Load: Load, DefaultConfig: DefaultConfig, Build: Build, Pipeline: Pipeline,
}

// Lifecycle declares WGS participation in every shared scenario owner. The
// pre-lift two-sample alignment/QC checkpoint is a different graph generation.
var Lifecycle = pipelines.LifecycleParticipation{
	GraphGeneration: GraphGeneration,
	Design:          true, Build: true, Customize: true, Run: true,
	Resume: true, Stop: true, Failure: true, PreLiftResumable: false,
}

var (
	_ func(io.Reader) ([]Sample, error) = Parse
	_ func(string) ([]Sample, error)    = Load
)
