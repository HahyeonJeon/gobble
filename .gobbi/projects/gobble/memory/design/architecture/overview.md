# Gobble — Overview

## Purpose

Gobble gives coding agents and Go authors a typed pipeline engine with
machine-readable planning, execution, inspection, failure, and recovery. It also
provides five local assay products built on that engine. The first application
goal is a local workspace where users and agents design, execute, control, and
monitor pipelines together. Gobble supplies the core engine and remains a
standalone Linux CLI tool. The recommended first delivery uses a thin Electron
host with React/TypeScript. The first integrated agent target is OpenAI through
ChatGPT subscription sign-in using the official Codex runtime integration.

Gobble is the shared engine and command surface. It is not a sixth assay. The
five products are WGS joint germline, bulk RNA-seq, Methyl-seq, ATAC-seq, and
scRNA-seq. Their current identities and selected paths live in
[Assay product family and ownership](../feature/assets.md).

## Current outcome

Each product has an assay-owned strict CSV loader, typed sample values, a fresh
`DefaultConfig`, a pure `Build`, a default `Pipeline()` command adapter, selected
command modules, required artifacts, one fixture and image manifest, and all
seven lifecycle outcomes: design, build, customize, run, resume, stop, and
failure.

The products form a consistent family, not an integrated multiomics workflow.
There is no universal sample model, cross-assay graph, join identity,
missing-modality policy, or combined scientific result.

## User outcome

An author can load assay data, change typed command policy, build and inspect a
plan, and use either the Go API or generic command. An operator can run the
selected graph in an exclusive local workspace, inspect structured state,
stop active work, reuse verified successes, and retry unfinished or affected
work through Resume.

The common distribution uses Docker Compose for agents and people alike. Go,
Gobble, and authoring dependencies are in the runtime image; project files stay
local. A detached controller survives the initiating terminal or Agent session.
Packed runners and the standalone Linux CLI remain supported entry points.
There is no separate beginner installation model.

Current monitoring includes structured Inspect and a sample-aware TUI. The
next product is a desktop app with an integrated agent, shared Go operations,
and the same independent analysis controllers. Electron and React/TypeScript
are selected; standalone browser delivery is later. See
[Application and monitoring](application.md) for the selected direction and
subscription integration boundary. These capabilities are not yet implemented.

The agent can open analysis views to present evidence, ask contextual questions,
or inspect results itself. Shared surfaces and direct user controls are central
to the product; layout and attention behavior require discussion before UI
implementation. See [Agent-centered workspace](../feature/agent-workspace.md).
React views can be developed in a browser without making a separately shipped
web app a prerequisite for the desktop outcome.

The Linux CLI retains independent installation, structured inspection, Watch,
and lifecycle operations without Electron, a provider login, or a running app
service. App and CLI changes share Go engine behavior and preserve workspace
identity and recovery guarantees.

Agent sign-in enables assisted authoring and operation. Existing local runs
must remain visible and controllable without an active provider session.
Pipeline data stays local; prompts and selected agent context use the provider.

## Scope and non-goals

Current analysis execution is engineering-only on trusted-local `linux/amd64`
with local files and Docker. Windows and macOS use the common container route;
actual Docker Desktop acceptance remains outstanding. Native launcher checks
do not prove Desktop execution. Apple Silicon uses amd64 emulation.

Gobble covers graph construction, declared command execution, artifacts,
provenance, structured failure, and recovery. Docker isolation is a
convenience, not a sandbox.

The family does not claim scientific, clinical, diagnostic, regulatory, or
production-scale validity. It does not imply nf-core support or endorsement.
WGS ends at an indexed, unfiltered joint callset. Integrated cross-assay
analysis, optional nf-core routes, extra assays, serialized product parameters,
a component registry, remote execution, HPC, cloud, and a persistent application
service remain outside the implemented result. The Electron desktop app,
integrated agent, and local service are now the next product direction, ahead
of remote backends.

## Release position

The immutable `v0.1.0` tag is the earlier engine preview. The current reviewed
development baseline is `39584ce8785aa66c14788c7d484e6ef088b58c2a`. A public
`ghcr.io/hahyeonjeon/gobble:develop` runtime is available; generated projects pin
the exact runtime digest. This development distribution is not a stable v0.2.0
release. See the [roadmap](../roadmap/project.md) for acceptance boundaries.

## Constraints and authority

One maintainer owns product defaults and support decisions. Upstream benchmark
changes do not update Gobble automatically. A default command, image, task ID,
port, destination, sheet meaning, or graph generation changes only through a
deliberate compatibility decision.

## Vocabulary

| Term | Meaning |
|---|---|
| Gobble | The shared Go library, engine, generic command, workspace, and recovery model used by all five products. |
| Gobble application | The planned local agent-centered workspace using Gobble as its core engine. |
| Surface | A managed application view that a user or agent can open, select, observe, or discuss; it need not be an OS window. |
| Product | One supported assay package, selected graph generation, typed contract, required outputs, pinned defaults, and lifecycle evidence. |
| Module | One command or subcommand represented by one Gobble task. It is not a remote registry unit. |
| Pipeline | One assay-owned typed graph under `assets/pipelines/<assay>`. |
| Lifecycle scenario | One cross-product actor outcome under `tests/scenarios/<outcome>`. |
| Graph generation | A named compatibility boundary for task and artifact identity. Named lifts require new workspaces. |
| Run workspace | The caller-owned local authority for one run's state, logs, artifacts, occupancy, and reuse decisions. |
