# Pipeline collaboration — Plan Part 01

> **Index:** [Plan](plan-index.md).<br>
> **Source:** [Task Hierarchy](tasks/tasks-index.md).<br>
> **Covers:** All five dependency-ordered implementation groups.

## Summary

| Aspect   | Definition                                                                                                                                                                                                                                               |
| -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Work     | One Agent-authored pipeline collaboration loop.                                                                                                                                                                                                          |
| Purpose  | Make source changes, Gobble Plans, authorized execution and observed results discussable in the same Project.                                                                                                                                            |
| Scope    | Five groups below; existing English App, Panes, Chat, references and CLI authority preserved. No Notebook editing, CSV chart generation, new viewers, broad provider shell, package or release work.                                                     |
| Output   | A qualified design → change → Plan → Run → failure/fix/Resume journey with exact identities.                                                                                                                                                             |
| Design   | Product philosophy is owner-accepted. [Detailed design](../design.md) is proposed; each group requires its concrete design/sketch checkpoint before implementation.                                                                                      |
| Grouping | Five separate user-visible outcomes and effect boundaries are the fewest practical review gates. Each combines two compatible leaves; splitting by file/layer would leave a partial contract, while merging crosses source-write or execution authority. |

## Shared Context

| Context              | Value                                                                                                                                                                                         | Applies to |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| Repository           | Gobble branch codex/project-workspace-design; actual reviewed dirty source is bound in ../subject.json. Rebind changed source before implementation.                                          | All groups |
| Platform             | Electron on current macOS/arm64; Gobble execution in pinned Linux/amd64 containers. Native root-library test failure is separately recorded, not a new host execution goal.                   | All groups |
| Governing boundaries | ../design.md; existing .gobbi Project workspace contract; latest owner request.                                                                                                               | All groups |
| Delivery rhythm      | Concrete sketch/schema before starting; summarize design/code/tests after each group and obtain owner approval before the next.                                                               | All groups |
| Preserved work       | Existing source changes, historical schemas/evidence, synced sources/, normal App profile, User data and analysis workspaces.                                                                 | All groups |
| Authority            | This planning result does not start production implementation or authorize changes to real analysis projects/runs, publication or release. Later controlled native checks use owned fixtures. | All groups |

## Task Groups

Lower order executes first. Requires edges are authoritative. Work is sequential;
accountable roles are responsibility labels, not requests to create other agents.

### `task-01-foundation` — Pipeline concepts and ownership foundation

| Attribute           | Definition                                                        |
| ------------------- | ----------------------------------------------------------------- |
| Order               | 1                                                                 |
| Combined task paths | 1.1, 1.2                                                          |
| Accountable role    | Pipeline architecture implementation owner                        |
| Compatible skills   | gobbi:go-design, gobbi:electron-contract, gobbi:typescript-typing |
| Requires            | None                                                              |

#### Decomposed Tasks

| Path | Title                                                   | Work                                                                                                                                                              | Boundary                                                                                                                                       | Output                                                                                                   |
| ---- | ------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| 1.1  | Define and expose pipeline identity                     | Define the core vocabulary and implement only the pipeline registration/read contract needed by the first consumer.                                               | Service owns definitions and source bindings; future Change/Plan/Operation definitions are documented, not speculative executable scaffolding. | Concept/ownership records, focused active contract and a usable registered-pipeline read path.           |
| 1.2  | Separate presentation lifetime and qualify the baseline | Extract the smallest supported resource/presentation lifetime seam while retaining the durable workspace writer; qualify the relevant supported runtime baseline. | Preserve transaction/revision behavior and frozen formats. Native root-library support is a separate decision, not automatic scope.            | Focused responsibility map, behavior-preserving extraction and honest host/Linux qualification evidence. |

#### Group Details

- **Why combined:** Definitions, minimal registration/read consumer and the presentation seam form one reviewable ownership boundary; separating them would leave types without a real consumer.
- **Group outcome:** A registered pipeline has a distinct identity and readable source binding; existing workspace behavior retains one writer and exact references.
- **Stop:** No source mutations, Plan jobs or execution controls. Do not scaffold unused workflow services or rewrite frozen schemas. End with an owner review checkpoint; do not advance automatically.
- **Constraints and authority:** Shared constraints apply. The detailed stage design is pending owner approval; record its actual approval before implementation. No generic approval inferred from ordinary Chat or prior unrelated stage acceptance.
- **Accepted design:** Accepted product philosophy in [design](../design.md); [relevant proposed detail](../design.md#concept-definitions) becomes the stage contract only after review.
- **Writer frontier:** Governing concept/ownership docs; focused internal/appservice pipeline registration/read modules and tests; active app/contracts; Main resource/presentation boundary and its existing consumers/tests; minimal pipeline navigation using existing views. No engine lifecycle changes.
- **Handoffs:** Receives this reviewed design; passes distinct pipeline identity, a source read path and stable workspace ownership to authoring.
- **Verification:** Pipeline registration/source navigation works; Project/Run/Pane concepts remain distinct; retained source/evidence and workspace restoration preserve observable behavior.
- **Metadata:** None beyond Shared Context and the qualification points below.

### `task-02-agent-changes` — Agent-authored changes and shared review

| Attribute           | Definition                                                                |
| ------------------- | ------------------------------------------------------------------------- |
| Order               | 2                                                                         |
| Combined task paths | 2.1, 2.2                                                                  |
| Accountable role    | Source collaboration implementation owner                                 |
| Compatible skills   | gobbi:go-development, gobbi:electron-development, gobbi:react-development |
| Requires            | `task-01-foundation`                                                      |

#### Decomposed Tasks

| Path | Title                                       | Work                                                                                                                                 | Boundary                                                                                                                                                       | Output                                                                                               |
| ---- | ------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| 2.1  | Own candidate changes and source acceptance | Implement bounded Agent proposals, immutable base/candidate retention, managed source acceptance and conflict/recovery outcomes.     | Only registered source/config text with explicit authoring scope. One managed writer; unsupported external-writer coexistence uses isolated candidate handoff. | Service-owned change-set revision and accepted-source lifecycle with retained evidence.              |
| 2.2  | Discuss exact source changes                | Connect Agent/User adapters to the same change service and add a read-only Changes view with exact hunk references in existing Chat. | Existing Panes, compact navigation and one composer; trusted User source acceptance is separate from Agent-authored content.                                   | User and Agent can inspect, point to, refine and accept the same exact changes without an editor UI. |

#### Group Details

- **Why combined:** Candidate lifecycle and shared diff/approval are one end-to-end authoring contract with the same source identity and effect boundary.
- **Group outcome:** An Agent produces bounded pipeline source changes; User and Agent discuss exact candidate hunks and can accept a conflict-checked source revision.
- **Stop:** No general shell, manual editor, binary editing, arbitrary file operations, dependency installation or analysis launch. Do not claim concurrent external-editor atomicity. End with an owner review checkpoint; do not advance automatically.
- **Constraints and authority:** Shared constraints apply. The detailed stage design is pending owner approval; record its actual approval before implementation. No generic approval inferred from ordinary Chat or prior unrelated stage acceptance.
- **Accepted design:** Accepted product philosophy in [design](../design.md); [relevant proposed detail](../design.md#agent-authoring-and-source-lifecycle) becomes the stage contract only after review.
- **Writer frontier:** Go service candidate/source-change modules and persistence; active change/reference contracts; Main scoped tool and User command adapters; renderer workspace/views/changes and inline Chat actions; affected tests and stage records.
- **Handoffs:** Receives pipeline/source identity and stable reference ownership; passes accepted candidate bytes and change provenance to Plan validation.
- **Verification:** A real registered multi-file text candidate is reviewable; stale bases and unsupported writes have explicit outcomes; exact hunk references survive refinement and restart; incomplete application cannot masquerade as accepted source.
- **Metadata:** None beyond Shared Context and the qualification points below.

### `task-03-plan-review` — Gobble validation and exact Plan discussion

| Attribute           | Definition                                                                |
| ------------------- | ------------------------------------------------------------------------- |
| Order               | 3                                                                         |
| Combined task paths | 3.1, 3.2                                                                  |
| Accountable role    | Plan integration implementation owner                                     |
| Compatible skills   | gobbi:go-development, gobbi:electron-development, gobbi:react-development |
| Requires            | `task-02-agent-changes`                                                   |

#### Decomposed Tasks

| Path | Title                                  | Work                                                                                                                                 | Boundary                                                                                                           | Output                                                                                                                          |
| ---- | -------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------- |
| 3.1  | Own validation jobs and Plan artifacts | Add isolated candidate validation, runtime/config binding, structured failures and Gobble-owned prepared payload/admission identity. | No host evaluation of Project Go code, no Docker socket in the validation job, no ambient dependency installation. | A versioned Plan artifact and private Gobble prepared payload, or explicit validation/setup failure tied to source and runtime. |
| 3.2  | Review the pre-run Plan together       | Display the Gobble-produced Plan in an existing Pane and expose exact node references and source links only when proven.             | Task definitions are distinct from Run expansions/attempts; no graph authoring or duplicated reuse logic.          | A shared read-only Plan view and Chat references with clear current/out-of-date states.                                         |

#### Group Details

- **Why combined:** Validation isolation, semantic Plan identity and shared projection must agree; treating a graph viewer as a separate milestone would omit its executable meaning.
- **Group outcome:** Gobble validates the exact candidate in a bounded job and both participants discuss the resulting Plan or defects.
- **Stop:** No analysis tasks or Start/Stop controls. A Plan is neither execution permission nor assumed executable JSON. End with an owner review checkpoint; do not advance automatically.
- **Constraints and authority:** Shared constraints apply. The detailed stage design is pending owner approval; record its actual approval before implementation. No generic approval inferred from ordinary Chat or prior unrelated stage acceptance.
- **Accepted design:** Accepted product philosophy in [design](../design.md); [relevant proposed detail](../design.md#plan-generation-and-executable-identity) becomes the stage contract only after review.
- **Writer frontier:** Focused Gobble Plan/admission-digest contract and CLI/job adapter; Go service validation jobs/artifacts; active Plan/reference schemas; Main Plan projections; renderer workspace/views/plan; affected tests and docs.
- **Handoffs:** Receives sealed candidate identity; passes a reviewed Plan/admission digest and declared input/runtime bindings to execution control.
- **Verification:** Plan nodes/commands/inputs/outputs and structured defects refer to the candidate; changed source invalidates the old Plan; validation cannot use controller authority; references resolve exact Plan nodes.
- **Metadata:** None beyond Shared Context and the qualification points below.

### `task-04-execution` — Authorized Start and conditional Stop

| Attribute           | Definition                                                             |
| ------------------- | ---------------------------------------------------------------------- |
| Order               | 4                                                                      |
| Combined task paths | 4.1, 4.2                                                               |
| Accountable role    | Execution lifecycle implementation owner                               |
| Compatible skills   | gobbi:go-concurrency, gobbi:go-development, gobbi:electron-development |
| Requires            | `task-03-plan-review`                                                  |

#### Decomposed Tasks

| Path | Title                                                      | Work                                                                                                                                                                                            | Boundary                                                                                                                    | Output                                                                                                                     |
| ---- | ---------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| 4.1  | Establish engine admission and durable operation ownership | Implement admission of the exact prepared payload without privileged Project-code reevaluation, discoverable launch correlation, service acceptance/reconciliation and engine conditional Stop. | Gobble owns occupancy and actual execution; service owns command durability; closing a caller only ends waiting.            | Recoverable command/lifecycle contract that distinguishes accepted, unknown, running, stop-requested and settled outcomes. |
| 4.2  | Expose exact execution intent in App and Agent tools       | Connect trusted User authorization and Agent requests to shared handlers; display exact Plan/target/effects and actual operation/Run states.                                                    | One right Chat and existing Run views; Agent cannot forge User authorization; no arbitrary paths/argv in renderer commands. | A reviewable Start/Stop journey with durable identity and truthful restart/recovery feedback.                              |

#### Group Details

- **Why combined:** Acceptance, engine admission, controller lifetime and Stop fencing define one coherent execution authority; UI controls are unsafe without the underlying contract.
- **Group outcome:** One exact authorized Plan starts a durably identifiable Run; Stop targets the observed owner; restart/timeout resolves accepted operations safely.
- **Stop:** No Resume before candidate-impact qualification; no App scheduler, checkpoint replica, automatic execution from generic chat answers or blind launch retry. End with an owner review checkpoint; do not advance automatically.
- **Constraints and authority:** Shared constraints apply. The detailed stage design is pending owner approval; record its actual approval before implementation. No generic approval inferred from ordinary Chat or prior unrelated stage acceptance.
- **Accepted design:** Accepted product philosophy in [design](../design.md); [relevant proposed detail](../design.md#commands-authorization-and-recovery) becomes the stage contract only after review.
- **Writer frontier:** Gobble public lifecycle/admission and engine occupancy/Stop seams; CLI protocol; Go service operation journal and effect controller adapter; focused App service/tool/User commands and inline controls; lifecycle tests and docs.
- **Handoffs:** Receives exact Plan/source approval inputs; passes durable operation and Run ownership semantics to failure/fix/Resume collaboration.
- **Verification:** Changed Plan admission is rejected before task effects; duplicate/ambiguous Start is reconciled; controller survives App exit; delayed Stop cannot affect a newer lease; UI states reflect engine-owned outcomes.
- **Metadata:** None beyond Shared Context and the qualification points below.

### `task-05-feedback` — Failure, refinement and Resume collaboration loop

| Attribute           | Definition                                                        |
| ------------------- | ----------------------------------------------------------------- |
| Order               | 5                                                                 |
| Combined task paths | 5.1, 5.2                                                          |
| Accountable role    | Pipeline feedback integration owner                               |
| Compatible skills   | gobbi:go-development, gobbi:electron-testing, gobbi:react-testing |
| Requires            | `task-04-execution`                                               |

#### Decomposed Tasks

| Path | Title                                         | Work                                                                                                                                             | Boundary                                                                                                                                        | Output                                                                                            |
| ---- | --------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| 5.1  | Preview and admit candidate Resume            | Expose engine-owned candidate-versus-prior-workspace reuse/rejection reasons and enforce their reviewed identity at Resume.                      | Reuse classifier remains in Gobble; changed source/runtime/input/workspace invalidates preview; preserve existing engine recovery rules.        | Qualified Resume preview and command linked to exact candidate and prior Run state.               |
| 5.2  | Complete the shared pipeline feedback journey | Integrate failure/log selection, Agent explanation/refinement, Plan review, authorized execution and monitored outcomes in one Project scenario. | Reuse current references/views; honest unavailable/unknown states; preserve User layout, drafts, retained evidence and independent Agent turns. | Scenario/checklist, visual evidence for changed UI and tested core-loop handoff for owner review. |

#### Group Details

- **Why combined:** Resume meaning and the monitor-to-change conversation close the same pipeline collaboration loop; both need the same prior Run/candidate identities.
- **Group outcome:** A researcher can move from a failed Run through exact evidence, Agent changes, a reviewed Plan and engine-owned reuse/Resume to a monitored result.
- **Stop:** End at one qualified complete core journey; no package/release, new scientific viewers, Notebook editor or general extension/MCP host. End with an owner review checkpoint; do not advance automatically.
- **Constraints and authority:** Shared constraints apply. The detailed stage design is pending owner approval; record its actual approval before implementation. No generic approval inferred from ordinary Chat or prior unrelated stage acceptance.
- **Accepted design:** Accepted product philosophy in [design](../design.md); [relevant proposed detail](../design.md#acceptance-scenarios-and-stage-gates) becomes the stage contract only after review.
- **Writer frontier:** Gobble candidate-aware Resume preview/admission using existing reuse classification; service Resume operations; Main references/projections; existing Run/log and Plan/Changes Chat flows; controlled integration fixtures and evidence.
- **Handoffs:** Receives source, Plan and execution ownership; returns a qualified core workflow and remaining real limitations to the owner before any broader product work.
- **Verification:** A controlled pipeline failure is discussed via exact instance/attempt/log references, corrected by Agent changes, and resumed with Gobble-produced reuse/rerun reasons; stale preview rejection and existing workspace behavior are demonstrable.
- **Metadata:** None beyond Shared Context and the qualification points below.

## Unresolved Metadata

| Item                                                           | Evidence                                                                      | Execution effect                                                                                                                          | Affected groups                                          |
| -------------------------------------------------------------- | ----------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------- |
| Actual validation/controller runtime artifact identity         | Current read-query runtime qualification does not prove a new effect profile. | Does not block stage 1; bind the tested runtime before stage 3 jobs or stage 4 execution are enabled.                                     | task-03-plan-review, task-04-execution, task-05-feedback |
| Representative authoring fixture and source-writer environment | No live new write flow ran during study.                                      | Does not block stage 1; stage 2 must qualify managed-writer scope and the isolated-candidate handoff limit before enabling source writes. | task-02-agent-changes                                    |

Detailed source/effect decisions are not hidden as metadata: they are explicit
proposed design choices subject to the owner's stage gates. Native provider
workspace-write qualification is deferred and is not a prerequisite for the selected
bounded-tool approach.
