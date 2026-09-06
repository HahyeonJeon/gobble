# Gobble — Product Lifecycle and Support

## Product family

Gobble supports five local assay products: WGS joint germline, bulk RNA-seq,
Methyl-seq, ATAC-seq, and scRNA-seq. Their exact packages, graph generations,
benchmarks, and selected paths are owned by
[Assay product family and ownership](../feature/assets.md). Gobble is their
shared engine, not another assay.

## First useful outcome

An author or external agent selects an assay, loads its strict sheet, changes
a fresh typed config, builds and reviews the graph, and runs it through the
shared engine. The common Compose runtime supplies Go and Gobble without a
host installation. Packed runners and direct Linux development remain
compatible entry points. No actor must learn a proprietary language or rebuild
the supported assay graph from individual tasks.

## Lifecycle

| Outcome | Product promise |
|---|---|
| Design | The product's typed values, selected stages, outputs, owner, and exclusions are discoverable. |
| Build | Explicit data and config produce a composed, validated, inspectable plan; invalid input fails before execution. |
| Customize | Named typed fields or safe argv extras have visible graph and command effects without mutating defaults. |
| Run | Required artifacts and reports are complete only after strict dependencies and fan-in succeed. |
| Resume | Matching work may be reused; changed and downstream work reruns under the compatible graph generation. |
| Stop | A durable lease-addressed request distinguishes acceptance from confirmed settlement and preserves inspectable state. |
| Failure | The failed unit, logs, reusable successes, and blocked descendants remain distinguishable without route fallback. |

Stop and failure are distinct outcomes. Inspect establishes the facts, Stop
settles active work, and Resume reconciles before admitting more work. Release
remains available for lower-level reconciliation but is unnecessary for routine
recovery. Assay packages add no recovery verbs.

## Audience and interfaces

Coding agents and people use the same Compose command contract. Go authors
can also use the library. Graph verbs accept `--sample PATH`; Inspect, Watch,
Stop, and Release do not use it. Structured commands return JSON or JSONL.
Watch is a sample-aware, read-only TUI.

The next product is an Electron desktop application with React/TypeScript,
an in-app agent, and shared Go application operations. ChatGPT subscription
sign-in through the official Codex integration is the first agent target.
Standalone browser delivery and additional providers come later. See
[Application and monitoring](../architecture/application.md). This is selected
direction, not a claim of shipped desktop or account integration.

## Support unit

A supported product is the exact tuple of package path, graph generation, typed
data and config, selected stages, task and artifact identities, required
outputs, default image digests, benchmark and fixture authority, and all seven
lifecycle outcomes.

Support is engineering-only on trusted-local `linux/amd64` Docker execution.
Pipeline source, config, OS user, and workspace are trusted. Docker is not a
sandbox. The current engine adds no account, service, upload, telemetry, or
secret store. The planned agent feature introduces provider authentication and
transmission of selected conversation context; local analysis and direct run
controls remain independent of that account. The caller owns local permissions,
retention, and deletion. Real Windows and
macOS Docker Desktop acceptance is still outstanding; current linux/amd64
images require emulation on Apple Silicon.

## Failure and recovery

The operator inspects identity, run state, errors, logs, instances, remaining
work, reuse, and lineage. A live scheduler excludes another owner. Stop and
Resume apply the shared settlement and reconciliation rules; the lower-level
Release operation still observes its owner/liveness gate.

An unresolved Docker identity is `unknown-backend`, not a failed task or a
released workspace. It keeps occupancy active and blocks Resume. The operator
restores the recorded Docker daemon's observability and retries Resume or the
lower-level Release operation. Gobble never signals or adopts an unproved PID.
Release does not delete controls or artifacts.

## Release and compatibility

The immutable `v0.1.0` tag is the earlier engine preview. It does not contain
the five product packages. The product family is available in the public
development runtime, with exact project pins, but has no stable v0.2.0 release.
A stable release carrying it must name all current graph generations and their
Go API, CLI, workspace, image, and recovery effects.

Release tags are immutable and supported instructions never use `@latest`. A
pre-1.0 patch intends no break to the Go API, CLI protocol, workspace schema,
product graph generations, or recovery behavior. A minor release may declare a
break and must name its effects.

The WGS, RNA-seq, and Methyl-seq lifts are named graph breaks. Their temporary
top-level constructor shims preserve source names only. Old proof workspaces
require new workspaces. ATAC-seq and scRNA-seq begin with their current first
generations.

## Current unsupported uses

The product family does not support scientific or clinical conclusions,
nf-core endorsement, untrusted or multi-user execution, integrated cross-assay
analysis, serialized product configuration, route fallback, partial required
fan-in, public Cancel/Retry/Diff/Repair/Clean, remote execution, HPC, cloud,
a shipped application service, or automatic artifact deletion. The planned
local service does not expand the assay products into scientific validation
or remote execution.

## Maintenance

Review a product when its benchmark changes a stage, command, image, sheet,
output, or fixture; when a pin becomes unavailable or receives a material
security or license advisory; or when a task ID, port, destination, data meaning,
required output, or lifecycle outcome changes. Update product source, its sole
manifest, focused evidence, lifecycle evidence, current design, and release
notes as one compatibility unit.
