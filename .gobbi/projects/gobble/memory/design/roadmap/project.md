# Gobble — Project Roadmap

Updated: 2026-09-06. This roadmap records direction and acceptance boundaries,
not dates or promised release numbers. Detailed designs are discussed before
implementation. The canonical app design is
[Application and monitoring](../architecture/application.md).

## Direction and its evolution

Keep one backend-independent Go engine, one structured lifecycle, and a bounded
family of five assay products. Extend that foundation into a local application
that people and external agents use to design, execute, monitor, and recover
analyses.

The engine came first, followed by the five-product family. Current distribution
uses one common Docker workflow without beginner and advanced installation
categories. The next direction is a browser application and shared operations,
followed by an optional desktop shell.

Earlier horizons placed service work after HPC. The current direction moves the
**local application service** ahead of remote execution because it directly
improves local observability and recovery. HPC and cloud remain separately
designed later backend capabilities.

## Current position

The reviewed implementation baseline is
[`39584ce`](https://github.com/HahyeonJeon/gobble/tree/39584ce8785aa66c14788c7d484e6ef088b58c2a).
It is a development baseline, not a stable v0.2.0 release.

| Area | Established baseline | Remaining boundary |
|---|---|---|
| Engine | Go authoring, structured planning, scheduling, artifact identity, inspection, Stop, and Resume | Application run/revision and operation contracts |
| Assays | WGS, RNA-seq, Methyl-seq, ATAC-seq, and scRNA-seq typed graphs | Preserve assay-specific scope and evidence; no integrated multiomics claim |
| Distribution | Public runtime image, common Compose entry, generated project pins | Stable release and real Desktop acceptance |
| Operations | Detached controllers, duplicate-owner refusal, Stop, interruption recovery | Application-level discovery and request deduplication |
| Monitoring | Sample-aware TUI, coherent JSON state, persistent logs | Web interface, multiple-run navigation, events, measured resources |
| Installed analysis evidence | Linux Docker WGS/RNA-seq outputs and unchanged Resume reuse | Other full installed assays and platform-specific evidence |

Native macOS launcher tests cover Intel and Apple Silicon compatibility
launchers, not Docker Desktop execution. Current analysis targets linux/amd64;
Apple Silicon relies on emulation. The immutable v0.1.0 tag is the earlier
engine preview and does not carry this development baseline.

## Delivery order

| Phase | Outcome | Dependency | Exit evidence |
|---|---|---|---|
| 0. Distribution acceptance | Repeatable local installation and recovery | Existing runtime | Actual Linux, Windows Desktop, and macOS Desktop install/recovery matrix with limits recorded |
| 1. Application contracts | Shared meaning for projects, runs, revisions, requests, and recovery | Current lifecycle and app direction | Accepted normal, interrupted, duplicate, stale, and incompatible operation examples |
| 2. Read-only web application | Find runs and understand progress, issues, and results | Phase 1 read contract | Browser agrees with Inspect; viewer/service restart leaves execution intact |
| 3. Shared control and MCP | UI, CLI, and agents control the same durable runs | Phase 1 write contract and Phase 2 viewer | Cross-client start/stop/resume and crash/retry journeys without duplicate execution |
| 4. Operational monitoring | History, measured resources, comparison, notifications | Stable run and operation identities | Reconnectable history, bounded collection, explainable failure/reuse views |
| 5. Optional desktop application | Native folders, tray, notifications, updates | Stable shared frontend/API and demonstrated native needs | Actual OS install/update/close/reopen checks preserving pinned runs |

Phase 0 acceptance and Phase 1 design can proceed independently. A browser
prototype may run on verified Linux while Desktop acceptance is open, but must
not be advertised as verified across all three host platforms.

## Phase 0 — Complete distribution evidence

Use the published image and actual fixture-backed pipelines users receive.
Keep the tiny installation check as the quick first result, followed by WGS and
RNA-seq execution. Expand full installed assay coverage according to available
resources; graph tests alone are not analysis proof.

Verify folder sharing, writable outputs, path translation, pinned runtime
restoration, detached execution, Stop, controller death, Docker restart, and
Resume. Cover Windows and macOS Intel/Apple Silicon separately, including
spaces and non-ASCII project names. Record OS, architecture, Docker version,
image digest, resources, outputs, and recovery outcome.

A stable release requires named compatibility effects and a support matrix.
Development-image publication is separate from that release decision.

## Phase 1 — Agree on application contracts

Define project registration, run/workspace identity, source revision capture,
task attempts, and command acceptance versus completion. Decide changed-source
Resume semantics without erasing previously executed definitions.

Specify the local service boundary, pinned-runtime routing, protocol versions,
request identity, and action eligibility. Separate connection freshness,
controller liveness, backend uncertainty, and execution outcome.

Review concrete examples before implementation: successful start, lost start
response after acceptance, repeated Stop, delayed Stop after Resume, service
restart, Docker unavailability, and application/runtime mismatch.

## Phase 2 — Deliver the first useful web monitor

Register explicitly shared project roots and discover runs. Show run lists,
overviews, sample progress, selected DAG context, task logs, errors, and results.
Reuse engine projections and aggregation rules.

Begin with bounded polling, observation times, and stale-state presentation.
Read selected logs by attempt and bounded range. Keep large graphs and task
lists searchable and bounded. Result access stays within registered project
authority.

Exit requires an actual pipeline visible through Inspect/TUI and the browser,
consistent sample/shared counts, missing-log behavior, reconnection, and proof
that viewer restart does not stop the controller. Measure and agree initial
performance budgets using representative runs instead of inventing scale or
ETA claims.

## Phase 3 — Add shared control and agent tools

Introduce start, stop, resume, and resume-preview operations behind the agreed
contract. MCP and application CLI adapters expose the same operations. Long
analyses return durable identifiers; request cancellation or an agent session
ending must not implicitly cancel an accepted pipeline.

Exercise control across different clients. Repeated requests, concurrent
clients, lost responses, crashes, and delayed commands must not launch duplicate
schedulers or control a newer owner. Resume reconciles and verifies reusable
outputs before admitting new work.

The first agent journey is: edit source, validate, inspect the plan, start,
reconnect by run ID, inspect a failure, preview recovery, and resume. An embedded
chat product or a new model runtime is unnecessary for this phase.

## Phase 4 — Make monitoring explanatory

Add ordered history with a defined checkpoint consistency rule, followed by
streaming with cursor recovery and snapshot fallback. Add measured resource
use, attempt timelines, provenance comparison, report export, and run-specific
notifications. Separate recorded evidence from diagnosis and estimates.

Verify event gaps, duplicate delivery, log rotation/truncation, collector
failure, and bounded overhead. Notifications deduplicate and open the relevant
run. Add ETA only after comparable execution evidence can validate usefulness.

## Phase 5 — Evaluate the desktop shell

Reuse the frontend and API. Evaluate Wails against native folder access,
packaging, signing, updates, tray behavior, and notifications on supported OSes.
Keep browser operation viable. The window process must not own long analyses.

Application updates preserve runtimes needed by existing runs. Opening the app
does not imply Docker or every analysis image is available or native to the
host CPU.

## Later horizons

- **Finish active tasks and wait:** Separately designed admission control,
  distinct from Stop and process suspension.
- **HPC:** A first adapter such as Slurm, including queue mapping, shared and
  node-local storage, and backend reconciliation; keep the core independent.
- **Cloud and remote operation:** Batch or Kubernetes, object storage,
  authentication, transfer, and remote workspace ownership.
- **Native ARM analysis:** Per-image and per-pipeline evidence with explicit
  emulation/fallback policy.
- **Ecosystem:** Discovery, richer plan comparison, and assisted diagnosis
  without replacing Go authoring or inventing cross-assay semantics.

Retention/deletion, cross-workspace caching, integrated multiomics, extra
assays, and bidirectional visual authoring remain separate decisions. Existing
durable deferrals stay in [Backlog Memory](../../backlogs/README.md). This
roadmap does not create implementation tickets for every possible feature.

## Next scope and replan rules

The next slice is **Phase 1 followed by the read-only web monitor in Phase 2**,
with outstanding platform acceptance tracked in parallel. Review domain/API
examples and screen structure before implementing application mutations or
choosing a desktop framework.

Replan if a service duplicates execution authority, changes old run identity,
treats uncertainty as success/failure, or hides incompatible runtime selection.
Return to user discussion for new execution semantics, persistence-schema
changes, automatic rerun, remote access, retention, and desktop framework choice.

Maintain product contracts, graph generations, artifact identity, required
outputs, image pins, fixture provenance, and recovery evidence throughout.
A user interface does not expand an assay's scientific claims.
