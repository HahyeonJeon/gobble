# R3b2 verification record

Subject: [baseline.json](baseline.json), [changed paths and digests](implementation-subject.json)
and [all final App source digests](final-app-source.json). Review mode: the implementation author
reviewed the affected owners and ran the Electron Testing workflow in this same task.
Unrelated pre-existing engine changes are excluded. This is a source-checkout checkpoint, not a release.

## Verification results

| Evidence | Result and scope |
| --- | --- |
| [Final full App check](check-3.log) | Passed formatting, process/tooling type checks, schema consistency, Go service build, **254 unit/contract tests in 23 files**, production Main/preload/renderer build and **42 native Electron scenarios**. |
| [Final subject](final-app-source.json) | Every App source digest was compared again after the final check and visual walkthrough; no App source changed. The affected set contains 65 new/modified App paths, largely version-boundary propagation and frozen readers. |
| [Native walkthrough images](visual-review.md) | Built App at normal size, split panes, historical attachment preview, 900×650 at 150%, and bounded list fallback. Reviewed by the author; synthetic data only. |
| [Compatibility audit](compatibility-audit.json) | Passed frozen bundles, archived capture/Run/tool request shapes, unchanged product dependencies and 78 untouched baseline non-App code files. |

`check-2.log` also passed all 254 unit/contract checks and 42 native scenarios before the final
no-op reveal correction. `check-3.log` is the final full result, including explicit coverage of
that correction and search-hidden selection recovery. No tests were skipped or disabled.

The [compatibility audit](compatibility-audit.json) passed: published v1–v10 schema bytes,
RunPresentation, archived observed captures and v4/v5/v6 Agent request shapes stay unchanged.
All 78 baseline non-App code files remain unchanged. Product manifests and lockfile are unchanged;
Plotly, React Flow and Dagre are absent from product dependencies. The v8 storage reader is compared
structurally with the published v10 bundle; the real storage writer backs up original bytes before v9 writes (unit integration coverage).

## Affected-owner review

- Frozen readers are named separately from current contracts. Reference v4 is activated only for
  User LocalSelection and captured evidence. Main uses the separate dependency revision; old task
  and log source revision algorithms keep their original inputs. Renderer coordinates are not IDs.
- Main obtains both presentations from one monitor read and retains it through navigation and
  capture. Refresh is the only newer read. The existing RenderSession bounds count the entire
  observation. Optional dependency context is dropped with an explanation if it alone would make
  Tasks inadmissible; the byte budget itself is unchanged.
- Mode, query/representation and camera have narrow separate command inputs. A delayed camera
  write cannot replace a search. Camera changes keep the semantic receipt; presentation changes
  invalidate it. A no-op Show action does not enter the presentation-loading lifecycle.
- RunView owns the single readiness/freshness notice and mode composition. Tasks and Dependencies
  retain their mounted navigation state while inactive content is hidden from accessibility. The
  dependency adapter owns bounded geometry; detail owns group/pair/member actions. Task filtering
  has explicit recovery for a selected instance outside the filter. Current-task navigation keeps
  prior dependency camera state and explicitly returns to Tasks.
- Capture uses the existing immutable blob publication, quota, draft preparation, preview and Send
  path. No new store, manager, polling loop or privilege bridge was introduced. Capture bytes are
  validated against their exact target and timestamp; later source changes never recapture a draft.
- The old Agent view tool refuses dependency displays instead of reporting hidden task rows as
  visible observations. Explicit source-preview reads remain v3; graph observe/point/marks and
  temporary Show/Return require R3b3.

## Classified failures and corrections

| Record | Classification and correction |
| --- | --- |
| construction-1 through construction-4 | Incomplete construction while propagating the new discriminated union: unused frozen imports and old `.origin`/`.text` accesses needed explicit version/content narrowing. No casts widened Agent inputs. construction-5 passed. |
| unit-1 | Three test expectations used 9 as an unknown future Workspace version. Version 9 is now current; unknown-version fixtures moved to 10. No migration acceptance check was removed. |
| native-1 | Test timing defect: `check()` required immediate DOM state for an IPC-controlled radio. Replaced with a real click and an awaited checked-state assertion. |
| native-2 | Test setup defect: the new Agent had not been selected as the composer recipient. Added the existing explicit recipient selection; Send remained disabled until a valid recipient was ready. |
| construction-7 | One obsolete imported test helper after splitting camera/navigation commands; removed. |
| native-3 | Hidden-mode duplicate freshness text made an unscoped text locator ambiguous. Review consolidated the common observation notice under RunView, its actual owner; native-4 passed both new journeys. |
| check-1 | 253 unit/contract checks and 41 of 42 native scenarios passed. One old native test's text locator also matched hidden dependency detail. It now explicitly asserts visible text. All scenarios remain enabled. |
| Inspection, followed by bounds-unit-1 | Optional dependency data could consume the existing Tasks byte allowance. The retained-data owner now omits that optional preview when needed. A realistic 1,000-instance fixture proves original Tasks remain usable and source identity unchanged. Six dependency integration tests passed. |
| Inspection, followed by final native scenario | Show with an already empty query would set readiness false without changing presentation identity. The Surface coordinator now treats this as a camera-only reveal. The final scenario asserts continued Discuss readiness. |

The first attempted native build used a relative log path from the wrong directory and did not
start; the corrected invocation used the repository's stage-review directory. Original logs are
retained. No system/account change, skipped scenario or relaxed exact-target validation was used.

## Limits

Tests run on macOS arm64 with Electron 44.2.0, React 19.2.8 and TypeScript 5.9.3, using the built
App/CSP, real Go Project service and temporary synthetic Projects. The Docker CLI and provider
executable are deterministic test substitutes. Synthetic provider input assertions prove captured
content delivery, not a real Agent's interpretation. There is no new live Docker or signed-in
Agent trial, Windows/Linux result, installed lifecycle/signing/update result, screen-reader study
or general usability claim. Owner walkthrough remains the next review. CSV chart creation stays
removed; existing CSV table selection and historical evidence stay covered by the regression suite.
