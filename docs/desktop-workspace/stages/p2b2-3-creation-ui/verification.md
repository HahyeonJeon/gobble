# P2B-2.3 verification

Request: [p2b2-3-r1](testing-request.md). Owner: Electron Development → Testing.
Target/environment: native macOS arm64, Electron44.2.0; actual installed local Linux
amd64 Gobble engine under Docker. Temporary Projects and isolated profiles. No real
research Project or user profile changes. Deterministic Agent peer; real Main/IPC,
native service and Gobble source checking. This proves integration, not LLM quality.

## Construction and native contracts

- **Passed:** final TypeScript construction including renderer/Main/preload/tool tests.
- **Passed:** generated contract bundle22 matches exports; frozen workspace17 migrates
  to18 without allowing new creation fields in the historical document schema.
- **Passed:** touched TypeScript/TSX/CSS/fixtures formatting.
- **Passed:** 350 Vitest tests in35 files. New tests cover per-message opt-in,
  source-read receipts/revocation, stale generation, messages-only access, closed
  identities, absence of raw runtime fields, retained exact native facts and response
  association checks. Existing tests changed only current-version expectations and
  narrowed fixture subjects where the new draft resource expands the union.
- **Passed:** `go test ./internal/appservice`, including local-only labeled engine
  discovery, deduplication, platform verification and unavailable selection refusal.
- **Passed:** complete development build. No dependency additions or package changes.

## Actual Electron cases

The creation scenario exercises native file selection, installed engine connection,
message opt-in, scoped Agent source authoring, real Gobble check, two green added steps,
No current version details, exact Quality threshold attachment, moving selection without
changing the attachment, read-before-point, independent Agent marks, restart, retained
unsent text, context removal independent of data, draft discard and unchanged source
bytes. Catalog remains4 with zero Pipelines and zero Runs.

Existing B proposal/check/adoption/restart and eight Workspace regression scenarios
passed as well: normal relaunch, keyboard tabs/duplicates/moves, compact panes and
context, cross-Project isolation, failed draft save recovery, one composer across
resizing, v1 geometry migration and selection navigation. Evidence is scoped to the
recorded development build; it is not an installed-app or distribution claim.

Screenshots were visually inspected. Corrections: new-step color was scoped to the
older comparison component and is now explicit for creation; numbered additions and
Agent marks no longer overlap; sidebar data names and discard update; Chat displays the
attached setting name/value rather than a generic target type. Narrow windows preserve
the existing Workspace/Chat switch instead of adding a second composer.

## Preserved failures and reruns

- `electron-r1.log`: locator expected the semantic New pipeline name but the decorative
  plus participated in the accessible name. Development supplied an explicit accessible
  label; subsequent runs reached creation/checking.
- `electron-r2.log`: test attempted to locate the composer while compact mode displayed
  Workspace. Product had preserved text; the test failed to switch back. Corrected test
  restores the wide viewport before asserting the composer. All preceding creation,
  pointing and restart observations passed.
- `electron-r3.log`: complete creation scenario passed after UI corrections.
- `electron-regression.log`: existing B and eight Workspace cases passed (9 total).
- Final formatting changed the emitted renderer bundles. Final creation/Workspace
  evidence is recorded separately against the rebuilt artifact in
  `electron-final.log`; Main/preload/native artifacts remained byte-identical.

The first image-label build used a broad temporary context and encountered an unrelated
filesystem xattr permission; the second lacked an explicit Linux target. A dedicated
empty build context and `--platform linux/amd64 --pull=false --network=none` produced the
installed label-only image from the exact prior qualification image. No image download
or engine source rebuild was used. Input: `sha256:4dbd1fb658d8aa6967bf8de6b2368a0dad2c59244306d2a2bf1c9e90e4d702dc`.
The selected image identity and host are in `environment.json`.

## Evidence identity and limits

`before.json` records the inherited source baseline; `after.json` records final source
hashes. `source-changes.json` distinguishes files lacking baseline coverage from known
changed files (unbaselined does not mean authored in this turn). `tested-build.json`
records the prior integration artifact; `final-build.json` records final emitted files.
Fixtures, runner, tests and schemas are in the source manifest. Logs and screenshots
are adjacent. No signing, packaging, OS installation/update, cross-OS Electron,
performance, real LLM acceptance, first creation adoption or Run execution was tested.
These excluded claims remain unsupported by this record; they are not passing proxies.

First adoption is the next owner gate. It must revalidate generation, data observation,
runtime and candidate before atomic Pipeline/Current registration. Historical check
success alone never authorizes registration or execution.

Final result: **passed**, final creation + eight Workspace cases (9/9, 1.4 minutes).
Prior unaffected B proposal/check/adoption/restart regression: **passed**. Ten distinct
Electron scenarios are covered across these records. Final source formatting, type,
schema, native and unit checks passed. The result is ready for the owner's Part3 review;
Part4 has not started.
