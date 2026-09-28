# P4 verification

## Accepted outcome

The complete approved P4 scope is implemented: copied input/FASTQ/tool/target check, explicit typed Start in existing Chat, exact durable admission and replay, independent controller lifetime, retained Run Flow with exact task discussion, and expected-owner Stop. Tests execute only synthetic research fixtures. No real account/provider request, real research dataset, remote release or packaged build was performed.

| Evidence | Result |
| --- | --- |
| `native-race-verified.log`, `native-vet-verified.log` | Native service race and vet checks pass, including copy validation/cancel, stale target, checksummed records, replay, ambiguous-admission guard and idempotent shutdown |
| `engine-release.log` | Admission, checkpoint downgrade refusal, Stop, prepared payload, cancellation during installed-tool check and CLI-focused regressions pass |
| `engine-regression-unprivileged.log` | Full engine and executor suites pass; two unrelated CLI pack tests fail (see limitation below) |
| `live-launch-fourth.log` | Actual 100-read Trim Galore → FastQC execution succeeds; identical Start is idempotent, another request is rejected, only one Run exists, completed controller removal preserves admission |
| `live-stop-restart-verified.log` | Actual 200,000-read launch, native service shutdown/reopen, exact owner Stop, stopped checkpoint with inactive occupancy, and reconnection pass |
| `vitest-verified.log` | 38 files / 363 tests pass |
| `typecheck-final-checked.log`, `schema-final-checked.log`, `build-verified.log` | Type checking, generated bundle25 check and development build pass |
| `electron-regression.log`, `electron-final-ui.log` | Broad Electron sweep: 66 pass, 8 opt-in scenarios skip; old bridge-inventory assertion was updated and all six foundation cases then pass |
| `electron-launch.log` | Full real Electron creation → preparation → copied data check → explicit Start → Run Flow → exact task discussion passes |
| `electron-confirmed.log` | Final retained-Run restart/compact-region/task-reference scenario and Agent Run/log reference regression both pass |

## Visual review and corrections

The actual App was operated through its normal controls in the Electron scenarios. Screenshots were inspected at full desktop size and compact width. Status badges initially crowded step names; they now sit in the node footer. The Run keeps its rounded right-angle Flow connections, status colors and independent selection. New Run titles are compact (`Analysis N`). Chat remains the sole composer. Compact width uses the existing Workspace/Chat region switch, with both regions explicitly exercised.

A fast Flow selection followed by Discuss needed a pending-selection barrier: actions now wait for the Workspace owner's acknowledgement, and the test checks that the newly attached evidence is **FastQC attempt 1**, not a prior task. The original unsent draft remains intact. One Project observation cache keeps the review panel, Chat card and Run Flow on the same launch-review read.

Final screenshots: [Run Flow](ui/run-flow.png), [compact workspace](ui/run-flow-compact.png), [compact Chat](ui/run-chat-compact.png). [Initial Start screen](ui/launch-ready-initial.png) records the actual admission scenario before the shared-cache/layout polish. The retained Run in the visual restart fixture was created with the earlier qualified P4 image; the final Stop image is recorded in `engine-image.json`.

## Failures found and resolved

- The P3 preparation image lacked the Linux Docker client required by an independent execution controller. The local execution image now includes the client from the already qualified stage6 runtime.
- Docker normalizes added capability names to `CAP_*`; exact controller verification now checks the actual normalized representation.
- A query container initially lacked the stable daemon host identity needed for Stop. The native host verifies the runtime and supplies that exact identity to its fixed query container.
- Cancellation during the installed-tool query was previously reported as an unknown backend. Before container creation is authorized, cancellation now retains its cancellation meaning and cannot manufacture an unknown task. Added a regression test.
- Repeated service shutdown reported an already-closed catalog lock during restart-test cleanup. Close is now idempotent and tested.
- Responsive/async screenshot assertions initially ran while a different region or previous observation was visible. Acceptance now explicitly switches compact regions and waits for exact task acknowledgement/state.

## Preserved limitations

The broad CLI suite reports `TestPackPrintpipeArtifact` and `TestPackHostpipeEmptyInspectProtocol` failures. Both reproduce with identical failure points against the P3 source image with the checkout's Git metadata mounted (`pack-baseline-with-git.log`). They are outside P4 and are not claimed fixed; packaging remains unqualified. Initial baseline attempts without `.git` failed earlier and are retained separately, not used as comparison evidence.

The all-App formatting sweep reports two unchanged frozen/reference fixtures (`contracts/src/storage/workspace-v17.json`, `desktop/tests/fixtures/creation-candidate.json`); their bytes are preserved. Changed files pass `changed-format.log`; hashes in `before.json` prove those two fixtures were unchanged. No whole-tree formatting or full-root macOS test claim is made. Linux engine qualification is amd64; native App/service qualification is macOS arm64. No Windows/Linux desktop or packaged-release qualification is claimed.

Unknown admission offers reconciliation only; interrupted execution offers recovery review, not automatic Resume. Ready/admitted data and controller evidence are retained. Future disk retention, remote execution, arbitrary workflows and P5 recovery require separate design.


Final source/build hashes are retained in `tested-build.json` and `changed-source.json`. Superseded task-owned P4 engine images and the completed synthetic UI controller were removed after evidence capture (`cleanup.log`); the final execution image and installed analysis tools remain local. The archived UI profile is historical test evidence, not a maintained user Project. Inherited images, source changes and user data were preserved.
