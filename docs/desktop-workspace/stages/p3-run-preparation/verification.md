# P3 verification

Request: `p3-preparation-20260912` (Development → Testing).

Final verdict: P3 passes within the qualified scope below. Initial failures, final results and tested build identity are retained separately.

## Initial findings retained

- Construction: the first TypeScript slice used an incorrect existing input-observation export name; corrected to `CreationInputObservationSchema`. A string-index nullability diagnostic was corrected with `charAt`. These were compile failures, not runtime passes.
- Boundary-test maintenance: the native import test initially refused the new `internal/preparation` package. This is a mechanism-free shared record contract, alongside `internal/pipelinereview`; it has no engine/provider dependencies. Added that exact allowed contract. Engine import remains prohibited.
- Product defect, actual isolated engine: the generated Go driver still classified `prepare` as an installed execution verb. Outer dispatch alone was insufficient. Corrected the inner driver guard, retained the failed live log, added a regression and rebuilt the local image. The failed intermediate image was removed so discovery will not offer it.
- Test defect: source-drift injection attempted to overwrite deliberately read-only retained source. The isolated fixture now explicitly makes that one test file writable before injecting drift. Production source permissions remain read-only.
- Review finding: durable Agent-mark validation initially accepted only comparison changes. Added the exact preparation-section alternative and changed the host test to validate a real Workspace document after publishing, including the originating Agent submission.
- Review finding: preparation marks must not evict earlier references at the retained limit. They now use the existing explicit-limit refusal.

Native and Electron qualification uses temporary Projects/profiles and fabricated metadata fixtures, never real research datasets. Actual research tools are never launched. Authenticated ChatGPT is not required in P3 acceptance: the deterministic provider peer exercises production Main authorization, immutable Chat context, tool calls and native Gobble. It is not evidence of live-model response quality.

- Architecture review finding: arbitrary Project Go must not be the final authority for its own safe review. Added a separate trusted `prepared-review` reader which decodes the exact private bytes without mounting or invoking source. A mismatched evaluator/reader projection now fails before publication; tests cover that refusal and regeneration of Flow across supported quality/input variants. The earlier passing live tests predate this stronger reader and are retained as intermediate evidence, not final qualification of it.

- Test maintenance: the capability inventory initially expected the previous mutation list. Updated it for the explicit User preparation/cancellation APIs; no Agent execution capability was added.
- Electron attempt 1: actual preparation succeeded; a broad `.review-context` selector matched both the composer and an earlier sent message. Scoped the test to the sole composer. The retained screenshot also revealed excessive scrolling from three stacked cards; section buttons now reveal one review card while preserving the Flow.
- Electron attempt 2: preparation, exact Agent read/point, restart, metadata drift and a second preparation succeeded. The final assertion incorrectly expected a sent reference to remain in the composer. Existing Send correctly consumes that draft reference; the sent submission retains it. Acceptance now explicitly reselects the historical message and verifies both original submission and new draft still name the first preparation.
- Formatting check scope: an initial root-level invocation included generated JSON and frozen storage snapshots. Formatted the newly introduced Workspace18 snapshot without changing its JSON value. All changed App source including that snapshot passes the App formatter; schema:check validates the generated bundle. Previously frozen files remain untouched.


## Final qualification

| Check | Result and evidence |
| --- | --- |
| TypeScript and production development build | Passed; [typecheck](types-final.log), [build](build-final.log). Electron44.2.0 / React19.2.8 / TS5.9.3. |
| App unit/contract/ownership checks | 359 tests in 37 files passed; [Vitest](vitest-final.log). Covers private-byte exclusion, exact response associations, Agent read-before-point/author policy, valid persisted marks, Workspace18→19 migration and original-byte archival. |
| Native service | Race-enabled service tests and focused vet passed; [race](native-final.log), [vet](vet.log). Covers replay, conflicts, source/data drift, cancellation/restart, payload tampering/privacy and evaluator/trusted-reader disagreement. |
| Actual Linux Gobble engine/root/CLI | Focused prepared/creation/review checks passed in all three packages; [engine checks](engine-final.log). Includes private codec round-trip/canonical integrity, explicit field compatibility, hidden-behavior refusal and independent safe-Flow regeneration across supported quality/input variants. |
| Actual native source/runtime upgrade | Passed in 322.09s; [live native](live-native-final.log). Older source engine creates/adopts and refines; final preparation engine independently prepares both exact Currents with replay and zero Runs. Printed `engine=` values in this log are runtime-binding digests, not image IDs; see [image identity](engine-image.json). |
| Actual Electron P3 journey | Passed in 3.6 minutes; [acceptance](electron-final.log). Creation/adoption, preparation, Data/Settings/Environment, exact Agent tools/marks, two restarts, metadata drift, second preparation, historical message reselection and zero Runs. Uses actual native service and both pinned engines with a deterministic Agent protocol peer. |
| Existing workspace/B regression | 9 tests passed in 3.5 minutes: B visual refinement/Agent discussion/adoption/restart plus eight workspace, keyboard, pane, small-window, persistence and migration scenarios; [regression log](electron-regression.log). |
| Schema and formatting | Bundle24 consistency and all changed App source formatting passed; [schema](schema-final.log), [format](format-final.log). |

[Source delta](source-delta.json) records final source hashes against the explicit [before manifest](before.json), not against the much larger inherited dirty Git baseline. [Tested build/environment](tested-build.json) identifies local output and the engine. The local image is reproducible using [the bounded build script](build-engine.py); it is a development qualification image, not a published release.

## Visual review

Actual Electron screenshots were inspected after acceptance. Flow and Chat remain the main surfaces; the compact review uses section buttons and its own bounded scroll area. Historical sent references display their immutable review time, independently of whichever history item the User currently browses. User can select that saved review through Review history. The saved reference does not silently retarget to the newest plan.

- [Prepared settings and Agent discussion](electron-final/pipeline-creation-Creation-39e6d-cussion-restart-and-prepare/preparation-review.png)
- [Metadata change makes the prior review historical](electron-final/pipeline-creation-Creation-39e6d-cussion-restart-and-prepare/preparation-stale.png)
- [Second preparation with the earlier exact Chat reference retained](electron-final/pipeline-creation-Creation-39e6d-cussion-restart-and-prepare/preparation-new-review.png)

The English messages naming fixture scenarios are deterministic acceptance inputs, not product example conversations or live-model responses. Intermediate failures/screenshots remain in `electron-attempt1` and `electron-attempt2`.

## Limits and next boundary

Qualification is local macOS arm64 App/native service with Linux/amd64 Gobble in Docker, within the App-created single-end Trim Galore → FastQC scope. Existing native-macOS root-engine support and earlier packed-executable gaps remain separate. This is not a full root-engine suite, packaged-app, cross-platform or live-model-quality claim.

Input metadata observation does not prove byte equality, FASTQ validity, tool availability, output workspace availability or a staged data snapshot. P4 must qualify those conditions and admit the exact reviewed private payload with explicit User launch intent, durable reconciliation and controller-owned conditional Stop. No Run, analysis process, document editor or CSV chart builder was added here.

The newly frozen Workspace18 snapshot was also compared directly to the published bundle23 schema after parsing: [exact storage-schema check](storage-snapshot.log).
