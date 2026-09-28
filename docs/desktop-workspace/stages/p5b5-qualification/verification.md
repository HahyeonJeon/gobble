# P5B-5 verification

Development qualification on macOS arm64 / Docker Desktop linux/amd64. Exact tools, daemon and image identities are in `environment.json`; engine build source inputs are in `engine-image.json`. No private research data, provider inference, registry publication or packaged installation is involved. The Agent peer and native file chooser are deterministic test doubles; production Electron, Main, native service, engine admission, Docker, Trim Galore and FastQC are real in the live scenario.

## Commands

Run from the repository root unless indicated. Logs retain each invocation; failures are classified below.

- `python3 docs/desktop-workspace/stages/p5b5-qualification/build-engine.py`
- App directory: `npm run build`, `npm run build:service`, `npm run typecheck`, `npm run schema:check`, `./node_modules/.bin/vitest run`.
- App directory: `GOBBLE_CREATION_LIVE=1 GOBBLE_CONTINUATION_LIVE=1 GOBBLE_PREPARATION_IMAGE=sha256:c7f3c4aa35767297db1f0acc84bc08477673f5f7429692ca594d06ba6e0c74e3 ./node_modules/.bin/playwright test desktop/tests/electron/pipeline-creation.spec.ts -g 'and launch' --output test-results/p5b5-live-N`.
- `GOTOOLCHAIN=local GOPROXY=off go test -race ./internal/appservice ./cmd/gobble-service` and matching `go vet`.
- Engine image, `/opt/gobble`, network disabled: `GOMAXPROCS=2 GOPROXY=off go test ./internal/engine ./internal/engine/exec ./internal/preparation`; `go test -race ./internal/engine -run 'Test(Continuation|PreparedHistory)' -count=1`; `go vet ./internal/engine ./internal/engine/exec ./internal/preparation`.
- Older engine regression: `GOBBLE_LAUNCH_IMAGE=sha256:5d7b7247b407844149bc80173e8da2bf656077b827dd3f2ae9149ee490258e76 GOTOOLCHAIN=local GOPROXY=off go test ./internal/appservice -run '^TestLivePreparedLaunch$' -count=1 -v`.

## Failures and classification

1. Initial type check found an unused local in the new scenario. The local now asserts that the admitted Run matches the catalog. No product defect. An initial live invocation was interrupted before Start after noticing the scenario used `execution_history` instead of the existing `executionHistory` checkpoint key; `electron-live-1.log` records the interruption, not a pass.
2. `electron-live-2.log`: initial Start, real Trim completion, real FastQC Stop, successful continuation review, missing-output refusal and changed-output refusal passed. The test then received EACCES writing the deliberately protected input copy. Corrected the opt-in fixture to restore exact input mode/content in `finally`. Product input protection was functioning correctly.
3. Live refusals were correctly blocked but all displayed a generic Docker error. `blocked-reason-before.log` reproduces the product defect through the real bounded process runner and native worker. The native transport now retains private bounded diagnostics, while the continuation adapter translates only known structured, path-free messages. Unknown and malformed diagnostics remain generic. This change creates no execution or permission path.
4. Engine tests and focused race tests pass. Standard engine vet reports two pre-existing unkeyed external struct literals in `internal/engine/prepared.go:89` and `internal/engine/prepared_test.go:15`; these files match the pre-edit baseline. The follow-up `-composites=false` run is separate limited static evidence, not a standard-vet pass. No unrelated cleanup or broad full-repository test claim is made.

## Evidence matrix

| Behavior | Evidence layer | Result |
| --- | --- | --- |
| Initial Start on the new engine; Trim success; FastQC Stop | Real Electron/native/Docker/tools, 1,000,000 synthetic reads | Passed, `electron-live-4.log` |
| Two same-Run confirmations; FastQC Attempts 2 then 3; latest success | Real tools | Passed; one Run / two receipts, `evidence/final.json` |
| Original Start admission, reused trimmed bytes and original logs preserved | Checkpoint equality, byte equality, SHA-256 | Passed; four original log hashes unchanged, eight logs retained |
| Missing output, changed output and changed checked input | Actual stopped workspace files | Passed; exact original bytes/mode restored, clear blocked reasons |
| Review after restoration uses the same content digest | Real review protocol | Passed |
| App restart before confirmation and during final execution; exact repeated confirm | Real Electron/native/engine | Passed; no extra Run, receipt or task attempt |
| Original captured review after blocked recheck | Actual Electron + immutable saved capture | Passed; old review still names planned Attempt 2 |
| Stop still settling, stale review, competing intent, old-owner Stop, changed installed tool, uncertain publication | Linux engine tests with controlled executor/daemon | Passed; real Docker tool replacement is not performed |
| Unknown receipt, interrupted service, protocol/epoch mismatch, exact controller binding, corrupt/cross-Run input | Native tests with controlled runtime + race checker | Passed |
| Safe known diagnostic translation; malformed/private stderr suppressed | Real OS process boundary with failed-Docker test peer | Passed; red/green regression retained |
| Main, contracts, evidence and existing P5A behavior | 41 Vitest files / 380 tests | Passed |
| Older qualified engine Start, service restart and completed controller removal | Real native service + tools, 100 synthetic reads | Passed, `legacy-launch.log` |
| Original-engine P5A visual refinement and separate new analysis | Real Electron/native/tools, scripted Agent | Passed, `electron-feedback.log`, 6.2 minutes |
| Agent exact pointing, Current-newer saved-design context, unknown acknowledgement, old-engine controls | Deterministic Electron runtime/Agent peers | Passed, `electron-regression.log`, 5 cases / 44.6 seconds |

`electron-live-4.log` completed in 3.5 minutes. The previous failed invocations remain diagnostic evidence and are not counted as successful qualifications.

## Limits

This qualifies a bounded saved single-end Trim Galore → FastQC development workflow. It does not qualify arbitrary pipelines, remote execution, broader crash recovery, packaged/released applications, Windows/Linux desktop, scientific correctness or representative-user/assistive-input usability. Unsupported-engine, tool-drift, lost-receipt and uncertain-publication cases use existing controlled engine/native/Electron tests, not destructive changes to shared installed tool images. Saved-current separation and exact Agent references are deterministic shared-view scenarios, not a real provider claim.

5. `electron-live-3.log`: all three drift refusals and App restart passed. The first receipt query ran before admission publication and correctly retained an unknown outcome. The test incorrectly assumed immediate admission without the existing explicit status refresh. A read-only exact receipt query subsequently proved admission (`receipt-diagnostic.json`). The scenario now uses the existing Check continuation status control until this same intent is verified, never re-confirming. Clarified the native unknown-outcome copy: analysis may already be running; Refresh submits no new work. The previous wording could imply that the original confirmation had not executed.


## Visual observations

Actual wide and compact completion screens show two succeeded Flow steps, the latest FastQC attempt 3 and retained original task attachment. Compact Workspace/Chat uses the existing region switch; task content scrolls within its region. The confirmed plan remains historical evidence and the current execution status is separately labelled. Two small copy corrections were made after this inspection: succeeded guidance now says the analysis is complete, and the Chat card explicitly labels its saved confirmation plan. Final restored screenshots verify those changes on the actual completed Run.

The App account is intentionally dormant after test restarts, and the composer displays its existing connection guidance. This is a deterministic provider fixture, not evidence of a real account connection failure. The inherited sidebar Run list still refreshes through its existing controls or Project/App reopen; P5B adds no new catalog subscription.


## Final regression commands and retained state

- App: `GOBBLE_CREATION_LIVE=1 GOBBLE_FEEDBACK_LIVE=1 GOBBLE_PREPARATION_IMAGE=sha256:5d7b7247b407844149bc80173e8da2bf656077b827dd3f2ae9149ee490258e76 ./node_modules/.bin/playwright test desktop/tests/electron/pipeline-creation.spec.ts -g 'and launch' --output test-results/p5b5-feedback`.
- App: `./node_modules/.bin/playwright test desktop/tests/electron/continuation-review.spec.ts desktop/tests/electron/continuation-transport.spec.ts desktop/tests/electron/observed-reference.spec.ts --output test-results/p5b5-regression`.
- After final copy changes and `npm run build`, App: `GOBBLE_CONTINUATION_RESTORE_FIXTURE=../docs/desktop-workspace/stages/p5b5-qualification/evidence/fixture.json ./node_modules/.bin/playwright test desktop/tests/electron/continuation-restored.spec.ts --output test-results/p5b5-restored`.

Six terminal controllers belonging to the disposable P5B attempts and two from the completed P5A regression were removed only after checking their state and exact fixture mount. Their logs and identities are retained in `controllers/` and `controllers.json`; earlier images and all synthetic Run files remain available. Completed controller removal is followed by the final restored-App check, so receipt retention is not inferred from a live container. No inherited process/image/source was removed, and no files were staged or committed.

The final `after.json` records this stage's added/modified files; `before.json` preserves the full inherited baseline. Build-output hashes are recorded separately in `tested-build.json`. These identify an uncommitted development result, not a release artifact.


Final restored-actual-fixture check passed (`electron-restored.log`, 3.3 seconds) on the rebuilt App, including complete-state guidance, saved-plan labelling, original planned Attempt 2 versus current successful Attempt 3, compact draft, no Resume action, one retained Run and two receipts after controller removal. The final wide, compact and historical-preview screenshots were inspected directly. Final build, type/schema, scoped format and diff checks pass. Native race/vet checks pass; the engine standard-vet limitation above remains. Existing frozen schema bundles and inherited files are preserved.
