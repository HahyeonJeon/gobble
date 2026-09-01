// Package atacseq owns the nf-core/atacseq 2.1.2-selected BWA ATAC-seq product.
//
// Build is the reusable typed entry point. Pipeline is the process-exclusive
// CLI adapter. The graph expands technical runs and biological replicates at
// compose time, resolves only explicit control links, and requires complete
// peak, BAM, count, QC, report, and session fan-in. Broad MACS2 peaks are the
// default; narrow peaks are a typed mode of the same path. The product has no
// alternate aligner, inferred control, contrast, IDR, motif, or footprinting
// route. DESeq2 output is cohort QC only.
package atacseq

import (
	"io"

	"github.com/HahyeonJeon/gobble"
	atacconsensuspeaks "github.com/HahyeonJeon/gobble/assets/modules/atac-consensus-peaks"
	atacfripscore "github.com/HahyeonJeon/gobble/assets/modules/atac-frip-score"
	atacreferenceintervals "github.com/HahyeonJeon/gobble/assets/modules/atac-reference-intervals"
	"github.com/HahyeonJeon/gobble/assets/modules/ataqv"
	ataqvmkarv "github.com/HahyeonJeon/gobble/assets/modules/ataqv-mkarv"
	bedgraphscale "github.com/HahyeonJeon/gobble/assets/modules/bedgraph-scale"
	bedtoolsgenomecov "github.com/HahyeonJeon/gobble/assets/modules/bedtools-genomecov"
	bedtoolsintersect "github.com/HahyeonJeon/gobble/assets/modules/bedtools-intersect"
	bwaindex "github.com/HahyeonJeon/gobble/assets/modules/bwa-index"
	bwamem "github.com/HahyeonJeon/gobble/assets/modules/bwa-mem"
	deeptoolscomputematrix "github.com/HahyeonJeon/gobble/assets/modules/deeptools-compute-matrix"
	deeptoolsplotfingerprint "github.com/HahyeonJeon/gobble/assets/modules/deeptools-plot-fingerprint"
	deeptoolsplotprofile "github.com/HahyeonJeon/gobble/assets/modules/deeptools-plot-profile"
	deseq2qc "github.com/HahyeonJeon/gobble/assets/modules/deseq2-qc"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	"github.com/HahyeonJeon/gobble/assets/modules/featurecounts"
	featurecountsmergematrices "github.com/HahyeonJeon/gobble/assets/modules/featurecounts-merge-matrices"
	homerannotatepeaks "github.com/HahyeonJeon/gobble/assets/modules/homer-annotate-peaks"
	igvsession "github.com/HahyeonJeon/gobble/assets/modules/igv-session"
	macs2callpeak "github.com/HahyeonJeon/gobble/assets/modules/macs2-callpeak"
	"github.com/HahyeonJeon/gobble/assets/modules/multiqc"
	picardcollectmultiplemetrics "github.com/HahyeonJeon/gobble/assets/modules/picard-collect-multiple-metrics"
	picardmarkduplicates "github.com/HahyeonJeon/gobble/assets/modules/picard-markduplicates"
	picardmergesamfiles "github.com/HahyeonJeon/gobble/assets/modules/picard-merge-sam-files"
	plothomerannotatepeaks "github.com/HahyeonJeon/gobble/assets/modules/plot-homer-annotatepeaks"
	plotmacs2qc "github.com/HahyeonJeon/gobble/assets/modules/plot-macs2-qc"
	samtoolsfaidx "github.com/HahyeonJeon/gobble/assets/modules/samtools-faidx"
	samtoolsflagstat "github.com/HahyeonJeon/gobble/assets/modules/samtools-flagstat"
	samtoolsidxstats "github.com/HahyeonJeon/gobble/assets/modules/samtools-idxstats"
	samtoolsindex "github.com/HahyeonJeon/gobble/assets/modules/samtools-index"
	samtoolssort "github.com/HahyeonJeon/gobble/assets/modules/samtools-sort"
	samtoolsstats "github.com/HahyeonJeon/gobble/assets/modules/samtools-stats"
	samtoolsview "github.com/HahyeonJeon/gobble/assets/modules/samtools-view"
	samtoolsviewcount "github.com/HahyeonJeon/gobble/assets/modules/samtools-view-count"
	trimgalore "github.com/HahyeonJeon/gobble/assets/modules/trim-galore"
	ucscbedgraphtobigwig "github.com/HahyeonJeon/gobble/assets/modules/ucsc-bedgraphtobigwig"
	wclines "github.com/HahyeonJeon/gobble/assets/modules/wc-lines"
	"github.com/HahyeonJeon/gobble/assets/pipelines"
)

const (
	// BenchmarkRelease is the dated upstream behavior benchmark.
	BenchmarkRelease = "nf-core/atacseq 2.1.2"
	// GraphGeneration names the first supported ATAC-seq graph generation.
	GraphGeneration = "atacseq-bwa-v1"
)

// ControlRef links one treatment replicate to an existing sample replicate.
// It does not assert that the control is scientifically appropriate.
type ControlRef struct {
	Sample    string
	Replicate int
}

// Run is one single- or paired-end technical sequencing run.
type Run struct {
	ID     string
	Fastq1 string
	Fastq2 string
}

// Replicate is one biological replicate with ordered technical runs and an
// optional explicit control dependency.
type Replicate struct {
	Number  int
	Runs    []Run
	Control *ControlRef
}

// Sample is one experimental sample or condition. Replicates must start at one
// without gaps and are copied by Build.
type Sample struct {
	Name       string
	Replicates []Replicate
}

// ReadyBWAIndex is an optional caller-staged BWA prefix and complete fixed
// sidecar Group. The zero value asks Build to prepare the index.
type ReadyBWAIndex struct {
	Prefix  gobble.PathSpec
	Members gobble.Group
}

// ReferenceConfig is the complete ATAC reference and annotation identity.
type ReferenceConfig struct {
	FASTA               gobble.PathSpec
	Annotation          gobble.PathSpec
	Blacklist           gobble.PathSpec
	BWAIndex            ReadyBWAIndex
	MitoName            string
	Organism            string
	ReadLength          int
	EffectiveGenomeSize string
}

// PeakMode selects broad or narrow output from the same MACS2 path.
type PeakMode string

const (
	PeakBroad  PeakMode = "broad"
	PeakNarrow PeakMode = "narrow"
)

// FilterPolicy owns selected ATAC alignment exclusions.
type FilterPolicy struct {
	MinimumMAPQ     int
	RemoveOrphans   bool
	RemoveMito      bool
	RemoveBlacklist bool
}

// PublicationPolicy names required result categories. They cannot be disabled
// while retaining this product identity.
type PublicationPolicy struct {
	FilteredAlignments bool
	Tracks             bool
	ReplicatePeaks     bool
	ConsensusMatrix    bool
	QC                 bool
	Reports            bool
	IGVSession         bool
}

// Config is the complete selected BWA analysis and task-build policy.
type Config struct {
	Reference        ReferenceConfig
	Results          gobble.Directory
	PeakMode         PeakMode
	Filters          FilterPolicy
	ConsensusMinimum int
	Publication      PublicationPolicy

	BWAIndex               bwaindex.Options
	SamtoolsFAIDX          samtoolsfaidx.Options
	ReferenceIntervals     atacreferenceintervals.Options
	FastQC                 fastqc.Options
	TrimGalore             trimgalore.Options
	BWAMem                 bwamem.Options
	SamtoolsSort           samtoolssort.Options
	SamtoolsIndex          samtoolsindex.Options
	SamtoolsStats          samtoolsstats.Options
	SamtoolsFlagstat       samtoolsflagstat.Options
	SamtoolsIdxstats       samtoolsidxstats.Options
	MergeRuns              picardmergesamfiles.Options
	MergeReplicates        picardmergesamfiles.Options
	MarkDuplicates         picardmarkduplicates.Options
	FilterBAM              samtoolsview.Options
	BlacklistFilter        bedtoolsintersect.Options
	CollectMultipleMetrics picardcollectmultiplemetrics.Options
	GenomeCoverage         bedtoolsgenomecov.Options
	ScaleCoverage          bedgraphscale.Options
	BedGraphToBigWig       ucscbedgraphtobigwig.Options
	ComputeMatrix          deeptoolscomputematrix.Options
	PlotProfile            deeptoolsplotprofile.Options
	PlotFingerprint        deeptoolsplotfingerprint.Options
	MACS2                  macs2callpeak.Options
	HOMER                  homerannotatepeaks.Options
	PlotMACS2QC            plotmacs2qc.Options
	PlotHOMERAnnotatePeaks plothomerannotatepeaks.Options
	PeakCount              wclines.Options
	PeakIntersect          bedtoolsintersect.Options
	ReadCount              samtoolsviewcount.Options
	FRiP                   atacfripscore.Options
	Consensus              atacconsensuspeaks.Options
	FeatureCounts          featurecounts.ATACOptions
	FeatureCountsMerge     featurecountsmergematrices.Options
	DESeq2QC               deseq2qc.Options
	Ataqv                  ataqv.Options
	Mkarv                  ataqvmkarv.Options
	IGV                    igvsession.Options
	MultiQC                multiqc.Options
}

// Lifecycle declares participation in every shared scenario. ATAC-seq has no
// pre-lift workspace because this generation introduces the product.
func Lifecycle() pipelines.LifecycleParticipation {
	return pipelines.CompleteLifecycle(GraphGeneration)
}

var (
	_ func(io.Reader) ([]Sample, error) = Parse
	_ func(string) ([]Sample, error)    = Load
)
