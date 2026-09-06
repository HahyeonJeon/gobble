# Gobble — Agent Application and Monitoring Design

Updated: 2026-09-06. Electron, React/TypeScript, agent-centered interaction,
ChatGPT subscription authentication as the first agent target, and continued
Linux CLI support are user-selected directions. After reconsidering local web
delivery, a thin desktop application remains the recommended first delivery.
Detailed UX choices require discussion; contracts and release qualification
remain implementation work. Existing and planned behavior are separated below.

Related: [Project roadmap](../roadmap/project.md), [Engine architecture](system.md),
[Agent-centered workspace](../feature/agent-workspace.md),
[Inspection](../feature/inspect-run.md), [Recovery](../feature/recover-run.md).

## Vision

The first application goal is a local workspace where a user and an agent
design, execute, control, and monitor bioinformatics pipelines together. The
agent can open analysis views, inspect their evidence, and bring specific
questions to the user. The first agent experience uses OpenAI models through
the user's ChatGPT subscription sign-in and the official Codex App Server.

Gobble is the core Go engine behind this application and remains a standalone
Linux CLI tool. The application makes analysis understandable and manageable;
the Go library supplies authoring, and the engine owns scheduling, execution,
artifacts, and recovery. All five assay products use those shared contracts.
Desktop delivery does not replace the engine or its headless command interface.

Use one installation and execution model across user experience levels.
The integrated agent and direct UI controls operate the same projects and runs.
Agent assistance is a first-product capability; monitoring and controlling an
existing run must still work when the agent is unavailable or signed out.

## Current foundation and intended extension

The implementation baseline reviewed for this design is
[`39584ce`](https://github.com/HahyeonJeon/gobble/tree/39584ce8785aa66c14788c7d484e6ef088b58c2a).

| Concern | Existing foundation | Intended extension |
|---|---|---|
| Distribution | Public runtime containing Go, Git, Gobble, and authoring dependencies; common Compose workflow | Electron desktop distribution managing the same container execution model |
| Execution | Detached controller; tool containers are siblings on the local daemon | Durable run registration independent of client connections |
| Inspection | Coherent JSON snapshots, sample-aware TUI, persistent logs | React/TypeScript desktop monitoring, multiple-run navigation, reconnectable updates, attempt history |
| Recovery | Lease-addressed Stop and reconciliation before Resume | Shared command contract, request deduplication, resume impact preview |
| Agents | External source editing and documented CLI operations | In-app subscription agent that opens/observes analysis surfaces, requests contextual input, and invokes Gobble operations |
| Linux CLI | Standalone Go command, structured Inspect, TUI, and lifecycle operations | Maintained headless interface sharing engine behavior with the app, without a desktop or provider dependency |
| Platforms | Linux/amd64 containers and cross-platform installation guidance | Electron installers and actual application plus Docker acceptance on supported hosts |

Linux Docker CI has exercised installed WGS and RNA-seq runs and Resume reuse.
Actual Windows and macOS Docker Desktop acceptance remains outstanding.
Native macOS launcher tests do not establish Docker Desktop acceptance.
The development image is not a stable v0.2.0 release.

## Selected product and technology direction

The first visual product is an **Electron desktop application with React and
TypeScript**. A standalone browser application is a later delivery option, not
a prerequisite. Reusable frontend components and API contracts should permit
that extension without delaying the desktop product.

| Layer | Selected direction |
|---|---|
| Desktop host | Electron; TypeScript main process and a narrow preload bridge |
| Interface | React and TypeScript; shared analysis workspace with agent-controlled surfaces and direct user interaction |
| First agent target | OpenAI models available through the user's ChatGPT subscription, integrated through Codex App Server |
| Agent transport | Host-managed official Codex runtime, initially using its stdio interface |
| Application operations | Independent Go service with versioned query and command contracts |
| Analysis execution | Pinned Gobble controllers and sibling analysis containers on local Docker |
| Later distribution | Browser access reusing the interface and application operations |

Electron replaces the earlier Wails candidate. Controlling the desktop
rendering environment and using established native integration and distribution
tools matter more here than keeping the desktop host in Go. The engine remains
Go; it is not moved into Electron's main process.

The application should guide project-folder selection, Docker readiness,
runtime acquisition, and agent sign-in. Users need not use a terminal for the
first desktop journey. The standalone Linux CLI, Go API, and Compose route
remain supported interfaces for operators, scripts, and external agents,
without audience tiers or a dependency on the desktop app.

A desktop installer does not eliminate the Linux environment required by
analysis containers. Electron host architecture, Codex runtime architecture,
Gobble controller architecture, and tool-image architecture have separate
support matrices. Current analysis images remain linux/amd64, with emulation on
Apple Silicon. Shipping an ARM desktop binary does not prove native ARM analysis.

### Delivery recommendation: thin desktop first

Agent-controlled panels, contextual questions, and observations can work inside
a local browser application. Agent-centered UX does not itself require native
windows. The recommendation to start with Electron follows this product's
combined local runtime, installation, file-access, and lifecycle requirements.

| Consideration | Electron first | Local web app first |
|---|---|---|
| Shared React workspace | Supports agent-opened tabs/panels and contextual interaction | Supports the same internal workspace model |
| Local runtime and project access | A desktop host can manage native integration and the official agent process | Requires a separately started local backend/launcher for native capabilities |
| Independent windows | Host manages app-owned windows when the UX needs them | New browser windows are subject to user-activation and popup rules; internal panels remain viable |
| Delivery cost | Native packages, signing, updater decisions, and bundled runtime overhead | Less initial native packaging, but local startup, connection, and process lifecycle still need a solution |
| First product evidence | Exercises the intended install/open/work/reopen journey early | Proves browser interaction first, with desktop-specific evidence still to collect |

Electron's main process owns native windows and application integration;
React views live in renderers. Browser popup policies constrain autonomous
creation of new browser windows, and browser directory picking requires user
interaction. A local service can supply native capabilities for a web client,
so these are integration tradeoffs, not a claim that a web product is impossible.
[Electron process model](https://www.electronjs.org/docs/latest/tutorial/process-model),
[browser windows](https://developer.mozilla.org/en-US/docs/Web/API/Window/open),
[directory picking](https://developer.mozilla.org/en-US/docs/Web/API/Window/showDirectoryPicker)

Start with a small Electron host and one usable React workspace. Keep view
components and typed application contracts independent of Electron, with a host
adapter at the boundary. A browser development harness may accelerate layout
work and component tests; it is not a separately released web MVP or an extra
product prerequisite. Qualify basic native packaging early and defer advanced
desktop features until the complete agent-driven journey works.

Reconsider web-first delivery if browser/remote access becomes the first user
need or measured desktop provisioning problems block the target hosts. Neither
condition is currently a reason to postpone the local desktop integration.

## Architecture and ownership

```mermaid
flowchart TD
    UI["React and TypeScript interface"] --> Host["Electron host and preload bridge"]
    Host --> Codex["Official Codex App Server"]
    Host --> Service["Local Go application service"]
    Codex --> Views["Workspace interaction tools"]
    Views --> Host
    Codex --> Tools["Gobble tools adapter"]
    Tools --> Service
    Service --> Controller["Independent pinned run controller"]
    Controller --> Tasks["Analysis tool containers"]
    Controller --> Workspace["State, logs, and artifacts"]
    Service --> Workspace
```

This is the intended application topology, not shipped behavior. The provider
runtime manages agent turns; Gobble manages analysis. An agent thread and a
pipeline run have different identities, state, and lifetimes.

| Component | Responsibility and boundary |
|---|---|
| React renderer | Presents shared conversation and analysis surfaces, returns scoped observations and user responses; has no direct provider credentials, Docker socket, or arbitrary process-launch interface |
| Electron main/preload | Owns native windows, dialogs, sign-in handoff, notifications, and validated host calls |
| Application workspace controller | Owns surface identity, layout, selection, pending questions, and acknowledgment of actual UI outcomes; accepts direct user actions and workspace interaction tools |
| Codex adapter | Connects the UI to the official runtime; maps conversation events and tool calls without implementing a replacement model loop |
| Workspace interaction tools | Open, update, observe, and release supported app surfaces and request contextual user input; independent of execution commands |
| Gobble tools adapter | Exposes structured project and run operations to the agent; uses the same commands as direct UI actions |
| Local Go service | Registers projects, discovers runs, serves queries, and routes commands; restart must not terminate controllers |
| Run controller | Owns scheduling, settlement, checkpoints, and reconciliation with its pinned runtime and verified daemon |
| Executors | Submit, observe, cancel, and reconcile analysis tasks on the local daemon |
| Workspace and catalog | Run workspace owns execution truth; app state owns registration/preferences, surface references and decisions, alongside rebuildable execution summaries |

Run the agent integration on the host so it can edit the selected local project
and use the host's sign-in environment. Keep compilation and analysis in the
Gobble container runtime. The proposed application service may initially run
in a runtime container; define its exact startup and mount contract before
implementation. No nested Docker daemon is required.

The service routes old runs through compatible pinned runtimes. Application,
Codex runtime, and analysis-runtime versions are managed independently. An app
update must not silently replace software required by an existing run.

Native folder selection initiates registration and Docker mount checks; a file
picker does not prove a folder is shareable. Preserve writable bind-mount
translation and containment. Arbitrary mounts, named-volume workspaces, remote
roots, and migration require separate designs.

Keep Electron renderer privileges narrow, with context isolation and validated
IPC. If the service exposes HTTP, publish only to loopback with a local session
credential and origin checks. Provider credentials remain with the host-side
agent runtime, outside renderer state, pipeline folders, logs, and images.

### Engine and Linux CLI independence

Keep `gobble` and `cmd/gobble` usable on supported Linux hosts without Electron,
Node.js, a browser, a ChatGPT account, or a running application service. Normal
compiler and executor requirements still apply. Retain Go authoring, validation,
planning, execution, structured inspection, Watch, Stop, and Resume for headless
operation, scripts, and external agents.

The service and CLI must use shared Go lifecycle behavior, workspace identity,
and owner/lease gates. Desktop-only concepts such as surface IDs and pending
questions stay outside engine state. The application catalog must not become
necessary to recover a run. Requests entering the application service use its
durable operation contract; standalone CLI calls retain their direct engine
path and the same ownership checks.

Before promising cross-interface control, verify that a compatible Linux CLI
can inspect, stop, and resume an app-started run, and that the app can discover
a CLI-started run. Use its recorded workspace and pinned runtime identity;
resolve container/host path mappings instead of bypassing identity checks.
Keep CLI documentation, headless tests, and distribution acceptance in each
engine-affecting release.

## First agent target: ChatGPT subscription sign-in

Use OpenAI's subscription OAuth through the official **Sign in with ChatGPT**
flow. This is the selected meaning of ChatGPT subscription authentication.
OpenAI documents Codex App Server as a product integration surface for
account authentication, conversation history, approvals, and streamed agent
events. This is the initial integration route for Gobble's shared agent workspace.
[Official App Server documentation](https://learn.chatgpt.com/docs/app-server)

Proposed sign-in flow: Electron requests ChatGPT login from App Server, opens
the returned authorization URL in the system browser, and observes completion.
App Server handles the callback and credentials. Model choices come from the
runtime's available-model list. Verify this flow against a pinned runtime on
each supported OS before claiming working subscription integration.

Subscription access applies to models and usage available to that account
through Codex. It does not establish access to every ChatGPT UI model or a
Platform API allowance. Keep subscription sign-in as the initial provider
path; an API-key mode and other providers are later choices with explicit
billing semantics. Account eligibility and limits remain provider-controlled.
[Official authentication documentation](https://learn.chatgpt.com/docs/auth)

The following are Gobble product requirements, not claims of completed provider
integration:

- Expose account connection, model selection, conversation, tool activity,
  reviewable source/plan changes, and run links in the desktop app. Connect the
  agent to the workspace interaction tools so it can present and inspect views
  and receive contextual user answers; account-connected chat alone is insufficient.
- Keep provider thread/turn identifiers separate from Gobble run/operation IDs.
  Restore the association when reopening a project.
- Preserve pending authorization and tool outcomes across UI reconnection;
  never translate a missing client response into permission to execute.
- Distinguish **interrupt agent turn** from **stop pipeline**. Signing out,
  expired authentication, a provider limit, or an agent crash must not stop an
  already accepted analysis or disable direct local Stop/Resume controls.
- Local analysis data and results remain on disk, while agent prompts and
  selected context are sent to OpenAI. Make context selection visible and avoid
  automatically attaching raw sequencing files or unrelated project data.
- Prefer runtime-supported OS credential storage; design and verify fallback
  behavior for systems without a usable credential store.

Choose the Codex binary acquisition method, version compatibility, and upgrade
policy during the initial integration work. Bundling versus a managed install,
license notices, and native host availability are release decisions. A successful
sign-in alone is not evidence that project editing and Gobble tool execution work.

## Project, revision, run, and attempt

These proposed application concepts must be mapped to existing workspace and
engine identities before choosing a new schema.

| Concept | Meaning |
|---|---|
| Project | Registered local root containing pipeline source and associated inputs and runs |
| Pipeline revision | Recorded source, typed configuration, dependency/runtime identity, and reviewed plan |
| Run | Durable analysis record associated with an exclusive run workspace |
| Task instance | Executable unit, including explicit sample ownership or expanded membership |
| Attempt | One task execution attempt; resumed work can create another attempt |
| Operation | Durable start, stop, or resume request, with its own acceptance and outcome |

Pipeline code and typed configuration remain authoring authority. Display the
graph emitted by Gobble's validated plan. Initial visual authoring must not
promise arbitrary round trips between edited nodes and Go source.

Record the revision used by each execution or recovery attempt. Source edits
must not rewrite already-executed definitions or provenance. For changed-source
recovery, show plan differences and expected reuse. Whether it creates a new
run or a revision within a run is an explicit compatibility decision.

## Agent-centered interaction and monitoring

The defining UX is a shared workspace that the agent can use alongside the
person. It opens a plan, selects a sample, reads a log, inspects a report, or
presents alternatives while discussing the analysis. Structured observations
and optional scoped captures let it check what the application actually shows.
User responses are associated with the evidence and its revision.

[Agent-centered workspace](../feature/agent-workspace.md) owns the proposed
layouts, surface lifecycle, attention rules, presentation-versus-inspection
behavior, contextual decisions, and concrete interaction examples. These require
discussion before UI implementation. The starting recommendation is a shared
work area with persistent conversation, tabs/splits for evidence, and always
accessible run controls. Detached windows are a separate layout choice.

The primary desktop journey is installation and dependency checks, ChatGPT
sign-in, project registration, in-app agent authoring, plan review, execution,
sample and issue monitoring, result inspection, and recovery. Browser use is
needed for official sign-in, not as the Gobble application interface.

| Surface | Main question | Required information |
|---|---|---|
| Setup and account | Can I start working? | Docker/runtime readiness, project-folder checks, account connection, available models |
| Agent conversation and activity | What is being proposed, shown, or checked? | Conversation linked to analysis surfaces, tool outcomes, source/plan changes, questions, linked runs |
| Shared work area | What evidence are we working with? | Agent- or user-opened views, selected context, comparison, freshness, pinning and view history |
| Contextual decision | What input is needed from me? | Specific question, evidence/revision, proposed effects, and response status |
| Projects and runs | What is active or needs attention? | Active, completed, interrupted, and actionable runs; observation freshness |
| Plan review | What will the analysis do? | Inputs, stages, commands, images, requested resources, destinations, validation findings |
| Run overview | What is happening now? | Preparation phase, active stages, counts, problems, available actions |
| Sample view | Which samples are delayed or failing? | Explicit sample ownership, stage progress, shared/cohort context, search |
| Task details | Why did this task behave this way? | Attempts, command, logs, timestamps, resource requests and measurements, recorded error |
| Results | What was produced, and how? | Artifacts, QC reports, provenance, execution history, reuse information |

Prioritize sample progress and attention items during execution. Use the DAG
for plan review, dependencies, and selected-stage context. Large runs require
grouping, filtering, and bounded rendering instead of an expanded node for
every task.

Display environment checks, input preparation, compilation, and image
acquisition as preparation phases when observations are available. Silence
before the first task must not look like a frozen analysis.

Keep these distinctions visible:

- Task-count percentage is not elapsed compute progress. Dynamic expansion
  can increase its denominator.
- Reused work is part of successful work and should be identifiable.
- Requested CPU and RAM differ from measured utilization. Docker VM capacity
  is not automatically the host machine's full capacity.
- Failed, blocked, skipped, incomplete, unfinalized, and unknown work retain
  their meanings even when the overview groups them.
- Execution success and scientific QC outcome are separate facts.
- Recorded errors and an agent's proposed diagnosis are labeled separately.
- ETA requires suitable historical evidence; otherwise omit it.

Use readable labels as well as color, keyboard access, and layouts that handle
dense tables and long names. Notifications lead to specific runs and actions;
the application remains useful without notifications.

## Lifecycle and recovery contract

Connection health, controller liveness, backend observability, and persisted
run outcome are separate facts. A disconnected client cannot declare a run
stopped or failed. Show the last valid observation and its time when fresh
state cannot be established.

| Scenario or action | Required application behavior |
|---|---|
| Window or agent closes | Accepted analysis continues while Docker and the computer remain running |
| Analysis surface closes or UI observation fails | Record a UI outcome; leave run state unchanged and retain recoverable questions |
| Agent interrupted, signed out, or limited | Agent activity stops or waits; direct local monitoring and pipeline controls remain usable |
| Start accepted | Return durable identifiers; acceptance is not execution success |
| Stop requested | Show request acceptance; report stopped only after settlement is proved |
| Stop repeated or delayed | Address the correct owner lease; do not stop a newer recovery owner |
| Resume | Reconcile, verify reuse, and execute unfinished or changed work in new attempts |
| Docker or controller interruption | Preserve uncertainty and reconcile before launching more work |
| Service restart | Rediscover runs and controllers without starting duplicate analysis |
| Request retried | Deduplicate by request identity and detect conflicting inputs |
| Application update | Preserve active runs and runtime pins; negotiate compatibility before control |

Current Resume is task-level recovery, not process-memory restoration. There
is no automatic rerun after Docker or computer restart.

An optional future **finish active tasks and wait** action would stop new
admission while allowing active tasks to finish. It needs a separate name and
admission/resume contract. It is distinct from current Stop, OS suspension, and
Docker pause, and is outside the initial application scope.

## Shared operations and Gobble tools

First expose reads backed by the existing Inspect projection. Add controls
after specifying versioning, request identity, concurrency, lifecycle outcomes,
and reconciliation.

Illustrative tools are `validate_pipeline`, `preview_run`, `start_run`,
`inspect_run`, `preview_resume`, `stop_run`, and `resume_run`. Their names and
schemas remain proposals. A Gobble MCP adapter is the initial candidate for
exposing these shared operations to Codex; its local transport and registration
are resolved during the integration work. It is distinct from the protocol
between Electron and Codex App Server.

The agent also receives application workspace tools for opening, observing, and
updating supported views and receiving contextual answers. Their owner is the
workspace controller, not the engine. Keep UI request identity separate from
analysis operation identity and validate tool-result delivery, including any
scoped images, against the pinned provider runtime.

Long operations return durable identifiers and remain observable after a
request or agent session ends. Transport cancellation must not silently become
pipeline cancellation after execution is accepted. All clients receive the
same factual state and action eligibility from the backend.

The integrated agent edits project source and invokes Gobble operations through
the approved project scope. External agents and CLI clients can use the same
contracts. Source changes and execution effects remain reviewable within the
user's established authorization. The engine, rather than the conversation,
is authoritative for run success and reusable outputs.

## State, events, logs, and metrics

Keep coherent checkpoints as the initial execution authority. Add event
history explaining who requested Stop, when tasks changed state, and why work
was reused or rerun. Define checkpoint/event ordering and crash recovery before
making history authoritative. Avoid independent writers for the same facts.

Start with bounded snapshot polling. Add an event stream, such as server-sent
events, after the event contract is ready. Reconnection needs a cursor and
full-snapshot fallback for gaps or expired history. Events need ordered
identities and consumer deduplication; do not assume exactly-once delivery.

Read logs by task attempt, stream, and byte range/cursor. Handle truncation,
rotation, and completed attempts. The current 4 KiB inspection tail is a preview,
not a full-history protocol. Render tool output as text.

Measurements and estimates include observation time and availability.
Monitoring collection must not determine task success or block the scheduler.
Timelines, run comparison, notifications, and reports follow the basic viewer.

A catalog database, potentially SQLite, can support discovery and preferences.
Application-owned surface references, conversation associations, and pending
decisions also need persistence; unlike execution summaries, they cannot all
be rebuilt from run files. Their format and location remain open. Large
artifacts and logs stay in files; catalog loss must not invalidate engine
workspace recovery or standalone CLI use.

## Decisions still requiring discussion

| Decision | Working recommendation | Resolve before |
|---|---|---|
| Agent workspace layout | Shared evidence area, persistent conversation, accessible run controls; compare with a conversation-led layout | First screen implementation, with user review |
| Surface and attention policy | Managed tabs/splits, scoped observation, contextual decisions; separate native-window choice | Workspace tools implementation, with user review |
| Codex distribution and compatibility | Official runtime with a pinned, tested host integration | First installer and agent integration |
| Agent state and tool bridge | Separate provider thread/turn IDs from Gobble operation/run IDs | First complete agent-driven analysis |
| Service and project registration | One local service with explicitly shared roots | Multi-project service implementation |
| Run/revision identity | Preserve executed definitions and link recovery attempts | Application persistence schema |
| API/runtime compatibility | Versioned operations routed to the pinned runtime | Application mutations |
| Event durability | Checkpoint-led state and recoverable history | Durable event publication |
| Catalog persistence | Rebuildable execution index; separately owned registration, preferences, surface references, and decisions | Persistent catalog |
| Background recovery | Reconnect and reconcile; no silent automatic rerun | Automatic recovery features |
| Finish-active-and-wait | Separate scheduling action | Adding a pause-like control |

The desktop framework, frontend language, agent-centered principle, initial
subscription-backed OpenAI provider, and standalone Linux CLI are selected.
The desktop-first recommendation was reconsidered against local web delivery
above. The table keeps unresolved UX and implementation contracts visible.

Standalone browser delivery, remote execution, HPC, cloud, native ARM analysis,
cross-workspace caching, retention/deletion, additional providers, and
bidirectional visual authoring remain later designs. The integrated agent is
part of the first useful desktop product.

## References

- [Container distribution](../../../../../../distribution/runtime/README.md)
  — installation, pinning, detached execution, and acceptance limits.
- [Monitoring](../../../../../../docs/monitoring.md)
  — projections, sample semantics, logs, and TUI boundaries.
- [Agent guide](../../../../../../docs/agent-guide.md)
  — current external-agent operation.
- [Electron process model](https://www.electronjs.org/docs/latest/tutorial/process-model)
  — desktop main, renderer, and preload responsibilities.
- [Electron distribution](https://www.electronjs.org/docs/latest/tutorial/distribution-overview)
  — packaging and release integration.
- [Codex App Server](https://learn.chatgpt.com/docs/app-server) and
  [authentication](https://learn.chatgpt.com/docs/auth)
  — official integration and subscription sign-in; checked 2026-09-06.
- [MCP tools specification](https://modelcontextprotocol.io/specification/2025-11-25/server/tools)
  — structured tool interfaces; Gobble tool names remain proposed.
