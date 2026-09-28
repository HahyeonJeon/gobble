# R3c3 verification record

Date: 2026-09-08, Asia/Seoul. Request **R3c3-2026-09-08**. Development and Testing worked
sequentially in this task. **Passed for the bounded Agent PDF integration; owner review pending.**
This is a source-build checkpoint, not an installed release decision.

## Identity and construction

- Final source inventory: `source-subject.json`, 361 code/config/test/schema/fixture files;
  SHA-256 `82a90e663c84edb5ceb25a5232b48876ebe7bcbe436519eae558522193737024`.
- Native build inventory: `tested-build.json`, 198 output files;
  SHA-256 `c4d51b78c85023263df0b86e16d8a97603ce579411103e2711c862f1a90b1de6`.
- Branch/HEAD: `codex/project-workspace-design` / `7fe3d5c9e10d0cc104a15b6e7aa3def0d15b98f1`
  plus uncommitted work. No commit, push or release was performed.
- macOS 26.5.2 (25F84), arm64; native Electron 44.2.0 / Chromium 152. Node 25.7.0,
  npm 11.10.1, Go 1.27.1; PDF.js 6.3.289, TypeScript 5.9.3, React 19.2.8,
  Vitest 4.1.11 and Playwright 1.63.0. Real provider: Codex 0.153.4, gpt-6-astra.
- Formatting, typecheck, published-schema consistency and production build passed; see
  `format.log`, `typecheck.log`, `schema.log`, `construction.log`. These are construction
  evidence, separate from behavioral results.

Relative to the frozen baseline: **47 code/config/test/schema files modified, 12 added**, no
protected file removed. The 13 published bundles v1–v13, 24 qualification files and 62 engine
files are byte-identical. New output is bundle v14. See `changes.json`. Existing engine edits
were preserved. Source inventories exclude documentation. Final verification-only changes to
native window targeting and measured viewport assertions do not change the native build.

## Behavioral evidence

| Layer                           | Pass condition                                                                                                                                                                                                                      | Result                                                                  |
| ------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| Unit / contracts / architecture | PDF identity, clipped-region scope, containment, stale viewport generations including scroll-away-and-back, turn ownership, source-preview rejection, frozen v11/v7 migration, equivalent provider array validation; prior behavior | **274 passed in 25 files**, `unit.log`                                  |
| Go service                      | Existing contained file/Run service behavior and PDF revision transport                                                                                                                                                             | **passed (cached)**, `go.log`; no Go implementation changed             |
| Native regression               | Existing UI, agents, evidence, tasks/logs/dependencies, storage, lifecycle and PDFs                                                                                                                                                 | **51 passed**, 2.7 minutes, `native-regression.log`                     |
| Final PDF verification          | All four Agent/User shared-reference cases and all four existing User PDF cases, after correcting native window selection and asserting actual viewport width                                                                       | **8 passed**, 25.5 seconds, `native-pdf.log`                            |
| Real signed-in Agent            | Decode visible synthetic page; display returned images; point to an exact smaller region; ask one captured-evidence question; preserve User work through arrival/Show/Return                                                        | **passed**, `live-review.json`, `live-delivery.json`, `live-tools.json` |
| Visual review                   | Actual central PDF / right Chat, authored marker readable without covering its selected text, measured 900×650 content viewport and Return access                                                                                   | **passed for expert review**, `visual-review.md` and native PNGs        |

There are **51 distinct passing native scenarios**, not 59: the eight final PDF scenarios repeat
cases in the regression run. The final PDF run uses the same native outputs. Only the folder chooser
and provider protocol peer are scripted in these scenarios; original PDF bytes, decoding, Main,
React, IPC and persistence are real. `pdf-scripted-delivery.json` and `pdf-scope-delivery.json`
record the controlled peer's exact delivered metadata/results.

Native Agent PDF scenarios prove:

1. Arrival does not alter draft/focus, Surface navigation or local selection. Show switches to
   the referenced page temporarily; Return preserves original page 3, scale, rotation, nonzero
   scroll and selection. Source change rejects Show, opens historical PNG and rejects a stale
   question reply without consuming its draft. Reference/question history survives restart.
2. Scrolling invalidates prior pointing authority. Source-preview, off-page selection, an
   outside-visible target and missing receipts fail; a fresh current observation can point.
3. User Share mark produces no Agent turn. A text-only model cannot observe PDF pixels.
4. Agent opening puts a PDF beside a protected User file and leaves the composer focus intact.

## Failures found and corrected

- **Product integration:** Question currentness used the generic reader, which intentionally
  rejects raw PDF delivery. `checkPdfSource` now verifies exact source identity without decoding
  another page. Native question creation and stale reply tests pass.
- **Provider compatibility:** Actual Codex rejected Draft 7 tuple-form coordinates at thread/start:
  `dynamic tool input schema is not supported for workspace_observe: invalid type: map, expected a string`.
  `codexToolSchema` now adapts fixed homogeneous tuples to equivalent arrays. Main validation and
  published schemas remain unchanged. A real runtime registration probe and the completed Agent
  trial verify the fix; the new unit case checks that invalid lengths/types remain invalid.
- **Presentation:** The full pointer note obscured selected PDF text. Final markers display only
  the author outside the rectangle. Full explanation remains in Chat.
- **Verification harness:** An exclusion-based window lookup could select a temporary PDF host
  while its URL was still empty. This made a nominal compact capture remain 1280 pixels wide.
  The harness now identifies the `app://gobble/` shell positively and asserts `innerWidth === 900`
  before capture. Final screenshots are 1800×1300 physical pixels at DPR 2. See
  `resize-diagnostic.log` for the rejected earlier assumption. No production window leak was
  inferred from that in-flight host.
- **Construction:** Exact-optional typing, current-version assertions, a test import and the
  package-boundary check were corrected before final verification; no check was disabled.

## Actual Agent and isolation

The successful trial used the real signed-in gpt-6-astra provider through pinned Codex 0.153.4.
A temporary byte-forwarding diagnostic wrapper was present; it supplied no replies and changed no
protocol payloads. The trace contains two image-display calls, an exact PDF point and one question.
The selected PDF rectangle was `[60, 570, 280, 640]` on page index 0, around the left heading and
caption. This is one successful synthetic trial, not a general guarantee of visual interpretation.

Only an isolated synthetic Project and temporary App profile were used. The existing login was
used locally without printing credentials; the normal Workspace was untouched, the original
credential file remained unchanged, and temporary credentials/profile were removed after the
owned process closed. `live-isolation.json` records this. The successful live trial preceded the
small marker-label presentation correction; final native screenshots validate that correction.

## Remaining limits and checkpoint

PDF page/region semantics only: no exact PDF text targeting, OCR, passwords, form/annotation
appearance, interactive PDF actions, off-page Agent browsing or CSV chart generation. Existing
size/deadline/image-delivery limits still apply. Source availability is explicit; no automatic
retargeting or execution controls were added. Installed/signed builds, Linux/Windows and general
representative-user usability were not tested here. The owner reviews this result before the next
View family, proposed R4 Notebook qualification.
