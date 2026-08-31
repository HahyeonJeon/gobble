# scRNA-seq Simpleaf product

`assets/pipelines/scrnaseq` owns graph generation `scrnaseq-simpleaf-v1`,
benchmarked against nf-core/scrnaseq 4.2.0. This generation introduces the
product. There is no earlier scRNA workspace to resume or migrate.

## Typed entry points

- `Parse` and `Load` accept exact columns `sample,fastq_1,fastq_2` plus optional
  `expected_cells,seq_center`. Unknown columns are errors. Repeated sample rows
  become ordered paired `Run` values. Both mates are required. Optional metadata
  must agree across repeated rows.
- `Protocol` is one of 10x V1, V2, V3, or V4 and is never inferred. The staged
  whitelist carries the same typed protocol. `DefaultConfig` selects V2 because
  the pinned official fixture is V2. V2–V4 also require their exact typed QCatch
  chemistry. QCatch has no V1 chemistry mapping, so V1 requires an explicit
  positive partition count instead.
- A source reference is an explicit FASTA and GTF. Build filters the GTF to
  reference sequences, extracts transcripts, produces a three-column
  transcript-to-gene relation, and builds a complete Simpleaf index `Tree`. A
  ready reference instead supplies a complete index `Tree` and a separate
  transcript-to-gene file. Source and ready forms cannot be mixed. A ready
  directory without `.gobble-tree.json` is incomplete and fails preflight.
- `DefaultConfig` returns fresh protocol, reference, UMI resolution, QCatch,
  conversion, output, immutable image, resource, argv, and publication policy.
  Its exact image-and-platform tuple is the supported tuple. A typed image
  replacement is recorded in task identity but leaves that support tuple.
- `Build(samples, config)` copies caller data and reads no process state,
  filesystem content, current directory, or network location. Invalid runs,
  protocols, reference forms, Trees, QCatch combinations, publication choices,
  and protected long, abbreviated, attached, or short aliases are structured
  compose defects.
- `Pipeline` is only the process-exclusive CLI adapter. It loads the injected
  samplesheet path and delegates to `Build` with fresh defaults. `cmd/gobble`
  remains asset-agnostic.

## Selected path and complete artifacts

Every original mate receives raw FastQC. Repeated paired runs are consolidated
mate by mate. Samples expand at compose time. Simpleaf maps and quantifies each
sample with the explicit protocol, complete index `Tree`, transcript-to-gene
file, whitelist, and UMI resolution. It emits separate complete map and
quantification `Tree` artifacts. QCatch consumes the complete quantification
`Tree` and emits its HTML report, metrics table, and separately filtered h5ad.
The raw Simpleaf matrix is converted independently to a provenance-named h5ad,
then to Seurat and SingleCellExperiment RDS files. Strict fan-in over every
sample raw h5ad creates `combined_raw_matrix.h5ad` with sample identity.
MultiQC consumes every FastQC report, quantification `Tree`, QCatch report, and
QCatch metrics table.

Cells and barcodes remain rows inside matrix artifacts. No runtime `Scatter`
represents a sample, barcode, or cell. No command scans an undeclared host
directory or fetches a reference, whitelist, read, or mapping at runtime.
`ExtraArgs` cannot replace the aligner, protocol, reads, index or quantification
Trees, relation, whitelist, QCatch input, or output roots.

## Lifecycle, recovery, and non-claims

A changed sample run affects that sample's consolidation, quantification,
QCatch, conversions, combined raw matrix, and aggregate report. A protocol,
whitelist, reference, or transcript-to-gene change affects every sample. A
QCatch-only setting preserves Simpleaf Trees and raw conversions while rerunning
QCatch and MultiQC. Required fan-in never accepts a missing sample matrix or
report. Stop, failure, Inspect, Release, and Resume use the shared engine
contract; this product adds no retry, fallback, repair, cleanup, or migration
verb.

Live fixture evidence fetches and verifies the nine exact official bytes,
stages them as ordinary product inputs, and drives every selected operation
through an input-consuming hermetic command boundary. This proves byte identity,
declared data flow, complete Tree publication, lifecycle state, and provenance
without Docker. It does not prove third-party scientific output parity or
registry availability.

Gobble does not claim that a barcode is a valid cell, that expected-cell
metadata is correct, that QCatch filtering is suitable, or that any raw,
filtered, or combined matrix is normalized, integrated, annotated, clustered,
or ready for scientific interpretation. Fixture cell metrics are not scientific
thresholds. CellBender, Cell Ranger, alternate aligners, custom chemistries,
Feature Barcode, VDJ, multiome, demultiplexing, differential analysis,
trajectory, RNA velocity, and nf-core endorsement are absent.
