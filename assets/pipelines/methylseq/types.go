// Package methylseq owns the supported nf-core/methylseq 4.2.0 directional
// Bismark Methyl-seq product.
//
// Build is the reusable typed entry point. Pipeline is the process-exclusive
// CLI adapter. The product supports one or more logical samples, repeated
// single- or paired-end runs, a generated or ready Bismark index Tree,
// deduplication, comprehensive context extraction, Bismark reports, and
// MultiQC. It has no DMR or special-library route.
package methylseq

import (
	"io"

	"github.com/HahyeonJeon/gobble"
	bismarkalign "github.com/HahyeonJeon/gobble/assets/modules/bismark-align"
	bismarkdeduplicate "github.com/HahyeonJeon/gobble/assets/modules/bismark-deduplicate"
	bismarkgenome "github.com/HahyeonJeon/gobble/assets/modules/bismark-genome"
	bismarkmethylationextractor "github.com/HahyeonJeon/gobble/assets/modules/bismark-methylation-extractor"
	bismarkreport "github.com/HahyeonJeon/gobble/assets/modules/bismark-report"
	bismarksummary "github.com/HahyeonJeon/gobble/assets/modules/bismark-summary"
	catfastq "github.com/HahyeonJeon/gobble/assets/modules/cat-fastq"
	"github.com/HahyeonJeon/gobble/assets/modules/fastqc"
	"github.com/HahyeonJeon/gobble/assets/modules/multiqc"
	trimgalore "github.com/HahyeonJeon/gobble/assets/modules/trim-galore"
	"github.com/HahyeonJeon/gobble/assets/pipelines"
)

const (
	// BenchmarkRelease is the dated upstream behavior benchmark.
	BenchmarkRelease = "nf-core/methylseq 4.2.0"
	// GraphGeneration names the first truthful Bismark Tree generation.
	GraphGeneration = "methylseq-bismark-v1"
)

// Run is one explicit sequencing run belonging to a logical sample. ID is a
// stable Gobble name. Fastq2 is empty for single-end data.
type Run struct {
	ID     string
	Fastq1 string
	Fastq2 string
}

// Sample is one logical Methyl sample. Runs retains samplesheet order and is
// copied by Build.
type Sample struct {
	Name string
	Runs []Run
}

// LibraryMode is the analysis-owned Methyl library route. The selected product
// supports only standard directional bisulfite libraries.
type LibraryMode string

const (
	// LibraryModeDirectional selects the standard directional Bismark route.
	LibraryModeDirectional LibraryMode = "directional"
)

// ReferenceConfig names either a caller-staged FASTA or a ready Bismark index
// Tree. A ready Tree takes precedence and omits genome preparation. At run, the
// engine requires its root .gobble-tree.json in the caller's workspace. A
// generated Tree contains the staged FASTA and every Bismark-created member.
type ReferenceConfig struct {
	FASTA        gobble.PathSpec
	BismarkIndex gobble.Tree
}

// PublicationPolicy identifies required results and optional intermediates.
// Required categories cannot be disabled without changing this product.
type PublicationPolicy struct {
	DeduplicatedBAMs bool
	MethylationCalls bool
	Reports          bool
	TrimmedReads     bool
	GeneratedIndex   bool
}

// Config is the complete selected directional Bismark analysis and task-build
// policy. It has no aligner selector, library preset, DMR stage, or skip list.
type Config struct {
	Reference   ReferenceConfig
	LibraryMode LibraryMode
	Results     gobble.Directory
	Publication PublicationPolicy

	CatFASTQ      catfastq.Options
	FastQC        fastqc.Options
	TrimGalore    trimgalore.Options
	BismarkGenome bismarkgenome.Options
	BismarkAlign  bismarkalign.Options
	Deduplicate   bismarkdeduplicate.Options
	Extractor     bismarkmethylationextractor.Options
	Report        bismarkreport.Options
	Summary       bismarksummary.Options
	MultiQC       multiqc.Options
}

// Lifecycle declares Methyl participation in every shared scenario owner. A
// pre-lift Group-index/FastP workspace is a different graph generation.
func Lifecycle() pipelines.LifecycleParticipation {
	return pipelines.CompleteLifecycle(GraphGeneration)
}

var (
	_ func(io.Reader) ([]Sample, error) = Parse
	_ func(string) ([]Sample, error)    = Load
)
