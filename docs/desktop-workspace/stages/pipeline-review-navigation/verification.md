# Verification record

Subject: the uncommitted renderer repair in this stage, on branch `codex/project-workspace-design`, base HEAD `7fe3d5c9e10d0cc104a15b6e7aa3def0d15b98f1`. Existing changes are preserved; nothing staged or committed. Before/final manifests bind source hashes, not just the branch label.

## Reproduction and repairs

- Before production edits, the deterministic Electron regression selected the older of two proposals and its second change. Once a two-Pane layout was maximized and the secondary Pane was gone, the comparison was absent: `Proposal history` could no longer be found. `baseline.log` retains the original-defect failure.
- Harness preparation failures: the first write used an extra `app/` path prefix and wrote nothing. Initial synthetic setting changes lacked matching flow settings and were correctly rejected by contract validation; the fixture now supplies matching before/proposed settings. The first maximize check did not await layout completion and inspected the old Current readiness marker; it now waits for the secondary Pane to disappear, ensures a real split layout, and asserts the restored comparison has no mounted Current flow. These are test defects, not product regressions.
- The native test window uses 80% zoom for a deterministic two-Pane wide case on this display, then 100% for compact mode. Captures scroll the selected detail into view; layout redesign remains outside this repair.
- Read-only advice identified two adjacent traps corrected within this slice: artifact viewport reset wiping review fields, and missing identity falling back silently to a different proposal/change.

## Proof and boundaries

- New Electron regression: older proposal / second change survive maximize, restore and compact Chat round trip; a third proposal arriving while hidden does not replace them. Confirmation is not retained. A duplicate View stays on Current independently. Returning Current retains step-list choice. Missing change/proposal gives explicit recovery without a substituted detail or actionable proposal. Closing/reopening starts fresh; restart starts Current. Current artifact records remain unchanged.
- Existing unit suite covers presentation invalidation and stale observation receipts; renderer regression confirms restored Changes does not mount/acknowledge the hidden Current flow. PresentedViews and useSurfaceLoad are unchanged (hashes verified).
- This fixture uses synthetic stored comparisons with the real Electron/IPC/native read path. It does not qualify engine checking, adoption, or a new Agent turn. The original actual-engine/Agent profile is revisited separately in a copy, without new engine or Agent actions.
- No contract, storage version or generated schema change. No cross-Project/restart retention is promised. Project-switch discard follows the existing Project-keyed ResourceWorkspace lifetime; no new persistence mechanism exists.

## Execution self-check

Applied Coding Review checklist categories: Project Fit, Affected Surfaces, Project Structure, Architecture, Correctness, Testing, Verification, Delivery, Usability, Compatibility. The task-specific questions have no unresolved in-scope problem in the verified paths. This is an execution self-check, not independent review or user acceptance.

Simplicity/modularization: seven existing renderer files change; one existing location record moves above the presentation gate, passed through explicit props. No registry service, context provider, class, schema or dependency added. ResourceWorkspace owns lifetime; the Pipeline presenters own location updates; backend owners retain all checked facts and permissions. The new test's four modularization terms are in README. Public API impact is limited to internal renderer props.

Inherited limitation: whole-app `npm run format:check` reports two untouched fixture files (`contracts/src/storage/workspace-v17.json`, `desktop/tests/fixtures/creation-candidate.json`). Changed files pass formatting. Do not rewrite unrelated fixtures in this repair.

## Final outcomes

All commands ran from `app/` using the existing lockfile and installed dependencies.

| Command | Result |
| --- | --- |
| `npm run typecheck` | Pass, including renderer, main, preload and test tooling |
| `npm run build` | Pass; native service and Electron bundles rebuilt |
| `npx vitest run` | 43 files / 397 tests pass on final production source |
| `npx playwright test pipeline-review-navigation.spec.ts workspace.spec.ts shared-context.spec.ts` | 12 scenarios pass, no skips, 29.0 s |
| `npm run schema:check` | Pass; generated bundle unchanged |
| `npx prettier --check` with the eight changed source/test files | Pass |
| `npm run format:check` | Two inherited fixture-format findings as listed above; no bulk formatting applied |
| `npx tsx ../docs/desktop-workspace/stages/pipeline-review-navigation/recheck.ts` | Pass: original actual 20 → 25 Phred comparison and draft survive compact Chat return |

Environment: macOS Darwin arm64, Node 25.7.0, npm 11.10.1, Go 1.27.1, Electron 44.2.0, TypeScript 5.9.3, Playwright 1.63.0, Vitest 4.1.11. Existing `playwright.config.ts`: one worker, actual Electron, isolated temporary profiles. No external Agent credentials needed. NO_COLOR/FORCE_COLOR emits an existing environment warning with no test effect.

Visual inspection: retained-wide.png / retained-compact.png and actual-profile-return.png show the retained Changes screen, with the duplicate View independently showing Current or the original saved report below. Short split panes and long proposal summaries require scrolling; this slice does not redesign their layout or preserve scroll position in Changes.

The recheck used a new copy of the earlier actual-run profile; the original remains untouched. No new Agent call, adoption or execution was requested. All test Electron applications closed normally.

Final identity: `after.json` contains SHA-256 hashes of changed sources/test plus unchanged presentation guards. `before.json` / `before-status.txt` preserve the inherited baseline. The verified tree remains uncommitted. Remaining product decisions (restart location restoration and Chat reading improvements) remain separate from this repair.

