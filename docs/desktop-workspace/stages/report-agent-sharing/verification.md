# Report Agent sharing — verification and handoff

Date: 2026-09-28 (Asia/Seoul). This is Coding Execution evidence, not an independent review or owner acceptance.

## Delivered scope

Accepted [FastQC report-sharing design](../../proposals/fastqc-report-sharing/accepted/ideation-01.md), slice 3. The user approved proceeding after the saved Report View result.

- `Attach report` places a whole saved report in the existing composer. Draft/sent previews reuse ReportView with the same original text, tables and charts.
- Initial Agent input contains complete original text/tables and chart inventory, with explicit notice that chart pixels have not yet been delivered.
- `read_report(attachmentId, imageId?)` reads only an attachment of the active message for its addressed Agent. It returns complete reading content or one original PNG, plus exact saved/producer identity and returned-image coverage.
- Shared-view access and image-capable recipient are checked at preparation, acceptance and reading. Account/turn checks remain in Coordinator; Project-session checks also cover cached replay. Closing a Pane does not revoke addressed evidence; changing Project or disconnecting the renderer does.
- User and Agent may point to the whole saved report. Agent pointing requires a current foreground observation receipt. Reading an attached chart never creates a pointing receipt. Report section changes invalidate Main's render identity.
- The existing evidence store retains one immutable report blob. Drafts, messages, report registry, Views and marks root that same identity. No source reread, image re-encoding or silent replacement.

Current versions: Workspace **21**, contract bundle **30**, catalog **5** unchanged, shared toolset **15**. Original saved report payload remains version 1. Existing shared-tool conversations require the normal explicit renewal path; history is retained.

## Ownership and code structure

| Owner / unit | Definition / responsibility | Boundary and consumers |
| --- | --- | --- |
| Contracts `run-report.ts` | Whole saved report address, report delivery representation, pure text/inventory projection | Plain records/functions. No I/O. Used by evidence, reference validation, shared tools and UI. Image IDs are reading addresses, not region selectors. |
| Evidence `report.ts` | Retained report validation and attachment materialization | Reuses existing immutable payload. `reportAsset`, `reportPreview`, `reportRepresentation` are shared by actual capture/preview/send/read callers. Private saved validation stays private; storage import is type-only to avoid a runtime cycle. |
| Existing EvidenceService / Coordinator | Preparation, acceptance and delivery | Report-specific text byte accounting uses verified representation, not retained image-bearing JSON size. Generic 64 KiB text, two initial images and final message envelope limits remain. |
| `shared-context/report.ts` | Current-message report access | One function, not a new service class. Uses authoritative ToolContext, exact sent manifest, retained resource reader and capability function; revalidates after I/O. No caller-selected Project, Agent, submission, hash or path. |
| Existing WorkspaceController / RenderSession | Project state and presentation lifetime | Single writer unchanged. `sentEvidenceGuard` captures the existing evidence epoch; report section is part of render fingerprint. No second session/cache owner. |
| Existing SharedContextHost / ObservedReads | Tool budgets and observation authority | Registers read_report under existing 32-call/8 MiB turn bounds; separate whole-report observation receipts for pointing. |
| ReportView / EvidencePreview | User reading and attachment preview | Same rendition component, English toolbar, existing modal/composer, separate User/Agent whole-report labels. No editor or chart calculation. |

No new pattern, inheritance tree, plugin registry or cache. Current forces are exact report identity, addressed delivery and independent viewport authority. Encapsulation keeps authorization in Main; plain data and functions handle projections. Public tool inputs stay flat. Canonical schema owner is `app/contracts/src`; generator is `app/contracts/scripts/export-schema.ts`, run with `npm run schema:generate`, checked with `npm run schema:check`. Frozen Workspace 20 and earlier schema bundles remain unchanged.

Subagent consultation: `/root/report_access_advice` inspected the actual Coordinator, historical sent-evidence lookup, provider image adapter, image renderer and byte accounting. Recommendations implemented: bind to current submission, recheck after I/O, bypass image resampling, update the explicit tool allowlist, and separate addressed reads from foreground receipts. This was read-only implementation advice, not review acceptance.

## Verification record

Environment: macOS Darwin ARM64; Node 25.7.0, npm 11.10.1, Electron 44.2.0, React 19.2.8, TypeScript 5.9.3, Playwright 1.63.0, Vitest 4.1.11, Go 1.27.1. Dependency lock unchanged. Commands from `app/` unless specified. Playwright uses the existing one-worker configuration.

| Check | Result / evidence |
| --- | --- |
| Pre-edit `npm test` | 389 tests / 42 files passed; `baseline.log` |
| Final `npm test` | **397 tests / 43 files passed**; `unit-final.log` |
| Final focused `npx vitest run desktop/tests/report-evidence.test.ts desktop/tests/report-sharing.test.ts` | 15 passed after final dependency visibility cleanup; `unit-shipping.log` |
| `npm run typecheck` | Passed; `typecheck-shipping.log`, final test-helper check in `typecheck-handoff.log` |
| `npm run schema:generate` then `npm run schema:check` | Passed; `schema-final.log` |
| `npm run build` | Passed; `build-shipping.log`. Final artifact hashes are identical to the batch-tested build; `build-comparison.json` |
| Changed authored files `npx prettier --check …` | Passed; exact inventory `format-files.json`, `format-handoff.log` |
| Repository `git diff --check` | Passed |
| Electron report-reader, saved-reports, evidence, shared-context, questions and workspace specs | 23 distinct scenarios have passing evidence across the runs below; **not a clean single batch** |

Electron command: `npx playwright test desktop/tests/electron/report-reader.spec.ts desktop/tests/electron/saved-reports.spec.ts desktop/tests/electron/evidence.spec.ts desktop/tests/electron/shared-context.spec.ts desktop/tests/electron/questions.spec.ts desktop/tests/electron/workspace.spec.ts`.

- `electron-final.log`: 20 passed, 3 failed. Report/text/table-image sharing calls were denied by the foreground-window gate. Error snapshots remain in `electron-failures/`.
- `electron-recheck.log`: 2 passed, 2 failed; the same foreground-dependent scenarios remained inconsistent. Production focus/authorization rules were not relaxed.
- The test harness now reuses its existing native focus operation before shared message sending and confirms no dialog is open; CDP clicks alone do not establish native macOS focus. This is an explicit test precondition, not a retry in production.
- `electron-focus.log`: report scenario, text pointing and conversation renewal passed. The remaining table/image scenario received `shared-pointt` instead of the test-authored `shared-point`; the mismatch is retained, not attributed to a product cause without evidence.
- `electron-isolated.log`: the remaining table/image scenario passed in a separate four-second run. Desktop focus/keyboard interference remains a qualification limitation of this interactive host.

New proof covers exact draft/prepared/sent content, wrong recipient/message/Project, unknown images, messages-only and non-image-capable recipients, changed preparation, cached capability/session revocation, in-flight authorization loss, sent-manifest mismatch, corrupt retained blob, combined text overflow, no arbitrary image transformation, current observation before pointing, stale section receipts, close/restart retention and frozen Workspace 20 migration/backup.

The expanded real Electron scenario opens the original saved report after source deletion and a newer attempt, attaches/previews/sends it, reads all eight chart images through the actual Codex adapter, compares every returned PNG byte with the retained source and publishes User/Agent whole-report pointers. Initial content has ten modules and no base64. The roughly 0.5 MB retained payload is successfully delivered as bounded text plus on-demand image calls, rather than rejected as 64 KiB text.

## Visual result

[Attachment preview](screenshots/report-attachment.png) · [Shared discussion and pointers](screenshots/report-discussion.png) · [Compact reader](screenshots/report-compact.png).

Inspected screenshots from the successful report scenario. Corrected a report preview footer that inherited Run wording, and scoped nested report header styles so the modal does not change its layout. Source text/status labels and original image bytes remain unchanged.

## Failure reconciliation and execution checklist

Earlier failures and repairs are recorded in [README](README.md) and logs, including test-peer tool allowlist, overly broad attachment locator, TypeScript integration mismatches and missing Main presentation invalidation. Passing results do not erase those records.

Applied categories: Project Fit, Affected Surfaces, Project Structure, Architecture, Design Pattern, Abstraction, Data Model, Public API, Parameters, Modularization, Reusability, Performance, Optimization, Overengineering, Code Complexity, Readability, Vocabulary, Naming Convention, Docstring, Correctness, Testing, Verification, Delivery, Usability, Operations and Compatibility; security, concurrency, migration and build overlays at affected boundaries.

No unresolved in-scope implementation failure was found in the exercised paths. Remaining undecided or out-of-scope items:

- Automated provider/engine peers were used for this slice's Electron scenario. The App, service file/provenance checks, actual original FastQC fixture and adapter image envelopes are real. This does **not** establish live model visual interpretation or scientific validity. Prior actual-engine qualification remains in [output evidence](../report-output-evidence/README.md).
- Some original FastQC PNGs have the previously qualified terminal IEND quirk. Original bytes reached the protocol peer unchanged; actual model decoding must be qualified in the next live walkthrough.
- Full-repository formatting still has the prior untouched `workspace-v17.json` and `creation-candidate.json` drift. Their pre-edit hashes are preserved. Changed files pass their scoped formatting check.
- No installer, release, cross-platform run, broad report corpus, throughput or assistive-technology audit was performed.
- Report-linked durable questions, reading earlier-message report attachments, chart-region/metric semantics and report editing are excluded. Use a direct current-message report attachment and ordinary Chat.
- Interactive desktop focus/keyboard interference is documented above; no claim of an all-green single Electron batch.

## Retained result and next boundary

Worktree `/Users/hahyeon/.codex/.chatgpt-projects/g-p-6a9bae3a3ef481919aec253b9710060a/gobble`, branch `codex/project-workspace-design`, HEAD `7fe3d5c9e10d0cc104a15b6e7aa3def0d15b98f1`.

Exact uncommitted app source and supporting evidence are identified by `before.json`, `reach.json` and `after.json`. Inherited changes are preserved; no staging, commit, reset or publication. Gobble/Go execution and native source acquisition are unchanged. Synced `sources/` remain read-only.

Owner result review precedes the next bounded slice: a new capable-engine Run with an actual Agent, using the selected single-FASTQ workflow to discuss the saved report, original charts and a concrete pipeline change through the UI. That live walkthrough is expert evaluation with synthetic data, not scientific validation.
