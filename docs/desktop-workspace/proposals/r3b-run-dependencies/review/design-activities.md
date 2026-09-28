# R3b design activity record

Reviewed 2026-09-08 by Codex. Subject: the proposed Run dependency workflow and its next
**R3b1 qualification checkpoint**, not acceptance of production UI or the full R3b family.
The project owner makes the concept/scope decision. This record follows Desktop Interface;
the approval checkpoint comes from the user's requested before/after stage workflow.

## Identity and evidence authority

The [Shared Research Workbench design](../../shared-research-workbench/design.md), accepted
Project/pane/Chat requirements in this task, and [live App tokens](../../../../../app/desktop/src/renderer/styles/tokens.css)
constrain the proposal. [R3a3](../../../stages/r3a3-agent-observation.md) and its
[live reference screenshot](../../../stages/r3a3-review/live-agent-reference.png) provide the
current interaction baseline. These establish continuity, not usability evidence for a new graph.
No project-wide identity document is created.

The claims and their required evidence classes are distinct:

| Claim under consideration                                                  | Evidence class needed for the decision                                                        |
| -------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| What an edge/group/attempt means and which owner supplies it               | Current local source and contract inspection.                                                 |
| A renderer can provide read-only layout and keyboard paths                 | Official primary documentation for candidacy; a pinned Electron qualification for acceptance. |
| The proposed flow preserves a draft/reference and keeps controls reachable | Interactive synthetic prototype and visual/keyboard inspection; later native integration.     |
| Research users understand the targets and can complete the task            | Observed representative-user tasks. Owner preference, code and automation cannot substitute.  |

## 1. Discovery research — Discovery evidence reviewed

**Disposition: Performed for the current subject.**

- Actor/owner, subject/scope: Codex; available Run facts, renderer candidates and current workflow.
- Inputs/method/evidence: inspect monitor, presenter, service and inspector; compare official
  React Flow/Dagre documentation. Exact sources and consequences are in [research](../research.md).
- Decision state: sufficient to propose R3b1; research-user needs and renderer behavior remain open.
- Counterevidence/failure/uncertainty/limits: missing edges are currently normalized to empty; a Plan
  DAG is separate; instance-level routing is unavailable. No current representative-user graph study.
- Dependencies/routes: R3b1 source-envelope and renderer qualification; owner may prioritize document
  reading instead. No execution changes follow from this evidence.
- Trace/reopen/evidence location: reopen if source fields lack availability/coherence or users prioritize
  papers/reports; [research](../research.md) and [design](../design.md).

## 2. Problem framing — Design requirements accepted for the current subject

**Disposition: Performed for the current subject.**

- Actor/owner, subject/scope: Codex proposes, project owner reviews; exact target discussion in one Project.
- Inputs/method/evidence: map the user's central work/Chat requirements to group → instance → logs;
  define ownership, semantic targets and immutable capture in [design](../design.md).
- Decision state: proposed requirements, pending owner acceptance; the result name is not an acceptance claim.
- Counterevidence/failure/uncertainty/limits: graph context may be less useful than broader file coverage;
  an arrow cannot establish failure cause. No general graph/Notebook feature is implied.
- Dependencies/routes: existing evidence and temporary-reference owners; R3b1 validates exact APIs.
- Trace/reopen/evidence location: reopen if a proposed grouping invents execution facts or changes an old
  selector/hash; definitions, diagram and compatibility section in [design](../design.md).

## 3. Concept alternatives — Concept decision recorded

**Disposition: Performed for the current subject.**

- Actor/owner, subject/scope: Codex recommends A; project owner chooses navigation scope.
- Inputs/method/evidence: compare A, one Run Surface with modes plus logs, against B, separate graph/list
  Surfaces, using the same fixture and target meanings in [sketch](../sketch.html).
- Decision state: A recommended, not finally selected. B is a separate state/placement model, not a style variant.
- Counterevidence/failure/uncertainty/limits: B provides more space for comparison; A's embedded member area
  must remain bounded. The three-node sketch cannot prove behavior at realistic graph scale.
- Dependencies/routes: owner walkthrough, then R3b1 qualification; keep pane placement and semantic selection separate.
- Trace/reopen/evidence location: reopen if persistent graph/table comparison proves more useful than direct
  logs; [A](concept-a.png), [B](concept-b.png), [research](../research.md).

## 4. Prototyping — Prototype evidence reviewed

**Disposition: Performed for the current subject.**

- Actor/owner, subject/scope: Codex; synthetic HTML proposal at 1280×840, 900×650 and 640×840.
- Inputs/method/evidence: exercise eight target/navigation/recovery scenarios in Chrome, inspect captured
  screenshots, correct defects and repeat affected scenarios. [Results](sketch-checks.json).
- Decision state: bounded sketch checks pass; production and general usability acceptance remain open.
- Counterevidence/failure: initial reference locking reduced graph text opacity; compact layout clipped
  controls; a generic synthetic log reused the wrong instance text. All three were corrected in the sketch.
- Uncertainty/limits: no real layout engine, pan/zoom, large graph, screen reader, Electron 150% zoom,
  engine read or Agent call. Camera restoration is an implementation obligation, not a demonstrated feature.
- Dependencies/routes: R3b1 qualifies renderer and bounds; R3b2 owns native pointer/keyboard/focus and
  assistive-input mechanics with Web Interaction and Electron testing guidance.
- Trace/reopen/evidence location: reopen for unreadable target/Return, stale substitution or focus loss;
  [check harness](check-sketch.cjs), [reference](concept-a-reference.png), [compact](compact-900.png).

## 5. Representative-user testing — Test evidence reviewed

**Disposition: Performed for the current subject.**

- Actor/owner, subject/scope: Codex reviewed the evidence gap and prepared a participant task; project owner
  is invited to perform the bounded walkthrough. **No participant test has been run for R3b.** The performed
  result is the evidence review; representative-user testing itself remains open.
- Inputs/method/evidence: distinguish earlier user preferences and R3a3 integration evidence from observations
  of people using this graph. Prepare the [walkthrough and success criteria](../implementation-plan.md).
- Decision state: returned open for insufficient participant evidence. A scope approval can start R3b1;
  it cannot establish that research users understand the final UI.
- Counterevidence/failure/uncertainty/limits: the author can complete a designed task; that cannot prove another
  person's comprehension. One owner walkthrough would cover that participant and context only.
- Dependencies/routes: owner feedback now; representative tasks and accessibility conditions before accepting
  consequential UI choices in R3b2. No participant recruitment or messaging has been initiated.
- Trace/reopen/evidence location: record task outcome and arrow/group interpretations in the sequential plan;
  mistaken causal reading, wrong attempt choice or lost draft reopens the concept.

## 6. Design–implementation collaboration — Obligations reconciled

**Disposition: Performed for the current subject.**

- Actor/owner, subject/scope: Codex design/code inspection; owner reviews the proposed checkpoint boundaries.
- Inputs/method/evidence: compare design intent against the current App's service, aggregate, evidence,
  reference and RunView responsibilities; locate missing availability semantics before choosing an API.
- Decision state: handoff defined for R3b1, not product implementation acceptance.
- Counterevidence/failure/uncertainty/limits: adding group metadata could accidentally change old revision
  hashes; no concrete version is allocated before the frozen-contract inventory. Library behavior is unqualified.
- Dependencies/routes: [sequential plan](../implementation-plan.md) assigns pure projection/resolution,
  one renderer adapter and existing stores; implementation findings must revise this design before promotion.
- Trace/reopen/evidence location: compatibility, source coherence and target scope are explicit reopen
  triggers in [design](../design.md); [baseline verification](baseline-verification.json) checks existing source hashes.

## 7. Post-release improvement — Post-release design review closed

**Disposition: Not applicable with exact reason.**

- Actor/owner, subject/scope: Codex; an unshipped synthetic proposal with no installed R3b runtime.
- Inputs/method/evidence: inspect the bounded file changes and release state. No R3b product usage or
  telemetry exists; production post-release observation cannot be performed for this subject.
- Decision state: no release/no post-release runtime change; maintain the accepted R3a3 baseline.
  This is not a no-change decision for a released graph or permission to close future measurement.
- Counterevidence/failure/uncertainty/limits: future adoption can expose confusion absent from the sketch.
  Current source/compatibility risks are handled before implementation; there is no post-release risk trigger yet.
- Dependencies/routes: after eventual release, owner consumes a dated, bounded review of target accuracy,
  successful Show/Return and wrong-attempt incidents. No monitoring or automation is created now.
- Trace/reopen/evidence location: reopen upon first R3b production use. Intended success means completing the
  correct task with retained context; guard against treating faster clicks or fewer questions as comprehension.
  [Plan](../implementation-plan.md) requires an explicit improvement/no-change and maintenance decision then.

## R3b1 outcome and scope correction — 2026-09-08

The owner approved R3b1 and explicitly removed CSV chart creation. The implemented retirement,
new ownership/API boundaries, native renderer comparison and limits are in the
[stage record](../../../stages/r3b1-foundation.md). Minimal HTML/SVG was selected after both
candidates passed the bounded technical qualification. Product graph navigation, camera/Return,
Agent pointing and representative owner interaction review remain R3b2/R3b3 work. This outcome
does not convert the sketch or technical harness into acceptance of the production graph UI.
