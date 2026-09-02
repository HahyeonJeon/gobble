# Gobble

Gobble is a pre-1.0 Go pipeline engine and a family of five trusted-local
bioinformatics pipeline products. It serves coding agents and Go authors through
the library and generic `gobble` command. A human operator can instead receive
one packed runner containing one pipeline.

Support is limited to engineering behavior on `linux/amd64`: graph construction,
local Docker command execution, declared artifacts, provenance, structured
failure, and recovery. It does not establish scientific, clinical, diagnostic,
or regulatory validity. Docker isolation is not a sandbox.

## Products

| Product | Selected path | Benchmark | Guide |
|---|---|---|---|
| WGS | BWA/GATK joint germline through an unfiltered joint VCF | nf-core/sarek 3.10.0 | [WGS](docs/authoring.md#wgs) |
| Bulk RNA-seq | STAR-Salmon, reference-guided outputs, cohort QC | nf-core/rnaseq 3.26.0 | [RNA-seq](docs/authoring.md#rna-seq) |
| Methyl-seq | Directional Bismark/Bowtie2 | nf-core/methylseq 4.2.0 | [Methyl-seq](docs/authoring.md#methyl-seq) |
| ATAC-seq | BWA, MACS2 peaks, consensus counts, cohort QC | nf-core/atacseq 2.1.2 | [ATAC-seq](docs/authoring.md#atac-seq) |
| scRNA-seq | Simpleaf, QCatch, raw matrix conversion and assembly | nf-core/scrnaseq 4.2.0 | [scRNA-seq](docs/authoring.md#scrna-seq) |

These are five independent assay products with common engineering contracts.
They are not an integrated multiomics analysis. No package joins modalities,
defines cross-assay identity, or produces a cross-assay scientific result.

## Documentation

- [Products](docs/products.md): exact product identities, ownership, package
  roles, dated paths, and family boundary.
- [Authoring](docs/authoring.md): samplesheets, typed configuration, selected
  stages, outputs, defects, and scientific limits.
- [Operations](docs/operations.md): installation, execution, trust, lifecycle,
  migration, occupancy, and recovery.
- [Provenance](docs/provenance.md): benchmark and pin authority, fixture
  staging, attribution, support, maintenance, and deferred routes.
- [Changelog](CHANGELOG.md): released history and current unreleased state.

## Release state

The immutable `v0.1.0` tag is the published engine preview. It predates the
five-product family and does not contain these product packages. The product
bytes documented here are frozen at commit
`f21a858c66a2d95ce8eff469e6db2bfa3240c3a5` (tree
`c90dfe77192c2528f8fd54d17f4d9547b09a6998`). No product-family release tag has
been assigned. This documentation closure changes no executable file. Until
release notes name these graph generations, use an exact trusted local checkout
containing that baseline and build the command from the same selected revision.
Do not expect `v0.1.0` or `@latest` to provide the products.

## Agent install

Agents, library consumers, and the machine creating a packed runner require Go
1.26 or newer. For the unreleased product baseline, put an exact trusted checkout
in the consumer module graph and build the command from that graph:

```sh
go mod edit -require=github.com/HahyeonJeon/gobble@v0.0.0
go mod edit -replace=github.com/HahyeonJeon/gobble=/absolute/path/to/gobble
go list -m github.com/HahyeonJeon/gobble
mkdir -p .gobbin
GOBIN="$PWD/.gobbin" go install github.com/HahyeonJeon/gobble/cmd/gobble
export PATH="$PWD/.gobbin:$PATH"
```

The unsuffixed `go install` uses the consumer's selected local module. The
`v0.0.0` requirement is only a local module-graph placeholder.

The released engine-only `v0.1.0` install remains:

```sh
go get github.com/HahyeonJeon/gobble@v0.1.0
GOBIN="$PWD/.gobbin" go install github.com/HahyeonJeon/gobble/cmd/gobble@v0.1.0
```

Keep the exact selected command on `PATH` for graph verbs. Consumer packages
under `internal/` are unsupported.

## First run

Choose one package from [Products](docs/products.md), prepare its host
samplesheet, and create an exclusive workspace. Stage every input named by the
sheet or typed reference config as a regular local file. Sheet cells and typed
reference paths are workspace-relative. Product construction and tasks do not
fetch them.

Run and Resume require a usable local Docker client in `/usr/bin` or `/bin` and
a reachable daemon. Before the first Run, the following command must succeed for
the same local user: `env -i PATH=/usr/bin:/bin docker info`. Gobble inspects
each exact Plan image and pulls it when it is absent from the daemon's local
image store. Offline use requires pre-staging every exact
`registry/repository:tag@sha256:digest` image before disconnecting. The
task-container `--network=none` setting does not apply to image acquisition.

```sh
PIPELINE=github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq
SHEET=/absolute/path/to/rnaseq-samplesheet.csv
WORKSPACE=/absolute/path/to/new-exclusive-workspace

mkdir "$WORKSPACE"
# Stage the sheet's reads and DefaultConfig's reference inputs in the workspace.
gobble compose "$PIPELINE" --sample "$SHEET"
gobble validate "$PIPELINE" --sample "$SHEET"
gobble plan "$PIPELINE" --sample "$SHEET" > plan.json
gobble run "$PIPELINE" --workspace "$WORKSPACE" --sample "$SHEET"
gobble inspect run --workspace "$WORKSPACE"
gobble release --workspace "$WORKSPACE"
```

Review Plan JSON before running. It exposes commands, parameters, resources,
images, binds, and destinations. The default adapter uses `samplesheet.csv` in
the process directory when `--sample` is omitted.

A Run or Resume start-preflight defect rejects the start before workspace
occupancy or task execution. Docker inspection or pull failure occurs later,
during task submission and before that task's command starts. It records a
contained failed task and leaves occupancy active. A task-command failure means
the container started and its command exited unsuccessfully. Inspect the failed
unit, then follow [Recovery](docs/operations.md#recovery).

For typed customization, call the selected package's `Load`, `DefaultConfig`,
and `Build` from a caller-owned Go package. Do not edit product source or use a
serialized parameter overlay. See [Authoring](docs/authoring.md#entry-roles).

## Human runner

An agent with Go and the exact consumer module creates a runner at an explicit
path:

```sh
gobble pack github.com/HahyeonJeon/gobble/assets/pipelines/rnaseq --output ./rnaseq-runner
```

The human uses the embedded package without Go, a package operand, or `pack`:

```text
./rnaseq-runner compose [--sample PATH]
./rnaseq-runner validate [--sample PATH]
./rnaseq-runner plan [--sample PATH]
./rnaseq-runner run --workspace DIR [--cap N] [--sample PATH]
./rnaseq-runner inspect VIEW --workspace DIR [--instance ID]
./rnaseq-runner release --workspace DIR
./rnaseq-runner resume --workspace DIR [--cap N] [--sample PATH]
```

Gobble portions are [MIT licensed](LICENSE). An embedded pipeline, its tools,
images, and data can carry other licenses. Packing does not relicense them.

## Engine contract

Supported operations are `Compose`, `Validate`, `BuildPlan`, `Run`, `Inspect`,
`Release`, and `Resume`. Composition uses `Module`, `Branch`, `Merge`, `Scatter`,
`Gather`, and `When`. Artifacts use `PathSpec` and File, Group, or Tree binds.
Failures use `Error`, `Defect`, and stable `DefectCode` values.

`WriteTo` is the supported plan-option constructor. Graph readers `Name`,
`TaskIDs`, `InputNames`, and `Edges`, plus Plan JSON, support inspection.
`LoadSampleSheetFile(path)` is the explicit-path concurrent core samplesheet
API. Other root-package exports remain provisional unless their own contract
says otherwise.

The caller supplies trusted pipeline code and an existing, exclusive workspace.
Gobble copies staged inputs into task isolates and publishes outputs by copying.
It does not hardlink or symlink staged or published data. Commands, parameters,
scripts, stdout, and stderr may persist caller content. Do not place secrets in
them. Inspect omits task environment values.

## Recovery

Occupancy remains active after success, failure, or cancellation. Recover with
**Inspect → Release → Resume**. Release reconciles known backend leftovers and
closes occupancy; it does not delete controls or artifacts. An unproved Docker
disposition remains `unknown-backend`, keeps occupancy active, and blocks
Resume. See [Operations](docs/operations.md#recovery) before acting on a failed
or interrupted workspace.

## Checks

The hermetic first check performs no fixture download:

```sh
GOTOOLCHAIN=local GOPROXY=off go test -count=1 ./...
```

Live suites have separate prerequisites and can fetch pinned public fixtures.
The installed engine path is exercised by `go test -tags=live
./tests/install-e2e`. Product-specific evidence and its limits are described in
[Provenance](docs/provenance.md#evidence).

## Exclusions

There is no supported Slurm, cloud, Kubernetes, service, remote-execution,
object-storage, or Podman backend. There is no GUI, Gobble DSL, YAML/JSON product
parameter surface, component catalog, public Cancel/Retry/Diff/Repair/Clean
verb, automatic retry or fallback, cross-workspace cache, hidden cleanup, or
integrated cross-assay graph. No support is claimed for musl, Windows, macOS,
or `linux/arm64`.
