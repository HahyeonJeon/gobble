// Package scrnaseq owns the nf-core/scrnaseq 4.2.0-selected Simpleaf product.
//
// Build is the reusable typed entry point. Pipeline is the process-exclusive
// CLI adapter. Samples and repeated paired runs expand at compose time. Cells
// remain rows in declared matrices and never become scheduler members. The
// product runs only Simpleaf, QCatch, raw matrix conversions, cohort raw h5ad
// assembly, and MultiQC. It has no CellBender, Cell Ranger, alternate aligner,
// custom chemistry, or downstream single-cell analysis route.
package scrnaseq

import (
	"io"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules/anndatar-convert"
	catfastq "github.com/HahyeonJeon/gobble/assets/modules/cat-fastq"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	gffreadtranscriptome "github.com/HahyeonJeon/gobble/assets/modules/gffread-transcriptome"
	gtfgenefilter "github.com/HahyeonJeon/gobble/assets/modules/gtf-gene-filter"
	gtftot2g "github.com/HahyeonJeon/gobble/assets/modules/gtf-to-t2g"
	h5adconcat "github.com/HahyeonJeon/gobble/assets/modules/h5ad-concat"
	matrixtoh5ad "github.com/HahyeonJeon/gobble/assets/modules/matrix-to-h5ad"
	"github.com/HahyeonJeon/gobble/assets/modules/multiqc"
	"github.com/HahyeonJeon/gobble/assets/modules/qcatch"
	simpleafindex "github.com/HahyeonJeon/gobble/assets/modules/simpleaf-index"
	simpleafquant "github.com/HahyeonJeon/gobble/assets/modules/simpleaf-quant"
	"github.com/HahyeonJeon/gobble/assets/pipelines"
)

const (
	// BenchmarkRelease is the dated upstream behavior benchmark.
	BenchmarkRelease = "nf-core/scrnaseq 4.2.0"
	// GraphGeneration names the first supported scRNA-seq graph generation.
	GraphGeneration = "scrnaseq-simpleaf-v1"
)

// Protocol is one supported barcode-based 10x chemistry. It is never inferred.
type Protocol string

const (
	Protocol10xV1 Protocol = "10XV1"
	Protocol10xV2 Protocol = "10XV2"
	Protocol10xV3 Protocol = "10XV3"
	Protocol10xV4 Protocol = "10XV4"
)

// UMIResolution is one Simpleaf 0.19.5-supported resolution strategy.
type UMIResolution string

const (
	ResolutionCRLike          UMIResolution = "cr-like"
	ResolutionCRLikeEM        UMIResolution = "cr-like-em"
	ResolutionParsimony       UMIResolution = "parsimony"
	ResolutionParsimonyEM     UMIResolution = "parsimony-em"
	ResolutionParsimonyGene   UMIResolution = "parsimony-gene"
	ResolutionParsimonyGeneEM UMIResolution = "parsimony-gene-em"
)

// Run is one paired barcode/read technical sequencing run.
type Run struct {
	ID     string
	Fastq1 string
	Fastq2 string
}

// Sample is one logical sample with ordered paired technical runs. Optional
// expected-cell and sequencing-center metadata agree across repeated rows.
type Sample struct {
	Name          string
	Runs          []Run
	ExpectedCells int
	SeqCenter     string
}

// WhitelistConfig explicitly binds one staged barcode whitelist to its 10x
// protocol. Gobble does not inspect or infer chemistry from its filename.
type WhitelistConfig struct {
	Protocol Protocol
	Path     gobble.PathSpec
}

// ReferenceConfig selects exactly one reference form. FASTA plus Annotation
// asks Build to normalize the reference and produce a Simpleaf index Tree and
// transcript-to-gene relation. A ready SimpleafIndex requires an explicit
// TranscriptToGene file and leaves FASTA and Annotation unset. The ready
// TranscriptToGene path and SimpleafIndex Tree root must render to paths
// distinct from each other, the barcode whitelist, and all sample reads.
type ReferenceConfig struct {
	FASTA            gobble.PathSpec
	Annotation       gobble.PathSpec
	SimpleafIndex    gobble.Tree
	TranscriptToGene gobble.PathSpec
	BarcodeWhitelist WhitelistConfig
}

// PublicationPolicy names required result categories. Disabling one changes
// the selected product and therefore fails composition.
type PublicationPolicy struct {
	Index          bool
	Quantification bool
	QCatch         bool
	RawH5AD        bool
	RawRDS         bool
	CombinedH5AD   bool
	MultiQC        bool
}

// Config is the complete selected Simpleaf analysis and task-build policy.
type Config struct {
	Protocol      Protocol
	Reference     ReferenceConfig
	Results       gobble.Directory
	UMIResolution UMIResolution
	Publication   PublicationPolicy

	Consolidate      catfastq.Options
	FastQC           fastqc.Options
	GTFFilter        gtfgenefilter.Options
	Transcriptome    gffreadtranscriptome.Options
	TranscriptToGene gtftot2g.Options
	SimpleafIndex    simpleafindex.Options
	SimpleafQuant    simpleafquant.Options
	QCatch           qcatch.Options
	MatrixToH5AD     matrixtoh5ad.Options
	AnnDataR         anndatarconvert.Options
	H5ADConcat       h5adconcat.Options
	MultiQC          multiqc.Options
}

// Lifecycle declares participation in every shared scenario. scRNA-seq has no
// pre-lift workspace because this generation introduces the product.
func Lifecycle() pipelines.LifecycleParticipation {
	return pipelines.CompleteLifecycle(GraphGeneration)
}

var (
	_ func(io.Reader) ([]Sample, error) = Parse
	_ func(string) ([]Sample, error)    = Load
)
