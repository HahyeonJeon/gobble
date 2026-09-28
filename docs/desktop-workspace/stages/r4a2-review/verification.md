# R4a2 verification record

Result: **passed; owner review pending**, 2026-09-09. Request R4a2-2026-09-09,
sequential Development → Testing roles in the same task. The stage was authorized
by the owner's acceptance of R4a1 and instruction to proceed. No next-stage
implementation or independent reviewer is claimed.

## Subject and runtime

- [Source identity](source-subject.json): branch `codex/project-workspace-design`,
  base HEAD `7fe3d5c9e10d0cc104a15b6e7aa3def0d15b98f1`, including uncommitted files.
- [Built artifact hashes](tested-build.json), [runtime tuple](runtime.json).
- macOS 26.5.2 (25F84), arm64; Electron 44.2.0, Chromium 152.0.7977.76,
  embedded Node 24.20.0. Build/test Node v25.7.0, Go 1.27.1,
  TypeScript 5.9.3, React 19.2.8, Playwright 1.63.0.
- Electron-vite's actual Main/preload/renderer and dedicated Notebook worker were
  built and launched. Each native scenario creates its own temporary Project,
  files and profile. Chooser/Account provider responses and crash-dialog choice
  are simulated; file service, worker, IPC, renderer and native image pixels are real.

## Completed checks

| Check                                    | Result                                                                                           | Evidence                                                           |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------ |
| TypeScript process configurations        | Pass                                                                                             | `types-complete.log`                                               |
| Generated schema v15                     | Pass                                                                                             | `schema-complete.log`                                              |
| Full unit/contracts/service suite        | 312 checks, 28 files passed                                                                      | `unit-complete.log`                                                |
| Native build and complete Electron suite | 59 scenarios passed, 3.1 minutes                                                                 | `electron-complete.log`                                            |
| Fresh Go app-service suite               | Pass, `-count=1`                                                                                 | `go-complete.log`                                                  |
| Whole app Prettier check                 | Pass                                                                                             | Recorded tool output: “All matched files use Prettier code style!” |
| Preservation                             | 521 of 579 baseline files unchanged; 58 intended changes; no missing files; 29 new subject files | `preservation.json`                                                |

The 241 protected baseline files, including engine, commands, memory, qualification
and 14 historical schema bundles, are unchanged. The new bundle is v15. Existing
pre-stage engine and memory edits are preserved; no normal App profile is used by
tests. No commit, push, release, live provider or Jupyter operation occurred.

Notebook coverage comprises 26 promoted parser/target/text-window checks, 6 public
contract/compatibility checks, 5 host/worker/evidence checks and one additional PDF
v12 migration check, accounting for the increase from 274 to 312. The eight new
native scenarios account for the increase from 51 to 59:

1. Registered saved source/output → selection → existing composer; offline preview,
   source change/deletion/retry, immutable attachments and draft restoration.
2. Native text drag with absolute display offsets; stale acknowledgment and foreign
   Project image requests refused.
3. Pointer/keyboard image selection at 150%, exact 180 × 81 crop at original
   coordinates (21,20), with native bitmap equality against the saved PNG.
4. 1,000 cells, two independent readers, persisted cell pages after restart,
   compact Workspace/Chat switching and enlarged display.
5. Malformed source/retry and forced renderer crash followed by native reload,
   verifying actual replacement-renderer readiness and restored draft/attachments.
6. 1 MiB text displayed in bounded windows; continuation offsets 2048–4096;
   oversized capture refused without changing existing attachments.
7. A 3.39 MB embedded-image Notebook opens through the dedicated source variant;
   raw generic bridge access refused, image loaded on demand and exact small crop
   captured. Hiding the image removes its display; active content is not executed.
8. Text and image captures sent through existing addressed preparation and explicit
   Send after source removal. A local fixture provider receives both typed target
   metadata and PNG content. This is not real Agent Notebook verification.

## First failures and corrective review

Earlier logs remain alongside final evidence. They are not successful-run claims.

| Observation                                                             | Classification and correction                                                                                                                                                                         |
| ----------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Initial typing/skeleton errors                                          | Construction defects corrected before final testing: discriminated union consumers, schema-version expectations, DOM helper shadowing, and a timeout parameter inferred as a literal.                 |
| First native refresh assertion read an old ready state                  | Readiness and test synchronization: revoke renderer readiness immediately; assert the new revision/cleared selection before reading the result.                                                       |
| Compact test required both regions concurrently                         | Test expectation mismatch: exercise the existing Workspace/Chat switch and verify each region plus retained reader state.                                                                             |
| Forced crash produced `Target crashed` / stale Page errors              | Test lifecycle: arm the crash event before triggering it, wait for native reload and inspect the replacement renderer through its BrowserWindow. Real renderer-loss recovery remains in product code. |
| Initial full units rejected TypeBox imports                             | Missing explicit dependency/boundary declaration: pin existing TypeBox directly in desktop and allow only the private worker/target validators.                                                       |
| Four old negative fixtures called v13 a future version                  | Fixture drift: keep migration success on current v13 and use explicit unsupported v999 for future-version refusal.                                                                                    |
| Rapid output Select → Add captured prior source                         | Product race: block capture while saving replacement selection and reset target-specific toolbar feedback. Keep the scenario's exact output assertion.                                                |
| Large text test watched the generic status; Send test omitted recipient | Test setup/assertion defects: check the actual error banner and explicitly choose the fixture recipient.                                                                                              |
| Review found raw Notebook through generic file IPC                      | Ownership defect: reject that path, retain bounded internal transport, add a real IPC refusal assertion.                                                                                              |

The corrected focused Notebook run passed all eight scenarios; the subsequent
rebuilt complete suite passed all 59, including the added two-reader restart check.
Final source received type/schema/format/unit verification and no further product
code edits followed the successful native run.

## Visual review and limits

Native full-window captures were visually inspected: `notebook-discussion.png`,
`notebook-offline-preview.png`, `notebook-image-selection.png`,
`notebook-compact-workspace.png`, `notebook-compact.png`, `notebook-zoom-150.png`.
Source/output labels, highlight/crop, attachment preview and the single composer
are clear and reachable. Compact and enlarged layouts preserve existing toggles;
no duplicate conversation or answer panel was introduced. Fixtures are synthetic.

See [architecture review](architecture-review.md) and the
[stage result / next ownership sketch](../r4a2-user-notebook.md). Saved nbformat
4.0–4.5 reading is bounded and passive. This is not a full nbformat validator,
Notebook editor, kernel integration or real Agent tool qualification. Agent scoped
reading, marks/questions and Show/Return are proposed R4a3; cross-platform packaging
and deployment remain outside this source-checkout result.
