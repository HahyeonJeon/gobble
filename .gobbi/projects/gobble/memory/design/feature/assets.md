# Assay product family and ownership

## Current products

The supported assay products are direct child packages below
`assets/pipelines`. Gobble is their shared engine and is not a sixth assay.

| Product | Package | Graph generation | Dated benchmark | Selected endpoint |
|---|---|---|---|---|
| WGS joint germline | `assets/pipelines/wgs` | `wgs-joint-germline-v1` | nf-core/sarek 3.10.0 | Indexed unfiltered gathered joint VCF |
| Bulk RNA-seq | `assets/pipelines/rnaseq` | `rnaseq-star-salmon-v1` | nf-core/rnaseq 3.26.0 | STAR-Salmon quantification, matrices, cohort QC, and reports |
| Methyl-seq | `assets/pipelines/methylseq` | `methylseq-bismark-v1` | nf-core/methylseq 4.2.0 | Directional Bismark deduplication, extraction, and reports |
| ATAC-seq | `assets/pipelines/atacseq` | `atacseq-bwa-v1` | nf-core/atacseq 2.1.2 | Filtered alignments, peaks, consensus counts, QC, and reports |
| scRNA-seq | `assets/pipelines/scrnaseq` | `scrnaseq-simpleaf-v1` | nf-core/scrnaseq 4.2.0 | Simpleaf and QCatch Trees, raw matrix conversions, and combined raw h5ad |

These are independent assay products with common engineering contracts. They do
not import one another or form an integrated multiomics graph.

## Ownership

One command or subcommand lives in one package below `assets/modules`. A module
owns typed `Options`, typed `Ports`, one task, its immutable default image,
resource request, argv-extra position, protected flags, and contained failure
boundary. Modules exist only when a selected product uses them. The module tree
is not a registry or independent distribution system.

One assay package owns strict CSV parsing, typed experiment values, complete
analysis config, fresh defaults, pure graph construction, default CLI adapter,
selected stages, required outputs, graph generation, and support limits. Package
`gobble` and `cmd/gobble` remain product-agnostic.

## Shared construction contract

Every assay package exposes `Parse`, `Load`, `DefaultConfig`, `Build`,
`Pipeline`, and `Lifecycle`. `Build` copies caller-owned values and performs no
ambient sheet, current-directory, filesystem, environment, or network lookup.
Only `Pipeline()` adapts the process-injected sheet path for the generic command.

Experiment data, analysis and task-build config, and engine run controls have
separate owners. `ExtraArgs` remains literal argv within the owning command and
cannot replace typed routes, inputs, outputs, or other protected options. File,
Group, and Tree ports match what each tool consumes. A ready Tree requires its
regular root manifest.

## Evidence and provenance

Command evidence follows each module under `tests/modules/<command>`. Assay
parsing, graph, stage, output, and fixture evidence follows each product under
`tests/pipelines/<assay>`. Each assay's schema-4
`testdata/manifest.json` is the sole authority for that assay's official bytes,
benchmark relation, and default-image inventory. Shared fetch and plan-check
code under `tests/internal` contains no assay fixture facts.

The seven lifecycle owners are `tests/scenarios/design`, `build`, `customize`,
`run`, `resume`, `stop`, and `failure`. They consume product packages and
assay-owned fixtures instead of copying graphs or manifests.

## Compatibility

The initial mechanical move of WGS, RNA-seq, and Methyl-seq preserved their old
graph facts. Their later main-path lifts are named graph generations and require
new workspaces. Temporary `assets.WGS`, `assets.RNASeq`, and
`assets.MethylSeq` shims now delegate to the lifted defaults. They preserve
source names only and are not permanent duplicate product owners.

ATAC-seq and scRNA-seq have no pre-lift workspace. Their current generations are
their first baselines.

## Retired proof owners

The former flat proof graphs are not current products. `LinkedQC` did not define
cross-assay identity or analysis. `OptionalMate` and synthetic operator graphs
were engine evidence. Useful behavior now belongs to module, pipeline, or
lifecycle tests. Removed proof-only `deseq2` and `merge-counts` modules have no
current authority; selected products use `deseq2-qc`, `tximport`, and
`featurecounts-merge-matrices` for their narrower current responsibilities.
