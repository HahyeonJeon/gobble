# P5B design review evidence

Date: 2026-09-13 KST. Actor: implementation/design agent. Subject: a proposed same-Run continuation interface for a cleanly stopped, sealed linear analysis. Owner decision pending. This report separates repository inspection, an operated HTML fixture and future production/user evidence.

## Evidence classes and identity

Source inspection tests the claim that existing ownership/admission boundaries can safely support the proposed semantics. Interactive expert walkthrough tests whether required states and controls are represented and reachable. Neither establishes what representative research users understand, nor whether the engine safely resumes. Representative task-based comprehension evidence and real-engine qualification remain required for broad acceptance.

The authority chain and compared concepts are recorded in [README](README.md). Existing live Flow/Chat conventions and owner instructions constrain hierarchy, density, connectors, colors and a single composer. The sketch uses repository-native HTML/CSS to expose exact interactions rather than a raster illustration. No new project-wide design system is prescribed.

## Seven activity results

Each result has one disposition. “Performed” below names the performed review work; an open result does not claim that missing participant or released-product evidence exists.

1. **Discovery evidence reviewed — Performed for the current subject.** Actor: design agent; owner: maintainer. Inputs: local engine/admission/checkpoint/Stop/reuse code and completed P5A record. Method: trace original Start, latest ownership, attempt creation, content checks and app-service storage. Evidence/trace: [architecture source table](architecture.md), baseline hashes. Decision: separate continuation admission and forbid old-format conversion in the first slice. Counterevidence: generic CLI Resume already exists, but permits broader behavior and rejects prepared Runs. Uncertainty: new format and true Linux execution remain unqualified. Route: engine contract/qualification phases. Result open until compatibility and execution assumptions are verified; reopen for another engine/workflow shape.

2. **Design requirements accepted for the current subject — Performed for the current subject.** Actor: design agent; owner: product maintainer. Inputs: owner's shared-UI philosophy, stage approval requirement, existing compact Flow/Chat, source constraints. Method: define saved Run versus Current, whole-task restart versus process resume, reuse versus observed completion. Evidence/trace: README scenarios and architecture vocabulary. Decision: proposed requirements, owner acceptance pending. Counterevidence: narrow supported scope may disappoint users expecting universal Resume; explicit older-engine and block states address discovery, not capability. Limit: only the sealed linear workflow. Route: owner review and plan phase 1; reopen if terminology or scope changes.

3. **Concept decision recorded — Performed for the current subject.** Actor: design agent; owner: maintainer. Inputs: same scenarios and authority chain. Method: compare in-Run review against sequential guided review, including final Chat confirmation. Evidence: [A](evidence/a-ready.png), [B](evidence/b-guided.png), interactive concept toggle. Decision: recommend A provisionally. Counterevidence: B may improve novice sequence comprehension. Uncertainty: no representative comparison. Route: owner choice then task sessions; reopen for confusion about saved design, kept output or execution consequence.

4. **Prototype evidence reviewed — Performed for the current subject.** Actor: design agent. Inputs: design fixture and scenarios below. Method: operate the in-app browser with pointer/keyboard, inspect accessibility tree and screenshots, test compact region switching. Evidence: walkthrough table and image files. Decision: candidate is reviewable after correcting misleading unchecked/blocked labels and reference preview language. Limit: simulated state/data, no backend, no provider, no reload persistence, no actual Stop or execution. Route: real Electron tests in phase 4/5. Result open for native accessibility, slow-check behavior and actual restoration; reopen on mismatch with real state or keyboard/focus defects.

5. **Test evidence reviewed — Performed for the current subject.** Actor: design agent; owner: maintainer. Scope performed: participant-test readiness and evidence-gap review only; no representative-user sessions occurred. Inputs: consequences, compatibility sensitivity, two candidates and tasks in the plan. Method: identify comprehension questions and harmful success proxies before acceptance. Evidence: [plan review tasks](plan.md). Decision: representative validation remains OPEN; expert/owner walkthrough is not substitute evidence. Counterevidence: established Flow familiarity does not prove understanding of continuation. Dependency: representative research participants and keyboard/assistive-input users. Route: bounded task sessions before broad acceptance; reopen immediately for wrong-design expectations or unintended work. No general usability claim is made.

6. **Design and implementation obligations reconciled for the current subject — Performed for the current subject.** Actor: design/implementation agent; owner: engine/App maintainer. Inputs: source findings and proposed UI. Method: map each state/action to renderer, Main, native and engine ownership; trace crash/replay/Stop and reference identity. Evidence: architecture diagrams and staged plan. Decision: handoff proposal complete enough for owner review; implementation reconciliation remains open until code demonstrates the invariants. Counterevidence: initial admission currently doubles as active lease, so direct wiring contradicts intent. Route: format-3 contract then engine before UI enablement. Reopen on API/state mismatch or implicit permission.

7. **Post-release design review closed — Not applicable with exact reason.** Actor for disposition: design agent; future owner/consumer: maintainer. Subject is an unshipped, non-executing design fixture; there is no released continuation cohort or telemetry to measure, so a post-release improvement/no-change decision cannot affect this pre-implementation checkpoint. Inputs: current design status and harm analysis. Evidence: no production delta and simulated labels. This disposition does not waive the compatibility/execution risk: those route to pre-release participant and engine qualification above. Future measures, bounded pilot window, guardrails and reopen triggers are in the plan. No post-release or Maintenance closure is claimed; this activity becomes applicable when a pilot is released, requiring explicit improvement/no-change and Maintenance decisions.

## Operated fixture scenarios

In-app browser, default 1280 × 720 and temporary 820 × 720 viewport; override reset afterward. This is an expert walkthrough, not an automated production regression suite.

| Scenario                          | Observed fixture outcome                                                                                                                                   |
| --------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Ready, concept A                  | Kept/restarted Flow labels, selected detail and one Chat confirmation visible; screenshot `a-ready.png`                                                    |
| Concept B                         | Eligibility hides Flow, then planned work, then focused Chat confirmation; keyboard Enter advances and explicitly enters simulated Running                 |
| Explicit Resume / completed scene | Same Analysis 12, Trim attempt 1 reused, quality check attempt 2; completed state changes its label; no second Resume action                               |
| Stopping                          | Resume disabled; status refresh reaches stopped-but-unchecked; Check continuation produces a review                                                        |
| Review changed → recheck          | No executable plan; missing Trim output blocks, downstream marked blocked, Resume disabled                                                                 |
| Reference before recheck          | Draft text and review 4 attachment survive recheck; preview still shows old reviewed action/attempt, marked Earlier review                                 |
| Earlier engine                    | Continuation unavailable, disabled Resume; inspection/new-analysis explanation                                                                             |
| Current newer                     | Explicitly retains saved Run design/data; Current changes not applied                                                                                      |
| Lost acknowledgement              | Confirmation pending, status action only; simulated status resolution returns same Analysis 12 continued                                                   |
| Compact Workspace / Chat          | Flow fills work area; attaching moves to Chat and focuses draft; switching retains text/reference; screenshots `compact-workspace.png`, `compact-chat.png` |
| Reference dialog keyboard         | Enter opens readable captured-review preview; Escape closes and returns focus to reference button                                                          |
| Send / Agent point                | Fixture Send clears draft but does not Resume; Agent link explicitly shows/marks Trim in Workspace                                                         |

## Fixes from the walkthrough

- Unchecked completed status no longer asserts verified reuse in the step detail.
- Stale/missing/pending states no longer show an executable reuse/restart count; blocked downstream is explicit.
- Current has a Current badge; accepted continuation uses kept/new-attempt wording instead of a future plan count.
- Running task capture uses its observed attempt/snapshot rather than the stopped review identity.
- Captured-reference preview uses ordinary analysis/step/attempt labels; technical identifiers stay behind the view.

Fixture limits: scenario controls jump among illustrative states and are not a persisted lifecycle; review IDs/digests are fake display fixtures; reload resets draft/history. The fixture has no slow-content-check progress, active-epoch Stop interaction or earlier-history browser. Those are explicit production qualification obligations; it cannot prove atomic admission, output preservation, duplicate suppression, authenticated Agent access or crash recovery. Source syntax/format and production hash checks are recorded in `evidence/checks.json`. No production TypeScript, Go or Electron tests were rerun because this checkpoint changes only design documents and the sketch.

## Remaining decision

Owner review of concept A and the scoped plan is next, as requested for each stage. This is a concrete design checkpoint, not an additional skill-imposed authorization demand. Representative comprehension, exact engine compatibility, read-only review and execution safety remain open evidence requirements during implementation and before broad acceptance.
