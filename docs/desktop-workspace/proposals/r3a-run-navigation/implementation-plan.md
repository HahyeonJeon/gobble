# R3a — Sequential delivery and review

Status: **R3a1–R3a3 implemented, verified and accepted, 2026-09-08.** The user's latest “네 진행해주세요” accepts [R3a3](../../stages/r3a3-agent-observation.md) and requests the next scope/ownership sketch. [R3b](../r3b-run-dependencies/design.md) is a proposal; its production implementation has not started.

## Goal

Demonstrate task → current-attempt stream → exact frozen excerpt → Project chat → Agent pointer → explicit reveal/return, including source advancement and restart. Preserve the User's other work and existing R1/R2 behavior.

## Checkpoints

| Checkpoint                               | Bounded implementation                                                                                                                                                                                                             | Completion review                                                                                                                                                                                            |
| ---------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| R3a1 — Contracts and observation capture | Typed Run/Log read models and selectors; current-attempt capability limits; bounded Main observation retention; immutable Run/log draft-capture path; compatibility readers and migrations. Existing view behavior remains usable. | Ownership/API review, adversarial contract/capture tests, migration preservation and relevant full App checks. Summarize before requesting R3a2.                                                             |
| R3a2 — User navigation and compact UI    | Task selection/search/state filter, explicit companion log opening, stdout/stderr tabs, selected excerpt → composer; explicit Refresh and historical capture UI.                                                                   | Native visual/keyboard review against the sketch, 1280×800/compact/150% scenarios, source-change/refresh-error flows. Summarize before requesting R3a3.                                                      |
| R3a3 — Agent observation and return      | Versioned tool inputs/output capabilities; exact task/log observe/point; separate User/Agent marks; current/historical reveal behavior and Return restoration.                                                                     | Scripted round-trip plus one real signed-in Agent trial using a synthetic fixture, captured payload/response inspection, normal quit/reopen and regression checks. User acceptance before another R3 family. |

Approval of the proposal starts **R3a1 only**. Later checkpoints retain the user's requested before/after review. R3a2 and R3a3 can refine their sketches from evidence, with changes shown before implementation.

## Code organization and API boundary

Keep the current repository structure. This is a responsibility map; do not create empty folders or abstractions to match it.

| Owner                                                                                 | Expected work and boundary                                                                                                                                                                                                                                                                                   |
| ------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `app/contracts/src/run.ts`, `run-presentation.ts`; proposed `log-presentation.ts`     | Validate consumed engine records and expose portable display facts. Add explicit count/preview/availability limitations. Keep transport envelopes separate from normalized read models.                                                                                                                      |
| `selection.ts`, `reference-target.ts`, `reference-resolution.ts`, evidence contracts  | Closed task/stream selector cases, precise coordinates, resource compatibility, content/provenance validation. Pure functions; no I/O or renderer types.                                                                                                                                                     |
| `main/service/run-presentation.ts`; proposed peer `log-presentation.ts`               | Normalize once at the service boundary. Preserve unknown status/absent facts. Validate exact returned log instance and reject ambiguous/mismatched records.                                                                                                                                                  |
| `main/workspace/service.ts`; proposed `run-observations.ts`                           | Resolve validated source data and retain bounded immutable loads. One scoped lifecycle owner with expected generation/revision checks; no independent durable Run store.                                                                                                                                     |
| `main/workspace/controller.ts`, `model.ts`, `render-session.ts`, `reference-views.ts` | Authorize/order commands and persist state; pure transitions; invalidate obsolete readiness; restore temporary reference views. Do not put log parsing or provider delivery inside the controller.                                                                                                           |
| `main/evidence/{service,materialize,storage,draft}.ts`                                | Reuse the existing asset store and quotas. Add explicit captured-observation source, integrity-bound previews, capture-before-draft ordering and reference-aware reclamation. Current storage has no orphan collector; missing/unknown document state must prevent deletion. Preserve addressed send checks. |
| `renderer/workspace/views/RunView.tsx`; proposed `LogView.tsx`                        | Render typed facts and emit semantic selection/navigation intents. Extract a task list or stream text component only when a distinct responsibility warrants it; keep Pane as layout only.                                                                                                                   |
| `main/shared-context/{host,catalog,observation}.ts` and existing tool contracts       | One semantic implementation for User/Agent; source-read versus visible-observation distinction; versioned receipt validation.                                                                                                                                                                                |
| `internal/appservice`                                                                 | Existing current-attempt/pinned-runtime query authority. Change validation only if an evidenced boundary gap requires it; never read engine internals from Electron or infer historical attempts.                                                                                                            |

Use named narrow operations such as `readRunObservation`, `readLogObservation`, `captureObservedTarget` and `revealObservedTarget` only where they correspond to real ownership. Inputs carry Project/resource identity and expected observation; outputs distinguish observed, historical, unavailable and unsupported. Renderer callbacks receive typed task/stream targets, never filenames, command strings or mutable engine objects. These names are design sketches, not an extra generic service layer.

Do not add a BasePanel, universal metadata bag, event bus, parallel Project repository, engine scheduling logic or new transport for this slice. Keep the Plotly boundary and existing table linkage isolated.

## Bounds and compatibility

- Initial Run UI preview: at most 1000 task instances from the bounded native response, with returned/available counts and filter scope. Preserve runtime order and instance identity. Transport retains its current response limit; an oversized response is an explicit error. Performance support beyond this bound needs a separate data contract.
- Logs: current attempt only, at most 4096 source bytes per stream as supplied by Monitor. Existing 64 KiB combined semantic-evidence and 16-attachment limits remain. Do not enlarge the service limit to compensate for a slow view.
- Retained observation cache: only current visible loads plus in-progress authorized capture; cap entries/bytes explicitly during R3a1 and test eviction. Capture either completes against its retained revision or fails clearly; no silent reacquisition of a newer revision.
- Freeze the v6 document reader and its nested schema membership before introducing v7. Back up exact original bytes on the first migration write; preserve v1–v6 reopening, unrelated workspace fields and existing asset hashes. Unknown future versions stay rejected/preserved under the current storage policy.
- Introduce a versioned new target/selector contract for task and stream selection. Retain v2 references and old flattened log coordinates verbatim. A legacy text range cannot be converted into a stream range without its original content, so do not guess. Old log evidence uses the legacy presenter when needed.
- Evidence payloads discriminate captured Run/log observations explicitly and preserve old asset decoders. Existing file/table/scatter capture and future/unknown payload rejection are regression subjects.
- Evolve the shared toolset from v4 only when R3a3 adds observable behavior. Existing provider threads use the established renewal path; do not silently reinterpret registered inputs or resend messages. Render receipt changes are versioned if the existing envelope cannot truthfully bind task/stream presentation.

## User scenarios and acceptance checklist

These are planned tests, not completed results.

| Scenario                                                             | Pass condition                                                                                                                                            |
| -------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Two expanded instances have the same task name                       | Selecting/opening logs addresses the exact instance and attempt; authored task ID is retained only as provenance.                                         |
| Task is a template or has attempt 0                                  | Facts remain discussable; unavailable log action has an honest reason; no fictitious attempt 1.                                                           |
| Failed task's stderr is selected using mouse or keyboard             | One attachment includes exact text, stream, attempt, preview coordinates, observation revision/time and tail limits; no Agent turn before Send.           |
| stdout and stderr contain identical text                             | Agent/User marks remain on their selected stream; no combined chronology or cross-stream remapping.                                                       |
| Log grows after Discuss and before Send                              | Original asset and excerpt stay unchanged; send labels the observation time and does not require the attempt still to be current.                         |
| Engine retries the instance before a new log query                   | Old-attempt query fails clearly; no automatic switch. Captured evidence remains readable. Explicit current navigation is labelled separately.             |
| Source bytes grow/change during native tail reading                  | UI claims only decoded preview exactness; no fabricated original byte/line offsets or atomic state/log claim.                                             |
| Empty/missing/unreadable tail, malformed log record, runtime failure | Distinct transport errors where available; empty text says No text returned. No invented successful log state. Last observation survives refresh failure. |
| Agent points elsewhere, then User selects Show and Return            | Agent mark is distinct. User selection, prior stream/filter/scroll, draft and protected tabs survive. Incoming pointer alone does not steal focus.        |
| Source closes/advances, normal quit/reopen, missing asset            | Captured content survives a normal restart; no persisted render readiness. Missing asset is explicit, never substituted from current source.              |
| Old flattened log reference and existing R2 evidence reopen          | Old coordinates/bytes retain their meaning; table/scatter membership and Show/Return remain correct.                                                      |
| Compact width, 150% text, keyboard-only use                          | Main content and Chat usable; no hover-only primary controls; Details collapsed by default; focus restoration predictable.                                |
| Run exceeds task preview bounds                                      | Counts/filter scope visible, no false whole-Run success/absence claim, no unbounded DOM.                                                                  |

## Evidence and review method

Use existing contract/unit/architecture and Electron suites. Add meaningful negative cases for subject/stream/attempt mismatch, invalid Unicode ranges, stale receipts, capture races, asset tampering and cross-Project access. Migration fixtures compare original evidence bytes and semantic identity, not merely successful parsing. Test failure between asset write and workspace commit and cleanup of unreferenced bytes.

Exercise UI with a small synthetic failed-attempt fixture first. A real-provider trial in R3a3 must record the delivered metadata/text and the returned distinct reference; scripted responses do not count as actual Agent integration. A real native Gobble read should verify the stated 4096-byte/current-attempt limitations, without changing unrelated engine work. Record any external runtime unavailability instead of claiming simulated evidence is live.

This proposal was reviewed against local source and the retained Stage 6 Run screenshot. No production test run, new live runtime test or new provider turn occurred for the design checkpoint. The sketch's separate visual check is recorded after rendering it.

## Proposal visual check — 2026-09-08

The synthetic sketch was exercised in installed Google Chrome and the Codex in-app
browser. Fixed demo excerpt → draft attachment → simulated Agent Show → Return
preserved the stderr selection while revealing stdout. The state filter returned
the one failed fixture task. No page-script errors or horizontal page overflow
were observed at 1280, 1024 and 740 px. See [check record](review/sketch-checks.json),
[selection layout](review/selection.png), [Agent reference](review/agent-reference.png)
and [compact layout](review/compact.png).

Visual review reduced task-row height and kept source IDs alongside labels, retained
tail limits when selected, and removed the misleading current-attempt label from an
observed log. This is a layout/interaction sketch: real text dragging, all task logs,
live refresh, provider delivery and native 150%/keyboard acceptance are not implemented
or verified here. Production source, runtime state and prior R2c evidence were not
changed. Document links and whitespace were checked.
