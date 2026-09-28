# Completed Run presentation

Date: 2026-09-27. Accepted scope: first implementation slice of the
[single-FASTQ walkthrough](../single-fastq-walkthrough/README.md). Work in the existing
checkout; preserve inherited uncommitted changes. No staging/commit authority.

## Implementation guide

| Slice | Existing owner and change | Proof / stop condition |
| --- | --- | --- |
| Completed state + Pane layout | RetainedRunFlow orders Flow before ancillary controls; ContinuationControls suppresses settled completed guidance only, retaining pending/unknown/checking; run CSS makes short stacked Panes scroll without clipping task actions. RunFeedbackControls makes normal design navigation compact. | Reproduce original compact obstruction before edit; real completed fixture at 1440×950 and 1100×800 with logs below, keyboard selection/logs/discussion; controlled stopped/unknown regression. |
| Preparation wording | RunPreparation labels readiness as a review, not a claim about all Runs. | Current navigation screenshot and existing preparation checks. |
| Run sidebar refresh | RunsBrowser observes existing shared launch-review cache; only changed admitted Run identities invalidate its native list. | Controlled admission after initial sidebar load; no manual refresh, no new submission. |

Reach: renderer components, existing launch read hook consumer, CSS, focused Electron
tests and checkpoint docs. Contracts, schemas/generators, Main shared references,
native service, execution engine, file/report transport and authentication are no-op.
No new production directory, class, public API or state store is needed. Existing
renderer ownership remains presentation; native service remains Run-list authority.
Report display/reference integration is the next separate contract design.

Baseline: prior walkthrough manifest verified; `before.json` captures renderer/test
sources. Prior walkthrough reproduced compact clipping and HTML source display.
New compact regression records the unfixed built App before production changes.

## Verification and handoff

Implemented, awaiting owner review before the report-contract step. Six production
files changed; no new production abstraction or schema. Three test files add/update
coverage. Exact changed-source/build identities are in `after.json` and `build-identity.json`.

- [Wide completed Run](verified-evidence/continuation-restored-comp-dc43f-al-review-without-executing/completed-wide.png)
- [Compact stacked Flow and logs](verified-evidence/continuation-restored-comp-83249-le-in-stacked-compact-Panes/completed-stacked-compact.png)
- [Compact Chat with exact Attempt 3 attachment](verified-evidence/continuation-restored-comp-83249-le-in-stacked-compact-Panes/completed-stacked-chat.png)

Production behavior:
1. One-line `Execution complete · Saved design`, then Flow before ancillary controls.
2. Completed, settled continuation controls disappear while their refresh effects stay
   mounted. Pending/unknown/checking/ready/error context is preserved.
3. The Run uses one outer scrolling area in short Panes, keeping the graph initially
   visible and task actions reachable. Generic Runs without retained Flow are unchanged.
4. `Prepared review` describes preparation only; Current need not infer past Run state.
5. Confirmed admission changes refresh the existing sidebar list; repeated status polls
   do not. Engine authority, Main references and exact task/log identities are unchanged.

Checks on macOS arm64, Node 25.7.0, npm 11.10.1, Electron 44.2.0, Playwright 1.63.0:

| Command (from `app/`) | Result | Record |
| --- | --- | --- |
| `npm run build` | Pass: native service and desktop build | final-build.log |
| `npm run typecheck` | Pass | final-typecheck.log |
| `npm run schema:check` | Pass; no schema changes | schema.log |
| `vitest run` | 41 files / 380 tests pass | final-unit.log |
| `playwright test continuation-restored.spec.ts continuation-review.spec.ts admitted-runs.spec.ts` with `GOBBLE_CONTINUATION_RESTORE_FIXTURE` pointing to the P5B-5 fixture | 6 pass, no skips | verified-electron.log |
| Prettier on all changed source/tests; root `git diff --check` | Pass | format-check.log; diff-check.log |

The actual completed fixture was inspected without starting another analysis. Controlled
process peers cover stopped/blocked/unknown/Agent-pointer cases and admission delivery.
This does not requalify scientific tools, real-provider behavior, engine changes or
packaging. Existing source/engine qualification limits remain unchanged.

Self-check categories: modularization, overengineering, correctness, testing,
verification, usability and compatibility. Existing owners retained; no public contract
or generated output edit needed; exact instance/attempt log and attachment behavior
verified; no execution authority moved into UI. The two unresolved broader outcomes
are report display/sharing (next approved direction, contract not yet implemented) and
representative-user validation. One transient test failure is recorded below.

Source tree retained uncommitted, with inherited changes preserved. Test-generated
browser profiles are archived outside the repository as recorded in
`fixture-archives.json`; only relevant screenshots, logs and failure records remain here.

## Implementation notes and learnings

- Subagent implementation consultation recommended the existing shared launch-review
  cache and a stable set of admitted Run identities. The sidebar remains a read-only
  consumer; it never requests reconciliation or starts work.
- Keep ContinuationControls mounted because its effects trigger Run refresh on review
  changes. Hide only settled completion after hooks; pending acknowledgements,
  checking/ready reviews and errors remain visible.
- The new compact regression failed on the original App with viewport ratio 0. After
  the layout fix Chromium reported 0.9999997019767761 for a fully visible scaled node.
  The assertion now permits subpixel rounding (ratio >= 0.999), not meaningful clipping.
- `admitted-runs.spec.ts` is a test-only delivery scenario: concept = observed confirmed
  admission; duty = prove navigation refresh; hidden knowledge = gated list responses;
  exposure = one Playwright test. It reuses a native fixture and the existing test-owned
  session-preload technique, never adding a production hook. It does not prove new
  engine admission, which remains covered by earlier qualification.

- The first combined Electron run clicked compact Chat before the asynchronous task
  attachment finished, invalidating its visible-view lease. The new test now waits
  for the second attachment before switching. No product visibility policy was weakened.
- In that run, the unchanged deterministic Agent-pointer test once reported “Shared
  review context missing.” An instrumented isolated rerun passed; the temporary fixture
  diagnostic was removed, and the final complete suite passed without it. The precise
  transient cause was not captured and is not claimed fixed or proven pre-existing.
  Retain `electron.log` and `diagnostic.log` beside the final evidence.
- Visually inspected final wide Run and compact Flow/log screenshots. The Current
  wording capture caught the other Pane reloading; it verifies the label, not a settled
  whole-workspace screenshot. No production report viewer is claimed in these images.
