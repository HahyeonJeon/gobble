# Products

This guide identifies the five current Gobble assay products and their owners.
It describes executable baseline
`f21a858c66a2d95ce8eff469e6db2bfa3240c3a5`. See [Authoring](authoring.md) for
inputs and outputs, [Operations](operations.md) for running and recovery, and
[Provenance](provenance.md) for evidence and support.

## Family boundary

“Multiomics” means a consistent family across five assay modalities. Each
product has its own data model, reference contract, graph, and outputs. The
products share typed Go construction, structured defects, immutable default
images, lifecycle evidence, and workspace recovery.

The family is **not** an integrated multiomics analysis. There is no universal
sample type, shared cohort table, cross-product import, multi-assay graph, join
identity, missing-modality policy, or cross-assay result. A future integration
needs a separate design that names all of those semantics.

## Product index

| Product | Package | Graph generation | Dated benchmark | Selected path |
|---|---|---|---|---|
| WGS joint germline | [`assets/pipelines/wgs`](../assets/pipelines/wgs) | `wgs-joint-germline-v1` | nf-core/sarek 3.10.0 | Paired FASTQ lanes through BWA/GATK preprocessing, interval HaplotypeCaller gVCFs, GenomicsDB, and gathered unfiltered joint VCF |
| Bulk RNA-seq | [`assets/pipelines/rnaseq`](../assets/pipelines/rnaseq) | `rnaseq-star-salmon-v1` | nf-core/rnaseq 3.26.0 | Repeated single- or paired-end runs through Trim Galore, STAR-Salmon, matrices, reference-guided outputs, cohort QC, and MultiQC |
| Methyl-seq | [`assets/pipelines/methylseq`](../assets/pipelines/methylseq) | `methylseq-bismark-v1` | nf-core/methylseq 4.2.0 | Repeated directional bisulfite runs through Trim Galore, Bismark/Bowtie2, deduplication, extraction, Bismark reports, and MultiQC |
| ATAC-seq | [`assets/pipelines/atacseq`](../assets/pipelines/atacseq) | `atacseq-bwa-v1` | nf-core/atacseq 2.1.2 | Technical runs and biological replicates through BWA, filtering, MACS2, strict consensus counts, cohort QC, ataqv, MultiQC, and IGV |
| scRNA-seq | [`assets/pipelines/scrnaseq`](../assets/pipelines/scrnaseq) | `scrnaseq-simpleaf-v1` | nf-core/scrnaseq 4.2.0 | Paired 10x V1–V4 runs through Simpleaf, QCatch, raw h5ad/RDS conversion, combined raw h5ad, and MultiQC |

An nf-core release is a dated behavior benchmark and source of selected prior
art. Gobble does not run Nextflow or copy every nf-core option. nf-core neither
supports nor endorses Gobble.

## Ownership

The exact repository ownership tree is:

```text
assets/
  modules/<command>/                 one command or subcommand
  pipelines/<assay>/                one supported assay graph
tests/
  modules/<command>/                 command-contract evidence
  pipelines/<assay>/                assay graph and data evidence
    testdata/manifest.json           sole assay fixture and image authority
  scenarios/<lifecycle-outcome>/     cross-product lifecycle evidence
```

The assay leaves are exactly `wgs`, `rnaseq`, `methylseq`, `atacseq`, and
`scrnaseq`. The lifecycle leaves are exactly `design`, `build`, `customize`,
`run`, `resume`, `stop`, and `failure`.

| Owner | Owns | Does not own |
|---|---|---|
| [`assets/modules/<command>`](../assets/modules) | One typed command task, options, ports, argv position, default image, and resources | Assay stage order, sample loops, cohort policy, or a module catalog |
| `assets/pipelines/<assay>` | Typed samples, strict sheet loading, defaults, graph policy, stage order, outputs, and graph generation | Engine occupancy, another assay, or an integrated family graph |
| [`tests/modules/<command>`](../tests/modules) | Focused command contract and execution evidence | Product construction or lifecycle policy |
| `tests/pipelines/<assay>` | Assay parsing, graph, stages, outputs, fixture preparation, manifest, and product evidence | Generic recovery policy |
| [`tests/scenarios/<outcome>`](../tests/scenarios) | One lifecycle outcome across every product | Duplicate builders, sheets, references, or manifests |
| Root package and [`testdata`](../testdata) | Engine and generic CLI fixtures | Assay fixtures |

`assets/modules` is a bounded source layer, not a registry. It has no remote
resolution, independent installation, or independent versioning. A command
module exists only because a selected product uses it.

## Package roles

Every command module has typed `Options` and `Ports`. Its parent adder records
one task in a product graph. Its standalone adapter wraps explicit File, Group,
or Tree inputs around that same adder for focused use and evidence. Standalone
modules do not define assay graphs.

Every assay package has the same public role split:

| Surface | Role |
|---|---|
| Typed `Sample`, `Run`, `Lane`, `Replicate`, or protocol values | Complete experiment-data model used during graph construction |
| `Parse(io.Reader)` and `Load(path)` | Convert the assay's exact CSV schema to typed samples; perform no graph work |
| `Config` | Complete analysis and task-build policy for the selected route |
| `DefaultConfig()` | Return a fresh, internally consistent default with workspace-relative inputs and immutable image choices; perform no I/O |
| `Build(samples, config)` | Copy caller-owned values, validate assay policy, and build the graph without process, directory, or network discovery |
| `Pipeline()` | Process-exclusive default CLI adapter; load the injected sheet path, create fresh defaults, and delegate to `Build` |
| `Lifecycle()` | Identify the current graph generation and participation in all seven scenario owners |

The generic [`cmd/gobble`](../cmd/gobble) remains product-agnostic. For graph
verbs it compiles a selected non-`internal` Go package and calls that package's
`Pipeline()`. A packed runner embeds one selected package and removes the
package operand and `pack` command.

## Support unit

A supported product is the exact tuple of package path, graph generation,
typed samples and config, selected stage set, task and artifact identities,
required outputs, default image digests, benchmark and fixture authority, and
the seven lifecycle outcomes. Changing a default command, task ID, port,
destination, image, sheet meaning, or graph generation is a compatibility and
recovery event. See [Migration](operations.md#migration) before reusing any
workspace across such a change.
