# Revision 2 — comparison design review

2026-09-09. Actor: Codex, current owner-led task. Scope: three local interactive
comparison concepts after the owner rejected the original toggle's intuitiveness.
Decision: replacement design awaiting owner review. No production code, runtime
contract, source adoption mechanism or Agent capability changes in this revision.

## Evidence and choices

Before choosing a layout, use the owner's current negative feedback as direct
evidence to reopen the original recommendation. Use prototype/DOM/screenshot
walkthrough for layout and state feasibility, and pending owner task feedback for
comprehension. No usability preference is established by the author walkthrough.

Identity authority: accepted Project-centered workspace and right Chat, followed
by the accepted flow polish and current App flow styling. The new brief explicitly
asks for simultaneous before/after context or more visible changes. Rounded
orthogonal edges, restrained green surfaces, compact controls, English UI and the
existing composer remain. This revision adds numbered change correspondence and
semantic previous/replacement color, not a new design system.

Alternatives differ in information hierarchy: A presents two complete versions;
B navigates one flow and inspects one change; C reads an ordered visual diff. Keep
the same example and Chat to isolate those differences. The comparison table and
captures in README.md document the tradeoffs; none is approved yet.

## Author walkthrough

Method: operate the authored local page through the in-App browser, inspect
semantic UI state and DOM geometry, and inspect screenshots. Actual default window
was **1280 × 720**. A compact **800 × 850** override was used for responsive testing
and reset afterwards. This is browser prototype evidence, not native Electron or
representative-user qualification.

| Path / question                               | Observed evidence                                                                                                                                                                                                                          |
| --------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| A: both complete flows visible?               | Initial fixed graph aspect ratio hid the new branch below the visible area. Adjusted the two regions to available Pane height; final default review body has 437 px content in 437 px, each graph region 143 px without vertical overflow. |
| B: where / what changed?                      | Selecting numbered Quality check updates the lower view to its prior absence and new FastQC input/output/parallel route. Quality selection shows 25 and 30 in paired labeled columns.                                                      |
| C: code-free diff readable in the viewport?   | Final default review body has no overflow; property row and added-branch neighborhood visible together. Expanded context deliberately becomes scrollable.                                                                                  |
| Same selection in A's two versions?           | Selecting quality highlights both Current and Proposed Trim adapters; matching layouts are fixed for this fixture. No general graph correspondence claim.                                                                                  |
| Attachment survives selection/layout changes? | Attached change 2 in B, switched to A and selected change 1, then switched to C: attachment still described change 2; draft stayed “Keep the alignment step as it is.”                                                                     |
| Keyboard and context                          | Enter on Show unchanged details exposes two unchanged rows; focus remains on the context control. Buttons/list of changes provide a non-spatial selection route. Full assistive-input coverage remains untested.                           |
| Incomplete / stale                            | Use disabled; incomplete warning explains an additional unrepresented change; stale columns say Base version rather than Current.                                                                                                          |
| Simulated adoption                            | Columns become Previous / Accepted; completion message says no analysis started; repeated Use disabled. No persistence or engine operation occurred.                                                                                       |
| Compact A and B                               | A content fits 524 px body; footer button remains within 850 px height. Discuss switches to Chat, focuses the existing draft and retains its text. Returning to Workspace preserves selection; B detail ends above the footer.             |

Corrections after inspection: clipping of vertically stacked graphs and B details;
excessive minimum sizes; ambiguity of signed values; a prior absence styled like a
removed step; imprecise “minimum read quality” wording, replaced with the supported
Quality threshold label. Added a Before/After legend. Detailed pixel choices are
provisional until the owner judges the comparison, and must be reconciled with
the real App's resizable Panes rather than copied as fixed graph coordinates.

Retained screenshots: A, B, C, B's added-step detail and compact A. The screenshot
files are actual JPEG captures. Syntax/format/link checks and unchanged-source
hash verification are in after.json. Production suites are not rerun for this
design-only revision.

## Seven design activity dispositions

Common actor/date/scope/evidence classes and trace are above. Each row records one
disposition; open conditions remain open rather than claiming acceptance.

| Activity / result                             | Disposition                       | Decision, limits and return route                                                                                                                                                                                                                                                                                                                            |
| --------------------------------------------- | --------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Discovery evidence reviewed                   | Reused current evidence           | Owner feedback on the actual previous sketch, 2026-09-09, contradicts its intuitiveness hypothesis for this owner and task. Reuse to reopen the design; do not generalize to all researchers. New task context or contrary walkthrough results reopens; owner supplies context.                                                                              |
| Design requirements accepted for this subject | Reused current evidence           | Standing UI-first Project/Chat requirements plus this explicit request for stacked versions or highlighted-flow details. Trace: current user request and prior accepted workspace. Requirement is current; choice among alternatives remains open. User need for a different comparison target reopens; design owner responds.                               |
| Concept decision recorded                     | Performed for the current subject | Codex built A/B/C with distinct information hierarchy; table/captures retain costs. No winner accepted. Owner completion/understanding evidence may reject or combine them.                                                                                                                                                                                  |
| Prototype evidence reviewed                   | Performed for the current subject | Codex exercised paths above and repaired observed layout failures. Sufficient fidelity for this comparison choice, not implementation proof. Large graphs, removed/rewired steps and native controls remain open for the chosen concept; implementation owner qualifies them.                                                                                |
| Test evidence reviewed                        | Reused current evidence           | The current owner directly reports the old design is not intuitive and proposes alternatives. That is valid negative feedback from the actual stakeholder using this context; it is not a structured study or new-concept approval. Tasks below are pending. Owner response reopens concept selection; broader use needs representative researcher evidence. |
| Design–implementation obligations reconciled  | Performed for the current subject | Codex keeps exact artifacts, references, User adoption and Gobble semantic ownership from the prior design; changes only presentation hypotheses. Real flow layout must preserve coherent graphs and version-scoped selection. Open until the selected design is implemented and reconciled against this artifact.                                           |
| Post-release design review closed             | Not applicable with exact reason  | This revision is an unreleased local simulation with no production users or telemetry. There is no release outcome to evaluate and no newly introduced production risk. Future pilot/adoption support reopens this activity; the existing P2B review guardrails remain applicable.                                                                           |

## Owner tasks and success criteria

For each concept, identify what 25 → 30 refers to, find the new quality check,
explain whether alignment receives a different input, and attach the relevant
change to Chat. Compare which view makes those answers easiest without switching
mental context or reading source code. Try the same questions after choosing
Unexplained change or Current version changed.

Success means accurate change/route/version understanding and preserved discussion.
It does not mean faster acceptance or more Use clicks. A mistaken negative quality
value, missing added branch, draft/reference loss, or acceptance mistaken for a Run
is a reason to reopen the design. The owner is the decision consumer; the next
implementation owner must retain these cases and the earlier P2B safety/ownership
tests. No background monitoring, recruitment or new task has been started.
