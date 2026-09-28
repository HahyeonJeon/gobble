# P5 — Discuss a result, review a change, run again

Status: P5A design and implementation sequence approved by the owner on 2026-09-12. See the [P5A implementation record](../../stages/p5a-feedback/README.md) for construction, storage refinement and verification. This page retains the approved design; P5B remains a separate checkpoint; its [continuation design proposal](../p5b-continuation/README.md) is now ready for owner review.

The User and Agent share a Run Flow, point to a specific result or failure, review an Agent-authored change in the existing Change spotlight view, and explicitly start a new analysis. The old Run keeps its own design and evidence. All product copy is English.

Open [the interactive sketch](sketch.html), [architecture and code findings](architecture.md), [implementation sequence](plan.md), and [design review evidence](review.md).

## Recommended scope

**P5A completes the feedback loop using a linked new Run.** It connects existing Run/task/log evidence, Agent authoring, checked visual proposals, preparation and Start. It adds no second composer, scientific editor, general recovery wizard or output cache.

**P5B separately qualifies Resume of the same admitted design.** Its first scope is an explicitly settled, stopped Run with an unchanged sealed preparation. General failed-task retry, interrupted-owner recovery and changed-design reuse need further engine qualification. They must not appear as enabled recovery actions merely because ordinary CLI Resume exists.

The sketch includes a future Resume explanation to make this distinction reviewable. It is not a claim that Resume is implemented or permission to implement it now.

## What the User sees

1. Open a failed or completed analysis. Its Flow shows the design actually executed, with status words and icons as well as color.
2. Select a step; details appear below. **Discuss this step** captures the exact observed task and puts it in the existing Chat draft. Sending remains the User's action. Logs are separate, explicit bounded references; selecting a step does not send arbitrary files or all logs.
3. The Agent distinguishes observed facts from a possible explanation. A failed process alone does not establish a scientific cause or justify a setting change.
4. For a supported change, the Agent proposes edits against the checked Current design. Change spotlight shows the changed setting above and Current → Proposed values below. A compact origin chip opens the earlier Run evidence.
5. **Use this version** adopts the checked proposal; it does not start work. Preparation and data checks then explain **New analysis · all steps run again · previous results kept**.
6. **Start new analysis** appears as the existing typed action in Chat. After admission, the new Run has an **Following Analysis 12** link. That link records the reason for this new analysis; it does not assert reused output or improved science.

Sample scenes are illustrative fixtures. The change example starts with an explicitly configured Minimum length of 30 bp and proposes 20 bp after the User says shorter reads are acceptable. It is not an inferred fix for the separate failure example. Unknown tool defaults remain unknown in production.

## Two action models considered

| Model                | Behavior                                                                                                                                               | Benefit                                                                   | Cost / failure mode                                                                                           | Decision                                                                         |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| In-context follow-up | Keep the Run, proposal and preparation as existing surfaces. Chat contains exact references and typed approvals. A small origin link follows the work. | Fits the accepted Project workbench and supports discussion across views. | Users can lose origin when Current advances; explicit provenance and stale-base states are essential.         | Recommended for P5A; owner review pending.                                       |
| Guided recovery path | Enter a dedicated center-panel sequence: evidence → diagnosis → change → execution; Chat stays alongside it.                                           | Makes sequence and incomplete steps visible.                              | Suggests all failures have a repair; adds a second navigation/state owner and constrains parallel discussion. | Retain as an alternative only if scenario review shows users losing their place. |

Changing the design in the same Run was also considered. Ordinary engine Resume supports graph changes, but the App's current admission and retained Flow contracts do not. For P5A, keep new designs in new Runs. Cross-Run reuse is not part of this decision.

## Design identity and interaction

Authority: the accepted [visual pipeline direction](../visual-pipeline-collaboration/README.md), [B / Change spotlight decision](../p2b-visual-change-review/alternatives-v2/README.md), and [P4 actual interface evidence](../../stages/p4-run-control/verification.md), followed by existing PipelineFlow and Run-launch styles. This constrains compact headers, central work, right Chat, rounded orthogonal edges, stage icons, and status colors. No project-wide design system is introduced.

Color always accompanies a word/icon. Red means observed failure; amber means changed design; teal means selection or action. A downstream unchanged step may be affected by a change without becoming a changed step. Fresh-run steps say “Runs again,” never “Reused.” Tool exit and scientific quality assessment stay separate.

At narrow widths the sketch uses Workspace / Chat switching and retains the draft. All required actions are named buttons, available before hover and keyboard reachable. Focus moves to the composer after attachment and to the new surface after navigation. Screen readers receive an action announcement; actions do not depend on animation. No new tray action, notification, global shortcut or window-chrome operation is needed; these must not become alternative execution authority. Existing menus continue to provide ordinary navigation. Desktop accessibility and assistive-input verification remain implementation obligations, not claims from this prototype.

## Decisions requested

Approve P5A's in-context flow, evidence and lineage boundaries, and implementation sequence. Keep P5B as a separate design/qualification checkpoint after P5A's result review. This approval does not add editors, viewers, automatic retries, broader scientific workflows or packaging.
