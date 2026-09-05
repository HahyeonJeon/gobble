# Provenance

This guide identifies the authority behind the five product defaults and the
limits of that evidence. It also defines maintenance and support boundaries.
The original executable baseline is
`f21a858c66a2d95ce8eff469e6db2bfa3240c3a5`; subsequent image corrections are
recorded below and in the manifest at the selected Gobble revision.

## Authority

Authority is singular by concern:

| Concern | Authority |
|---|---|
| Product graph and typed defaults | `assets/pipelines/<assay>` source at the exact selected Gobble revision |
| Command contract | The selected `assets/modules/<command>` package |
| Assay fixture and default-image evidence | One schema-4 `tests/pipelines/<assay>/testdata/manifest.json` |
| Run state and reuse | The caller-owned workspace |
| Observed graph and state | Plan JSON and Inspect output; these are observations, not configuration stores |

An ignored host cache is disposable and never authoritative. A mutable branch,
tag without a content digest, public URL, container registry state, copied
scenario fixture, or successful run does not replace the sources above.

## Benchmarks

The dated benchmarks were selected on 2026-08-30. Each manifest binds the
release to an exact upstream source commit and exact test-data commit.

| Product | Selected release | Pipeline commit | Test-data commit | Manifest |
|---|---|---|---|---|
| WGS | nf-core/sarek 3.10.0 | `8ccac7ad37b05dd792447763bf9671b719824587` | `6c82958a6f302d8471a20855023ac59f9974fa8a` | [WGS](../tests/pipelines/wgs/testdata/manifest.json) |
| RNA-seq | nf-core/rnaseq 3.26.0 | `e7ca46272c8f9d5ceee3f71759f4ba551d3217a4` | `626c8fab639062eade4b10747e919341cbf9b41a` | [RNA-seq](../tests/pipelines/rnaseq/testdata/manifest.json) |
| Methyl-seq | nf-core/methylseq 4.2.0 | `5aa56467a85a5e2d6795ea72dfa5a5f0c9babc23` | `e7e1fb8940fc14e2336101147a31ce8e0eda6264` | [Methyl-seq](../tests/pipelines/methylseq/testdata/manifest.json) |
| ATAC-seq | nf-core/atacseq 2.1.2 | `1a1dbe52ffbd82256c941a032b0e22abbd925b8a` | `cd022b097372b078a68d8afadb172ad7342fd91f` | [ATAC-seq](../tests/pipelines/atacseq/testdata/manifest.json) |
| scRNA-seq | nf-core/scrnaseq 4.2.0 | `3fc17b4f971a89e47c88337de71d0e777ffad8cc` | `d934d6e8367fe2626184496b1889671cf2b02dab` | [scRNA-seq](../tests/pipelines/scrnaseq/testdata/manifest.json) |

The release is prior art and a behavior benchmark. It is not a runtime
dependency, complete feature-parity claim, scientific validation, or nf-core
endorsement. Gobble selects only the route documented in
[Products](products.md#product-index).

## Image pins

Fixture `entries` and image rows are separate record types. Every image row in
the five manifests records `reference`, `digest`, `tool`, `command`, `version`,
`license`, and `platform`. The remaining image fields differ by assay:

- WGS records `module`, `task_name`, `module_commit`, and `module_source`.
- RNA-seq and Methyl-seq record `modules`, `benchmark_pipeline`,
  `benchmark_release`, `module_commit`, `module_source`, `provenance`, and
  `license_source`. They do not record `task_name`.
- ATAC-seq and scRNA-seq record `module`, `task_name`, and `source`.

No image row records `license_authority` or `redistribution`. WGS's MultiQC
override additionally records `source` and `provenance`; other WGS rows and the
ATAC-seq/scRNA-seq rows omit those additional provenance fields. The image table
is therefore a pin and inventory record, not a complete license, provenance, or
redistribution authority.

The tag and digest form one identity. `latest`, a missing digest, and a
tag/digest disagreement are rejected. Mirroring remains inside the supported
tuple only when the resolved bytes match the accepted digest. A typed image
override is recorded in task identity and provenance, but behavioral support
for arbitrary replacement software is not claimed.

New upstream tags or releases never update defaults automatically. Change an
image only through a product release decision that assesses command behavior,
ports, outputs, license, security, and workspace-resume effects together.

During the v0.2.0 installed-assay checks, eight WGS digest references failed to
resolve to the manifests served for their existing tags. The
[image audit](../tests/pipelines/wgs/testdata/image-manifest-audit.json) records
the old identities and verified manifest hashes. The selected Wave MultiQC
build also failed to expose its CLI on PATH, so WGS now uses the upstream
MultiQC **v1.35** image with a pinned linux/amd64 manifest. Its command, output
paths, and declared tool version are retained. Installed Docker tests check
tool versions and real outputs. These corrections apply to new projects using
the updated runtime; existing runtime locks do not migrate automatically.

## Fixture staging

Each assay manifest records every directly or transitively consumed official
byte by logical role, immutable source URL and commit, byte count, SHA-256,
provenance, license source, redistribution rule, stage use, and benchmark
relation. Complete archive and Tree membership is explicit where applicable.

Hermetic tests never fetch. Live test support and the explicit `gobble demo`
preparation command may:

1. download an exact commit URL into a verified cache (the assay's ignored
   `cache/` for evidence tests, or `.gobble-cache/fixtures` for demo preparation);
2. verify the declared byte count and SHA-256;
3. verify required archive members where applicable; and
4. copy the verified bytes into a caller-created workspace as ordinary inputs.

A changed byte, unavailable source, missing or extra member, unknown required
license, or checksum mismatch is preparation failure. The preparer must not
accept the byte and rewrite the manifest automatically. Product source,
`DefaultConfig`, `Build`, and selected product tasks do not fetch fixtures,
references, reads, annotations, whitelists, or mappings. Operators stage their
own local data under the same declared input contract.

### Fixture limits

| Product | Official fixture role | Evidence limit |
|---|---|---|
| WGS | Two paired-read lineages plus split preprocessing and joint-call scenarios | The tiny region and tumor-derived engineering input do not establish variant quality; the final VCF remains unfiltered |
| RNA-seq | Small yeast repeated-run, single-/paired-end, and strandedness cases | Depth and reference do not establish expression validity, differential expression, or scientific thresholds |
| Methyl-seq | E. coli single-/paired-end reads and generated/ready Bismark indexes | Conversion and methylation values are not acceptance thresholds |
| ATAC-seq | Official osmotic-stress PE/SE runs, replicates, FASTA, and GTF | The hermetic command boundary proves declared byte flow, not third-party output parity, peak quality, or registry availability |
| scRNA-seq | Official 10x V2 repeated runs, chr19 reference/GTF, and whitelist | The oracle and output double prove command/bind identity and lifecycle behavior, not third-party execution, barcode validity, filtering quality, or registry availability |

No fixture cell, read, mapping, peak, FRiP, PCA, conversion, or replicate metric
is a product scientific threshold.

## Licenses

Gobble source is licensed under the [MIT License](../LICENSE). That license does
not automatically cover an embedded pipeline, container image, tool, reference,
annotation, whitelist, read set, accession, or generated artifact.

Before fetching, committing, packing, mirroring, or redistributing third-party
material, use the record for that material:

1. For a fixture or source byte in `entries`, read `license`, `license_source`,
   `redistribution`, and `provenance`. WGS, ATAC-seq, and scRNA-seq entries also
   record `license_authority`; RNA-seq and Methyl-seq entries do not.
2. For a container image or its tools, use the image row only as the pin and
   inventory described in [Image pins](#image-pins). No image row states
   redistribution terms. Verify permission and required notices from the
   applicable upstream license and image contents before mirroring or
   redistributing; do not infer permission from a public registry or pin.
3. Preserve the named copyright, permission notice, accession, dataset,
   reference, and tool attribution.
4. Apply any originating data or reference terms in addition to repository
   licensing.
5. Stop when permission or required provenance is unknown.

A public URL is not proof of redistribution permission. Ignored fixture caches
are not committed or redistributed. Packing a runner does not relicense its
embedded pipeline or dependencies. Do not put user-study data, secrets, or
credentials in a repository manifest.

## Evidence

The minimum hermetic regression check is:

```sh
GOTOOLCHAIN=local GOPROXY=off go test -count=1 ./...
```

It covers source contracts and lifecycle simulations without network access.
Live suites are explicit, prerequisite-dependent evidence. They may stage
pinned public data. Some product live paths use hermetic command boundaries or
frozen oracles instead of third-party tools; their manifest and fixture README
state the exact limit. Do not report a simulated output as tool parity or
scientific validation.

The seven scenario owners under [`tests/scenarios`](../tests/scenarios) cover
design, build, customize, run, resume, stop, and failure for every product.
They consume assay owners instead of duplicating graphs or manifests.

## Support

Support covers the current package paths, graph generations, typed contracts,
selected stages, required artifacts, pinned defaults, structured failures, and
Inspect–Release–Resume lifecycle on trusted-local `linux/amd64`. It is
engineering-only.

Support does not cover study design, reference recommendation, biological
thresholds, experimental suitability, clinical interpretation, scientific
validity, realistic cohort scale, throughput, fairness, autoscaling,
distributed filesystems, job arrays, retries, or remote execution. Reports
expose metrics; they do not issue Gobble scientific pass/fail decisions. Users
own consent, lawful use, local permissions, retention, and deletion of sensitive
data.

The product family is present only in current unreleased source. The immutable
`v0.1.0` tag predates it. A release carrying the family must name all five graph
generations and their Go API, CLI, workspace, image, and recovery effects.

## Releases

Release tags are immutable and supported install instructions never use
`@latest`. Before 1.0, a patch release means no intended break to the Go API,
CLI protocol, workspace schema, product graph generations, or recovery
behavior. A minor release may declare a break. Its notes must name effects on
the Go API, CLI, product packages, graph generations, workspaces, and recovery.
A release candidate uses `v0.x.y-rc.1` only when explicitly requested.

## Maintenance

Review a product when any of these triggers occurs:

- a selected nf-core release changes or removes a stage, command, image, sheet,
  output, or fixture;
- a default image is unavailable, changes digest, or receives a material
  security or license advisory;
- an official byte disappears, changes identity, or gains unclear provenance
  or redistribution terms;
- a command option, port, task ID, artifact kind, destination, sample meaning,
  or graph edge changes;
- a required output or one of the seven lifecycle outcomes stops passing; or
- support expands to a new platform, backend, route, protocol, assay, or claim.

For each trigger, compare the old and proposed state by stage, command, image,
data schema, output, license, support, and workspace recovery. Update pipeline
source, the sole manifest, focused command and product evidence, lifecycle
evidence, these guides, and release notes together. Do not let an upstream
addition silently become a Gobble requirement. Default changes and task-ID
drift require an explicit compatibility decision.

## Deferred work

Deferred means absent now, not experimental or partially supported:

- **WGS:** VQSR; somatic and tumor-normal calling; WES; SV, CNV, MSI, and
  annotation; UMI, GPU, and alternate callers.
- **RNA-seq:** study-specific differential expression; alternate aligners and
  pseudoaligners; UMI, rRNA, contaminant, RustQC, and de novo routes.
- **Methyl-seq:** BWA-Meth and TAPS; PBAT, RRBS, EM-seq, SLAM-seq, targeted,
  NOMe, Qualimap, Preseq, and DMR routes.
- **ATAC-seq:** alternate aligners, Preseq, IDR, motif discovery,
  footprinting, and differential-accessibility analysis.
- **scRNA-seq:** STARsolo, Kallisto/Bustools, Cell Ranger, CellBender, custom
  chemistries, Feature Barcode, VDJ, multiome, demultiplexing, annotation,
  clustering, integration, differential analysis, trajectory, and velocity.
- **Family and engine:** integrated cross-assay analysis; more assays;
  scientific benchmarking; HPC, cloud, services, remote execution, object
  storage, and institutional profiles.

Each deferred route needs a separate accepted design, exact benchmark, suitable
data, typed contract, outputs, provenance, lifecycle evidence, support boundary,
and compatibility decision.

## Rejected claims

The following proof-era statements are not current product capabilities:

- old limited WGS alignment/QC, RNA featureCounts/two-group DESeq2, and Methyl
  FastP/enumerated-index graphs are not supported final products;
- `LinkedQC` was aggregate QC over two files, not a sixth product or integrated
  multiomics analysis;
- `OptionalMate` and synthetic Scatter/Gather/When graphs were engine evidence,
  not assay products;
- a mechanical package move does not make a named graph lift resumable;
- official small data does not establish scientific or production-scale
  behavior; and
- using nf-core-selected paths, images, or data does not imply nf-core support
  or endorsement.
