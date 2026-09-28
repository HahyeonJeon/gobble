# P2B design review evidence

2026-09-09. Author: Codex in the current owner-led task. **Design/prototype review;
owner decision and production implementation are pending.** No external users,
independent reviewers or Agents participated in this turn's walkthrough.

## Scope and governing evidence

The owner accepted P2A-2 and asked to proceed. Their standing instruction requires
a sketch before implementation and result review between parts. This turn defines
P2B-1 and previews P2B-2; it does not implement proposal APIs, service publication,
new Agent permissions or runtime comparison. The previous source baseline is
[P2A-2 after.json](../../stages/p2a2-flow-discussion/after.json), source hash
`f33086ffa7ea66a1675e2ddb416cb12ef4456006933d1080f49ff80b048eee84`.

Identity and product requirements come from the owner's UI-first direction, the
accepted workspace and [visual collaboration proposal](../visual-pipeline-collaboration/README.md),
the [accepted flow styling](../../stages/p2a1-flow-polish/README.md), current flow
components/styles, and [P2A-2](../../stages/p2a2-flow-discussion/README.md). This
evidence applies to the current Project-centered App; it does not establish how
other researchers will understand source ownership or comparison coverage.

Before choosing a concept, the evidence classes were: source inspection for
ownership/contract constraints; official platform references for publication
limits; interactive author walkthrough for state/layout feasibility; pending
owner/representative-user task evidence for comprehension and usability. The
one-flow recommendation remains provisional, not a measured preference result.

## Design-relevant code findings

| Observed source                                                                                                                 | Consequence / disposition                                                                                                                                                                                                           |
| ------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [pipeline_inspection.go](../../../../pipeline_inspection.go): v2 display/settings and ordinal edge IDs                          | A visual diff alone misses processing behavior. Add Gobble review facts; do not match revisions by edge ordinal.                                                                                                                    |
| [engine plan](../../../../internal/engine/plan.go): command/script, params, environment, inputs and control                     | Enumerate every execution-bearing field. Unknown or unrepresented changes must be visible gaps.                                                                                                                                     |
| [Trim Galore](../../../../assets/modules/trim-galore/trim_galore.go) and [FastQC](../../../../assets/modules/fastqc/product.go) | Actual argv includes defaults/resources/extra arguments, not just visible settings. Verify complete canonical definitions. Both import Gobble, so verification must avoid reverse imports; use a small shared leaf where necessary. |
| [source retention](../../../../internal/appservice/pipeline_source.go)                                                          | Reuse explicit bounded manifests and containment. Retention is a useful foundation, not a completed managed-source store.                                                                                                           |
| [inspection lifecycle](../../../../internal/appservice/pipeline_inspection.go)                                                  | Existing `candidates` are check attempts. Proposal identity, check identity and adopted revision need separate lifetimes.                                                                                                           |
| [catalog writer](../../../../internal/appservice/catalog.go)                                                                    | A single existing writer can own conditional source-pointer publication and receipt. New crash guarantees need fault tests; do not claim them from rename alone.                                                                    |
| [Main policy](../../../../app/desktop/src/main/codex/policy.ts)                                                                 | Existing shared-view permission cannot silently become authoring permission. Add a scoped tool capability; adoption remains User-only.                                                                                              |
| [Workspace controller](../../../../app/desktop/src/main/workspace/controller.ts)                                                | Keep the existing single Workspace writer. Source lifecycle and semantic comparison belong outside this controller.                                                                                                                 |

## Interactive author walkthrough

Method: control the served local sketch through the in-App browser, inspect
accessibility/DOM state and screenshots. Default viewport observed: 1166 × 969;
explicit compact test: 800 × 850, then override reset. All values and outcomes are
simulated in memory. Screenshots show this prototype, not the Electron product.

| Scenario                                   | Observed result                                                                                                                                                              |
| ------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Default proposal                           | Changed quality and added branch visible; original alignment connection preserved; one flow and Chat readable.                                                               |
| Two-flow alternative                       | Content is 900 px in a 619 px canvas; horizontal scrolling hides part of Proposed. Reason to defer B as default.                                                             |
| Attach, switch side, select another change | Draft stays `Keep the alignment step as it is.`; attachment stays Proposal 3, 25 → 30. Added-step selection switches to Proposed.                                            |
| Revised proposal 4                         | Detail becomes 28; earlier attachment remains 30 and gains Earlier proposal.                                                                                                 |
| Unexplained change                         | Known changes remain visible with a third coverage-gap entry; Use is disabled.                                                                                               |
| Current version changed                    | Base v2 remains identifiable; update-proposal action available; Use disabled.                                                                                                |
| Checking / invalid                         | Candidate adoption disabled; current flow and draft available.                                                                                                               |
| Ask for revision                           | Reference added to the existing composer without replacing draft; composer receives focus.                                                                                   |
| Adoption interrupted                       | Check outcome produces the simulated recorded completion; action becomes disabled and history labels become Previous/Accepted. This tests UI state only, not crash recovery. |
| First managed version / Use                | Ownership explanation visible at the decision; local completion says no analysis started. No files or service were touched.                                                  |
| New pipeline                               | Example data reference and goal action use existing Chat; premature adoption controls hidden. This is entry-flow exploration only.                                           |
| Compact / keyboard                         | Workspace/Chat retain draft; Tab reaches Proposed and Enter activates it; Step list supplies a non-spatial route. Default compact flow fits its canvas after correction.     |

Corrections made during review: invisible preview-toolbar labels; missing input
arrow; ambiguous Current labels after adoption or conflict; an added-step selection
left on the old side; a new data attachment incorrectly called a proposal; clipped
compact flow; mobile switch events bubbling through the container and losing the
pressed state. Each was corrected in prototype files and the relevant path revisited.

Retained captures: [A](concept-a.jpg), [B](concept-b.jpg),
[incomplete](incomplete.jpg), [first managed version](first-managed-version.jpg),
[compact](compact.jpg). Prototype syntax/format and final source preservation are
recorded in [after.json](after.json). Native service/App suites are not rerun for
this design-only change; historical P2A-2 test counts are not current results.

## Seven design activity results

Each activity has one disposition. “Open” means the scoped evidence does not close
that activity for product acceptance. Common scope, sources, date, actor and trace
are defined above; the rows add activity-specific evidence and return conditions.

| Activity / result                                | Disposition                       | Method, evidence and decision                                                                                                                                                                                                                                  | Limit / reopen condition / next owner                                                                                                                                                                                                                 |
| ------------------------------------------------ | --------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Discovery evidence reviewed                      | Performed for the current subject | Codex inspected current code, accepted design and primary platform contracts; findings above and source-publication section in design.md.                                                                                                                      | Open on human context: additional research workflows may invalidate the two-recipe starting scope. Owner feedback or contradictory engine behavior reopens; implementation owner verifies facts.                                                      |
| Design requirements accepted for current subject | Reused current evidence           | Owner's accepted Project/Pane/Chat and UI-first direction, 2026-09-09 current checkpoint, plus linked stage docs constrain the design. Applies to the same owner and App context; source-free discussion is falsified if a completion task requires Go review. | Existing requirements reused; new ownership/handoff requirements remain proposed. New data/root constraints or owner rejection reopens; owner decides, implementation owner reconciles.                                                               |
| Concept decision recorded                        | Performed for the current subject | Codex built two materially different comparison models; measured visible width/scroll and reviewed captures. A is recommended; B deferred.                                                                                                                     | Open pending owner tasks. Lower switch cost/readability is a hypothesis; large or removed-step graphs may favor B. Owner/representative-user evidence can reverse the choice.                                                                         |
| Prototype evidence reviewed                      | Performed for the current subject | Codex exercised the state and layout scenarios above, corrected defects and retained captures. Adequate to present the sketch for review.                                                                                                                      | Open for native behavior, assistive-input coverage, large graphs, network/storage failures and real content. Production implementation owner must reconcile those before completion.                                                                  |
| Test evidence reviewed                           | Performed for the current subject | Codex prepared the representative-user tasks below and presents this runnable artifact to the owner. Test preparation is complete; no participant responses or human completion evidence collected.                                                            | Open; empirical testing has not occurred. Author walkthrough is insufficient to accept novel adoption/ownership comprehension. Owner task feedback and a subsequent representative researcher session must evaluate it before broader product claims. |
| Design and implementation obligations reconciled | Performed for the current subject | Codex traced each proposed responsibility to the current owner, found inspection/command and import-cycle gaps, and added plan acceptance cases. Design-to-code reconciliation is documented, not implemented.                                                 | Open until actual Gobble/service/Main/UI behavior passes plan cases. Constructor drift, unsupported source/data binding or lost historical evidence requires revising the design before enabling adoption.                                            |
| Post-release design review closed                | Not applicable with exact reason  | This artifact is an unreleased, local, in-memory prototype with no users, publication or telemetry; there is no release outcome to measure. No post-release maintenance decision can change this scoped draft through absent production signals.               | No product-release conclusion. Guardrails below govern the future consequential feature; implementation owner must schedule a bounded review after a pilot. Release/pilot or real data adoption reopens the activity.                                 |

## Human task review and future measures

Ask the owner now, then a representative bioinformatics researcher using an owned
fixture during implementation acceptance, to:

1. Identify the changed threshold and explain whether alignment receives a
   different input when the quality report is added.
2. Point at that change, ask a question in Chat, then open a revised proposal and
   explain what the earlier attachment still refers to.
3. Explain why Use is unavailable in incomplete/stale states and choose the next
   action without source-code inspection.
4. Explain which analysis version becomes current, whether imported files change,
   and whether analysis starts after Use.

Observe pointer and keyboard/list routes; native screen-reader/focus testing is
still owed. Success is accurate understanding with recoverable action, not fast
clicking or high adoption rate. Wrong source ownership, lost draft/reference,
unexplained behavior accepted, or confusing adoption with execution are guardrail
failures; reduce scope or revise the UI when observed.

For a future pilot, the implementation owner reviews these tasks and volunteered
failure reports with the Project owner after the first three fixture sessions.
Use bounded manual notes; collect no background source contents or Chat telemetry.
The consumer is the owner deciding the next part. Record improve/no-change and
maintenance ownership explicitly then. This document starts no monitoring or
automation and makes no claim of usability across users, platforms or arbitrary
pipelines.
