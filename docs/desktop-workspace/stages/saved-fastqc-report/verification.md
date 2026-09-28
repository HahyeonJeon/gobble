# Saved FastQC report: execution handoff

Date: 2026-09-28 (Asia/Seoul). Accepted scope: [FastQC sharing design, slice 2](../../proposals/fastqc-report-sharing/accepted/ideation-01.md). This records implementation verification, not an independent review or owner acceptance.

## Retained identity

Worktree: `/Users/hahyeon/.codex/.chatgpt-projects/g-p-6a9bae3a3ef481919aec253b9710060a/gobble`.
Branch: `codex/project-workspace-design`; HEAD: `7fe3d5c9e10d0cc104a15b6e7aa3def0d15b98f1`.
The inherited dirty worktree is preserved. Nothing was staged, committed, reset, or published. `before.json` captures the pre-edit app source; `reach.json` lists the 66 changed/new app files and their hashes; `after.json` identifies final app sources, supporting documents, screenshots and built artifacts. Build outputs are verification artifacts, not source changes.

Current versions: Workspace **20**, schema bundle **29**, catalog **5** (unchanged), shared tools **14**. Canonical definitions are in `app/contracts/src`; `app/contracts/scripts/export-schema.ts` generates schemas via `npm run schema:generate`. `npm run schema:check` confirms consistency. Workspace 19 is frozen in its own source/schema before migration; original profile backup remains available.

## Completed behavior and boundaries

A completed admitted quality step opens its saved FastQC report in the lower Pane while Flow remains above. The report contains ordered source text, tables, literal status labels and all original PNG chart bytes. Section navigation survives maximize/move/restart. Existing Pane controls provide focus and close.

Project `savedReports` metadata owns retention independently of open Views. Reopening uses the saved content and exact producing attempt, including after the source file is removed or a later attempt starts. Corrupt saved content fails explicitly; it never substitutes a fresh report or newer logs. Workspace publishes the metadata and View in one writer transaction; unsuccessful publication leaves only a managed orphan for the existing cleanup path.

| Unit | Concept and owner | Boundary / current reason | Proof |
| --- | --- | --- | --- |
| `run-report.ts` | Source, content and saved identity contracts | Defines provenance and bounded immutable payload independently of View state | Contract checks, migration and report-evidence tests |
| `report-reader/reader.ts`, `profile.ts` | Qualified FastQC 0.12.1 reading profile | Strict detached XML interpretation; owns known grammar/assets, rejects unknown content | Actual source equality and negative parser tests |
| `FastqcReader` | One disposable decoder job | Owns hidden sandbox, deadline and shutdown; no privileged bridge | Electron sandbox/cancel/timeout tests |
| `ReportEvidence` | Verified retained report | Injected decoder and existing storage; owns source digest and saved content consistency | Retention/corruption/quota/publication tests |
| `EvidenceStorage` | Immutable Project blob storage | Minimal hash/length descriptor serves existing attachments and reports without fabricating an attachment | Existing evidence tests plus report tests |
| Workspace controller | Project metadata / View writer | Capture first, publish together; host validates navigation and saved ownership | Controller tests and app restart scenario |
| `ReportView` | Readable presentation of saved evidence | React text/tables/original charts; View section address is separate from report identity | Real Electron interaction and screenshots |

No new inheritance hierarchy, plugin registry, report editor, recalculated scientific data, or separate cache was introduced. Gobble/native service remains the authority for execution and source provenance. Agent report open/observe/select/attach is explicitly unavailable until slice 3.

## Verification

Commands ran from `app/` unless noted. Existing logs retain earlier failures; repairs and their causes are recorded in [README](README.md).

| Command / observation | Result / evidence |
| --- | --- |
| Pre-edit `npm test` | 380 tests / 41 files passed; `baseline.log` |
| Final `npm test` | **389 tests / 42 files passed**; `unit-final.log` |
| `npm run typecheck` | Passed; `typecheck-final.log`, rerun after final test setup edit in `typecheck-shipping.log` |
| `npm run schema:check` | Passed; `schema-final.log` |
| `npm run build` | Passed; `build-final.log` |
| `npx playwright test desktop/tests/electron/report-reader.spec.ts desktop/tests/electron/saved-reports.spec.ts desktop/tests/electron/shared-context.spec.ts desktop/tests/electron/workspace.spec.ts` | **16 passed**; `electron-final.log` |
| `npx playwright test desktop/tests/electron/report-reader.spec.ts desktop/tests/electron/saved-reports.spec.ts` against final shipping decoder | **5 passed**; `electron-shipping.log` |
| Changed authored-file Prettier check | Passed; file inventory `format-files.json`, `format-changed-final.log`; final test file `format-shipping.log` |
| Repository `git diff --check` | Passed |
| Full `npm run format:check` | Two pre-existing failures remain: `contracts/src/storage/workspace-v17.json` and `desktop/tests/fixtures/creation-candidate.json`; both hashes equal pre-edit baseline |

The parser spec initially rebuilt the decoder independently. A final build comparison found only that decoder output differed from the prior tested build. Removed this redundant test build so the spec consumes the shipping artifact. The five focused tests then passed with decoder SHA-256 `0143b4ea491222a9bca0e6035f29604761597679ba7cce46f6e550e5a09b2e39` unchanged before/after. Main, preload and renderer output were byte-identical to the 16-test run. This reconciles the identity difference without claiming unverified build equivalence.

Environment: macOS Darwin ARM64; Node 25.7.0, npm 11.10.1, Electron 44.2.0, React 19.2.8, TypeScript 5.9.3, Playwright 1.63.0, Vitest 4.1.11, Go 1.27.1. Dependencies remain locked in `app/package-lock.json`; tests use `app/playwright.config.ts` (one worker) and `app/vitest.config.ts`.

The fixture is the byte-exact **537181-byte** FastQC 0.12.1 output from the preceding [actual-engine qualification](../report-output-evidence/). Source SHA-256: `8f045465fd56bbfb18360592e1b41c4978b45c6054fb6bc2cef52aa724413d52`. Decoder qualification preserves ten modules and eight original charts. `qualified-content.json` is the verified rendition; `measurements.json` records its bounds. Source license/provenance is documented beside the fixture.

The UI test uses the real App, Main, native service provenance/file checks and real FastQC source, with an isolated simulated engine query peer. It does **not** claim a fresh real-tool pipeline run or live Agent delivery. Prior actual-engine evidence remains separate. No production user profile was used.

## Visual verification

Checked wide 1600×1000 and compact 1000×760 windows, stacked and focused Panes, original chart decoding, section navigation, close, restart and saved reopen. No horizontal overflow in these scenarios. Visual inspection exposed loss of section on maximize; persisting the validated View section fixed it and the test covers it.

- [Flow and report](screenshots/report-stacked.png)
- [Original chart below Flow](screenshots/report-chart.png)
- [Focused report](screenshots/report-focused.png)
- [Compact window](screenshots/report-compact.png)
- [Saved result after restart/source removal](screenshots/report-restored.png)

## Execution checklist pass

Applied the Coding Execution checklist to this retained identity. Categories answered: Project Fit, Affected Surfaces, Project Structure, Architecture, Design Pattern, Abstraction, Data Model, Public API, Parameters, Modularization, Reusability, Performance, Optimization, Overengineering, Code Complexity, Readability, Vocabulary, Naming Convention, Docstring, Correctness, Testing, Verification, Delivery, Usability, Operations and Compatibility. Security, concurrency, build and migration concerns were checked at the affected boundaries.

The implementation evidence does not show an unresolved in-scope failure under the admitted profile/scenarios. Current callers justify the two concrete lifecycle owners; immutable contracts and dependency injection keep decoder, storage and View concerns separate. Existing single-writer transaction, quota and cleanup mechanisms are reused. Strict whole-report validation, bounded sandbox disposal and explicit unsupported shared-tool paths avoid partial or misattributed evidence.

Recorded concerns / undecided coverage:

- Full-repository formatting is not clean because of the two untouched inherited files above; this slice does not repair them.
- Representative performance is bounded and exercised with the qualified synthetic source; this is not a broad corpus or throughput qualification.
- Cross-platform installers, Windows/Linux execution and assistive-technology auditing were not run. No release/package claim is made.
- The profile recognizes one qualified FastQC version/grammar, with explicit size/count/dimension limits. Unsupported reports fail completely, not partially.
- Some qualified original PNGs omit the final 1–2 IEND CRC bytes. Only that exact terminal suffix is admitted after preceding CRC checks and successful bitmap decoding. Original bytes are retained unchanged. Future Agent image transport must qualify those exact bytes; provider compatibility is unproven here.
- Agent attachment, recipient-scoped `read_report`, whole-report pointing and live Agent interpretation belong to the next slice. No current Agent-read claim is made.

The accepted slice is implemented and verified; owner result review is the next boundary before slice 3.
