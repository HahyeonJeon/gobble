# Pipeline collaboration — Architecture code review

> **Record state:** partial · **Relationship:** author, disclosed self-review.<br>
> **Reviewed:** 2026-09-09. Prepared review material: an independent evaluator should
> inspect the same subject unaided before reading this report.<br>
> **Authority:** review only; no implementation, approval, merge or release authority.

## Summary

The review found two concrete capability gaps against the owner's **new pipeline
collaboration direction**: the App cannot represent the pre-run source/change/Plan
relationship (P1), and its Agent has no authoring capability (P2). These are not
regressions of the accepted read-only first slice. Execution admission/recovery
needs a new qualified capability (G2), rather than a few new buttons.

Two optional structural improvements are justified before growth: separate
presentation-session lifetime from the durable workspace writer (I1), and make
current contract ownership easier to navigate while preserving frozen formats (I2).
Existing engine/service separation, provider ports and exact-reference transactions
are strengths to retain.

A native root Go test command failed to compile. Further inspection established
that README targets Linux containers and Linux/amd64 cross-compilation succeeds.
This is the target-specific qualification limit G4, not a demonstrated supported
runtime regression. The native App service and 32 focused App tests passed.

## Subject and Scope

| Field | Value |
|---|---|
| Caller / consumer | Project owner requesting architecture review before the next implementation stage. |
| Review questions | Concepts, directory/file ownership, abstraction/pattern boundaries, public APIs and the missing design → change → Plan → Run → monitor workflow. |
| Included | Traces across root Go APIs, CLI driver, engine plan/reuse/Stop; Go service capabilities/runtime/catalog; TS contracts, Main workspace/provider/shared tools and renderer navigation. |
| Excluded | Exhaustive line review of all files, new production effects, dataset/scientific validity, live provider authoring, deep UI/accessibility/scale audit and packaging qualification. |
| Governing sources | Latest owner request; root AGENTS.md; bound design records, especially project-workspace-contract, compose-pipeline and validate-plan; README runtime contract. |
| Subject kind | Working-tree file set, 523 files in [subject.json](subject.json), including prior uncommitted work. |
| Exact revision | SHA256 `ec2b38e675f9b6e535839b8fd9a436f7402f6a71ab970ebfff19f9f9ee6d1821`; HEAD `7fe3d5c9e10d0cc104a15b6e7aa3def0d15b98f1` is context only. |
| Added dependency reach | [dependency-evidence.json](dependency-evidence.json) binds containerenv after a test exposed a dependency outside the initial file set. It does not rewrite the original subject or unaided record. |
| Read-only boundary | Production/governing source remains unchanged. This operation writes this report; study/design/plan/doc navigation are separate authorized roles. |
| Invalidation | Any bound source/governing-byte change makes its related claims historical; rebind before reviewing a fix. |

## Method

| Item | Detail |
|---|---|
| Actual code first | Inspected model and callers, runtime/transport/store lifetimes, representative validators/tests and current Project/Run navigation before the checklist. |
| Affected surfaces | User/Agent discussion, source authoring, Plan identity, execution ownership, stored references, pinned provider/runtime and standalone CLI consumers. |
| Locked identity | Original source manifest plus the unchanged unaided record copied below. |
| Checklist | [Canonical source](/Users/hahyeon/.codex/plugins/cache/gobbi-workspace/gobbi/1.2.4/skills/code-review/checklist.md), SHA256 `89aaf24bad0d5f0deddba8df22103f7380c6aa348effe3468c5573cbc43fce07`. |
| Commands / environment | [Verification](verification.md) lists exact focused commands, outputs, platform and limitations. Source inspection used file reads and searches. |
| Sampling | Architectural call/data/lifetime traces; binding 523 files is not claiming 523-file exhaustive inspection. |
| Uninspected reach | Complete implementation bodies, broad adversarial behavior, migration matrix, accessibility, scale, packaging and new effect integration. |
| Uncertainty | Pinned native authoring protocol; executable admission/recovery; source-application coexistence with external editors; supported runtime execution. |

### Checklist-free critical review

The following text is preserved verbatim from the pre-checklist record. Leads P1,
P2 and P3 were provisional identifiers. Later reconciliation classifies them; the
native build finding was discovered later and is not backfilled here.

<details>
<summary>Locked unaided record</summary>

```text
## Locked checklist-free critical review

Completed before loading any base checklist or previous report for this subject.
Direct evidence scope: root Pipeline/Graph/BuildPlan/Run/Resume/Stop API; CLI driver;
engine plan/reuse/Stop seams; Go service routes/runtime/catalog; TS Project/resource/
Run/Workspace contracts; Codex policy/provider binding; shared tools; workspace
controller/resource adapter; representative tests and Project/Run navigation.

1. **Design, intent and best supported version.** Public Pipeline is a mutable Go
   builder, Graph is composed data and Plan is validated engine output. App Project
   resources expose files, attached Runs and attempt logs. There is no App identity
   for a pipeline definition, candidate source revision, change set or reviewed Plan.
   Existing Run dependency presentation intentionally reads a monitor snapshot. It
   cannot serve as a pre-run Plan by relabeling. Lead P1: missing pre-run domain seam.
2. **Failure, misuse and cosmetic compliance.** Agent bind/start uses an isolated
   discussion cwd and read-only/no-network policy; shell is disabled. Shared tools
   can observe and point but cannot author code. Merely changing role instructions
   cannot satisfy the revised purpose. Lead P2: missing authoring port/profile.
   CLI plan/validate compiles and invokes userpipe.Pipeline(), so opening Plan support
   must account for arbitrary project code; “no analysis tasks” is not “no code runs”.
3. **State, effects and resources.** Service only registers Projects/attaches Runs
   and performs bounded reads. Existing request receipts are registration receipts,
   not execution intents. Public Stop chooses the current lease when invoked; it
   cannot accept a caller's previously observed lease. A durable async execution
   bridge needs correlation and engine-side fencing, not just POST endpoints. Lead
   P3: effect command ownership gap. Preserve current no-replay turn recovery.
4. **Change, integration and compatibility.** WorkspaceController serializes one
   stored document and implements collaboration/evidence/question ports. This is
   valuable atomicity, but presentation sessions and view-specific resource methods
   are coupled into the same integration surface. Improvement I1: extract a focused
   presentation/resource seam before growing pipeline collaboration, while retaining
   one workspace writer and separate service-owned workflow records. Frozen version
   files are compatibility assets, not clutter to delete. Improvement I2: organize
   active contract access without moving or rewriting historical schemas casually.
5. **Absences across the lifecycle.** No real edit → diff → validated Plan → exact
   user authorization → Start → monitored result journey is implemented. Existing
   tests explicitly refuse start routes and pin read-only Agent behavior. This is a
   new product capability gap, not evidence that prior read-only tests are wrong.
   Native authoring isolation and precise plan-to-execution binding remain unproven.
   Gap G1: production authoring transport; G2: launch/recovery admission contract;
   G3: full architecture/security/accessibility/performance audit beyond traced seams.

Strengths: engine/service/provider independence, narrowed renderer bridge, opaque
resource identity, frozen formats, bounded projections, versioned observation
receipts, immutable explicit evidence, protected User view state, and durable
uncertain-turn reconciliation. No bug was established in their existing behavior.

Contrary evidence: governing project-workspace-contract.md already anticipates
change sets, a durable operation journal, launch correlation and stale-lease Stop.
The correction is to realize and refine those seams, not invent a second engine or
replace the working reference subsystem. Public Stop is correctly lease-addressed
once it starts; the missing capability is expected-lease admission from an earlier
App observation. A wider native Agent workspace is an alternative, but its actual
file-read and command scope has not been qualified on the pinned provider.

Unaided record SHA256: `28635a4cc10e891ab13336ff8b2a5c80ccd0d58dc560c3c1d7782425f6cf5fdc`.

```

</details>

### Core category applicability

| Category | Result | Exact reach / reason | Affected signs | Gap |
|---|---|---|---|---|
| Project Fit | applicable | Owner direction versus read-only session-plan and service capability list. | Supported subset below; others explicitly missing | G3 |
| Affected Surfaces | applicable | Go API → CLI → service and provider → Main → shared references. | Supported subset below; others explicitly missing | G3 |
| Project Structure | applicable | Actual main/workspace, service, contracts and renderer view locations. | Supported subset below; others explicitly missing | G3 |
| Architecture | applicable | WorkspaceController:79, service.go:54, boundaries_test.go:12, engine API imports. | Supported subset below; others explicitly missing | G3 |
| Design Pattern | applicable | ConversationProvider port, ServiceResources adapter and serialized workspace writer. | Supported subset below; others explicitly missing | G3 |
| Abstraction | applicable | WorkspaceResources:18 optional format/session methods; distinct Go Graph and Plan. | Supported subset below; others explicitly missing | G3 |
| Data Model | applicable | resource.ts file/run/log union; run.ts registration; no pipeline/change/Plan identity. | Supported subset below; others explicitly missing | G3 |
| Public API | applicable | BuildPlan/Run/Resume/Stop; service capabilities and intentional unsupported mutations. | Supported subset below; others explicitly missing | G3 |
| Parameters | applicable | Service decoders and exact request/Project/turn binding in traced callers. | Supported subset below; others explicitly missing | G3 |
| Modularization | applicable | Workspace controller, resource adapters and provider boundary imports. | Supported subset below; others explicitly missing | G3 |
| Reusability | applicable | Gobble reuse classifier, shared User/Agent reference operations. | Supported subset below; others explicitly missing | G3 |
| Performance | evidence missing | No representative performance measurements collected. | Supported subset below; others explicitly missing | G3 |
| Optimization | evidence missing | No optimization subject/workload qualified. | Supported subset below; others explicitly missing | G3 |
| Unintended Overengineering | applicable | Frozen formats have real stored consumers; new workflow not yet implemented. | Supported subset below; others explicitly missing | G3 |
| Code Complexity | applicable | Controller enqueue/load ticket flow and provider reconciliation sampled. | Supported subset below; others explicitly missing | G3 |
| Readability | applicable | CLI driver operation ordering and public API comments sampled. | Supported subset below; others explicitly missing | G3 |
| Vocabulary | applicable | Go Pipeline/Graph/Plan, Run presentation label and historical instance field. | Supported subset below; others explicitly missing | G3 |
| Naming Convention | applicable | Current exported contracts and provider port sampled; no full convention audit. | Supported subset below; others explicitly missing | G3 |
| Docstring | applicable | BuildPlan/Run/Stop and resource adapter contracts sampled. | Supported subset below; others explicitly missing | G3 |
| Correctness | applicable | Current service negative cases and provider tests; no new effect path exists. | Supported subset below; others explicitly missing | G3 |
| Testing | applicable | 32 focused App tests, fresh service suite and explicit native-root compile failure. | Supported subset below; others explicitly missing | G3 |
| Verification | applicable | Exact environment/commands and separate cached, fresh, compile-only outcomes. | Supported subset below; others explicitly missing | G3 |
| Delivery | applicable | No artifact built or delivered; current source/reference compatibility sampled. | Supported subset below; others explicitly missing | G3 |
| Usability | applicable | Project/Run navigation lacks source-change/Plan collaboration journey. | Supported subset below; others explicitly missing | G3 |
| Operations | applicable | Uncertain-turn reconciliation; future execution recovery absent. | Supported subset below; others explicitly missing | G3 |
| Compatibility | applicable | Frozen formats and pinned runtime/provider contracts sampled. | Supported subset below; others explicitly missing | G3 |

### Overlay applicability

| Overlay | Result | Activation evidence / source qualification | Uninspected reach | Gap |
|---|---|---|---|---|
| Security | activated | Existing IPC sender checks; official Electron sender-validation guidance. | New authoring/job isolation and full adversarial audit. | G1, G3 |
| Privacy | evidence missing | Model context and project source are material; existing isolated policy read, no complete disclosure analysis. | All future write/profile data exposure. | G1, G3 |
| Concurrency | activated | Governing single-writer/no-replay/lease contract and traced coordinator, workspace queue, Stop. | New multi-file application and launch races. | G2, G3 |
| Accessibility | evidence missing | Operating React desktop UI; no native accessibility exercise this turn. | Focus, screen reader and new views. | G3 |
| Localization | evidence missing | English UI requirement; no full string/locale audit. | All new controls and locale-sensitive ranges. | G3 |
| Dependencies | activated | Pinned Codex generator and package lock; containerenv dependency discovered by compilation. | Native authoring protocol/runtime compatibility. | G1, G4 |
| Build | activated | README's Linux/amd64 container contract and Go target rules; native and cross-compile evidence. | Actual Linux test execution. | G4 |
| Packaging | not applicable | This review delivers documents and preserves source; no package candidate or packaging behavior is being changed. | Historical stage-7 work remains deferred. | None |
| Release | not applicable | No release candidate, publication or promotion in this subject decision. | Release qualification remains outside this report. | None |
| Deployment | not applicable | No deployment target or change in this review. | None within this review. | None |
| Configuration | activated | Existing pinned runtime/image/daemon policy and read-only provider profile are load-bearing. | Validation-job and effect-controller profiles. | G1, G2 |
| Observability | activated | Durable submission/uncertain reconciliation in coordinator and engine-owned monitor. | Future operation recovery/diagnostics. | G2 |
| Migration | activated | Frozen schemas and AtomicStateFile migration/backup owner. | Full historic format matrix and future workflow persistence. | G3 |
| Deprecation | not applicable | No API or stored format is removed/deprecated by this review. | None within this change. | None |
| Retirement | not applicable | No product/service/data retirement action. | None within this change. | None |

### Reconciliation

| Finding or lead | Origin | Checklist relation | Resolution |
|---|---|---|---|
| P1 pre-run identity | both | Data Model required state/relationship and identity. | Retained as capability gap against latest direction; no prior-slice regression claim. |
| P2 Agent authoring | both | Usability ordinary task completion. | Retained as absent capability, not a defective read-only sandbox. |
| P3 execution ownership lead | critical review | Public API/Operations reach; no implemented effect endpoint to judge. | Classified as G2 capability/evidence gap; existing contract already assigns owners. |
| I1 presentation boundary | critical review | Architecture/Modularization. | Optional focused extraction; no unsupported claim that queue atomicity is broken. |
| I2 contract organization | critical review | Structure/Compatibility. | Optional explicit current navigation; preserve historical formats. |
| Native compile failure | specialist overlay | Build target qualification. | G4 after README/HEAD/cross-compile reinspection; not a Linux runtime regression. |
| Broad item coverage | base checklist | Sampled categories/overlays. | G3 narrows the review; no all-green inference from uninspected code. |

## Checklist Review

Exact negative signs are copied in source order. Checked means a supported problem
against the stated scope. Missing evidence is not a failure and not a pass.

<details>
<summary>Base checklist with bounded per-item evidence</summary>


### Project Lifecycle

#### Project Fit

##### The code work lacks accepted project direction or control

- [ ] The code work has no current purpose or intended result.
  Result: `no problem found` · Evidence: README pipeline purpose and latest owner direction provide a concrete result.

- [ ] The accepted scope of the code work is unclear.
  Result: `no problem found` · Evidence: Current session-plan explicitly excludes effects; latest request authorizes study/design first.

- [ ] The code work has no observable completion condition.
  Result: `evidence missing` · Evidence: Owner direction versus read-only session-plan and service capability list. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] The implemented behavior conflicts with the accepted project purpose.
  Result: `evidence missing` · Evidence: Owner direction versus read-only session-plan and service capability list. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] The implemented behavior satisfies the form of an accepted decision while missing its intended result.
  Result: `evidence missing` · Evidence: Owner direction versus read-only session-plan and service capability list. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] The code conflicts with an applicable project rule, decision, or governance constraint.
  Result: `no problem found` · Evidence: Sampled read-only service/provider behavior matches the accepted first-slice contract.

- [ ] An accepted project baseline changes without a traceable change-control decision.
  Result: `no problem found` · Evidence: This review preserves source; latest owner direction is recorded as a proposal before implementation.

- [ ] A project result remains presented as current after it becomes temporary, superseded, or archived.
  Result: `evidence missing` · Evidence: Owner direction versus read-only session-plan and service capability list. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Affected Surfaces

##### The change leaves affected work unknown or inconsistent

- [ ] A material affected surface cannot be identified from the change and its governing sources.
  Result: `evidence missing` · Evidence: Go API → CLI → service and provider → Main → shared references. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Code and one of its callers express different versions of the same contract.
  Result: `evidence missing` · Evidence: Go API → CLI → service and provider → Main → shared references. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A material consistency-bound surface remains stale after the change.
  Result: `evidence missing` · Evidence: Go API → CLI → service and provider → Main → shared references. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Two implementations of the same project rule remain inconsistent after the change.
  Result: `evidence missing` · Evidence: Go API → CLI → service and provider → Main → shared references. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

##### Affected work cannot be coordinated

- [ ] Coordination ownership or handoff is unclear for a material affected surface.
  Result: `evidence missing` · Evidence: Go API → CLI → service and provider → Main → shared references. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Affected owners have conflicting coordination expectations.
  Result: `evidence missing` · Evidence: Go API → CLI → service and provider → Main → shared references. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

### Design and Development Lifecycle

#### Project Structure

##### Project layout is incoherent or misleading

- [ ] Code placement conflicts with the project's accepted structural convention.
  Result: `evidence missing` · Evidence: Actual main/workspace, service, contracts and renderer view locations. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] One file combines code with independent ownership or change reasons.
  Result: `evidence missing` · Evidence: Actual main/workspace, service, contracts and renderer view locations. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A coherent unit is fragmented across files without a project reason.
  Result: `evidence missing` · Evidence: Actual main/workspace, service, contracts and renderer view locations. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Code ordering inside a file conflicts with the project's accepted structure.
  Result: `evidence missing` · Evidence: Actual main/workspace, service, contracts and renderer view locations. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Code visibility does not match its structural owner.
  Result: `evidence missing` · Evidence: Actual main/workspace, service, contracts and renderer view locations. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] The source, generator, inputs, output location, or regeneration owner is unclear for generated or derived code.
  Result: `no problem found` · Evidence: app/scripts/generate-codex-protocol.ts and narrowed generated.ts identify pinned regeneration for inspected provider surface.

##### Refactoring leaves the project structure worse

- [ ] A refactoring leaves duplicated or obsolete code locations.
  Result: `evidence missing` · Evidence: Actual main/workspace, service, contracts and renderer view locations. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A refactoring fragments one structural owner across misleading locations.
  Result: `evidence missing` · Evidence: Actual main/workspace, service, contracts and renderer view locations. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A refactoring leaves the declared structure inconsistent with the actual ownership boundary.
  Result: `evidence missing` · Evidence: Actual main/workspace, service, contracts and renderer view locations. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Architecture

##### Responsibilities, dependencies, or ownership sit in the wrong boundaries

- [ ] A system unit owns materially unrelated responsibilities.
  Result: `evidence missing` · Evidence: WorkspaceController:79, service.go:54, boundaries_test.go:12, engine API imports. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] One responsibility is divided across units without a clear owner.
  Result: `evidence missing` · Evidence: WorkspaceController:79, service.go:54, boundaries_test.go:12, engine API imports. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A dependency points against the accepted responsibility direction.
  Result: `no problem found` · Evidence: internal/appservice/boundaries_test.go forbids engine/third-party imports; fresh service tests passed.

- [ ] State or data ownership is unclear across system boundaries.
  Result: `evidence missing` · Evidence: WorkspaceController:79, service.go:54, boundaries_test.go:12, engine API imports. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Resource lifetime ownership is unclear across system boundaries.
  Result: `evidence missing` · Evidence: WorkspaceController:79, service.go:54, boundaries_test.go:12, engine API imports. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Failure ownership is unclear across system boundaries.
  Result: `evidence missing` · Evidence: WorkspaceController:79, service.go:54, boundaries_test.go:12, engine API imports. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A failure in one concern spreads into an unrelated concern because their boundaries are coupled.
  Result: `evidence missing` · Evidence: WorkspaceController:79, service.go:54, boundaries_test.go:12, engine API imports. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Responsibilities that must succeed or fail as one consistency unit are split across boundaries without one transaction or recovery owner.
  Result: `evidence missing` · Evidence: WorkspaceController:79, service.go:54, boundaries_test.go:12, engine API imports. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Design Pattern

##### A design pattern does not fit the current problem

- [ ] A recognizable design pattern conflicts with the accepted design.
  Result: `no problem found` · Evidence: ConversationProvider is a narrow port and ServiceResources is an adapter; no mismatch in traced call sites.

- [ ] A design pattern hides control, data, state, or failure flow that its consumers must understand.
  Result: `evidence missing` · Evidence: ConversationProvider port, ServiceResources adapter and serialized workspace writer. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Participants in one design pattern follow conflicting roles or lifecycle rules.
  Result: `evidence missing` · Evidence: ConversationProvider port, ServiceResources adapter and serialized workspace writer. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Abstraction

##### A concept boundary hides or exposes the wrong details

- [ ] An abstraction leaks a private implementation detail into its consumer contract.
  Result: `evidence missing` · Evidence: WorkspaceResources:18 optional format/session methods; distinct Go Graph and Plan. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] One abstraction combines concepts with independent meanings or change reasons.
  Result: `evidence missing` · Evidence: WorkspaceResources:18 optional format/session methods; distinct Go Graph and Plan. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Consumer-specific branching makes one abstraction serve conflicting concepts.
  Result: `evidence missing` · Evidence: WorkspaceResources:18 optional format/session methods; distinct Go Graph and Plan. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] An abstraction hides an invariant, effect, state, or failure that consumers must reason about.
  Result: `evidence missing` · Evidence: WorkspaceResources:18 optional format/session methods; distinct Go Graph and Plan. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] The abstraction level forces current consumers to work above or below the concept boundary they need.
  Result: `evidence missing` · Evidence: WorkspaceResources:18 optional format/session methods; distinct Go Graph and Plan. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Data Model

##### The data model cannot preserve required domain meaning and invariants

- [x] The data model cannot represent a required domain state or relationship.
  Result: `problem found` · Evidence: resource.ts/run.ts lack pipeline definition, source change and Plan artifact relationships required by the new direction.
  Finding: [P1](#p1-pre-run-pipeline-identity-is-absent)

- [ ] The data model represents a state or relationship that the domain forbids.
  Result: `evidence missing` · Evidence: resource.ts file/run/log union; run.ts registration; no pipeline/change/Plan identity. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [x] Identity is missing, unstable, or ambiguous where behavior or references depend on it.
  Result: `problem found` · Evidence: No App identity exists for a candidate or reviewed Plan; current runtime Run labels are not that identity.
  Finding: [P1](#p1-pre-run-pipeline-identity-is-absent)

- [ ] One fact has multiple independently writable representations that can disagree.
  Result: `evidence missing` · Evidence: resource.ts file/run/log union; run.ts registration; no pipeline/change/Plan identity. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] The data model collapses distinct absence, default, unknown, and invalid states that behavior must distinguish.
  Result: `evidence missing` · Evidence: resource.ts file/run/log union; run.ts registration; no pipeline/change/Plan identity. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Public API

##### The public entry surface makes correct use hard

- [ ] Public entry points do not make the ordinary use path clear.
  Result: `evidence missing` · Evidence: BuildPlan/Run/Resume/Stop; service capabilities and intentional unsupported mutations. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A public operation does not distinguish completion states that require different consumer responses.
  Result: `evidence missing` · Evidence: BuildPlan/Run/Resume/Stop; service capabilities and intentional unsupported mutations. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Required public operation ordering is unclear.
  Result: `evidence missing` · Evidence: BuildPlan/Run/Resume/Stop; service capabilities and intentional unsupported mutations. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Ownership, mutation, lifetime, cleanup, or retry obligations for public data, resources, and effects are hidden from the consumer.
  Result: `evidence missing` · Evidence: BuildPlan/Run/Resume/Stop; service capabilities and intentional unsupported mutations. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Public failure or recovery behavior is hidden from its consumer.
  Result: `evidence missing` · Evidence: BuildPlan/Run/Resume/Stop; service capabilities and intentional unsupported mutations. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Overlapping public entry points leave the intended choice unclear.
  Result: `evidence missing` · Evidence: BuildPlan/Run/Resume/Stop; service capabilities and intentional unsupported mutations. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A caller must inspect private implementation details to use the public surface correctly.
  Result: `evidence missing` · Evidence: BuildPlan/Run/Resume/Stop; service capabilities and intentional unsupported mutations. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Parameters

##### Parameters make valid calls hard to express correctly

- [ ] A parameter makes the caller supply a decision that the implementation already owns.
  Result: `evidence missing` · Evidence: Service decoders and exact request/Project/turn binding in traced callers. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Parameter order makes distinct values easy to exchange accidentally.
  Result: `evidence missing` · Evidence: Service decoders and exact request/Project/turn binding in traced callers. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A parameter name does not identify the decision or value the caller supplies.
  Result: `evidence missing` · Evidence: Service decoders and exact request/Project/turn binding in traced callers. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Related parameter values are grouped in a way that permits contradictory combinations.
  Result: `evidence missing` · Evidence: Service decoders and exact request/Project/turn binding in traced callers. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A parameter type or representation admits states outside the accepted input model.
  Result: `evidence missing` · Evidence: Service decoders and exact request/Project/turn binding in traced callers. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Parameter validation occurs outside the boundary that owns the input contract.
  Result: `no problem found` · Evidence: service.go decodeRequest validates incoming requests; http_test.go negative cases ran in fresh service suite.

- [ ] A parameter default or omission changes behavior in a way the caller cannot predict.
  Result: `evidence missing` · Evidence: Service decoders and exact request/Project/turn binding in traced callers. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A mode parameter combines behaviors with different contracts in one call surface.
  Result: `evidence missing` · Evidence: Service decoders and exact request/Project/turn binding in traced callers. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A parameter does not identify the unit, encoding, normalization, or reference frame needed to interpret its value.
  Result: `evidence missing` · Evidence: Service decoders and exact request/Project/turn binding in traced callers. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A parameter representation collapses absent, default, unknown, and invalid caller states that the operation must distinguish.
  Result: `evidence missing` · Evidence: Service decoders and exact request/Project/turn binding in traced callers. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Modularization

##### Unit boundaries do not match cohesive ownership and change

- [ ] Code that changes together is split across units that require repeated coordinated edits.
  Result: `evidence missing` · Evidence: Workspace controller, resource adapters and provider boundary imports. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Independently changing concerns share one unit and force unrelated edits.
  Result: `evidence missing` · Evidence: Workspace controller, resource adapters and provider boundary imports. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A unit exposes more surface than its consumers need.
  Result: `evidence missing` · Evidence: Workspace controller, resource adapters and provider boundary imports. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A unit cannot be understood or tested without unrelated parts of the system.
  Result: `evidence missing` · Evidence: Workspace controller, resource adapters and provider boundary imports. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A dependency cycle makes unit ownership or change order unclear.
  Result: `evidence missing` · Evidence: Workspace controller, resource adapters and provider boundary imports. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Reusability

##### Shared behavior has no stable, proportionate owner

- [ ] Current consumers duplicate one behavior or domain rule in implementations that must agree but can drift independently.
  Result: `evidence missing` · Evidence: Gobble reuse classifier, shared User/Agent reference operations. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Shared code branches by consumer because it combines different responsibilities.
  Result: `evidence missing` · Evidence: Gobble reuse classifier, shared User/Agent reference operations. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Reuse forces independent consumers to coordinate unrelated changes.
  Result: `evidence missing` · Evidence: Gobble reuse classifier, shared User/Agent reference operations. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Performance

##### Observable resource behavior fails a current need

- [ ] Measured latency exceeds a current representative need.
  Result: `evidence missing` · Evidence: No representative performance measurements collected. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Measured processing capacity falls below a current representative need.
  Result: `evidence missing` · Evidence: No representative performance measurements collected. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Measured resource use violates a current budget or supported operating range.
  Result: `evidence missing` · Evidence: No representative performance measurements collected. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A change regresses a representative performance baseline.
  Result: `evidence missing` · Evidence: No representative performance measurements collected. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Optimization

##### An attempted optimization is not justified by its observed effect

- [ ] The optimization uses a measurement frame that does not represent the current need.
  Result: `evidence missing` · Evidence: No optimization subject/workload qualified. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] The optimization has no demonstrated benefit on its target workload.
  Result: `evidence missing` · Evidence: No optimization subject/workload qualified. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] The optimization shifts material cost outside its target workload.
  Result: `evidence missing` · Evidence: No optimization subject/workload qualified. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] The optimization's measured benefit is disproportionate to its added non-performance cost.
  Result: `evidence missing` · Evidence: No optimization subject/workload qualified. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Unintended Overengineering

##### A mechanism has no support from current requirements or observed need

- [ ] A structural boundary has no current requirement or current consumer.
  Result: `evidence missing` · Evidence: Frozen formats have real stored consumers; new workflow not yet implemented. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] An optional behavior or configuration surface has no current need.
  Result: `evidence missing` · Evidence: Frozen formats have real stored consumers; new workflow not yet implemented. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A fallback has no current requirement or observed failure to handle.
  Result: `evidence missing` · Evidence: Frozen formats have real stored consumers; new workflow not yet implemented. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A compatibility path has no current supported consumer or version.
  Result: `no problem found` · Evidence: Frozen Workspace/evidence formats support stored prior documents; their presence alone is not speculative complexity.

- [ ] A generic mechanism or future variant is shaped around hypothetical consumers rather than current use.
  Result: `evidence missing` · Evidence: Frozen formats have real stored consumers; new workflow not yet implemented. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A forwarding layer adds no ownership, policy, transformation, or stable boundary.
  Result: `evidence missing` · Evidence: Frozen formats have real stored consumers; new workflow not yet implemented. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A mechanism remains after its load-bearing current-need premise is disproved.
  Result: `evidence missing` · Evidence: Frozen formats have real stored consumers; new workflow not yet implemented. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A mechanism that serves no current execution or supported path remains in the affected surface.
  Result: `evidence missing` · Evidence: Frozen formats have real stored consumers; new workflow not yet implemented. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Code Complexity

##### Required behavior is harder to reason about than its domain requires

- [ ] Avoidable control flow makes behavior difficult to trace.
  Result: `evidence missing` · Evidence: Controller enqueue/load ticket flow and provider reconciliation sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Avoidable mutable state makes an operation's result difficult to predict.
  Result: `evidence missing` · Evidence: Controller enqueue/load ticket flow and provider reconciliation sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Avoidable dependency structure makes a change difficult to reason about.
  Result: `evidence missing` · Evidence: Controller enqueue/load ticket flow and provider reconciliation sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] One behavior is scattered across more units than its current domain needs.
  Result: `evidence missing` · Evidence: Controller enqueue/load ticket flow and provider reconciliation sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Local behavior requires tracing distant unrelated dependencies.
  Result: `evidence missing` · Evidence: Controller enqueue/load ticket flow and provider reconciliation sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Necessary domain complexity is mixed with accidental implementation complexity.
  Result: `evidence missing` · Evidence: Controller enqueue/load ticket flow and provider reconciliation sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Readability

##### Local code expression obscures behavior

- [ ] Dense or indirect expression makes local behavior difficult to follow.
  Result: `evidence missing` · Evidence: CLI driver operation ordering and public API comments sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Local flow or ordering hides the sequence of effects.
  Result: `evidence missing` · Evidence: CLI driver operation ordering and public API comments sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A local expression hides an important invariant or domain rule.
  Result: `evidence missing` · Evidence: CLI driver operation ordering and public API comments sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Local formatting obscures code grouping or flow.
  Result: `evidence missing` · Evidence: CLI driver operation ordering and public API comments sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] An internal comment conflicts with the code it describes.
  Result: `evidence missing` · Evidence: CLI driver operation ordering and public API comments sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Internal comments restate syntax while a non-obvious reason or constraint remains hidden.
  Result: `evidence missing` · Evidence: CLI driver operation ordering and public API comments sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Vocabulary

##### Code uses an inaccurate or unstable domain vocabulary

- [ ] The same domain concept uses conflicting terms across nearby code and contracts.
  Result: `evidence missing` · Evidence: Go Pipeline/Graph/Plan, Run presentation label and historical instance field. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] One term names different domain concepts in the affected surface.
  Result: `evidence missing` · Evidence: Go Pipeline/Graph/Plan, Run presentation label and historical instance field. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A term describes an accidental mechanism instead of the domain concept it represents.
  Result: `evidence missing` · Evidence: Go Pipeline/Graph/Plan, Run presentation label and historical instance field. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A term misstates the responsibility, state, value, or effect it names.
  Result: `no problem found` · Evidence: run-presentation.ts explicitly labels a display pipeline name, not a Pipeline/Plan registration; no contrary use in traced view.

#### Naming Convention

##### Identifiers violate the applicable naming owner

- [ ] An identifier violates the applicable casing or word-form convention.
  Result: `evidence missing` · Evidence: Current exported contracts and provider port sampled; no full convention audit. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] An abbreviation or spelling conflicts with the applicable project or language convention.
  Result: `evidence missing` · Evidence: Current exported contracts and provider port sampled; no full convention audit. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A visibility or namespace name conflicts with the role of the identified code.
  Result: `evidence missing` · Evidence: Current exported contracts and provider port sampled; no full convention audit. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Definitions and uses apply inconsistent names to the same program role.
  Result: `evidence missing` · Evidence: Current exported contracts and provider port sampled; no full convention audit. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Docstring

##### Caller-facing contract documentation is missing or wrong

- [ ] A unit that requires a docstring has none.
  Result: `evidence missing` · Evidence: BuildPlan/Run/Stop and resource adapter contracts sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A docstring conflicts with the behavior it describes.
  Result: `no problem found` · Evidence: ServiceResources describes resource resolution without Project/engine ownership, consistent with traced read methods.

- [ ] A docstring omits a caller-relevant contract detail.
  Result: `evidence missing` · Evidence: BuildPlan/Run/Stop and resource adapter contracts sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A docstring restates the signature without explaining the caller contract.
  Result: `evidence missing` · Evidence: BuildPlan/Run/Stop and resource adapter contracts sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A docstring example no longer represents current behavior.
  Result: `evidence missing` · Evidence: BuildPlan/Run/Stop and resource adapter contracts sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Correctness

##### Required behavior or failure handling is incomplete

- [ ] Ordinary valid use produces behavior that conflicts with the governing contract.
  Result: `evidence missing` · Evidence: Current service negative cases and provider tests; no new effect path exists. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A materially different valid path is rejected or handled as invalid.
  Result: `evidence missing` · Evidence: Current service negative cases and provider tests; no new effect path exists. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Invalid input or state is accepted without the required rejection or containment.
  Result: `no problem found` · Evidence: Fresh service HTTP boundary tests reject ambiguous/oversized/untrusted requests and unsupported effect routes in this bounded reach.

- [ ] A boundary or state transition violates a required invariant.
  Result: `evidence missing` · Evidence: Current service negative cases and provider tests; no new effect path exists. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A failure leaves state inconsistent.
  Result: `evidence missing` · Evidence: Current service negative cases and provider tests; no new effect path exists. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Failure recovery does not restore the required state.
  Result: `evidence missing` · Evidence: Current service negative cases and provider tests; no new effect path exists. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] An error is lost or transformed so its consumer cannot respond correctly.
  Result: `evidence missing` · Evidence: Current service negative cases and provider tests; no new effect path exists. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A prohibited state or effect can be reached by bypassing the expected path.
  Result: `evidence missing` · Evidence: Current service negative cases and provider tests; no new effect path exists. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A retried or repeated operation produces an unintended duplicate or conflicting effect.
  Result: `no problem found` · Evidence: Focused collaboration tests and coordinator reconciliation show no automatic replay of uncertain provider turns; execution not covered.

- [ ] A multi-step state or data change exposes a partial result where the contract requires atomicity.
  Result: `evidence missing` · Evidence: Current service negative cases and provider tests; no new effect path exists. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A resource is acquired, retained, released, or restored outside its required lifetime on a supported terminal path.
  Result: `evidence missing` · Evidence: Current service negative cases and provider tests; no new effect path exists. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] An operation reports completion before its required effect is complete or durably owned.
  Result: `evidence missing` · Evidence: Current service negative cases and provider tests; no new effect path exists. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Required behavior changes with time, ordering, randomness, locale, or environment without a contract for that variation.
  Result: `evidence missing` · Evidence: Current service negative cases and provider tests; no new effect path exists. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

##### A change fails to preserve required behavior

- [ ] A refactoring or maintenance change alters required observable behavior.
  Result: `evidence missing` · Evidence: Current service negative cases and provider tests; no new effect path exists. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Testing

##### Behavior and risk coverage is incomplete

- [ ] A material observable behavior lacks a direct applicable test.
  Result: `evidence missing` · Evidence: 32 focused App tests, fresh service suite and explicit native-root compile failure. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A materially different valid path with distinct behavior or risk lacks a direct test.
  Result: `evidence missing` · Evidence: 32 focused App tests, fresh service suite and explicit native-root compile failure. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] An invalid or adversarial input, state, or operation with material side-effect risk lacks a direct test.
  Result: `evidence missing` · Evidence: 32 focused App tests, fresh service suite and explicit native-root compile failure. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A material boundary, state transition, time, ordering, concurrency, cancellation, or timeout behavior lacks a controlled test.
  Result: `evidence missing` · Evidence: 32 focused App tests, fresh service suite and explicit native-root compile failure. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A relevant failure or recovery path lacks a test.
  Result: `evidence missing` · Evidence: 32 focused App tests, fresh service suite and explicit native-root compile failure. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A regression-prone behavior lacks a test that distinguishes the prior defect.
  Result: `evidence missing` · Evidence: 32 focused App tests, fresh service suite and explicit native-root compile failure. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

##### Test code does not provide trustworthy, maintainable checks

- [ ] A test asserts an incidental implementation detail instead of governing behavior.
  Result: `evidence missing` · Evidence: 32 focused App tests, fresh service suite and explicit native-root compile failure. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A test fixture or test double does not represent the condition or real boundary contract named by the test.
  Result: `evidence missing` · Evidence: 32 focused App tests, fresh service suite and explicit native-root compile failure. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A test oracle cannot distinguish the expected result from a material wrong result.
  Result: `evidence missing` · Evidence: 32 focused App tests, fresh service suite and explicit native-root compile failure. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A test result depends on uncontrolled order, shared state, time, randomness, network, process state, or machine state.
  Result: `evidence missing` · Evidence: 32 focused App tests, fresh service suite and explicit native-root compile failure. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A skipped, disabled, or suppressed test hides a relevant result.
  Result: `evidence missing` · Evidence: 32 focused App tests, fresh service suite and explicit native-root compile failure. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Test setup obscures the behavior under test.
  Result: `evidence missing` · Evidence: 32 focused App tests, fresh service suite and explicit native-root compile failure. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A test can pass without executing the behavior it claims to verify.
  Result: `evidence missing` · Evidence: 32 focused App tests, fresh service suite and explicit native-root compile failure. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A retry, rerun, quarantine, or broad tolerance hides an unresolved intermittent test result.
  Result: `evidence missing` · Evidence: 32 focused App tests, fresh service suite and explicit native-root compile failure. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A test leaves a resource or shared state changed after a success, failure, skip, timeout, or cancellation path.
  Result: `evidence missing` · Evidence: 32 focused App tests, fresh service suite and explicit native-root compile failure. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Verification

##### Verification evidence cannot support the claimed result

- [ ] Verification ran against a different subject identity than the reviewed subject.
  Result: `no problem found` · Evidence: subject.json and handoff-checks.json preserve original bytes; added build dependency reach is explicitly separate.

- [ ] The exact verification environment is not recorded.
  Result: `no problem found` · Evidence: verification.md records OS, architecture, runtime versions and native/cross-compile settings.

- [ ] The exact verification tool identity is not recorded.
  Result: `no problem found` · Evidence: verification.md and bound package lock identify Go/Node/Vitest and pinned provider.

- [ ] The exact verification command identity is not recorded.
  Result: `no problem found` · Evidence: verification.md records exact commands; linux-compile.json retains concrete argv.

- [ ] The exact verification configuration identity is not recorded.
  Result: `evidence missing` · Evidence: Exact environment/commands and separate cached, fresh, compile-only outcomes. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A claimed verification step was not run.
  Result: `no problem found` · Evidence: Claims are limited to observed focused completions, native compilation failure and Linux compile-only success.

- [ ] Verification output cannot be tied reproducibly to one complete execution.
  Result: `evidence missing` · Evidence: Exact environment/commands and separate cached, fresh, compile-only outcomes. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] The claim extends beyond the paths, inputs, environments, modes, repetitions, or terminal results actually verified.
  Result: `no problem found` · Evidence: Verification explicitly excludes complete suite, live Agent, new effects, Linux runtime and package acceptance.

- [ ] A later passing run replaces rather than reconciles a conflicting earlier result.
  Result: `no problem found` · Evidence: Cached service result, fresh service pass, native failure and Linux compile-only success are retained separately.

- [ ] A verification result omits a relevant failure, flake, skip, quarantine, unavailable prerequisite, unsupported environment, or evidence limit.
  Result: `no problem found` · Evidence: Native compile failure and historical packed-runner limits are disclosed; no full-suite success claimed.

- [ ] A cache, stale generated output, suppression, exclusion, or baseline can hide a material result from the recorded run.
  Result: `evidence missing` · Evidence: Exact environment/commands and separate cached, fresh, compile-only outcomes. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] An applicable project-owned check for the affected surface is absent from the recorded verification evidence without a subject reason.
  Result: `evidence missing` · Evidence: Exact environment/commands and separate cached, fresh, compile-only outcomes. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Delivery

##### The handed-off result is not the reviewed and verified implementation

- [ ] A delivered artifact changes the reviewed behavior.
  Result: `evidence missing` · Evidence: No artifact built or delivered; current source/reference compatibility sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] The delivered result cannot be reproduced from its recorded dependencies and configuration.
  Result: `evidence missing` · Evidence: No artifact built or delivered; current source/reference compatibility sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] The handoff does not identify the exact delivered result.
  Result: `evidence missing` · Evidence: No artifact built or delivered; current source/reference compatibility sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] The handoff omits an operating condition needed to use the result.
  Result: `evidence missing` · Evidence: No artifact built or delivered; current source/reference compatibility sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] The handoff omits information needed to recover the result.
  Result: `evidence missing` · Evidence: No artifact built or delivered; current source/reference compatibility sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A delivery failure leaves neither the prior result nor the new result safely usable.
  Result: `evidence missing` · Evidence: No artifact built or delivered; current source/reference compatibility sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A required artifact, metadata file, schema, generated output, or runtime asset is absent from the delivered result.
  Result: `evidence missing` · Evidence: No artifact built or delivered; current source/reference compatibility sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] The artifact exercised in verification is not the exact artifact handed to its consumers.
  Result: `evidence missing` · Evidence: No artifact built or delivered; current source/reference compatibility sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Build, package, release, deployment, live verification, and observed health are reported as one result.
  Result: `no problem found` · Evidence: Verification distinguishes focused tests, compile-only evidence and unperformed delivery/live checks.

### Product Lifecycle

#### Usability

##### Consumers face avoidable learning or use burden

- [x] A consumer cannot complete an ordinary task from the public surface and its immediate guidance.
  Result: `problem found` · Evidence: Newly requested Agent source modification cannot execute through current read-only policy and shared tools.
  Finding: [P2](#p2-agent-authoring-is-not-an-implemented-capability)

- [ ] Similar tasks require conflicting mental models or interaction patterns.
  Result: `evidence missing` · Evidence: Project/Run navigation lacks source-change/Plan collaboration journey. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A common task requires avoidable interaction or implementation knowledge.
  Result: `evidence missing` · Evidence: Project/Run navigation lacks source-change/Plan collaboration journey. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Feedback does not make the task result or current state clear.
  Result: `evidence missing` · Evidence: Project/Run navigation lacks source-change/Plan collaboration journey. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Failure information does not support a safe next action or recovery.
  Result: `evidence missing` · Evidence: Project/Run navigation lacks source-change/Plan collaboration journey. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Operations

##### Runtime behavior is difficult to observe, support, or recover

- [ ] A material failure cannot be distinguished from ordinary behavior with the available diagnostics.
  Result: `evidence missing` · Evidence: Uncertain-turn reconciliation; future execution recovery absent. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Failures that require different responses appear indistinguishable.
  Result: `evidence missing` · Evidence: Uncertain-turn reconciliation; future execution recovery absent. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Partial failure is invisible while the product remains degraded.
  Result: `evidence missing` · Evidence: Uncertain-turn reconciliation; future execution recovery absent. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Operators cannot determine whether recovery restored the required service, state, or data condition.
  Result: `evidence missing` · Evidence: Uncertain-turn reconciliation; future execution recovery absent. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Diagnostic output omits context needed for support.
  Result: `evidence missing` · Evidence: Uncertain-turn reconciliation; future execution recovery absent. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A supported operating configuration has an unclear runtime effect.
  Result: `evidence missing` · Evidence: Uncertain-turn reconciliation; future execution recovery absent. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A retry, fallback, or recovery loop has no attempt, time, or resource bound.
  Result: `evidence missing` · Evidence: Uncertain-turn reconciliation; future execution recovery absent. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A stalled or degraded operation has no bounded detection and safe recovery path.
  Result: `evidence missing` · Evidence: Uncertain-turn reconciliation; future execution recovery absent. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

#### Compatibility

##### Change breaks a supported consumer or lifecycle transition

- [ ] A supported runtime, operating system, architecture, environment, or consumer integration stops working after the change.
  Result: `evidence missing` · Evidence: Frozen formats and pinned runtime/provider contracts sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Established public behavior changes without an explicit compatibility decision.
  Result: `evidence missing` · Evidence: Frozen formats and pinned runtime/provider contracts sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Public behavior conflicts with a supported version promise.
  Result: `evidence missing` · Evidence: Frozen formats and pinned runtime/provider contracts sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] Supported version combinations cannot operate together as required.
  Result: `evidence missing` · Evidence: Frozen formats and pinned runtime/provider contracts sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A supported transition between versions breaks consumer behavior or prevents required stored or serialized data from remaining readable and valid.
  Result: `evidence missing` · Evidence: Frozen formats and pinned runtime/provider contracts sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A supported replacement leaves consumers without a working transition.
  Result: `evidence missing` · Evidence: Frozen formats and pinned runtime/provider contracts sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

- [ ] A supported exit leaves consumers without continuity or a defined state disposition.
  Result: `evidence missing` · Evidence: Frozen formats and pinned runtime/provider contracts sampled. This sign was not fully qualified in that reach.
  Finding: [G3 — bounded review coverage](#gaps)

</details>

### Specialist overlays

| Overlay / owning source question | Result | Evidence / finding |
|---|---|---|
| Security — Electron: “Validate the sender of all IPC messages” | no problem found | Main service/workspace/collaboration IPC check workspace webContents, main frame and trusted origin. This is static trace evidence, not comprehensive attack testing. |
| Concurrency — governing contract: “An uncertain turn submission is reconciled through provider history; it is not automatically submitted again.” | no problem found | coordinator.ts retains request ID and reconciles history; focused collaboration tests passed. |
| Concurrency — governing contract: “Late Stop must be conditional on the observed lease” | evidence missing | Existing engine Stop selects its own current lease; public Stop has no expected-lease input. Future App effect contract needs G2. |
| Dependencies — local generated protocol is tied to Codex 0.153.4 | no problem found | package.json lock, generator and conversation read-only policy agree for the current narrowed surface. Native authoring is not qualified (G1). |
| Build — README: “Go and Gobble run in Linux/amd64 containers, emulated on Apple Silicon.” | evidence missing | Linux cross-compilation passes; supported container execution not rerun. Native root compile fails separately (G4). |
| Configuration — governing contract requires pinned image/daemon/workspace runtime routing | no problem found | runtime.go checks existing read-query binding; fresh service tests pass within test scope. No effect profile claimed. |
| Observability — governing contract requires durable submission ID before contacting provider | no problem found | coordinator.ts persists submission before provider call and keeps uncertain states visible. |
| Migration — existing contract keeps versioned UI state and backups | evidence missing | AtomicStateFile has validation/migration/backup/atomic-write paths; complete format round-trip matrix not rerun (G3). |

## Problems

### P1 Pre-run pipeline identity is absent

- **Primary category:** Data Model. **Found during:** both.
- **Expectation and source:** Latest owner goal requires a pipeline that can be
  designed/revised before it has any Run, and exact shared review of changes/Plan.
- **Observation:** Current resource union is file/run/log. RunRegistration binds an
  existing engine workspace; the presentation pipeline string explicitly is not a
  Pipeline registration. There is no source revision/change/Plan relationship.
- **Impact:** A UI cannot reliably say which candidate was reviewed or which Plan a
  new Run must match. Reusing a Run label would lose identity and chronology.
- **Evidence:** [resource.ts](../../../../app/contracts/src/resource.ts),
  [run.ts](../../../../app/contracts/src/run.ts),
  [run-presentation.ts](../../../../app/contracts/src/run-presentation.ts),
  [Go catalog types](../../../../internal/appservice/types.go).
- **Cause:** Deliberate observation-first implementation scope, now extended by the owner.
- **Uncertainty:** New wire schemas/retention limits remain proposed; no existing
  read-only reference corruption was demonstrated.
- **Related effects:** Data Model identity sign; Usability P2.
- **Responsible owner:** Pipeline domain/service contract owner.

### P2 Agent authoring is not an implemented capability

- **Primary category:** Usability. **Found during:** both.
- **Expectation and source:** Latest owner goal makes the Agent the pipeline editor.
- **Observation:** Discussion sessions use read-only isolated cwd, no network and
  disabled shell. Shared tools only query/observe/point/question; no source-change
  operation exists. This is correct for the prior slice, insufficient for the new one.
- **Impact:** An Agent can explain a requested edit but cannot produce a host-owned,
  reviewable, applied change through the current product contract.
- **Evidence:** [policy.ts](../../../../app/desktop/src/main/codex/policy.ts),
  [conversations.ts](../../../../app/desktop/src/main/codex/conversations.ts),
  [shared-tools.ts](../../../../app/contracts/src/shared-tools.ts), focused App tests.
- **Cause:** Missing scoped authoring use case, not merely a prompt deficiency.
- **Uncertainty:** A broader provider mode may work, but pinned protocol, scope and
  file-change/approval behavior have not been qualified (G1).
- **Related effects:** Ordinary new-purpose task completion; execution remains G2.
- **Responsible owner:** Pipeline collaboration application owner.

## Improvements

### I1 Separate transient presentation lifetime from the workspace writer

- **Primary category:** Modularization. **Found during:** critical review.
- **Acceptable current condition:** One WorkspaceController transaction keeps chat,
  questions and evidence coherent. Render/reference helpers already own parts of the work.
- **Evidence:** [controller.ts:79](../../../../app/desktop/src/main/workspace/controller.ts),
  [WorkspaceResources:18](../../../../app/desktop/src/main/workspace/service.ts),
  loadSurface ticketed loading and disconnect/session cleanup.
- **Supported benefit:** Plan/diff presentation can grow without adding service-owned
  lifecycle state or more format-specific optional methods to the durable writer.
- **Suggestion:** A small presentation-session coordinator/read-resource port;
  preserve one durable queue and its revision/epoch checks.
- **Trade-off:** Extraction can break reference lifetimes if it splits a transaction.
  Keep effect records in the Go service and verify old reference/restore behavior.
- **Uncertainty:** No bug established merely from file size; exact extraction is stage 1 design.
- **Decision owner:** Workspace architecture owner.

### I2 Make active contract ownership explicit without rewriting history

- **Primary category:** Project Structure. **Found during:** critical review.
- **Acceptable current condition:** Frozen version files have real compatibility
  consumers and explicit schema versions. Their count is not itself overengineering.
- **Evidence:** [contracts index](../../../../app/contracts/src/index.ts),
  [workspace-document](../../../../app/contracts/src/workspace-document.ts), versioned
  evidence/document content and storage migration code.
- **Supported benefit:** New pipeline/source/operation types have clear current owners
  and do not accidentally become UI document facts or provider-specific event types.
- **Suggestion:** Focused current modules/exports and a short ownership map; defer
  physical history moves unless a concrete consumer needs them.
- **Trade-off:** Another facade with no ownership would add noise; avoid a sweeping reorganization.
- **Uncertainty:** Full historic format compatibility has not been rerun (G3).
- **Decision owner:** Shared contract owner.

## Strengths

### S1 Engine authority and service isolation are already explicit

- **Primary category:** Architecture. **Found during:** both.
- **Verified benefit:** Service uses an adapter to pinned runtime queries; it does
  not link engine internals. Go model/CLI remains independent of App/provider.
- **Evidence:** [boundaries_test.go](../../../../internal/appservice/boundaries_test.go),
  [runtime.go](../../../../internal/appservice/runtime.go), fresh service tests;
  [reuse.go](../../../../internal/engine/reuse.go) owns engine classification.
- **Must preserve:** New effects go through the same engine authority, not duplicated
  TypeScript scheduling/reuse decisions or a native service engine import.
- **Uncertainty:** New effect adapter is not implemented.

### S2 Exact shared references and one durable UI writer have a useful owner

- **Primary category:** Architecture. **Found during:** critical review.
- **Verified benefit:** Source/projection revisions, observation receipts and explicit
  evidence have separate meanings; queued writes and view epochs protect User state.
- **Evidence:** Controller, render-session/reference-view helpers, source checks in
  ServiceResources and AtomicStateFile storage inspected in the bound traces.
- **Must preserve:** No silent retarget; candidate diff and Plan references use these
  semantics. Pipeline operations do not join the UI writer's persisted state.
- **Uncertainty:** Full native shared-view matrix not repeated this turn.

### S3 Provider uncertainty is modeled instead of blindly retried

- **Primary category:** Operations. **Found during:** both.
- **Verified benefit:** Durable submission IDs, narrow provider port and history
  reconciliation keep ambiguous delivery distinct from failed delivery.
- **Evidence:** [provider.ts](../../../../app/desktop/src/main/collaboration/provider.ts),
  [coordinator.ts](../../../../app/desktop/src/main/collaboration/coordinator.ts),
  32 focused Codex/collaboration tests passed.
- **Must preserve:** Analysis operations have distinct identity and cancellation;
  a turn timeout is never permission to repeat a potentially accepted Start.
- **Uncertainty:** Test doubles do not establish live native authoring.

## Gaps

| Gap | Found during | Affected claims | Effect / needed evidence | Recovery owner and first action |
|---|---|---|---|---|
| G1 Authoring capability qualification | critical review | P2; Security/Privacy/Dependencies | No production write scope or native provider authoring validated. Bounded tool/candidate path is a proposal; wider sandbox not assumed safe. | Collaboration owner: implement and qualify scoped candidate authoring after its design checkpoint. |
| G2 Executable admission and recovery | critical review | P3 lead; Public API/Operations/Concurrency | Service start routes absent; BuildPlan and Run construct separately; expected-lease Stop unavailable. Need same-document admission, durable launch correlation, conditional ownership and candidate Resume preview. | Gobble lifecycle owner: add the focused engine contract before enabling App effect controls. |
| G3 Bounded review coverage | base checklist | All explicitly missing checklist rows and specialist depths | Sampling cannot establish complete code-quality/security/accessibility/performance/compatibility results. | Each implementation-stage owner: inspect and test affected behavior; independent review requires a separate unaided same-subject pass. |
| G4 Target-specific root test limit | specialist overlay | Build/runtime claims | darwin/arm64 root compile fails; README's Linux/amd64 target compiles. No actual Linux test run this turn. Do not infer an App service failure or add host execution scope automatically. | Go runtime owner: qualify relevant tests in the supported container target before lifecycle changes; separately decide whether native library testing is required. |

## Handoff

| Field | Value |
|---|---|
| Rechecked subject identity | [handoff-checks.json](handoff-checks.json) records unchanged original subject and added dependency bytes; same subject SHA256 as above. |
| Record-state reason | Partial because architecture was sampled and new effects/platform runtime have unqualified evidence. |
| Caller report ownership | The caller owns this caller-bound report after handoff. |
| Next owner | Project owner reviewing the proposed design and first implementation checkpoint. |
| First action | Review [design](design.md) and [plan](plan/plan-index.md); authorize stage 1 only when its concrete boundaries are acceptable. |
| Problems handed over | P1 pre-run identity; P2 authoring capability. |
| Gaps handed over | G1–G4 above. |
| Authority boundary | This report grants no target mutation, approval, merge, publication or release authority. |
| Restart condition | Changed source or accepted contract requires rebinding; completed implementation gets its own review evidence. |
