# Gobble — System

## Composition

Gobble supplies one shared engine for five assay products. Its dependency
direction is acyclic:

| Layer | Depends on |
|---|---|
| Command modules | Public Gobble model |
| Assay pipelines | Command modules and public model |
| Tests | The module, pipeline, or lifecycle boundary under test |

The intended application layer is defined in
[Application and monitoring](application.md). Electron and React/TypeScript
provide the first UI; a host-side Codex adapter supplies the initial ChatGPT
subscription agent experience. The independent Go service and analysis
controllers retain engine authority. Provider turns and pipeline runs have
separate identities and cancellation rules. Browser delivery is a later option.

The app is a Project-centered shared workspace with multiple attached agents:
its controller manages Project-owned views,
observations, and contextual user decisions through separate workspace tools.
These UI concepts do not enter the engine's execution state. Gobble remains
the core engine and a standalone Linux CLI. Neither the Go library nor CLI
requires Electron, Node.js, provider authentication, or the application service.
The CLI and service use shared Go lifecycle behavior and the same workspace
identity and owner gates. See [Agent-centered workspace](../feature/agent-workspace.md).

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
`Inspect`, `Stop`, `Release`, and `Resume`. Composition uses `Module`, `Branch`,
`Merge`, `Scatter`, `Gather`, and `When`. Failures use structured `Error`,
`Defect`, and `DefectCode` values. Structured CLI success is JSON or JSONL;
failure stdout is empty; exits are 0, 1, or 2. Interactive Watch is a separate
read-only TUI. The monitor view projects coherent state and selected log tails
without occupying the run.

`Run` and `Resume` require one effective execution identity. `inspect identity`
remains readable on mismatch; other reads and mutations fail closed. The
workspace schema remains 2. Older schemas have no migration path.

## Execution and recovery

The current analysis runtime boundary is trusted-local `linux/amd64`. The
common Compose image contains the compiler and Gobble. Project pins preserve
runtime identity, and analysis containers are siblings on the host daemon.
Desktop host acceptance and CPU emulation are separate validation concerns.

An empty task image selects the process executor; an exact non-empty image
selects Docker. Docker uses the caller UID/GID and `--network=none` for the task
container, but this is
not a sandbox. Image inspection and acquisition occur through the local Docker
client before task command launch and may require registry network access.

Occupancy belongs to one Run/Resume controller. Detached Compose execution
keeps it independent of the initiating client. Stop writes a durable request
addressed to the current owner lease, and distinguishes request acceptance
from proved settlement. Resume reconciles and acquires its next lease under
one continuous run lock; a separate Release is unnecessary for routine
recovery. Release remains the lower-level actor-gated reconciliation operation.

Unresolved Docker state remains `unknown-backend` and blocks new work. Gobble
does not signal or adopt unproved PIDs. Recovery preserves state and artifacts.
Client disconnection is not execution cancellation; Docker/computer restart
does not automatically resume a pipeline.

## Provenance and support

Each assay's schema-4 manifest is the sole authority for its official fixture
bytes and default-image inventory. Fixture rows record immutable URLs, exact
commits, byte counts, SHA-256 values, provenance, stage use, and license or
redistribution facts. Image rows bind exact tag and digest identities, but they
are not a uniform image license or redistribution authority.

The reviewed product baseline is available as a public development runtime,
with project-level digest pinning. No stable v0.2.0 release is implied;
`v0.1.0` is the earlier engine-only release. Product support is engineering-only
and does not include scientific validity, nf-core endorsement, realistic cohort
scale, remote backends, or arbitrary replacement software.

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
