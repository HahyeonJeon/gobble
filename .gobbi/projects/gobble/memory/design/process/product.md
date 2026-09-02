# Gobble — Product Lifecycle and Support

## Product family

Gobble supports five local assay products: WGS joint germline, bulk RNA-seq,
Methyl-seq, ATAC-seq, and scRNA-seq. Their exact packages, graph generations,
benchmarks, and selected paths are owned by
[Assay product family and ownership](../feature/assets.md). Gobble is their
shared engine, not another assay.

## First useful outcome

An author selects one assay package, loads its strict sheet, changes a fresh
typed config, builds and reviews the graph, and runs it through the Go API or
generic command. A human may receive one packed runner for the same selected
package. No actor must learn a proprietary language or rebuild the supported
assay graph from individual tasks.

## Lifecycle

| Outcome | Product promise |
|---|---|
| Design | The product's typed values, selected stages, outputs, owner, and exclusions are discoverable. |
| Build | Explicit data and config produce a composed, validated, inspectable plan; invalid input fails before execution. |
| Customize | Named typed fields or safe argv extras have visible graph and command effects without mutating defaults. |
| Run | Required artifacts and reports are complete only after strict dependencies and fan-in succeed. |
| Resume | Matching work may be reused; changed and downstream work reruns under the compatible graph generation. |
| Stop | Context cancellation is structured and leaves state inspectable and occupancy active. |
| Failure | The failed unit, logs, reusable successes, and blocked descendants remain distinguishable without route fallback. |

Stop and failure are distinct outcomes. Both use the shared Inspect, Release,
and Resume recovery model. Assay packages add no recovery verbs.

## Audience and interfaces

Coding agents and Go authors use the library and generic `gobble` command.
Humans may use a packed `linux/amd64` runner. Both expose the same seven verbs.
Graph verbs accept `--sample PATH`; Inspect and Release do not. Success output
is JSON or JSONL. There is no TUI or GUI.

## Support unit

A supported product is the exact tuple of package path, graph generation, typed
data and config, selected stages, task and artifact identities, required
outputs, default image digests, benchmark and fixture authority, and all seven
lifecycle outcomes.

Support is engineering-only on trusted-local `linux/amd64` Docker execution.
Pipeline source, config, OS user, and workspace are trusted. Docker is not a
sandbox. Gobble adds no account, service, upload, telemetry, or secret store.
The caller owns local permissions, retention, and deletion.

## Failure and recovery

Run and Resume retain occupancy after return. The operator inspects identity,
run state, errors, logs, instances, remaining work, reuse, and lineage. Release
is allowed only through the owner-process or later-process actor gate and
reconciles backend state before closing occupancy.

An unresolved Docker identity is `unknown-backend`, not a failed task or a
released workspace. It keeps occupancy active and blocks Resume. The operator
restores the same Docker client's backend observability and retries actor-gated
Release. Gobble never signals or adopts an unproved PID. Release does not delete
controls or artifacts.

## Release and compatibility

The immutable `v0.1.0` tag is the earlier engine preview. It does not contain
the five product packages. The product family is local and unreleased. A future
release that carries it must name all current graph generations and their Go
API, CLI, workspace, image, and recovery effects.

Release tags are immutable and supported instructions never use `@latest`. A
pre-1.0 patch intends no break to the Go API, CLI protocol, workspace schema,
product graph generations, or recovery behavior. A minor release may declare a
break and must name its effects.

The WGS, RNA-seq, and Methyl-seq lifts are named graph breaks. Their temporary
top-level constructor shims preserve source names only. Old proof workspaces
require new workspaces. ATAC-seq and scRNA-seq begin with their current first
generations.

## Refused and unsupported uses

The product family does not support scientific or clinical conclusions,
nf-core endorsement, untrusted or multi-user execution, integrated cross-assay
analysis, serialized product configuration, route fallback, partial required
fan-in, public Cancel/Retry/Diff/Repair/Clean, remote execution, HPC, cloud,
services, or automatic artifact deletion.

## Maintenance

Review a product when its benchmark changes a stage, command, image, sheet,
output, or fixture; when a pin becomes unavailable or receives a material
security or license advisory; or when a task ID, port, destination, data meaning,
required output, or lifecycle outcome changes. Update product source, its sole
manifest, focused evidence, lifecycle evidence, current design, and release
notes as one compatibility unit.
