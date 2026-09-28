# P5B — Continue a stopped analysis

Status: owner approved the recommended design and scope on 2026-09-13 (Asia/Seoul). The [P5B-1 history foundation](../../stages/p5b1-history/README.md) and [P5B-2 engine continuation](../../stages/p5b2-continuation/README.md) are implemented. [P5B-3 native/Main transport](../../stages/p5b3-transport/README.md) is also implemented. Shared Flow/Chat integration and actual tool qualification remain pending.

A User and Agent inspect the same saved Run Flow, discuss exactly which results can be kept and which task must restart, then the User explicitly confirms continuation. The Run keeps its identity, saved design, data and earlier attempts. A stopped task restarts from its beginning; this is not a process checkpoint.

Open the [interactive sketch](sketch.html?scene=ready&concept=context), [architecture and source study](architecture.md), [implementation sequence](plan.md), and [review record](review.md). The sketch is entirely simulated. Scenario buttons belong to the design study, not the product.

## Recommended interaction

1. Open the stopped analysis. Until Stop settles, show **Stopping** with status refresh only.
2. **Check continuation** asks Gobble for a read-only review. Completed status alone never means reusable output.
3. The center Flow shows **Can reuse · Attempt 1** and **Will restart · Attempt 2**. Selecting a step explains its decision below the Flow.
4. **Discuss this step** adds an exact saved reference to the existing Chat draft. It does not send, execute or grant permission. Agent references point to the same immutable review.
5. The existing Chat action card states **Keep 1 result · Restart 1 step**, saved design/data, preserved attempt history, and the whole-task restart consequence. **Resume analysis** is the only execution confirmation.
6. Gobble checks the reviewed facts again under its execution lock. If anything material changed, it refuses and requests another check. If accepted, the same analysis shows the new attempt alongside the reused result.

Current can advance independently. **Current is newer than this Run** explains that its changes will not be applied. To use them, the User returns to the existing proposal → preparation → new analysis flow.

## Two concepts

These A/B labels are local to P5B. They do not replace the previously approved **B / Change spotlight** for design diffs.

| Concept                               | Action model                                                                      | Benefit                                                                                         | Cost and reopen condition                                                                                                                 |
| ------------------------------------- | --------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| **A — In the Run view (recommended)** | Review directly in the Flow; selected detail below; confirmation in existing Chat | Keeps results and discussion visible, supports questions in any order, adds no navigation owner | Users may miss the scope of continuation; reopen if scenario participants confuse saved Run with Current or Resume with mid-task recovery |
| B — Guided review                     | Center sequence: eligibility → planned work → confirm in Chat                     | Gives an explicit order and reveals progress                                                    | Hides Flow during the first step and adds navigation; prefer only if it measurably improves understanding without disrupting discussion   |

Evidence class for the recommendation is source study and expert walkthrough against the owner's accepted workbench direction. It is a reversible design proposal, not a claim of representative-user validation.

## First implementation boundary

Only App-created, continuation-capable, cleanly stopped single-end **Trim Galore → FastQC** Runs with unchanged sealed design, staged data, exact engine/tool bindings and verified complete outputs qualify. A completed task can keep its original attempt; the stopped task gets a fresh attempt and separate logs.

Missing or changed completed output, ambiguous published output, changed input/tool identity, active owner, failed/interrupted execution, unsupported graph or older admission format blocks this path. Gobble must not quietly turn a reviewed reuse into more work. Changed designs use a separate new analysis. No arbitrary retry, cross-Run cache, data editor or new viewer is introduced.

**Compatibility:** the first slice creates new Runs with the new engine and continuation-aware history from their initial Start. Earlier P4/P5A Runs remain inspectable but do not gain Resume through an in-place conversion. Existing identity checks bind Resume to the original installed engine; replacing its binary is not a valid migration. New readers must retain supported older inspection. This limit must be visible as **This analysis uses an earlier engine**.

## Visual identity and scope

Authority: existing [Project visual pipeline direction](../visual-pipeline-collaboration/README.md), [Change spotlight](../p2b-visual-change-review/alternatives-v2/README.md), [P4 verified interface](../../stages/p4-run-control/verification.md), and the owner's compact center-Flow/right-Chat brief. Keep English UI, small headers, rounded orthogonal connectors, stage icons and words alongside colors. No new composer or top-level Pane type.

At compact widths, Workspace / Chat switching preserves the draft and reference. Buttons are visible before hover. Pointer and keyboard operate the same scoped actions; no execution command is added to tray, notifications, global shortcuts or window chrome. Existing navigation/menu behavior remains the route to the Run view. Native accessibility and assistive-input qualification are obligations for implementation. Motion is unnecessary for this state comparison.

## Review requested

Approve the recommended in-Run review, the limited stopped-Run scope, the explicit older-engine boundary, and the staged implementation plan. This is the owner's requested pre-implementation design checkpoint. Representative-user comprehension and actual engine qualification remain open evidence requirements; the prototype does not settle them.
