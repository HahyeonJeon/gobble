# scRNA-seq Simpleaf product

`assets/pipelines/scrnaseq` owns graph generation `scrnaseq-simpleaf-v1`,
benchmarked against nf-core/scrnaseq 4.2.0. This generation introduces the
product. There is no earlier scRNA workspace to resume or migrate.

## Typed entry points

- `Parse` and `Load` accept exact columns `sample,fastq_1,fastq_2` plus optional
  `expected_cells,seq_center`. Unknown columns are errors. Repeated sample rows
  become ordered paired `Run` values. Both mates are required. Every FASTA, GTF,
  whitelist, and read input must render to a distinct workspace path. Reads also
  cannot reuse one path across samples, mates, or technical runs. Optional
  metadata must agree across repeated rows.
- `Protocol` is one of 10x V1, V2, V3, or V4 and is never inferred. The staged
  whitelist carries the same typed protocol. `DefaultConfig` selects V2 because
  the pinned official fixture is V2. V2–V4 also require their exact typed QCatch
  chemistry. QCatch has no V1 chemistry mapping, so V1 requires an explicit
  positive partition count instead.
- A source reference is an explicit FASTA and GTF. Build filters the GTF to
  reference sequences, extracts transcripts, produces a three-column
  transcript-to-gene relation, and builds a complete Simpleaf index `Tree`. A
  ready reference instead supplies a complete index `Tree` and a separate
  transcript-to-gene file. The ready transcript-to-gene path and index `Tree`
  root must render to paths distinct from each other, the whitelist, and every
  read. Source and ready forms cannot be mixed. A ready directory without
  `.gobble-tree.json` is incomplete and fails preflight.
- `DefaultConfig` returns fresh protocol, reference, UMI resolution, QCatch,
  conversion, output, immutable image, resource, argv, and publication policy.
  Its exact image-and-platform tuple is the supported tuple. A typed image
  replacement is recorded in task identity but leaves that support tuple.
- `Build(samples, config)` copies caller data and reads no process state,
  filesystem content, current directory, or network location. Invalid runs,
  protocols, reference forms, Trees, QCatch combinations, publication choices,
  and protected long, abbreviated, attached, or short aliases are structured
  compose defects. One module-owned policy derived from Simpleaf 0.19.5
  `IndexOpts` and `MapQuantOpts` at commit
  `935c013ff6123cc307791501dca86b92b1f6dd16` protects every documented
  spelling that competes with typed direct-reference inputs, quantification
  inputs, outputs, resources, permit-list filtering, or the selected Piscem
  route. Standalone adders and pipeline validation use the same policy.
- `Pipeline` is only the process-exclusive CLI adapter. It loads the injected
  samplesheet path and delegates to `Build` with fresh defaults. `cmd/gobble`
  remains asset-agnostic.

## Selected path and complete artifacts

Every original mate receives raw FastQC. Repeated paired runs are consolidated
mate by mate with Gobble's pinned 2026-08-30 `cat_fastq` image tuple. Its bytes
were previously recorded for nf-core/rnaseq 3.26.0; it is not an
nf-core/scrnaseq 4.2.0 module or image claim. Samples expand at compose time.
Simpleaf maps and quantifies each sample with the explicit protocol, complete
index `Tree`, transcript-to-gene file, whitelist, and UMI resolution. It emits
separate complete map and quantification `Tree` artifacts. QCatch consumes the
complete quantification `Tree` and emits its HTML report, metrics table, and
separately filtered h5ad.
The raw Simpleaf matrix is converted independently to a provenance-named h5ad,
then to Seurat and SingleCellExperiment RDS files. Strict fan-in over every
sample raw h5ad creates `combined_raw_matrix.h5ad` with sample identity.
MultiQC consumes every FastQC report, quantification `Tree`, QCatch report, and
QCatch metrics table.

Cells and barcodes remain rows inside matrix artifacts. No runtime `Scatter`
represents a sample, barcode, or cell. No command scans an undeclared host
directory or fetches a reference, whitelist, read, or mapping at runtime.
`ExtraArgs` cannot replace the aligner, expected read orientation, protocol,
reads, index or quantification Trees, relation, whitelist, QCatch input, or
output roots.

## Lifecycle, recovery, and non-claims

A changed sample run affects that sample's consolidation, quantification,
QCatch, conversions, combined raw matrix, and aggregate report. A protocol,
whitelist, reference, or transcript-to-gene change affects every sample. A
QCatch-only setting preserves Simpleaf Trees and raw conversions while rerunning
QCatch and MultiQC. Required fan-in never accepts a missing sample matrix or
report. Stop, failure, Inspect, Release, and Resume use the shared engine
contract; this product adds no retry, fallback, repair, cleanup, or migration
verb.

Live fixture evidence fetches and verifies the nine exact official bytes and
stages them as ordinary product inputs. An independent frozen oracle validates
each selected task's complete command-specific argv and compares every declared
input bind's producer port and exact SHA-256 identity set with its expected set.
The hermetic output double then
proves only engine occupancy, complete Tree publication, and lifecycle state;
its reads and placeholder bytes are not selected-command consumption evidence.
This path does not execute the selected third-party commands or prove their
scientific output parity or registry availability.

Gobble does not claim that a barcode is a valid cell, that expected-cell
metadata is correct, that QCatch filtering is suitable, or that any raw,
filtered, or combined matrix is normalized, integrated, annotated, clustered,
or ready for scientific interpretation. Fixture cell metrics are not scientific
thresholds. CellBender, Cell Ranger, alternate aligners, custom chemistries,
Feature Barcode, VDJ, multiome, demultiplexing, differential analysis,
trajectory, RNA velocity, and nf-core endorsement are absent.
