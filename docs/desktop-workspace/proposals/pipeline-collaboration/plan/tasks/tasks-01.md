# Pipeline collaboration — Task Hierarchy Part 01

> **Index:** [Task Hierarchy](tasks-index.md).<br>
> **Result:** [Planning](../plan-index.md).<br>
> **Covers:** One Project-centered pipeline collaboration work item.

## Work Summary

### Agent-authored pipeline collaboration

| Work aspect     | Definition                                                                                                                                                                                                               |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Purpose         | Design, revise, validate, execute and monitor Gobble pipelines together through shared views and Chat.                                                                                                                   |
| Scope           | Core concepts/ownership, bounded source changes, Plan review, durable Start/Stop and candidate-aware Resume. Existing viewers support communication; no manual editor, chart builder, new viewer family or release work. |
| Output          | A qualified small pipeline journey with exact source/Plan/Run identities and clear User/Agent/Gobble ownership.                                                                                                          |
| Design          | Owner accepted the product philosophy; [detailed design](../../design.md) is proposed and must be reviewed before its stage executes.                                                                                    |
| Hierarchy paths | 1, 1.1, 1.2, 2, 2.1, 2.2, 3, 3.1, 3.2, 4, 4.1, 4.2, 5, 5.1, 5.2.                                                                                                                                                         |

## Task Hierarchy

Numeric paths express parent/child relationships, not execution order or agent assignments.

### 1 — Pipeline concepts and ownership foundation

> **Type:** `Group`

- **Work items:** Agent-authored pipeline collaboration.
- **Work:** A registered pipeline has a distinct identity and readable source binding; existing workspace behavior retains one writer and exact references.
- **Boundary:** No source mutations, Plan jobs or execution controls. Do not scaffold unused workflow services or rewrite frozen schemas.
- **Output:** A registered pipeline has a distinct identity and readable source binding; existing workspace behavior retains one writer and exact references.

### 1.1 — Define and expose pipeline identity

> **Type:** `Task`

- **Work items:** Agent-authored pipeline collaboration.
- **Work:** Define the core vocabulary and implement only the pipeline registration/read contract needed by the first consumer.
- **Boundary:** Service owns definitions and source bindings; future Change/Plan/Operation definitions are documented, not speculative executable scaffolding.
- **Output:** Concept/ownership records, focused active contract and a usable registered-pipeline read path.

### 1.2 — Separate presentation lifetime and qualify the baseline

> **Type:** `Task`

- **Work items:** Agent-authored pipeline collaboration.
- **Work:** Extract the smallest supported resource/presentation lifetime seam while retaining the durable workspace writer; qualify the relevant supported runtime baseline.
- **Boundary:** Preserve transaction/revision behavior and frozen formats. Native root-library support is a separate decision, not automatic scope.
- **Output:** Focused responsibility map, behavior-preserving extraction and honest host/Linux qualification evidence.

### 2 — Agent-authored changes and shared review

> **Type:** `Group`

- **Work items:** Agent-authored pipeline collaboration.
- **Work:** An Agent produces bounded pipeline source changes; User and Agent discuss exact candidate hunks and can accept a conflict-checked source revision.
- **Boundary:** No general shell, manual editor, binary editing, arbitrary file operations, dependency installation or analysis launch. Do not claim concurrent external-editor atomicity.
- **Output:** An Agent produces bounded pipeline source changes; User and Agent discuss exact candidate hunks and can accept a conflict-checked source revision.

### 2.1 — Own candidate changes and source acceptance

> **Type:** `Task`

- **Work items:** Agent-authored pipeline collaboration.
- **Work:** Implement bounded Agent proposals, immutable base/candidate retention, managed source acceptance and conflict/recovery outcomes.
- **Boundary:** Only registered source/config text with explicit authoring scope. One managed writer; unsupported external-writer coexistence uses isolated candidate handoff.
- **Output:** Service-owned change-set revision and accepted-source lifecycle with retained evidence.

### 2.2 — Discuss exact source changes

> **Type:** `Task`

- **Work items:** Agent-authored pipeline collaboration.
- **Work:** Connect Agent/User adapters to the same change service and add a read-only Changes view with exact hunk references in existing Chat.
- **Boundary:** Existing Panes, compact navigation and one composer; trusted User source acceptance is separate from Agent-authored content.
- **Output:** User and Agent can inspect, point to, refine and accept the same exact changes without an editor UI.

### 3 — Gobble validation and exact Plan discussion

> **Type:** `Group`

- **Work items:** Agent-authored pipeline collaboration.
- **Work:** Gobble validates the exact candidate in a bounded job and both participants discuss the resulting Plan or defects.
- **Boundary:** No analysis tasks or Start/Stop controls. A Plan is neither execution permission nor assumed executable JSON.
- **Output:** Gobble validates the exact candidate in a bounded job and both participants discuss the resulting Plan or defects.

### 3.1 — Own validation jobs and Plan artifacts

> **Type:** `Task`

- **Work items:** Agent-authored pipeline collaboration.
- **Work:** Add isolated candidate validation, runtime/config binding, structured failures and Gobble-owned prepared payload/admission identity.
- **Boundary:** No host evaluation of Project Go code, no Docker socket in the validation job, no ambient dependency installation.
- **Output:** A versioned Plan artifact and private Gobble prepared payload, or explicit validation/setup failure tied to source and runtime.

### 3.2 — Review the pre-run Plan together

> **Type:** `Task`

- **Work items:** Agent-authored pipeline collaboration.
- **Work:** Display the Gobble-produced Plan in an existing Pane and expose exact node references and source links only when proven.
- **Boundary:** Task definitions are distinct from Run expansions/attempts; no graph authoring or duplicated reuse logic.
- **Output:** A shared read-only Plan view and Chat references with clear current/out-of-date states.

### 4 — Authorized Start and conditional Stop

> **Type:** `Group`

- **Work items:** Agent-authored pipeline collaboration.
- **Work:** One exact authorized Plan starts a durably identifiable Run; Stop targets the observed owner; restart/timeout resolves accepted operations safely.
- **Boundary:** No Resume before candidate-impact qualification; no App scheduler, checkpoint replica, automatic execution from generic chat answers or blind launch retry.
- **Output:** One exact authorized Plan starts a durably identifiable Run; Stop targets the observed owner; restart/timeout resolves accepted operations safely.

### 4.1 — Establish engine admission and durable operation ownership

> **Type:** `Task`

- **Work items:** Agent-authored pipeline collaboration.
- **Work:** Implement admission of the exact prepared payload without privileged Project-code reevaluation, discoverable launch correlation, service acceptance/reconciliation and engine conditional Stop.
- **Boundary:** Gobble owns occupancy and actual execution; service owns command durability; closing a caller only ends waiting.
- **Output:** Recoverable command/lifecycle contract that distinguishes accepted, unknown, running, stop-requested and settled outcomes.

### 4.2 — Expose exact execution intent in App and Agent tools

> **Type:** `Task`

- **Work items:** Agent-authored pipeline collaboration.
- **Work:** Connect trusted User authorization and Agent requests to shared handlers; display exact Plan/target/effects and actual operation/Run states.
- **Boundary:** One right Chat and existing Run views; Agent cannot forge User authorization; no arbitrary paths/argv in renderer commands.
- **Output:** A reviewable Start/Stop journey with durable identity and truthful restart/recovery feedback.

### 5 — Failure, refinement and Resume collaboration loop

> **Type:** `Group`

- **Work items:** Agent-authored pipeline collaboration.
- **Work:** A researcher can move from a failed Run through exact evidence, Agent changes, a reviewed Plan and engine-owned reuse/Resume to a monitored result.
- **Boundary:** End at one qualified complete core journey; no package/release, new scientific viewers, Notebook editor or general extension/MCP host.
- **Output:** A researcher can move from a failed Run through exact evidence, Agent changes, a reviewed Plan and engine-owned reuse/Resume to a monitored result.

### 5.1 — Preview and admit candidate Resume

> **Type:** `Task`

- **Work items:** Agent-authored pipeline collaboration.
- **Work:** Expose engine-owned candidate-versus-prior-workspace reuse/rejection reasons and enforce their reviewed identity at Resume.
- **Boundary:** Reuse classifier remains in Gobble; changed source/runtime/input/workspace invalidates preview; preserve existing engine recovery rules.
- **Output:** Qualified Resume preview and command linked to exact candidate and prior Run state.

### 5.2 — Complete the shared pipeline feedback journey

> **Type:** `Task`

- **Work items:** Agent-authored pipeline collaboration.
- **Work:** Integrate failure/log selection, Agent explanation/refinement, Plan review, authorized execution and monitored outcomes in one Project scenario.
- **Boundary:** Reuse current references/views; honest unavailable/unknown states; preserve User layout, drafts, retained evidence and independent Agent turns.
- **Output:** Scenario/checklist, visual evidence for changed UI and tested core-loop handoff for owner review.
