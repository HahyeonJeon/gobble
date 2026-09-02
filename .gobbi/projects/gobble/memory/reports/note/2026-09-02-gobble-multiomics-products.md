# Gobble multiomics products

The completed work establishes five local, unreleased, research-ready
engineering products on Gobble: WGS joint germline, bulk RNA-seq, Methyl-seq,
ATAC-seq, and scRNA-seq. Gobble is their shared Go engine and command surface;
it is not a sixth assay or an integrated multiomics orchestrator.

## Result

The executable product baseline is commit
`f21a858c66a2d95ce8eff469e6db2bfa3240c3a5`, tree
`c90dfe77192c2528f8fd54d17f4d9547b09a6998`. Current public documentation is
at accepted work head `4f2b922b170a5ede6b311c24d41da7665dff1583`, tree
`55677117c75c7d95a94cc6aa8b94c13eb10b1f2e`.

Each product owns strict assay data, typed `Config`, fresh `DefaultConfig`, pure
`Build`, a default `Pipeline()` adapter, selected command modules, required
artifacts, one immutable fixture and image manifest, and design, build,
customize, run, resume, stop, and failure evidence.

| Product | Current selected path |
|---|---|
| WGS | BWA/GATK preprocessing, interval HaplotypeCaller gVCFs, GenomicsDB, and an indexed unfiltered gathered joint VCF |
| RNA-seq | STAR-Salmon, tximport matrices, reference-guided outputs, cohort QC, and MultiQC |
| Methyl-seq | Directional Bismark/Bowtie2 with Trim Galore, deduplication, extraction, Bismark reports, and MultiQC |
| ATAC-seq | BWA, filtering, MACS2 peaks, strict consensus counts, cohort QC, ataqv, MultiQC, and IGV |
| scRNA-seq | Simpleaf, QCatch, raw h5ad and RDS conversion, combined raw h5ad, and MultiQC |

Command ownership now lives under `assets/modules/<command>`. Product graphs
live under `assets/pipelines/<assay>`. Assay and fixture authority lives under
`tests/pipelines/<assay>`, and cross-product lifecycle behavior lives under
`tests/scenarios/<outcome>`. Removed flat proof graphs and unused proof-only
modules no longer provide competing current authority.

## Compatibility

The WGS, RNA-seq, and Methyl-seq lifts are named graph changes. Pre-lift
workspaces do not resume with the current defaults. Temporary `assets.WGS`,
`assets.RNASeq`, and `assets.MethylSeq` shims preserve source names only. ATAC
and scRNA begin with their current first graph generations.

The immutable `v0.1.0` tag predates the five products. No product-family release
tag has been assigned. Consumers use an exact trusted local checkout and a
command built from the same selected module graph.

## Evidence

Planning and all nine execution groups earned PASS. The accepted worktree passed
`GOTOOLCHAIN=local GOPROXY=off go test -count=1 ./...` and
`git diff --check`. The root README and `docs/{products,authoring,operations,provenance}.md`
are the current public narrative.

Live Docker, registry, network, fixture-download, and third-party command
execution were not run during final closure verification. Their absence is an
evidence limit, not a failure and not a completion claim. Product-specific
hermetic command boundaries and frozen oracles retain their documented limits.

## Deferred outcomes

WGS VQSR, Methyl DMR, RNA multi-group differential analysis, and other retained
specific outcomes remain in [Backlog Memory](../../backlogs/README.md). The
current WGS endpoint remains unfiltered.

Salmon/tximport RNA quantification, assay-owned repeated-run or multi-lane
sheets, and WGS typed sheet construction are now current product behavior, so
their old backlog entries were removed. Related-file output `From` readiness is
resolved through Plan wait paths; no deferred run-local outcome remains.

## Limits

Support is engineering-only on trusted-local `linux/amd64` Docker execution.
Docker is not a sandbox. The work makes no scientific, clinical, diagnostic,
regulatory, production-scale, or nf-core-endorsement claim. It adds no
cross-assay analysis, remote backend, service, YAML/JSON product parameter
surface, component registry, public retry, or cleanup verb.
