# Gobble — System

## Composition

Gobble supplies one shared engine for five assay products. Its dependency
direction is acyclic:

```text
gobble Go API and engine
        ↑
assets/modules/<command>
        ↑
assets/pipelines/<assay>
        ↑
tests/modules | tests/pipelines | tests/scenarios
```

Package `gobble` and `cmd/gobble` do not import product packages. The generic
command selects a non-`internal` package, compiles a child, and calls its
`Pipeline()` adapter. A packed runner embeds one selected package.

## Parts and responsibilities

| Part | Responsibility |
|---|---|
| Public Go model | Own `Pipeline`, immutable `Graph`, tasks, modules, operators, PathSpec, and File, Group, or Tree binds. |
| Product command modules | Own one executable command or subcommand, typed options and ports, one immutable default image, resources, and argv ownership. |
| Assay pipelines | Own strict assay data, typed configuration, selected stage order, graph generation, required outputs, and scientific limits. |
| Validator and planner | Reject graph and bind defects and emit inspectable Plan JSON and a DAG before execution. |
| Scheduler | Admit ready identities by DAG, CPU, memory, and run-level cap. Runtime Scatter/Gather membership uses reserved identity. |
| Executors | Run local processes or exact Docker images through Submit, Poll, Cancel, and Reconcile. |
| Workspace state | Persist schema-2 run and task state, occupancy, logs, artifacts, lineage, fingerprints, and reuse decisions. |
| Evidence owners | Keep command evidence under `tests/modules`, assay and fixture authority under `tests/pipelines`, and lifecycle behavior under `tests/scenarios`. |

## Product construction

Every assay package exposes typed samples, `Parse`, `Load`, `Config`,
`DefaultConfig`, `Build`, `Pipeline`, and `Lifecycle`. `Build` copies mutable
caller values and reads no process global, current directory, filesystem state,
environment, or network. Only the default process-exclusive `Pipeline()`
adapter reads the injected sheet path.

Assay sheets are strict and product-owned. There is no universal multiomics
sheet. Sample, lane, run, replicate, and protocol expansion happens at compose
time. Runtime Scatter/Gather is used where declared artifact membership is the
true work unit, notably WGS intervals.

## Data and flow

Trusted Go source and typed config define a graph. The operator stages regular
input files, complete Groups, and ready Trees in an existing exclusive
workspace. A ready Tree requires a regular root `.gobble-tree.json`; directory
presence alone is incomplete.

Run copies staged inputs into task isolates and copies declared outputs to their
destinations. It does not hardlink or symlink staged or published data. Product
construction and selected tasks do not fetch reads, references, annotations,
whitelists, or fixture bytes. Test-only live preparation may fetch exact
commit-bound bytes into an ignored assay-owned cache, verify size and SHA-256,
and copy them into a workspace.

Resume reuses succeeded work only when task identity, command or script,
parameters, environment digest, runtime software identity, staged-input
fingerprints, and published-destination checksums still match. Missing proof is
a reuse miss. Cross-workspace result caching is not part of the product.

## Interfaces

The public lifecycle verbs are `Compose`, `Validate`, `BuildPlan`, `Run`,
`Inspect`, `Release`, and `Resume`. Composition uses `Module`, `Branch`,
`Merge`, `Scatter`, `Gather`, and `When`. Failures use structured `Error`,
`Defect`, and `DefectCode` values. CLI success is JSON or JSONL; failure stdout
is empty; exits are 0, 1, or 2.

`Run` and `Resume` require one effective execution identity. `inspect identity`
remains readable on mismatch; other reads and mutations fail closed. The
workspace schema remains 2. Older schemas have no migration path.

## Execution and recovery

The supported runtime boundary is trusted-local `linux/amd64`. Empty task image
selects the process executor; an exact non-empty image selects Docker. Docker
uses the caller UID/GID and `--network=none` for the task container, but this is
not a sandbox. Image inspection and acquisition occur through the local Docker
client before task command launch and may require registry network access.

Occupancy belongs to the process that called Run or Resume and remains active
after success, failure, or cancellation. Recovery is Inspect, actor-gated
Release, then Resume. Release reconciles backend state and closes occupancy only
when every disposition is proved. An unresolved Docker identity becomes
`unknown-backend`, leaves occupancy active, and blocks Resume. Gobble does not
signal or adopt an unproved PID. Release does not delete state or artifacts.

## Provenance and support

Each assay's schema-4 manifest is the sole authority for its official fixture
bytes and default-image inventory. Fixture rows record immutable URLs, exact
commits, byte counts, SHA-256 values, provenance, stage use, and license or
redistribution facts. Image rows bind exact tag and digest identities, but they
are not a uniform image license or redistribution authority.

The supported product baseline is local and unreleased. `v0.1.0` is the earlier
engine-only release. Product support is engineering-only and does not include
scientific validity, nf-core endorsement, realistic cohort scale, remote
backends, or arbitrary replacement software.

## Stack and checks

| Component | Current boundary |
|---|---|
| Go | Module `github.com/HahyeonJeon/gobble`, Go 1.26 or newer |
| Platform | `linux/amd64` |
| Runtime | Local process and Docker |
| State and artifacts | Caller-owned local files |
| Hermetic first check | `GOTOOLCHAIN=local GOPROXY=off go test -count=1 ./...` |

The first check covers source contracts and all seven lifecycle scenario owners
without fixture downloads. It does not prove live Docker, registry, network,
third-party command execution, scientific outputs, or production scale. Those
claims require separately identified evidence and must remain explicit when not
run.

## Open policy

Retention and guarded deletion remain unset. The caller owns workspace files
and may delete them outside Gobble. There is no public Clean verb or automatic
artifact retirement policy.
