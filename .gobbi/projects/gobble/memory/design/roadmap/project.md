# Gobble — Project Roadmap

Updated: 2026-09-06. Electron, React/TypeScript, agent-centered interaction,
ChatGPT subscription sign-in as the first agent target, and standalone Linux
CLI support are selected directions. Thin desktop-first delivery remains the
recommendation after comparing it with a local web application.
This roadmap defines sequence and acceptance, not dates or release promises.
The canonical design is [Application and monitoring](../architecture/application.md).

## Direction and its evolution

Keep one backend-independent Go engine, a structured lifecycle, and five
bounded assay products. The first application goal is a local workspace where
people work with an agent to design, execute, control, and monitor bioinformatics
pipelines. Gobble is its core engine and remains independently usable as a
Linux CLI tool.

Agent-centered UX is a first-product requirement: the agent opens relevant
analysis views, uses them to inspect evidence, asks contextual questions, and
continues from the user's answers. Review this interaction and the layout with
the user before implementing screens. The canonical UX proposal lives in
[Agent-centered workspace](../feature/agent-workspace.md).

The engine and assay family are already implemented. Current distribution uses
one common Docker workflow without beginner/advanced categories. The desktop
app will manage that execution foundation through native setup and project
flows. Compose, the Go API, and the standalone Linux CLI remain maintained
interfaces for operators, scripts, and external agents.

The earlier browser-first plan and optional Wails shell are superseded.
**Electron with React and TypeScript is the first application**, with an
integrated OpenAI agent accessed through ChatGPT subscription authentication.
Standalone browser delivery moves to a later horizon. A browser may still be
used for official sign-in; that is not the Gobble application interface.

A local application service remains ahead of HPC and cloud. Its purpose is to
support durable analysis independent of the desktop window and agent session.
The agent runtime and pipeline scheduler remain separate authorities.

### Why keep desktop first

The [delivery comparison](../architecture/application.md#delivery-recommendation-thin-desktop-first)
recommends a small Electron host because local agent provisioning, native
project access, app windows, and reopen/reconnect behavior belong to the initial
user journey. Local web delivery can support the same internal agent-controlled
panels, but still needs a local process for engine and agent access.

Use portable React views and typed host adapters. A browser development harness
can help explore and test layouts; shipping a browser product is not a phase
before Electron. Keep the initial desktop scope small: prove shared agent/UI
interaction and an actual pipeline before adding advanced window management,
extensive dashboards, or automatic updates.

## Current position

The reviewed executable baseline remains
[`39584ce`](https://github.com/HahyeonJeon/gobble/tree/39584ce8785aa66c14788c7d484e6ef088b58c2a).
Subsequent design-memory edits do not implement the app or authentication.
This development baseline is not a stable v0.2.0 release.

| Area | Existing evidence | Work required for the desktop product |
|---|---|---|
| Engine | Go authoring, plans, scheduling, artifacts, Inspect, Stop, and Resume | Application run/revision and operation contracts |
| Linux CLI | Independent Go command, structured output, and Watch | Preserve headless installation/use and verify compatible cross-interface control |
| Assays | WGS, RNA-seq, Methyl-seq, ATAC-seq, and scRNA-seq typed graphs | Preserve assay contracts and evidence |
| Distribution | Public runtime, common Compose entry, generated project pins | Desktop dependency setup, installers, and host-specific acceptance |
| Operations | Detached controllers, duplicate-owner refusal, interruption recovery | Application discovery and request deduplication |
| Monitoring | Sample-aware TUI, JSON state, persistent logs | React desktop UI, run navigation, events, measured resources |
| Agents | External coding agents using the documented CLI | In-app subscription integration, workspace interaction tools, contextual decisions, and observations; not yet tested in Gobble |
| Analysis evidence | Installed Linux Docker WGS/RNA-seq outputs and Resume reuse | Desktop-driven analysis and remaining platform/assay coverage |

Native macOS launcher tests cover compatibility launchers, not the Electron app
or Docker Desktop execution. Current analysis images target linux/amd64, using
emulation on Apple Silicon. Electron and Codex host architecture support require
their own evidence. The immutable v0.1.0 tag is the earlier engine preview.

## Delivery order

| Phase | Outcome | Dependency | Exit evidence |
|---|---|---|---|
| 0. Execution baseline | Repeatable local container installation and recovery | Current runtime | Actual host/Docker matrix with limits recorded |
| 1. Shared workspace and agent integration proof | Reviewed UX, thin Electron/React/TypeScript app, setup, sign-in, and agent-controlled analysis surfaces | User review of layout/attention rules and focused application contracts | Agent opens a real surface, consumes its observation, asks a question, and receives the user's answer; facts agree with Gobble |
| 2. First complete desktop workflow | Agent-assisted design, plan review, Run, monitoring, Stop, and Resume | Phase 1 plus accepted write-operation contracts | Actual fixture-backed analysis from the app, including interruption and recovery |
| 3. Distribution and reliability | Qualified native installation, updates, and reconnection | Phase 2 workflow | OS-specific installer/update/recovery evidence without changing active analysis pins |
| 4. Explanatory monitoring | Events, measured resources, attempt timelines, comparison, reports | Stable run and operation identities | Reconnectable history and bounded collection overhead |
| Later. Additional delivery and backends | Browser client, further providers, remote execution | Separate accepted designs | Capability-specific evidence; not prerequisites for the first desktop app |

Phase 0 acceptance can proceed alongside desktop development. Native packaging
begins in Phase 1 so sign-in, paths, and process behavior are tested early;
Phase 3 qualifies distribution rather than introducing Electron for the first
time. Phase 2 is the first complete application outcome. Full event history,
detached native windows, and a separately shipped web app do not block it.

Linux CLI independence is a continuing requirement across these phases. Keep
headless lifecycle tests and documented commands working without the desktop,
Node.js, provider login, or an app service. Application features must not add
those dependencies to the Go engine or CLI.

## Phase 0 — Complete execution evidence

Use the published runtime and actual fixture-backed pipelines users receive.
Keep the tiny installation check, then WGS/RNA-seq execution. Expand full
installed assay coverage according to resources; graph tests are not analysis
execution evidence.

Verify shared folders, writable outputs, path translation, image restoration,
detached execution, Stop, controller death, Docker restart, and Resume. Cover
Windows and macOS Intel/Apple Silicon explicitly, including spaces and non-ASCII
project names. Record OS, architecture, Docker version, runtime digest,
resources, outputs, and recovery outcomes.

A development image is not a stable release. Stable claims require named
compatibility effects and a truthful support matrix.

## Phase 1 — Review the workspace UX and prove agent interaction

Before implementation, show the user low-fidelity workspace layouts and a
concrete design-to-recovery interaction. Compare the recommended shared work
area with a conversation-led alternative. Decide the starting layout, how the
agent opens views without disrupting user attention, how questions attach to
evidence, and which surface types are initially needed. Keep a separate decision
for detachable OS windows. Implement screens after this discussion.

Review a compact domain/API contract. Define registered roots, run/workspace
identity, source revision capture, task attempts, request acceptance, and outcome.
Separate UI connection,
agent turn, surface readiness, user decision, controller liveness, backend
observability, and pipeline status. Keep UI state out of engine workspaces.

Build the Electron main process, narrow preload interface, and React/TypeScript
renderer. Provide setup checks, native folder selection, project registration,
persistent conversation, and the minimum shared surfaces needed for the proof,
such as a real plan/run view and its selected task/log context. Use bounded
reads from existing engine projections; maintain identity and containment.
Keep React views independent of host APIs and use a browser development harness
where useful. A basic installer or native packaged build is part of this phase.

Implement a host adapter for the official Codex App Server and prove ChatGPT
subscription sign-in, a model selection, an agent turn, and reconnectable
conversation state. Add workspace tools to open/reuse a surface, observe its
actual state, request contextual input, and receive the response. Demonstrate
agent consumption of structured observations and, for a visual inspection case,
a scoped rendered capture. Do not assume App Server supplies Gobble-specific
UI tools. The exact runtime provisioning and tool bridge are resolved here,
with pinned compatibility and native host tests. The provider behavior is
specified in [the agent design](../architecture/application.md#first-agent-target-chatgpt-subscription-sign-in).

Review packaging versus managed acquisition of the official Codex binary,
notices, credential handling, native availability, and upgrade behavior before
committing the first distributable. App Server is the chosen integration route;
a general-purpose model proxy is not a required architecture component.

Exit requires desktop facts matching Inspect/TUI, graceful missing-log/stale
state handling, and a complete open/observe/ask/answer interaction. Test repeated
surface requests, delayed or failed rendering, pinned/dismissed views, and
answers after relevant source changes. Prove that closing a surface, the app,
or the agent does not stop an existing detached analysis. Record account, model,
and runtime compatibility without retaining credential material in test artifacts.

## Phase 2 — Deliver the first complete agent-driven analysis

Extend the shared workspace through source changes, sample inputs, plan review,
run progress, task/log inspection, and results. The agent uses these views to
discuss and verify analysis; the user can navigate and act directly. Source files
remain versionable Go and typed configuration. The displayed DAG is derived
from Gobble's validated plan. Scope reports to known artifact viewers with an
isolated path for HTML; arbitrary executable agent-generated UI is later design.

Expose shared start, stop, resume, and resume-preview operations to direct UI
controls and the Gobble agent tools adapter. Resolve its MCP transport and
registration during implementation; it calls the same Go service operations.
Long analysis returns durable identifiers, and a lost response must not cause
an automatic duplicate start.

The first end-to-end desktop journey is: install/open, check dependencies, sign
in, choose a project, ask the agent to prepare a real test pipeline, inspect the
views it opens, answer a contextual question, review the plan, run, inspect
progress/results together, stop when needed, preview recovery, and resume.
Use the existing tiny check and actual assay fixtures as execution evidence.

| Scenario | Required evidence |
|---|---|
| Agent presents a plan or asks about selected input | View, question, answer, and affected operation remain linked to the correct evidence revision |
| Agent opens logs or a QC report to check a result | Actual scoped observation is consumed; interpretation is distinguishable from engine facts and scientific validity |
| View is closed, rendering fails, or question is dismissed | Explicit UI outcome; no implied answer and no pipeline cancellation |
| Agent request and direct UI action overlap | One authoritative operation and one controller owner |
| Start response is lost or a tool call is retried | Existing operation can be recovered without duplicate execution |
| User interrupts the agent turn | Accepted analysis continues; pipeline Stop remains a separate action |
| Login expires, user signs out, or provider usage is limited | Existing runs remain visible and controllable locally |
| Agent proposes changed pipeline source | Executed revision is preserved; reuse/re-execution impact is visible |
| Stop is repeated or arrives after a new Resume | Correct owner lease is addressed; the newer run owner is protected |
| Controller or Docker fails | Unknown state is preserved; reconciliation precedes new work |
| Application closes and reopens | The same project, linked run, and recorded conversation association can be recovered |
| CLI and app operate on the same analysis | Compatible Linux CLI can inspect/stop/resume app-started work; app can discover CLI-started work, preserving recorded runtime identity and owner gates |

An account-connected chat pane alone does not satisfy this phase. The agent
must author/review a pipeline, use the shared views with the person, and invoke
Gobble operations successfully. Keep direct controls usable independently of
agent turns, and verify standalone CLI use with the app and service absent.
No implicit switch from subscription access to separately billed API calls is
part of the first workflow.

## Phase 3 — Qualify distribution and reliability

Qualify Electron installers, signing/notarization where applicable, dependency
setup, callback handling, credential-store behavior, folder access, tray status,
notifications, and update/rollback behavior on supported host platforms. Choose
an explicit Linux update strategy; cross-platform packaging does not imply one
identical updater on every OS.

Keep app, Codex runtime, and analysis-runtime versions independently compatible.
An update must preserve active run controllers and their pinned images. Test
native app failure, service failure, provider-process failure, Docker restart,
and machine restart separately. Automatic analysis rerun remains a separate
policy decision.

Measure startup, idle memory, large-list/graph rendering, and log throughput on
representative machines. Set budgets from measurements and product needs.
Qualify additional installed assays and architectures before expanding claims.
Retain Linux CLI installation, command documentation, and headless recovery
evidence alongside desktop release acceptance. Native distribution work must
not make the engine or CLI require the application catalog or a provider account.

## Phase 4 — Make monitoring explanatory

Add ordered event history with a checkpoint consistency rule, then streaming
with cursor recovery and snapshot fallback. Add measured resource use, attempt
timelines, provenance comparison, report export, and run-specific notifications.
Keep execution events and provider conversation/tool events distinguishable.

Verify gaps, duplicate delivery, log rotation/truncation, collector failure,
and bounded overhead. Notifications open the relevant run. Add ETA only when
comparable execution evidence supports it. Keep recorded errors, agent diagnosis,
and scientific QC interpretation distinct.

## Later horizons

- **Browser application:** Reuse React components and service contracts after
  the desktop workflow is useful; not a prerequisite for desktop delivery.
- **Additional agent access:** Explicitly selected API-key billing and other
  providers, while preserving the initial ChatGPT subscription experience.
- **Finish active tasks and wait:** Separate admission control, distinct from
  pipeline Stop and agent-turn interruption.
- **HPC:** An adapter such as Slurm, including queue/storage mapping and backend
  reconciliation, without making the core scheduler backend-specific.
- **Cloud and remote operation:** Authentication, data transfer, remote workspace
  authority, object storage, and batch/Kubernetes execution.
- **Native ARM analysis:** Per-image and per-pipeline qualification, independent
  of native Electron or Codex availability.
- **Ecosystem:** Discovery, plan comparison, and assisted diagnosis while
  retaining Go authoring and assay-specific semantics.

Retention/deletion, cross-workspace caching, integrated multiomics, extra assays,
and bidirectional visual authoring remain separate decisions. Existing durable
deferrals stay in [Backlog Memory](../../backlogs/README.md).

## Next scope and replan rules

The next slice is **a user-reviewed agent workspace layout and interaction
storyboard, then Phase 1's thin Electron app with official subscription login
and open/observe/ask/answer proof**. Phase 2 delivers the complete local analysis
journey. Track outstanding execution acceptance and Linux CLI compatibility
alongside that work. Electron and React/TypeScript are selected; precise UX
behavior remains a design discussion before screen implementation.

Replan if the app duplicates execution authority, silently changes runtime
identity or billing mode, couples provider availability to pipeline control,
or treats backend uncertainty as a terminal outcome. Also replan if the agent
cannot consume the evidence its tools open, repeatedly disrupts user attention,
or app development makes Linux CLI use dependent on the desktop. Return to
discussion for new execution semantics, persistence migration, automatic rerun, remote access,
and retention policy. Report unsupported provider capabilities explicitly
rather than promising every ChatGPT model or unlimited subscription usage.

Maintain graph generations, artifact identity, required outputs, image pins,
fixture provenance, and recovery evidence throughout. A desktop UI or agent
does not expand an assay's scientific claims.
