# Stage 3 codebase design review

> Record state: **historical** — design-focused author self-review, partial specialist reach.
> The correction phase began after read-only handoff; findings refer only to the bound pre-correction identity.
> Prepared review material; no target mutation or stage approval authority.

## Summary

Five supported design/API problems and four focused structural improvements are recorded below. Existing engine isolation, pure transitions and persistence checks should be preserved. Live-provider/runtime, release and independent usability claims remain outside available evidence.

## Subject and Scope

- Relationship: **author — disclosed self-review**. No independent reviewer claim.
- State: in progress; source remains read-only until this review report is handed off to implementation.
- Repository: `/Users/hahyeon/.codex/.chatgpt-projects/g-p-6a9bae3a3ef481919aec253b9710060a/gobble`.
- Branch: `codex/project-workspace-design`.
- Baseline HEAD: `7fe3d5c9e10d0cc104a15b6e7aa3def0d15b98f1`; the implemented app is uncommitted, so HEAD alone does not identify it.
- Bound content SHA-256: `e31b593c3798ebe981ae629862a65c0c086b917bcab68cee7db93b5b34d45f12` (117 files, sorted relative pathname + NUL + bytes + NUL).
- Subject: all nonignored files under app/, internal/appservice/, cmd/gobble-service/, the design memory and four session-plan documents; desktop CI and review README.
- Governing sources: user-approved Project workflow, English app, stages 1–3, this turn's design review/correction request and root instructions. Synced sources/ are read-only and outside implementation scope.
- Exclusions: node_modules, build/test outputs, generated concept images outside app/, historical stage reports, and this report. Existing engine/CLI source is a reference for Pipeline/Plan/Run semantics, not an authorized broad engine refactor.
- Questions: domain definitions and metadata ownership; directory/file/API boundaries; lifecycle and view composition; minimal maintainable patterns before agents are added.
- Report owner: user; path does not overlap bound subject. Any subject or governing change invalidates current review claims and must be recorded before correction.
- Inspection method: read-only source/contract/test/document inspection; report is the only review write.


## Method

Actual code was inspected before the checklist. Read-only commands were `rg`, `cat`, `sed`, `git status/ls-files/rev-parse` and content hashing. Source reach is the contracts, host/service/controller/model, leaf views, native catalog/query boundaries, test/config declarations cited below. Static inspection is sampled; it is not a fresh full runtime/security/performance audit. The exact toolchain is recorded in app/package.json and prior stage 3 evidence.

### Checklist-Free Critical Review

Locked before checklist exposure. The source identity still matches the binding.

- Design/intent: portable Project, Surface and Evidence records plus a pure transition already separate layout from engine execution. However `project.ts` also defines agent/resource contracts; `service.ts` combines Project, file, runtime, raw Monitor, transport and rendering text; `workspace-session.ts` combines persisted document and IPC. These files change for unrelated reasons (I1).
- Failure/misuse: `ResourceRef(kind=log).taskId` is populated from Monitor `identity`, while `monitor/snapshot.go` explicitly distinguishes `task_id` and instance `identity`. RunView sends this misleading field back as the service `instance` query; a future caller using the authored task ID selects the wrong scope (P1).
- State/data/effects: `WorkspaceController` owns durable documents/revisions and ephemeral renderer leases/session/load concurrency. Both have valid logic but unrelated lifetimes; add a separate render-session collaborator before provider tools reuse observation state (I2). `paneOf` silently returns secondary for an absent surface, hiding a precondition (P3).
- Change/integration: raw `snapshot` JSON reaches RunView, which interprets unknown values and edges itself; the Pipeline label is available in the engine but has no typed UI read model. Metadata interpretation needs a single explicit adaptation boundary (P2). Service URL construction/response parsing is duplicated between service IPC and workspace ServiceResources (I3).
- Absences: no exact glossary resolves app Workspace vs execution workspace, definition vs Plan vs Run, source Resource vs opened Surface vs Pane vs OS Window, or future Pipeline presentation. Existing contracts already support a bounded answer; future pipeline discovery must not be fabricated from Monitor.pipeline (P4).
- Direct UI lead: RunView chooses literal secondary for logs, even if RunView itself is secondary; leaf view owns layout policy. Sidebar owns both native navigation composition and Files/Runs data effects. Replace broad command props in RunView with a log-open callback and let pane composition choose the destination; extract focused resource browsers (P5/I4).
- Strengths: native engine-independent service; private authenticated transport; strict sandbox; runtime pinning; atomic profile state; Project-scoped selection and request checks; real Electron and native-process tests.
- Gaps: this is author self-review and design-focused; no independent usability, live Docker/engine, packaging, provider or cross-platform qualification. Existing native engine is a read-only vocabulary reference.

Evidence: app/contracts/src/{project,service,workspace-session,surface,workspace,validation}.ts;
app/desktop/src/main/{workspace/{controller,model,service,storage,selection},service/{client,ipc}}.ts;
app/desktop/src/renderer/workspace/{Sidebar,Pane,WorkspaceApp,views/{SurfaceView,RunView,TableView,TextView}}.tsx;
internal/appservice/{types,runs,service,catalog}.go; monitor/snapshot.go; pipeline.go; plan.go;
app/tests/architecture.test.ts; app/desktop/tests/{workspace,service,security}.test.ts; app/contracts/tests/contracts.test.ts.

This unaided record will not be backfilled from checklist findings.

### Core Category Applicability

| Category | Result | Evidence / reason | Gap |
|---|---|---|---|
| Project Fit | applicable | docs/desktop-workspace/session-plan/plan-01.md, task-03-workspace: Project UI and read-only scope; stage 4 is not implemented. | None |
| Affected Surfaces | applicable | app/contracts/src/workspace-session.ts, main/workspace/controller.ts, preload/index.ts and renderer/workspace/useWorkspace.ts share command/response types. | None |
| Project Structure | applicable | app/contracts/src/service.ts and workspace-session.ts each combine domain schemas and transport; see I1. | None |
| Architecture | applicable | main/workspace/model.ts transition, storage.ts AtomicStateFile, internal/appservice/service.go Service and runtime.go adapter each have explicit owners; see I2/I3. | None |
| Design Pattern | applicable | main/workspace/model.ts transition is pure; controller serializes writes; ServiceResources adapts the native boundary without inheritance. | None |
| Abstraction | applicable | renderer/workspace/views/RunView.tsx record/text/edge filtering interprets engine JSON; see P2. | None |
| Data Model | applicable | contracts/src/validation.ts parseWorkspace enforces Project, surface and pane membership; monitor/snapshot.go distinguishes authored task IDs and instances. | None |
| Public API | applicable | main/workspace/model.ts paneOf; main/workspace/service.ts describe/read; service/client.ts request; see P3. | None |
| Parameters | applicable | contracts/src/project.ts ResourceRefSchema log.taskId and RunView command construction; see P1. | None |
| Modularization | applicable | contracts/src/service.ts, workspace-session.ts and renderer/workspace/Sidebar.tsx own independently changing concerns; see I1/I4. | None |
| Reusability | applicable | main/service/ipc.ts and main/workspace/service.ts repeat URLs and parsing; workspace model and document parser repeat resource identity; see I3. | None |
| Performance | evidence missing | No fresh measurements or artifact execution in the read-only design pass; static declarations alone cannot prove this category. | G2 |
| Optimization | evidence missing | No fresh measurements or artifact execution in the read-only design pass; static declarations alone cannot prove this category. | G2 |
| Unintended Overengineering | applicable | private profile store, request receipts, split/duplicate and narrow IPC all have current stage 1–3 consumers; no DI/plugin framework exists. | None |
| Code Complexity | applicable | main/workspace/controller.ts queue and render lease paths; model.ts single transition; renderer RunView parses unknown raw metadata; see P2/I2. | None |
| Readability | applicable | workspace/model.ts named transition cases and service/client.ts bounded startup/request/stop; CSS is separated from React rendering. | None |
| Vocabulary | applicable | contracts/src/project.ts log.taskId; internal/appservice/types.go Run is registration; monitor/snapshot.go Pipeline is a label; see P1/P4. | None |
| Naming Convention | applicable | PascalCase React components and public classes, camelCase TypeScript functions, Go package-local helpers; Run naming precision is P4. | None |
| Docstring | applicable | main/workspace/service.ts WorkspaceResources signatures and model.ts paneOf have implicit preconditions; see P3. | None |
| Correctness | applicable | contracts/validation.ts and workspace/controller.ts checks and storage.ts commit-before-publish; RunView secondary destination is P5. | None |
| Testing | applicable | desktop/tests/workspace.test.ts exercises stale/foreign/duplicate/restoration/late loads; contracts/tests/contracts.test.ts validates reference invariants; runtime fixture lacks distinct authored/instance IDs (P1). | None |
| Verification | evidence missing | No fresh measurements or artifact execution in the read-only design pass; static declarations alone cannot prove this category. | G2 |
| Delivery | evidence missing | No fresh measurements or artifact execution in the read-only design pass; static declarations alone cannot prove this category. | G2 |
| Usability | applicable | WorkspaceApp, Pane, Discussion and SurfaceView expose explicit loading/error/selection/save states; RunView destination inconsistency is P5. | None |
| Operations | applicable | service/client.ts startup/request/stop deadlines; workspace/storage.ts preserves failures and main/index.ts protects unsaved drafts. Live-runtime recovery is G1. | None |
| Compatibility | applicable | workspace/storage.ts rejects future schemas without overwriting; current v1 log field is a persisted compatibility constraint, see P1. | None |

### Overlay Applicability

Sources frozen to the bound project contract, interface design, code/config and this checklist. Reviewer qualification: implementing maintainer; no independent specialist certification is claimed.

| Overlay | Result | Source and activation | Reach / limit |
|---|---|---|---|
| Security | activated | Project contract Renderer isolation/Host to Go service; actual main/security and service.Handler authorization | Named sender/payload checks and profile/process isolation inspected; no penetration-test claim. |
| Privacy | activated | Project contract Persistence and Multi-agent coordination; Discussion and workspace/storage | Local drafts and selections; no provider copies. Account/retention/deletion integration is future G1. |
| Concurrency | activated | Project contract session/revision semantics; controller queue; Service.coalescedQuery | Serialized writes and observation invalidation are explicit; live runtime shutdown needs G1. |
| Accessibility | activated | feature/agent-workspace.md Desktop lifetime/accessibility; Pane/Splitter/TextView | Named roles and keyboard paths inspected. Screen-reader/manual usability qualification is G1. |
| Localization | activated | User English request; renderer labels and Intl.DateTimeFormat en | App-owned labels are English; filenames remain original. No multilingual release claim. |
| Dependencies | activated | app/package.json and package-lock.json | Pinned declarations inspected; no fresh vulnerability or supply-chain audit, G2. |
| Build | activated | app/package.json, electron.vite.config.ts, tsconfig and tests/architecture.test.ts | Process builds and schema generator are explicit; post-correction execution is G2. |
| Packaging | not applicable | Private source workspaces only; installable .app is stage 7 | No packaging change or supported installed artifact in this review. |
| Release | not applicable | Session plan stage 7 deferred | No release/version publication authorized. |
| Deployment | not applicable | Local source/build only | No rollout or remote environment exists in this slice. |
| Configuration | activated | app/README.md isolated profile and pinned runtime; service/client.ts | Private profile and fixed service handshake; no arbitrary renderer command setting. |
| Observability | activated | SurfaceView errors; main/index.ts save failures; service/client.ts timeouts | User-visible failures inspected; support telemetry/full live diagnosis is G1. |
| Migration | activated | workspace/storage.ts unknown-version preservation; current v1 contracts | Existing user profiles require wire preservation; no automatic reset. Future field rename needs explicit versioned migration. |
| Deprecation | not applicable | No feature removal planned; v1 data retained | No supported phase-out implemented. |
| Retirement | not applicable | Stage 3 active local development | No retirement/consumer exit requested. |

### Reconciliation

P1–P5 originate in `critical review`; checklist links identify their related effects without creating duplicate problems. I1–I4 remain optional structural improvements. G1/G2 narrow general verification claims. No checklist observation was added to the locked unaided record.

## Checklist Review

Checklist: [/Users/hahyeon/.codex/plugins/cache/gobbi-workspace/gobbi/1.2.4/skills/code-review/checklist.md](/Users/hahyeon/.codex/plugins/cache/gobbi-workspace/gobbi/1.2.4/skills/code-review/checklist.md), SHA-256 `89aaf24bad0d5f0deddba8df22103f7380c6aa348effe3468c5573cbc43fce07`. Items preserve source order and wording. “No problem found” means within the cited static reach only.

### Project Fit

#### The code work lacks accepted project direction or control

- [ ] The code work has no current purpose or intended result.
  Result: `no problem found`.
  Evidence: docs/desktop-workspace/session-plan/plan-01.md, task-03-workspace: Project UI and read-only scope; stage 4 is not implemented.

- [ ] The accepted scope of the code work is unclear.
  Result: `no problem found`.
  Evidence: docs/desktop-workspace/session-plan/plan-01.md, task-03-workspace: Project UI and read-only scope; stage 4 is not implemented.

- [ ] The code work has no observable completion condition.
  Result: `no problem found`.
  Evidence: docs/desktop-workspace/session-plan/plan-01.md, task-03-workspace: Project UI and read-only scope; stage 4 is not implemented.

- [ ] The implemented behavior conflicts with the accepted project purpose.
  Result: `no problem found`.
  Evidence: docs/desktop-workspace/session-plan/plan-01.md, task-03-workspace: Project UI and read-only scope; stage 4 is not implemented.

- [ ] The implemented behavior satisfies the form of an accepted decision while missing its intended result.
  Result: `no problem found`.
  Evidence: docs/desktop-workspace/session-plan/plan-01.md, task-03-workspace: Project UI and read-only scope; stage 4 is not implemented.

- [ ] The code conflicts with an applicable project rule, decision, or governance constraint.
  Result: `no problem found`.
  Evidence: docs/desktop-workspace/session-plan/plan-01.md, task-03-workspace: Project UI and read-only scope; stage 4 is not implemented.

- [ ] An accepted project baseline changes without a traceable change-control decision.
  Result: `no problem found`.
  Evidence: docs/desktop-workspace/session-plan/plan-01.md, task-03-workspace: Project UI and read-only scope; stage 4 is not implemented.

- [ ] A project result remains presented as current after it becomes temporary, superseded, or archived.
  Result: `no problem found`.
  Evidence: docs/desktop-workspace/session-plan/plan-01.md, task-03-workspace: Project UI and read-only scope; stage 4 is not implemented.

### Affected Surfaces

#### The change leaves affected work unknown or inconsistent

- [ ] A material affected surface cannot be identified from the change and its governing sources.
  Result: `no problem found`.
  Evidence: app/contracts/src/workspace-session.ts, main/workspace/controller.ts, preload/index.ts and renderer/workspace/useWorkspace.ts share command/response types.

- [ ] Code and one of its callers express different versions of the same contract.
  Result: `no problem found`.
  Evidence: app/contracts/src/workspace-session.ts, main/workspace/controller.ts, preload/index.ts and renderer/workspace/useWorkspace.ts share command/response types.

- [ ] A material consistency-bound surface remains stale after the change.
  Result: `no problem found`.
  Evidence: app/contracts/src/workspace-session.ts, main/workspace/controller.ts, preload/index.ts and renderer/workspace/useWorkspace.ts share command/response types.

- [ ] Two implementations of the same project rule remain inconsistent after the change.
  Result: `no problem found`.
  Evidence: app/contracts/src/workspace-session.ts, main/workspace/controller.ts, preload/index.ts and renderer/workspace/useWorkspace.ts share command/response types.

#### Affected work cannot be coordinated

- [ ] Coordination ownership or handoff is unclear for a material affected surface.
  Result: `no problem found`.
  Evidence: app/contracts/src/workspace-session.ts, main/workspace/controller.ts, preload/index.ts and renderer/workspace/useWorkspace.ts share command/response types.

- [ ] Affected owners have conflicting coordination expectations.
  Result: `no problem found`.
  Evidence: app/contracts/src/workspace-session.ts, main/workspace/controller.ts, preload/index.ts and renderer/workspace/useWorkspace.ts share command/response types.

### Project Structure

#### Project layout is incoherent or misleading

- [ ] Code placement conflicts with the project's accepted structural convention.
  Result: `no problem found`.
  Evidence: app/contracts/src/service.ts and workspace-session.ts each combine domain schemas and transport; see I1.

- [ ] One file combines code with independent ownership or change reasons.
  Result: `no problem found`.
  Evidence: app/contracts/src/service.ts and workspace-session.ts each combine domain schemas and transport; see I1.

- [ ] A coherent unit is fragmented across files without a project reason.
  Result: `no problem found`.
  Evidence: app/contracts/src/service.ts and workspace-session.ts each combine domain schemas and transport; see I1.

- [ ] Code ordering inside a file conflicts with the project's accepted structure.
  Result: `no problem found`.
  Evidence: app/contracts/src/service.ts and workspace-session.ts each combine domain schemas and transport; see I1.

- [ ] Code visibility does not match its structural owner.
  Result: `no problem found`.
  Evidence: app/contracts/src/service.ts and workspace-session.ts each combine domain schemas and transport; see I1.

- [ ] The source, generator, inputs, output location, or regeneration owner is unclear for generated or derived code.
  Result: `no problem found`.
  Evidence: app/contracts/src/service.ts and workspace-session.ts each combine domain schemas and transport; see I1.

#### Refactoring leaves the project structure worse

- [ ] A refactoring leaves duplicated or obsolete code locations.
  Result: `no problem found`.
  Evidence: app/contracts/src/service.ts and workspace-session.ts each combine domain schemas and transport; see I1.

- [ ] A refactoring fragments one structural owner across misleading locations.
  Result: `no problem found`.
  Evidence: app/contracts/src/service.ts and workspace-session.ts each combine domain schemas and transport; see I1.

- [ ] A refactoring leaves the declared structure inconsistent with the actual ownership boundary.
  Result: `no problem found`.
  Evidence: app/contracts/src/service.ts and workspace-session.ts each combine domain schemas and transport; see I1.

### Architecture

#### Responsibilities, dependencies, or ownership sit in the wrong boundaries

- [ ] A system unit owns materially unrelated responsibilities.
  Result: `no problem found`.
  Evidence: main/workspace/model.ts transition, storage.ts AtomicStateFile, internal/appservice/service.go Service and runtime.go adapter each have explicit owners; see I2/I3.

- [ ] One responsibility is divided across units without a clear owner.
  Result: `no problem found`.
  Evidence: main/workspace/model.ts transition, storage.ts AtomicStateFile, internal/appservice/service.go Service and runtime.go adapter each have explicit owners; see I2/I3.

- [x] A dependency points against the accepted responsibility direction.
  Result: `problem found`.
  Evidence: main/workspace/model.ts transition, storage.ts AtomicStateFile, internal/appservice/service.go Service and runtime.go adapter each have explicit owners; see I2/I3.
  Finding: [P5](#p5).

- [ ] State or data ownership is unclear across system boundaries.
  Result: `no problem found`.
  Evidence: main/workspace/model.ts transition, storage.ts AtomicStateFile, internal/appservice/service.go Service and runtime.go adapter each have explicit owners; see I2/I3.

- [ ] Resource lifetime ownership is unclear across system boundaries.
  Result: `no problem found`.
  Evidence: main/workspace/model.ts transition, storage.ts AtomicStateFile, internal/appservice/service.go Service and runtime.go adapter each have explicit owners; see I2/I3.

- [ ] Failure ownership is unclear across system boundaries.
  Result: `no problem found`.
  Evidence: main/workspace/model.ts transition, storage.ts AtomicStateFile, internal/appservice/service.go Service and runtime.go adapter each have explicit owners; see I2/I3.

- [ ] A failure in one concern spreads into an unrelated concern because their boundaries are coupled.
  Result: `no problem found`.
  Evidence: main/workspace/model.ts transition, storage.ts AtomicStateFile, internal/appservice/service.go Service and runtime.go adapter each have explicit owners; see I2/I3.

- [ ] Responsibilities that must succeed or fail as one consistency unit are split across boundaries without one transaction or recovery owner.
  Result: `no problem found`.
  Evidence: main/workspace/model.ts transition, storage.ts AtomicStateFile, internal/appservice/service.go Service and runtime.go adapter each have explicit owners; see I2/I3.

### Design Pattern

#### A design pattern does not fit the current problem

- [ ] A recognizable design pattern conflicts with the accepted design.
  Result: `no problem found`.
  Evidence: main/workspace/model.ts transition is pure; controller serializes writes; ServiceResources adapts the native boundary without inheritance.

- [ ] A design pattern hides control, data, state, or failure flow that its consumers must understand.
  Result: `no problem found`.
  Evidence: main/workspace/model.ts transition is pure; controller serializes writes; ServiceResources adapts the native boundary without inheritance.

- [ ] Participants in one design pattern follow conflicting roles or lifecycle rules.
  Result: `no problem found`.
  Evidence: main/workspace/model.ts transition is pure; controller serializes writes; ServiceResources adapts the native boundary without inheritance.

### Abstraction

#### A concept boundary hides or exposes the wrong details

- [x] An abstraction leaks a private implementation detail into its consumer contract.
  Result: `problem found`.
  Evidence: renderer/workspace/views/RunView.tsx record/text/edge filtering interprets engine JSON; see P2.
  Finding: [P2](#p2).

- [ ] One abstraction combines concepts with independent meanings or change reasons.
  Result: `no problem found`.
  Evidence: renderer/workspace/views/RunView.tsx record/text/edge filtering interprets engine JSON; see P2.

- [ ] Consumer-specific branching makes one abstraction serve conflicting concepts.
  Result: `no problem found`.
  Evidence: renderer/workspace/views/RunView.tsx record/text/edge filtering interprets engine JSON; see P2.

- [ ] An abstraction hides an invariant, effect, state, or failure that consumers must reason about.
  Result: `no problem found`.
  Evidence: renderer/workspace/views/RunView.tsx record/text/edge filtering interprets engine JSON; see P2.

- [x] The abstraction level forces current consumers to work above or below the concept boundary they need.
  Result: `problem found`.
  Evidence: renderer/workspace/views/RunView.tsx record/text/edge filtering interprets engine JSON; see P2.
  Finding: [P2](#p2).

### Data Model

#### The data model cannot preserve required domain meaning and invariants

- [ ] The data model cannot represent a required domain state or relationship.
  Result: `no problem found`.
  Evidence: contracts/src/validation.ts parseWorkspace enforces Project, surface and pane membership; monitor/snapshot.go distinguishes authored task IDs and instances.

- [ ] The data model represents a state or relationship that the domain forbids.
  Result: `no problem found`.
  Evidence: contracts/src/validation.ts parseWorkspace enforces Project, surface and pane membership; monitor/snapshot.go distinguishes authored task IDs and instances.

- [ ] Identity is missing, unstable, or ambiguous where behavior or references depend on it.
  Result: `no problem found`.
  Evidence: contracts/src/validation.ts parseWorkspace enforces Project, surface and pane membership; monitor/snapshot.go distinguishes authored task IDs and instances.

- [ ] One fact has multiple independently writable representations that can disagree.
  Result: `no problem found`.
  Evidence: contracts/src/validation.ts parseWorkspace enforces Project, surface and pane membership; monitor/snapshot.go distinguishes authored task IDs and instances.

- [ ] The data model collapses distinct absence, default, unknown, and invalid states that behavior must distinguish.
  Result: `no problem found`.
  Evidence: contracts/src/validation.ts parseWorkspace enforces Project, surface and pane membership; monitor/snapshot.go distinguishes authored task IDs and instances.

### Public API

#### The public entry surface makes correct use hard

- [ ] Public entry points do not make the ordinary use path clear.
  Result: `no problem found`.
  Evidence: main/workspace/model.ts paneOf; main/workspace/service.ts describe/read; service/client.ts request; see P3.

- [x] A public operation does not distinguish completion states that require different consumer responses.
  Result: `problem found`.
  Evidence: main/workspace/model.ts paneOf; main/workspace/service.ts describe/read; service/client.ts request; see P3.
  Finding: [P3](#p3).

- [ ] Required public operation ordering is unclear.
  Result: `no problem found`.
  Evidence: main/workspace/model.ts paneOf; main/workspace/service.ts describe/read; service/client.ts request; see P3.

- [ ] Ownership, mutation, lifetime, cleanup, or retry obligations for public data, resources, and effects are hidden from the consumer.
  Result: `no problem found`.
  Evidence: main/workspace/model.ts paneOf; main/workspace/service.ts describe/read; service/client.ts request; see P3.

- [ ] Public failure or recovery behavior is hidden from its consumer.
  Result: `no problem found`.
  Evidence: main/workspace/model.ts paneOf; main/workspace/service.ts describe/read; service/client.ts request; see P3.

- [ ] Overlapping public entry points leave the intended choice unclear.
  Result: `no problem found`.
  Evidence: main/workspace/model.ts paneOf; main/workspace/service.ts describe/read; service/client.ts request; see P3.

- [ ] A caller must inspect private implementation details to use the public surface correctly.
  Result: `no problem found`.
  Evidence: main/workspace/model.ts paneOf; main/workspace/service.ts describe/read; service/client.ts request; see P3.

### Parameters

#### Parameters make valid calls hard to express correctly

- [ ] A parameter makes the caller supply a decision that the implementation already owns.
  Result: `no problem found`.
  Evidence: contracts/src/project.ts ResourceRefSchema log.taskId and RunView command construction; see P1.

- [ ] Parameter order makes distinct values easy to exchange accidentally.
  Result: `no problem found`.
  Evidence: contracts/src/project.ts ResourceRefSchema log.taskId and RunView command construction; see P1.

- [x] A parameter name does not identify the decision or value the caller supplies.
  Result: `problem found`.
  Evidence: contracts/src/project.ts ResourceRefSchema log.taskId and RunView command construction; see P1.
  Finding: [P1](#p1).

- [ ] Related parameter values are grouped in a way that permits contradictory combinations.
  Result: `no problem found`.
  Evidence: contracts/src/project.ts ResourceRefSchema log.taskId and RunView command construction; see P1.

- [ ] A parameter type or representation admits states outside the accepted input model.
  Result: `no problem found`.
  Evidence: contracts/src/project.ts ResourceRefSchema log.taskId and RunView command construction; see P1.

- [ ] Parameter validation occurs outside the boundary that owns the input contract.
  Result: `no problem found`.
  Evidence: contracts/src/project.ts ResourceRefSchema log.taskId and RunView command construction; see P1.

- [ ] A parameter default or omission changes behavior in a way the caller cannot predict.
  Result: `no problem found`.
  Evidence: contracts/src/project.ts ResourceRefSchema log.taskId and RunView command construction; see P1.

- [ ] A mode parameter combines behaviors with different contracts in one call surface.
  Result: `no problem found`.
  Evidence: contracts/src/project.ts ResourceRefSchema log.taskId and RunView command construction; see P1.

- [ ] A parameter does not identify the unit, encoding, normalization, or reference frame needed to interpret its value.
  Result: `no problem found`.
  Evidence: contracts/src/project.ts ResourceRefSchema log.taskId and RunView command construction; see P1.

- [ ] A parameter representation collapses absent, default, unknown, and invalid caller states that the operation must distinguish.
  Result: `no problem found`.
  Evidence: contracts/src/project.ts ResourceRefSchema log.taskId and RunView command construction; see P1.

### Modularization

#### Unit boundaries do not match cohesive ownership and change

- [ ] Code that changes together is split across units that require repeated coordinated edits.
  Result: `no problem found`.
  Evidence: contracts/src/service.ts, workspace-session.ts and renderer/workspace/Sidebar.tsx own independently changing concerns; see I1/I4.

- [ ] Independently changing concerns share one unit and force unrelated edits.
  Result: `no problem found`.
  Evidence: contracts/src/service.ts, workspace-session.ts and renderer/workspace/Sidebar.tsx own independently changing concerns; see I1/I4.

- [ ] A unit exposes more surface than its consumers need.
  Result: `no problem found`.
  Evidence: contracts/src/service.ts, workspace-session.ts and renderer/workspace/Sidebar.tsx own independently changing concerns; see I1/I4.

- [ ] A unit cannot be understood or tested without unrelated parts of the system.
  Result: `no problem found`.
  Evidence: contracts/src/service.ts, workspace-session.ts and renderer/workspace/Sidebar.tsx own independently changing concerns; see I1/I4.

- [ ] A dependency cycle makes unit ownership or change order unclear.
  Result: `no problem found`.
  Evidence: contracts/src/service.ts, workspace-session.ts and renderer/workspace/Sidebar.tsx own independently changing concerns; see I1/I4.

### Reusability

#### Shared behavior has no stable, proportionate owner

- [ ] Current consumers duplicate one behavior or domain rule in implementations that must agree but can drift independently.
  Result: `no problem found`.
  Evidence: main/service/ipc.ts and main/workspace/service.ts repeat URLs and parsing; workspace model and document parser repeat resource identity; see I3.

- [ ] Shared code branches by consumer because it combines different responsibilities.
  Result: `no problem found`.
  Evidence: main/service/ipc.ts and main/workspace/service.ts repeat URLs and parsing; workspace model and document parser repeat resource identity; see I3.

- [ ] Reuse forces independent consumers to coordinate unrelated changes.
  Result: `no problem found`.
  Evidence: main/service/ipc.ts and main/workspace/service.ts repeat URLs and parsing; workspace model and document parser repeat resource identity; see I3.

### Performance

#### Observable resource behavior fails a current need

- [ ] Measured latency exceeds a current representative need.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] Measured processing capacity falls below a current representative need.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] Measured resource use violates a current budget or supported operating range.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] A change regresses a representative performance baseline.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

### Optimization

#### An attempted optimization is not justified by its observed effect

- [ ] The optimization uses a measurement frame that does not represent the current need.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] The optimization has no demonstrated benefit on its target workload.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] The optimization shifts material cost outside its target workload.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] The optimization's measured benefit is disproportionate to its added non-performance cost.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

### Unintended Overengineering

#### A mechanism has no support from current requirements or observed need

- [ ] A structural boundary has no current requirement or current consumer.
  Result: `no problem found`.
  Evidence: private profile store, request receipts, split/duplicate and narrow IPC all have current stage 1–3 consumers; no DI/plugin framework exists.

- [ ] An optional behavior or configuration surface has no current need.
  Result: `no problem found`.
  Evidence: private profile store, request receipts, split/duplicate and narrow IPC all have current stage 1–3 consumers; no DI/plugin framework exists.

- [ ] A fallback has no current requirement or observed failure to handle.
  Result: `no problem found`.
  Evidence: private profile store, request receipts, split/duplicate and narrow IPC all have current stage 1–3 consumers; no DI/plugin framework exists.

- [ ] A compatibility path has no current supported consumer or version.
  Result: `no problem found`.
  Evidence: private profile store, request receipts, split/duplicate and narrow IPC all have current stage 1–3 consumers; no DI/plugin framework exists.

- [ ] A generic mechanism or future variant is shaped around hypothetical consumers rather than current use.
  Result: `no problem found`.
  Evidence: private profile store, request receipts, split/duplicate and narrow IPC all have current stage 1–3 consumers; no DI/plugin framework exists.

- [ ] A forwarding layer adds no ownership, policy, transformation, or stable boundary.
  Result: `no problem found`.
  Evidence: private profile store, request receipts, split/duplicate and narrow IPC all have current stage 1–3 consumers; no DI/plugin framework exists.

- [ ] A mechanism remains after its load-bearing current-need premise is disproved.
  Result: `no problem found`.
  Evidence: private profile store, request receipts, split/duplicate and narrow IPC all have current stage 1–3 consumers; no DI/plugin framework exists.

- [ ] A mechanism that serves no current execution or supported path remains in the affected surface.
  Result: `no problem found`.
  Evidence: private profile store, request receipts, split/duplicate and narrow IPC all have current stage 1–3 consumers; no DI/plugin framework exists.

### Code Complexity

#### Required behavior is harder to reason about than its domain requires

- [ ] Avoidable control flow makes behavior difficult to trace.
  Result: `no problem found`.
  Evidence: main/workspace/controller.ts queue and render lease paths; model.ts single transition; renderer RunView parses unknown raw metadata; see P2/I2.

- [ ] Avoidable mutable state makes an operation's result difficult to predict.
  Result: `no problem found`.
  Evidence: main/workspace/controller.ts queue and render lease paths; model.ts single transition; renderer RunView parses unknown raw metadata; see P2/I2.

- [ ] Avoidable dependency structure makes a change difficult to reason about.
  Result: `no problem found`.
  Evidence: main/workspace/controller.ts queue and render lease paths; model.ts single transition; renderer RunView parses unknown raw metadata; see P2/I2.

- [ ] One behavior is scattered across more units than its current domain needs.
  Result: `no problem found`.
  Evidence: main/workspace/controller.ts queue and render lease paths; model.ts single transition; renderer RunView parses unknown raw metadata; see P2/I2.

- [ ] Local behavior requires tracing distant unrelated dependencies.
  Result: `no problem found`.
  Evidence: main/workspace/controller.ts queue and render lease paths; model.ts single transition; renderer RunView parses unknown raw metadata; see P2/I2.

- [ ] Necessary domain complexity is mixed with accidental implementation complexity.
  Result: `no problem found`.
  Evidence: main/workspace/controller.ts queue and render lease paths; model.ts single transition; renderer RunView parses unknown raw metadata; see P2/I2.

### Readability

#### Local code expression obscures behavior

- [ ] Dense or indirect expression makes local behavior difficult to follow.
  Result: `no problem found`.
  Evidence: workspace/model.ts named transition cases and service/client.ts bounded startup/request/stop; CSS is separated from React rendering.

- [ ] Local flow or ordering hides the sequence of effects.
  Result: `no problem found`.
  Evidence: workspace/model.ts named transition cases and service/client.ts bounded startup/request/stop; CSS is separated from React rendering.

- [ ] A local expression hides an important invariant or domain rule.
  Result: `no problem found`.
  Evidence: workspace/model.ts named transition cases and service/client.ts bounded startup/request/stop; CSS is separated from React rendering.

- [ ] Local formatting obscures code grouping or flow.
  Result: `no problem found`.
  Evidence: workspace/model.ts named transition cases and service/client.ts bounded startup/request/stop; CSS is separated from React rendering.

- [ ] An internal comment conflicts with the code it describes.
  Result: `no problem found`.
  Evidence: workspace/model.ts named transition cases and service/client.ts bounded startup/request/stop; CSS is separated from React rendering.

- [ ] Internal comments restate syntax while a non-obvious reason or constraint remains hidden.
  Result: `no problem found`.
  Evidence: workspace/model.ts named transition cases and service/client.ts bounded startup/request/stop; CSS is separated from React rendering.

### Vocabulary

#### Code uses an inaccurate or unstable domain vocabulary

- [ ] The same domain concept uses conflicting terms across nearby code and contracts.
  Result: `no problem found`.
  Evidence: contracts/src/project.ts log.taskId; internal/appservice/types.go Run is registration; monitor/snapshot.go Pipeline is a label; see P1/P4.

- [x] One term names different domain concepts in the affected surface.
  Result: `problem found`.
  Evidence: contracts/src/project.ts log.taskId; internal/appservice/types.go Run is registration; monitor/snapshot.go Pipeline is a label; see P1/P4.
  Finding: [P4](#p4).

- [ ] A term describes an accidental mechanism instead of the domain concept it represents.
  Result: `no problem found`.
  Evidence: contracts/src/project.ts log.taskId; internal/appservice/types.go Run is registration; monitor/snapshot.go Pipeline is a label; see P1/P4.

- [x] A term misstates the responsibility, state, value, or effect it names.
  Result: `problem found`.
  Evidence: contracts/src/project.ts log.taskId; internal/appservice/types.go Run is registration; monitor/snapshot.go Pipeline is a label; see P1/P4.
  Finding: [P1](#p1).

### Naming Convention

#### Identifiers violate the applicable naming owner

- [ ] An identifier violates the applicable casing or word-form convention.
  Result: `no problem found`.
  Evidence: PascalCase React components and public classes, camelCase TypeScript functions, Go package-local helpers; Run naming precision is P4.

- [ ] An abbreviation or spelling conflicts with the applicable project or language convention.
  Result: `no problem found`.
  Evidence: PascalCase React components and public classes, camelCase TypeScript functions, Go package-local helpers; Run naming precision is P4.

- [ ] A visibility or namespace name conflicts with the role of the identified code.
  Result: `no problem found`.
  Evidence: PascalCase React components and public classes, camelCase TypeScript functions, Go package-local helpers; Run naming precision is P4.

- [ ] Definitions and uses apply inconsistent names to the same program role.
  Result: `no problem found`.
  Evidence: PascalCase React components and public classes, camelCase TypeScript functions, Go package-local helpers; Run naming precision is P4.

### Docstring

#### Caller-facing contract documentation is missing or wrong

- [ ] A unit that requires a docstring has none.
  Result: `no problem found`.
  Evidence: main/workspace/service.ts WorkspaceResources signatures and model.ts paneOf have implicit preconditions; see P3.

- [ ] A docstring conflicts with the behavior it describes.
  Result: `no problem found`.
  Evidence: main/workspace/service.ts WorkspaceResources signatures and model.ts paneOf have implicit preconditions; see P3.

- [x] A docstring omits a caller-relevant contract detail.
  Result: `problem found`.
  Evidence: main/workspace/service.ts WorkspaceResources signatures and model.ts paneOf have implicit preconditions; see P3.
  Finding: [P3](#p3).

- [ ] A docstring restates the signature without explaining the caller contract.
  Result: `no problem found`.
  Evidence: main/workspace/service.ts WorkspaceResources signatures and model.ts paneOf have implicit preconditions; see P3.

- [ ] A docstring example no longer represents current behavior.
  Result: `no problem found`.
  Evidence: main/workspace/service.ts WorkspaceResources signatures and model.ts paneOf have implicit preconditions; see P3.

### Correctness

#### Required behavior or failure handling is incomplete

- [x] Ordinary valid use produces behavior that conflicts with the governing contract.
  Result: `problem found`.
  Evidence: contracts/validation.ts and workspace/controller.ts checks and storage.ts commit-before-publish; RunView secondary destination is P5.
  Finding: [P5](#p5).

- [ ] A materially different valid path is rejected or handled as invalid.
  Result: `no problem found`.
  Evidence: contracts/validation.ts and workspace/controller.ts checks and storage.ts commit-before-publish; RunView secondary destination is P5.

- [ ] Invalid input or state is accepted without the required rejection or containment.
  Result: `no problem found`.
  Evidence: contracts/validation.ts and workspace/controller.ts checks and storage.ts commit-before-publish; RunView secondary destination is P5.

- [ ] A boundary or state transition violates a required invariant.
  Result: `no problem found`.
  Evidence: contracts/validation.ts and workspace/controller.ts checks and storage.ts commit-before-publish; RunView secondary destination is P5.

- [ ] A failure leaves state inconsistent.
  Result: `no problem found`.
  Evidence: contracts/validation.ts and workspace/controller.ts checks and storage.ts commit-before-publish; RunView secondary destination is P5.

- [ ] Failure recovery does not restore the required state.
  Result: `no problem found`.
  Evidence: contracts/validation.ts and workspace/controller.ts checks and storage.ts commit-before-publish; RunView secondary destination is P5.

- [ ] An error is lost or transformed so its consumer cannot respond correctly.
  Result: `no problem found`.
  Evidence: contracts/validation.ts and workspace/controller.ts checks and storage.ts commit-before-publish; RunView secondary destination is P5.

- [ ] A prohibited state or effect can be reached by bypassing the expected path.
  Result: `no problem found`.
  Evidence: contracts/validation.ts and workspace/controller.ts checks and storage.ts commit-before-publish; RunView secondary destination is P5.

- [ ] A retried or repeated operation produces an unintended duplicate or conflicting effect.
  Result: `no problem found`.
  Evidence: contracts/validation.ts and workspace/controller.ts checks and storage.ts commit-before-publish; RunView secondary destination is P5.

- [ ] A multi-step state or data change exposes a partial result where the contract requires atomicity.
  Result: `no problem found`.
  Evidence: contracts/validation.ts and workspace/controller.ts checks and storage.ts commit-before-publish; RunView secondary destination is P5.

- [ ] A resource is acquired, retained, released, or restored outside its required lifetime on a supported terminal path.
  Result: `no problem found`.
  Evidence: contracts/validation.ts and workspace/controller.ts checks and storage.ts commit-before-publish; RunView secondary destination is P5.

- [ ] An operation reports completion before its required effect is complete or durably owned.
  Result: `no problem found`.
  Evidence: contracts/validation.ts and workspace/controller.ts checks and storage.ts commit-before-publish; RunView secondary destination is P5.

- [ ] Required behavior changes with time, ordering, randomness, locale, or environment without a contract for that variation.
  Result: `no problem found`.
  Evidence: contracts/validation.ts and workspace/controller.ts checks and storage.ts commit-before-publish; RunView secondary destination is P5.

#### A change fails to preserve required behavior

- [ ] A refactoring or maintenance change alters required observable behavior.
  Result: `no problem found`.
  Evidence: contracts/validation.ts and workspace/controller.ts checks and storage.ts commit-before-publish; RunView secondary destination is P5.

### Testing

#### Behavior and risk coverage is incomplete

- [ ] A material observable behavior lacks a direct applicable test.
  Result: `no problem found`.
  Evidence: desktop/tests/workspace.test.ts exercises stale/foreign/duplicate/restoration/late loads; contracts/tests/contracts.test.ts validates reference invariants; runtime fixture lacks distinct authored/instance IDs (P1).

- [x] A materially different valid path with distinct behavior or risk lacks a direct test.
  Result: `problem found`.
  Evidence: desktop/tests/workspace.test.ts exercises stale/foreign/duplicate/restoration/late loads; contracts/tests/contracts.test.ts validates reference invariants; runtime fixture lacks distinct authored/instance IDs (P1).
  Finding: [P1](#p1).

- [ ] An invalid or adversarial input, state, or operation with material side-effect risk lacks a direct test.
  Result: `no problem found`.
  Evidence: desktop/tests/workspace.test.ts exercises stale/foreign/duplicate/restoration/late loads; contracts/tests/contracts.test.ts validates reference invariants; runtime fixture lacks distinct authored/instance IDs (P1).

- [ ] A material boundary, state transition, time, ordering, concurrency, cancellation, or timeout behavior lacks a controlled test.
  Result: `no problem found`.
  Evidence: desktop/tests/workspace.test.ts exercises stale/foreign/duplicate/restoration/late loads; contracts/tests/contracts.test.ts validates reference invariants; runtime fixture lacks distinct authored/instance IDs (P1).

- [ ] A relevant failure or recovery path lacks a test.
  Result: `no problem found`.
  Evidence: desktop/tests/workspace.test.ts exercises stale/foreign/duplicate/restoration/late loads; contracts/tests/contracts.test.ts validates reference invariants; runtime fixture lacks distinct authored/instance IDs (P1).

- [ ] A regression-prone behavior lacks a test that distinguishes the prior defect.
  Result: `no problem found`.
  Evidence: desktop/tests/workspace.test.ts exercises stale/foreign/duplicate/restoration/late loads; contracts/tests/contracts.test.ts validates reference invariants; runtime fixture lacks distinct authored/instance IDs (P1).

#### Test code does not provide trustworthy, maintainable checks

- [ ] A test asserts an incidental implementation detail instead of governing behavior.
  Result: `no problem found`.
  Evidence: desktop/tests/workspace.test.ts exercises stale/foreign/duplicate/restoration/late loads; contracts/tests/contracts.test.ts validates reference invariants; runtime fixture lacks distinct authored/instance IDs (P1).

- [ ] A test fixture or test double does not represent the condition or real boundary contract named by the test.
  Result: `no problem found`.
  Evidence: desktop/tests/workspace.test.ts exercises stale/foreign/duplicate/restoration/late loads; contracts/tests/contracts.test.ts validates reference invariants; runtime fixture lacks distinct authored/instance IDs (P1).

- [ ] A test oracle cannot distinguish the expected result from a material wrong result.
  Result: `no problem found`.
  Evidence: desktop/tests/workspace.test.ts exercises stale/foreign/duplicate/restoration/late loads; contracts/tests/contracts.test.ts validates reference invariants; runtime fixture lacks distinct authored/instance IDs (P1).

- [ ] A test result depends on uncontrolled order, shared state, time, randomness, network, process state, or machine state.
  Result: `no problem found`.
  Evidence: desktop/tests/workspace.test.ts exercises stale/foreign/duplicate/restoration/late loads; contracts/tests/contracts.test.ts validates reference invariants; runtime fixture lacks distinct authored/instance IDs (P1).

- [ ] A skipped, disabled, or suppressed test hides a relevant result.
  Result: `no problem found`.
  Evidence: desktop/tests/workspace.test.ts exercises stale/foreign/duplicate/restoration/late loads; contracts/tests/contracts.test.ts validates reference invariants; runtime fixture lacks distinct authored/instance IDs (P1).

- [ ] Test setup obscures the behavior under test.
  Result: `no problem found`.
  Evidence: desktop/tests/workspace.test.ts exercises stale/foreign/duplicate/restoration/late loads; contracts/tests/contracts.test.ts validates reference invariants; runtime fixture lacks distinct authored/instance IDs (P1).

- [ ] A test can pass without executing the behavior it claims to verify.
  Result: `no problem found`.
  Evidence: desktop/tests/workspace.test.ts exercises stale/foreign/duplicate/restoration/late loads; contracts/tests/contracts.test.ts validates reference invariants; runtime fixture lacks distinct authored/instance IDs (P1).

- [ ] A retry, rerun, quarantine, or broad tolerance hides an unresolved intermittent test result.
  Result: `no problem found`.
  Evidence: desktop/tests/workspace.test.ts exercises stale/foreign/duplicate/restoration/late loads; contracts/tests/contracts.test.ts validates reference invariants; runtime fixture lacks distinct authored/instance IDs (P1).

- [ ] A test leaves a resource or shared state changed after a success, failure, skip, timeout, or cancellation path.
  Result: `no problem found`.
  Evidence: desktop/tests/workspace.test.ts exercises stale/foreign/duplicate/restoration/late loads; contracts/tests/contracts.test.ts validates reference invariants; runtime fixture lacks distinct authored/instance IDs (P1).

### Verification

#### Verification evidence cannot support the claimed result

- [ ] Verification ran against a different subject identity than the reviewed subject.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] The exact verification environment is not recorded.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] The exact verification tool identity is not recorded.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] The exact verification command identity is not recorded.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] The exact verification configuration identity is not recorded.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] A claimed verification step was not run.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] Verification output cannot be tied reproducibly to one complete execution.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] The claim extends beyond the paths, inputs, environments, modes, repetitions, or terminal results actually verified.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] A later passing run replaces rather than reconciles a conflicting earlier result.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] A verification result omits a relevant failure, flake, skip, quarantine, unavailable prerequisite, unsupported environment, or evidence limit.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] A cache, stale generated output, suppression, exclusion, or baseline can hide a material result from the recorded run.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] An applicable project-owned check for the affected surface is absent from the recorded verification evidence without a subject reason.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

### Delivery

#### The handed-off result is not the reviewed and verified implementation

- [ ] A delivered artifact changes the reviewed behavior.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] The delivered result cannot be reproduced from its recorded dependencies and configuration.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] The handoff does not identify the exact delivered result.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] The handoff omits an operating condition needed to use the result.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] The handoff omits information needed to recover the result.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] A delivery failure leaves neither the prior result nor the new result safely usable.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] A required artifact, metadata file, schema, generated output, or runtime asset is absent from the delivered result.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] The artifact exercised in verification is not the exact artifact handed to its consumers.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

- [ ] Build, package, release, deployment, live verification, and observed health are reported as one result.
  Result: `evidence missing`.
  Evidence: No fresh benchmark or artifact verification in this review; post-correction execution is required (G2).
  Finding: [G2](#gaps).

### Usability

#### Consumers face avoidable learning or use burden

- [ ] A consumer cannot complete an ordinary task from the public surface and its immediate guidance.
  Result: `no problem found`.
  Evidence: WorkspaceApp, Pane, Discussion and SurfaceView expose explicit loading/error/selection/save states; RunView destination inconsistency is P5.

- [x] Similar tasks require conflicting mental models or interaction patterns.
  Result: `problem found`.
  Evidence: WorkspaceApp, Pane, Discussion and SurfaceView expose explicit loading/error/selection/save states; RunView destination inconsistency is P5.
  Finding: [P5](#p5).

- [ ] A common task requires avoidable interaction or implementation knowledge.
  Result: `no problem found`.
  Evidence: WorkspaceApp, Pane, Discussion and SurfaceView expose explicit loading/error/selection/save states; RunView destination inconsistency is P5.

- [ ] Feedback does not make the task result or current state clear.
  Result: `no problem found`.
  Evidence: WorkspaceApp, Pane, Discussion and SurfaceView expose explicit loading/error/selection/save states; RunView destination inconsistency is P5.

- [ ] Failure information does not support a safe next action or recovery.
  Result: `no problem found`.
  Evidence: WorkspaceApp, Pane, Discussion and SurfaceView expose explicit loading/error/selection/save states; RunView destination inconsistency is P5.

### Operations

#### Runtime behavior is difficult to observe, support, or recover

- [ ] A material failure cannot be distinguished from ordinary behavior with the available diagnostics.
  Result: `no problem found`.
  Evidence: service/client.ts startup/request/stop deadlines; workspace/storage.ts preserves failures and main/index.ts protects unsaved drafts. Live-runtime recovery is G1.

- [ ] Failures that require different responses appear indistinguishable.
  Result: `no problem found`.
  Evidence: service/client.ts startup/request/stop deadlines; workspace/storage.ts preserves failures and main/index.ts protects unsaved drafts. Live-runtime recovery is G1.

- [ ] Partial failure is invisible while the product remains degraded.
  Result: `no problem found`.
  Evidence: service/client.ts startup/request/stop deadlines; workspace/storage.ts preserves failures and main/index.ts protects unsaved drafts. Live-runtime recovery is G1.

- [ ] Operators cannot determine whether recovery restored the required service, state, or data condition.
  Result: `no problem found`.
  Evidence: service/client.ts startup/request/stop deadlines; workspace/storage.ts preserves failures and main/index.ts protects unsaved drafts. Live-runtime recovery is G1.

- [ ] Diagnostic output omits context needed for support.
  Result: `no problem found`.
  Evidence: service/client.ts startup/request/stop deadlines; workspace/storage.ts preserves failures and main/index.ts protects unsaved drafts. Live-runtime recovery is G1.

- [ ] A supported operating configuration has an unclear runtime effect.
  Result: `no problem found`.
  Evidence: service/client.ts startup/request/stop deadlines; workspace/storage.ts preserves failures and main/index.ts protects unsaved drafts. Live-runtime recovery is G1.

- [ ] A retry, fallback, or recovery loop has no attempt, time, or resource bound.
  Result: `no problem found`.
  Evidence: service/client.ts startup/request/stop deadlines; workspace/storage.ts preserves failures and main/index.ts protects unsaved drafts. Live-runtime recovery is G1.

- [ ] A stalled or degraded operation has no bounded detection and safe recovery path.
  Result: `no problem found`.
  Evidence: service/client.ts startup/request/stop deadlines; workspace/storage.ts preserves failures and main/index.ts protects unsaved drafts. Live-runtime recovery is G1.

### Compatibility

#### Change breaks a supported consumer or lifecycle transition

- [ ] A supported runtime, operating system, architecture, environment, or consumer integration stops working after the change.
  Result: `no problem found`.
  Evidence: workspace/storage.ts rejects future schemas without overwriting; current v1 log field is a persisted compatibility constraint, see P1.

- [ ] Established public behavior changes without an explicit compatibility decision.
  Result: `no problem found`.
  Evidence: workspace/storage.ts rejects future schemas without overwriting; current v1 log field is a persisted compatibility constraint, see P1.

- [ ] Public behavior conflicts with a supported version promise.
  Result: `no problem found`.
  Evidence: workspace/storage.ts rejects future schemas without overwriting; current v1 log field is a persisted compatibility constraint, see P1.

- [ ] Supported version combinations cannot operate together as required.
  Result: `no problem found`.
  Evidence: workspace/storage.ts rejects future schemas without overwriting; current v1 log field is a persisted compatibility constraint, see P1.

- [ ] A supported transition between versions breaks consumer behavior or prevents required stored or serialized data from remaining readable and valid.
  Result: `no problem found`.
  Evidence: workspace/storage.ts rejects future schemas without overwriting; current v1 log field is a persisted compatibility constraint, see P1.

- [ ] A supported replacement leaves consumers without a working transition.
  Result: `no problem found`.
  Evidence: workspace/storage.ts rejects future schemas without overwriting; current v1 log field is a persisted compatibility constraint, see P1.

- [ ] A supported exit leaves consumers without continuity or a defined state disposition.
  Result: `no problem found`.
  Evidence: workspace/storage.ts rejects future schemas without overwriting; current v1 log field is a persisted compatibility constraint, see P1.

### Specialist Overlays

#### Security

Owning question: Project contract Renderer isolation/Host to Go service; actual main/security and service.Handler authorization.
Result: `no problem found`.
Evidence: Named sender/payload checks and profile/process isolation inspected; no penetration-test claim.

#### Privacy

Owning question: Project contract Persistence and Multi-agent coordination; Discussion and workspace/storage.
Result: `no problem found`.
Evidence: Local drafts and selections; no provider copies. Account/retention/deletion integration is future G1.

#### Concurrency

Owning question: Project contract session/revision semantics; controller queue; Service.coalescedQuery.
Result: `no problem found`.
Evidence: Serialized writes and observation invalidation are explicit; live runtime shutdown needs G1.

#### Accessibility

Owning question: feature/agent-workspace.md Desktop lifetime/accessibility; Pane/Splitter/TextView.
Result: `evidence missing`.
Evidence: Named roles and keyboard paths inspected. Screen-reader/manual usability qualification is G1. [Gaps](#gaps).

#### Localization

Owning question: User English request; renderer labels and Intl.DateTimeFormat en.
Result: `no problem found`.
Evidence: App-owned labels are English; filenames remain original. No multilingual release claim.

#### Dependencies

Owning question: app/package.json and package-lock.json.
Result: `evidence missing`.
Evidence: Pinned declarations inspected; no fresh vulnerability or supply-chain audit, G2. [Gaps](#gaps).

#### Build

Owning question: app/package.json, electron.vite.config.ts, tsconfig and tests/architecture.test.ts.
Result: `evidence missing`.
Evidence: Process builds and schema generator are explicit; post-correction execution is G2. [Gaps](#gaps).

#### Configuration

Owning question: app/README.md isolated profile and pinned runtime; service/client.ts.
Result: `no problem found`.
Evidence: Private profile and fixed service handshake; no arbitrary renderer command setting.

#### Observability

Owning question: SurfaceView errors; main/index.ts save failures; service/client.ts timeouts.
Result: `evidence missing`.
Evidence: User-visible failures inspected; support telemetry/full live diagnosis is G1. [Gaps](#gaps).

#### Migration

Owning question: workspace/storage.ts unknown-version preservation; current v1 contracts.
Result: `no problem found`.
Evidence: Existing user profiles require wire preservation; no automatic reset. Future field rename needs explicit versioned migration.

## Problems

### P1

**Log resource confuses authored task and runtime instance**

- Primary category: Vocabulary. Found during: `critical review`.
- Expectation: the user requests precise concepts and maintainable API/ownership before agents.
- Observation and impact: The public log target says taskId but requires an instance identity. Scatter makes these different. Introduce instanceId-oriented construction/access APIs and typed Run rows; preserve existing v1 JSON field explicitly until a versioned migration.
- Evidence: app/contracts/src/project.ts ResourceRefSchema; renderer/workspace/views/RunView.tsx View logs; monitor/snapshot.go Task.Identity/TaskID and Edge.
- Cause: domain/presentation vocabulary and ownership were implicit in the first UI slice.
- Uncertainty: Pipeline authoring/agent behavior remains future scope; these findings do not claim live engine failures.
- Related effects: corresponding item links above. Responsible owner: desktop contracts/application maintainer.

### P2

**Engine metadata interpretation leaks into React**

- Primary category: Abstraction. Found during: `critical review`.
- Expectation: the user requests precise concepts and maintainable API/ownership before agents.
- Observation and impact: Provide an explicit typed Run presentation model and one host-side projection from the native snapshot. Include pipelineName as observed metadata; never infer a Pipeline registration or source identity from that label.
- Evidence: app/contracts/src/service.ts RunSnapshotSchema; renderer/workspace/views/RunView.tsx record/text/edges.
- Cause: domain/presentation vocabulary and ownership were implicit in the first UI slice.
- Uncertainty: Pipeline authoring/agent behavior remains future scope; these findings do not claim live engine failures.
- Related effects: corresponding item links above. Responsible owner: desktop contracts/application maintainer.

### P3

**Pane lookup silently invents a location for an absent view**

- Primary category: Public API. Found during: `critical review`.
- Expectation: the user requests precise concepts and maintainable API/ownership before agents.
- Observation and impact: Return null for absence or throw an explicit not-found error. Export Pane/Layout contracts and document lookup semantics; callers must not silently receive secondary.
- Evidence: app/desktop/src/main/workspace/model.ts paneOf.
- Cause: domain/presentation vocabulary and ownership were implicit in the first UI slice.
- Uncertainty: Pipeline authoring/agent behavior remains future scope; these findings do not claim live engine failures.
- Related effects: corresponding item links above. Responsible owner: desktop contracts/application maintainer.

### P4

**App and engine concepts lack one implementation glossary**

- Primary category: Vocabulary. Found during: `critical review`.
- Expectation: the user requests precise concepts and maintainable API/ownership before agents.
- Observation and impact: Define Project, app Workspace, execution workspace, Pipeline definition, Plan, Run registration, Task instance, Resource, Surface, Pane and Window; name current and deferred metadata owners.
- Evidence: internal/appservice/types.go Run; contracts/project.ts and workspace.ts; architecture/project-workspace-contract.md.
- Cause: domain/presentation vocabulary and ownership were implicit in the first UI slice.
- Uncertainty: Pipeline authoring/agent behavior remains future scope; these findings do not claim live engine failures.
- Related effects: corresponding item links above. Responsible owner: desktop contracts/application maintainer.

### P5

**Run content chooses workspace placement**

- Primary category: Architecture. Found during: `critical review`.
- Expectation: the user requests precise concepts and maintainable API/ownership before agents.
- Observation and impact: A Run already in secondary opens logs in its own pane. A leaf content component should emit onOpenLogs with a semantic instance target; its enclosing Pane chooses the other pane.
- Evidence: renderer/workspace/views/RunView.tsx View logs hard-codes secondary.
- Cause: domain/presentation vocabulary and ownership were implicit in the first UI slice.
- Uncertainty: Pipeline authoring/agent behavior remains future scope; these findings do not claim live engine failures.
- Related effects: corresponding item links above. Responsible owner: desktop contracts/application maintainer.

## Improvements

### I1

**Separate contract files by change reason**

- Primary category: Modularization. Found during: `critical review`.
- Acceptable current condition: existing process boundaries and tests work; source size alone is not a defect.
- Evidence: contracts/src/project.ts, service.ts, workspace-session.ts.
- Supported benefit and suggestion: Use project/agent/resource/file/run schemas, persisted workspace document and bridge definitions with one public barrel. Keep v1 definitions and JSON shape stable.
- Cost: coordinated imports and regression checks; avoid one-class-per-file or speculative plugin abstractions.
- Uncertainty: no performance benefit claimed. Decision owner: user-authorized implementation maintainer.

### I2

**Separate ephemeral rendering lifetime**

- Primary category: Modularization. Found during: `critical review`.
- Acceptable current condition: existing process boundaries and tests work; source size alone is not a defect.
- Evidence: main/workspace/controller.ts session/leases/load generation.
- Supported benefit and suggestion: Extract a render-session collaborator; leave serialization and durable document commit in the controller. Do not introduce a general event bus or DI framework.
- Cost: coordinated imports and regression checks; avoid one-class-per-file or speculative plugin abstractions.
- Uncertainty: no performance benefit claimed. Decision owner: user-authorized implementation maintainer.

### I3

**Give current service callers one typed gateway**

- Primary category: Modularization. Found during: `critical review`.
- Acceptable current condition: existing process boundaries and tests work; source size alone is not a defect.
- Evidence: main/service/ipc.ts and main/workspace/service.ts.
- Supported benefit and suggestion: Keep transport/lifetime in ProjectServiceClient; centralize named routes, parsing and association in ProjectService. Reuse resource identity helpers.
- Cost: coordinated imports and regression checks; avoid one-class-per-file or speculative plugin abstractions.
- Uncertainty: no performance benefit claimed. Decision owner: user-authorized implementation maintainer.

### I4

**Separate navigation composition from resource browsers**

- Primary category: Modularization. Found during: `critical review`.
- Acceptable current condition: existing process boundaries and tests work; source size alone is not a defect.
- Evidence: renderer/workspace/Sidebar.tsx Files/Runs.
- Supported benefit and suggestion: Extract FilesBrowser and RunsBrowser with narrow callbacks. Keep workspace orchestration in WorkspaceApp/Pane, and data rendering in views.
- Cost: coordinated imports and regression checks; avoid one-class-per-file or speculative plugin abstractions.
- Uncertainty: no performance benefit claimed. Decision owner: user-authorized implementation maintainer.

## Strengths

### Explicit authoritative boundaries

Primary category: Architecture. Found during: `critical review`. Pure transition, single profile writer, private native service and scoped runtime adapter are directly visible in model.ts, storage.ts, service/client.ts and internal/appservice/runtime.go. Preserve no renderer Node/engine imports, request/revision checks, and read-only engine query policy. Static structure is verified; live deployment remains G1.

## Gaps

| Gap | Found during | Effect | Needed evidence / owner |
|---|---|---|---|
| G1 | critical review | No independent review, representative usability/screen reader, live Docker/provider or installed-platform qualification | Stage 4–7 owners must provide the corresponding real environment evidence; this pass only changes current design/code |
| G2 | base checklist | Fresh performance, supply-chain, complete artifact verification are not established by static inspection | Implementation owner runs relevant unit/Electron/Go checks after correction; no new performance/audit claims without measurement |

## Handoff

- Rechecked bound identity: `e31b593c3798ebe981ae629862a65c0c086b917bcab68cee7db93b5b34d45f12`, unchanged before handoff.
- Record state: partial because specialist/runtime reach is explicitly limited; all base categories and overlays are accounted for.
- Caller owns this report. Next owner: the same maintainer acting in the separately user-authorized correction phase.
- First action: present the domain/ownership sketch, document the definitions, then correct P1–P5 and selected I1–I4 with regression tests.
- This report grants no target mutation, approval, merge, publication or release authority; the user’s current review-and-fix instruction supplies correction authority.
- Any subsequent source edit makes this report historical for the bound identity. Correction outcomes belong in a separate stage record, not rewritten finding evidence.
