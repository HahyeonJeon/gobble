# R3b sequential plan and review checklist

Status: R3b1–R3b3 implemented and accepted by the owner on 2026-09-08. CSV chart creation remains removed.
Current formats: Workspace v10, contract bundle v12, Agent toolset shared-views-v7.
[R3b3 implementation and verification](../../stages/r3b3-agent-references.md) records 263 unit/contract tests, 43 native scenarios and a real Agent trial.
The next proposed View family is [R3c PDF shared reading](../r3c-pdf-shared-reading/design.md).

## Delivery checkpoints

| Checkpoint                                       | Concrete work                                                                                                                                                                                                                                                                                             | Evidence before next approval                                                                                                                                                                                    |
| ------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| R3b1 — Facts, targets and renderer qualification | Inspect monitor availability/coherence; define group/dependency observation, new selectors and frozen-version readers; pure projection/resolution/capture; compare one read-only React Flow + Dagre adapter with minimal HTML/SVG against bounded fixtures. Choose/pin libraries only from that evidence. | Original v2/v3 hashes/captures unchanged; missing/empty/partial/invalid graph cases; exact group/edge scope; bounded CPU/memory and CSP/keyboard findings for the candidate. Summarize final APIs and ownership. |
| R3b2 — User dependency navigation                | Tasks/Dependencies mode, group/dependency selection, observed instance list → existing logs; Discuss → frozen attachment; explicit camera, list fallback and compact/zoom interaction.                                                                                                                    | Native pointer/keyboard scenarios, same IDs/counts through mode changes, no graph editing, source-change/historical handling, normal restart, representative owner walkthrough.                                  |
| R3b3 — Agent graph references                    | Versioned observe/point for groups and directed dependencies; distinct marks; temporary Show/Return across modes; captured historical preview and explicit current search.                                                                                                                                | Scripted integration and one real signed-in Agent trial on synthetic data, delivered group/edge content, stale/cross-scope refusal, restored User mode/camera/selection/draft, regression checks.                |

The R3b1 and R3b2 approvals cover their respective implemented checkpoints only. R3b3 retains the user's
before/after design and approval process. No time estimate treats a checkpoint as one turn.

## First implementation request

Work on the existing App tree. Preserve unrelated engine work and all synced sources. First
freeze exact source hashes and old schema/hash fixtures. Keep runtime projection, reference
resolution, capture and graph rendering independently testable. Write no placeholder managers,
empty extension directories, event bus or second evidence cache. The proof adapter lives in a
bounded qualification harness until a candidate meets the contract and is selected.

A later renderer choice must record the exact Electron/React/library/OS tuple and imports,
license, build/CSP behavior, input semantics and failure fallback. CSV chart creation and Plotly rendering are retired by user request. Existing captured
metadata remains readable; CSV table selection remains supported. This graph does not use scientific X/Y coordinates or infer sample edges.

## Scenarios and expected behavior — unchecked acceptance criteria

- [ ] A mixed-state Align group shows separate observed counts and template facts; it never gets
      an invented overall execution state. Pick `align:S03`, attempt 2, then open only those logs.
- [ ] Two instances share a label. Selection, capture and Agent pointing use exact identity,
      never the label or a row/canvas index.
- [ ] Select Prepare → Align. Capture contains that direction, source revision and bounded endpoint
      context; reversing the pair, changing the Run or inventing an unreturned edge fails.
- [ ] A group has omitted or unmapped members. The view/capture declares partial membership and
      does not treat “none in preview” as absence from the Run.
- [ ] Topology is missing, empty, cyclic, malformed or too large. Tasks/logs remain usable; the
      dependency view explains its state and does not invent a graph.
- [ ] Change mode/camera/filter, then Discuss. The draft contains the intended semantic target;
      nothing is sent until Send and no unrelated log content is attached.
- [ ] An Agent points elsewhere while a User draft is focused. Arrival preserves focus and selection.
      Show reveals its group/edge; Return restores the User's mode/camera/selection/draft.
- [ ] Refresh changes topology or attempts. Old marks never relocate; existing immutable evidence
      remains readable. Old contracts/hashes and normal restart remain compatible.
- [ ] Use keyboard and assistive input to choose a node/edge/member and return to source. Thin edges
      have a list equivalent. No announced editing action is enabled accidentally.
- [ ] Use 1280×840, 900×650 and 150% zoom. Exact target and Discuss/Return remain reachable without
      hover; large synthetic fixtures demonstrate a measured bound before accepting the limits.

## Proposed owner walkthrough

In the sketch: choose **Align reads**, pick **align:S03 · attempt 2**, open logs, discuss the
Prepare → Align dependency, add the simulated Agent pointer, then Show/Return. Explain what the
arrow proves and whether it identifies a failed sample. Try the separate-pane alternative and
compare access to logs. Owner feedback can choose this workflow but is not general usability proof.

Record observed mistakes and task completion, not click count as a productivity proxy. Reopen the
design if a group is repeatedly mistaken for an attempt, an arrow is read as causal/sample evidence,
or compact mode loses the target/Return. No analytics or ongoing monitoring is created here.

## R3b3 checkpoint — 2026-09-08

Owner-authorized implementation is complete and returned for review. Exact Agent group/pair
observations and marks, temporary Show/Return, captured-evidence lookup and explicit current-source
search are implemented. CSV chart creation remains excluded. See the
[stage report and evidence](../../stages/r3b3-agent-references.md). Further work requires the
owner's next checkpoint decision.
