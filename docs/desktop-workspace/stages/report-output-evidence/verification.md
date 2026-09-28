# Output evidence — verification and handoff

2026-09-28. Retained uncommitted result; no staging, commit or publication.
[Environment](environment.json), [build input/image identity](engine-image.json),
[initial inventory](before.json), [final inventory](after.json), [reach](reach.json).

## Final checks

| Subject | Command / observation | Result |
| --- | --- | --- |
| Native service, macOS arm64 | `go test -race ./internal/appservice` | Passed, 20.545 s; [log](native-final.log). Opt-in live test skips without its explicit environment. |
| Engine + CLI, Linux amd64 | `go test -vet=off ./internal/engine ./cmd/gobble -run 'Test(OutputEvidence\|Continuation\|PreparedContinuation\|AdmissionPublishedBeforeTaskAndNeverRescheduled)' -count=1` in the recorded image against mounted final source | Passed, engine 8.811 s and CLI 0.067 s; [log](engine-final.log). |
| App consumers + rebuilt service | `npm test` from app | 380 tests in 41 files passed; [log](app-tests.log). |
| Native file guards, Linux amd64 | `go test ./internal/appservice -run 'Test(RunReport\|ReportQueryErrors\|ServiceHasNoEngineOrThirdPartyImports)' -count=1` | Passed; [log](linux-native.log). |
| Native static checks | `go vet ./internal/appservice` | Passed; [log](native-vet-final.log). |
| Formatting / retained work | gofmt on changed Go files; `git diff --check` | Passed. Inherited changes preserved, affected reach enumerated. |
| Actual engine capabilities | Separate output and prepared queries; older image label inspection | Output v1; original prepared launch v2 / continuation v1 unchanged; old image has no report label. [Record](capabilities.json). |
| Actual fresh Run | `GOBBLE_LAUNCH_IMAGE=<recorded image> GOBBLE_REPORT_EVIDENCE_DIR=<stage>/live-final go test ./internal/appservice -run '^TestLivePreparedLaunch$' -v -count=1 -timeout 10m` | Passed, 144.22 s; [log](live-final.log), [actual returned bytes and attribution](live-final/actual-report.json). |

The engine test command keeps the inherited `-vet=off` setting for this Linux
package suite; it does not claim a clean whole-engine vet or root-library test run.
Native service vet is separately clean. Earlier Stage 6 packed-executable failures
are outside this slice and are not reported as repaired.

## Actual qualification

A fresh synthetic 100-read single-end analysis was created, checked, adopted,
prepared and started through existing native owners on the new exact engine.
Trim Galore and FastQC both completed. The service restarted without replacing the
controller or Run. The new authenticated report endpoint returned **537,181 bytes**,
recipe `fastqc-v1`, FastQC attempt 1, declared `html` port and the original launch
attribution. Their SHA-256 exactly equals the engine-recorded hash and source file:
`8f045465fd56bbfb18360592e1b41c4978b45c6054fb6bc2cef52aa724413d52`.

Changing bytes without changing length failed; deleting the file failed; restoring
original bytes returned identical content and producer evidence. The temporary Run
and native profile are test-owned and cleaned after completion; the returned source
bytes/attribution are retained in the stage evidence. No User research data was used.
This is native integration with real tools, not a new Electron UI or Agent test.

The local image is `sha256:f5f69ad6d965fcbb2217dd89a621effc4b3a60f13a113781571915fee4b0c971`.
Its compiled Gobble production inputs match the final engine/library/CLI tree.
Later native-only file guards and test changes are verified from the final host
checkout; they do not enter the Gobble executable. The local image carries the new
output label, and the production Dockerfile now declares it for future builds.
The full distributed runtime image was not rebuilt or published.

## Failure reconciliation

- Wrong-Run engine fixture attempted an illegal checkpoint rewrite; existing admission
  immutability rejected setup. Test now requests a wrong Run without modifying history.
- Native fixture omitted its report parent directory; fixed setup. HTTP test then
  decoded the envelope incorrectly; fixed to read `value`.
- Cleanup relocation left an unused import; removed.
- Initial component traversal used x/sys. The existing native standard-library-only
  boundary test rejected it. Replaced it with pinned no-follow file handles and
  os.Root identity checks; boundary/race tests pass without relaxing that test.
- Initial actual run passed before that last platform revision. A second fresh Run
  and final native code passed again in live-final.log; prior log remains retained.

## Limits and next boundary

Engine/native provenance and byte acquisition are complete. Next slice adds the
bounded FastQC reader, Main retention and lower-Pane Report View. TypeScript report
contracts and generated schemas begin there, when those consumers exist. Whole-report
Agent attachments, chart-read authorization and saved-view references follow it.
No workspace schema, bundle version or shared-toolset change occurred here. No report
UI, live Agent consumption, historical-attempt output access or old-engine substitution
is claimed. Per-stage result approval remains in effect.
