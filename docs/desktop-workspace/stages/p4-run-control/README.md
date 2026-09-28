# P4 Run control — implementation

Status: accepted P4 scope implemented and verified. Owner accepted the concrete P4 design and requested implementation on 2026-09-12. Parent: [accepted design](../../proposals/p4-run-control/README.md).

Author work covers engine prepared admission/expected-owner Stop, native launch-review/staging/operation reconciliation, typed App bridge and existing Flow/Chat integration. No Project code is evaluated at Start. Test effects are limited to synthetic input, owned temporary Projects/controllers and local build/cache outputs. No real research data, credentialed provider call, publication or packaged release.

Compatibility refinement: existing control schema2 remains the ordinary Run contract. Admission-bearing checkpoints use pointer format2 (old readers reject it) and an explicit version1 admission record inside run.json. The new engine supports pointer formats1/2 and validates the admission-bearing pointer/record pair. A dedicated admission query conveys the new command receipt; the existing monitor projection remains schema2. Prepared payload v1 is unchanged. This isolates the new durable admission boundary without a gratuitous global rewrite of unrelated Run DTOs.

Dependency order: shared admission DTO → Gobble codec/admission/Stop and CLI → native staging/launch records and runtime → closed IPC/Main/preload → compact React run controls and inline action card → behavior/visual acceptance. Existing scheduler, checkpoints, Run monitor, evidence and Chat writers retain ownership. Deletion is limited to owned incomplete staging after terminal cancellation, never admitted data/results or inherited code.

Dynamic handoff — Development → Testing: `p4-run-control-20260912`. Test prepared-byte/input/runtime/target mismatch, duplicate and interrupted dispatch/admission, late-owner Stop, copy cancellation and corrupt FASTQ, durable intent/restart, exact Run/step communication, truthful unavailable status and original P3/B/workspace regressions. Lowest layers: engine fault injection, native isolated/race checks, App closed-contract/authorization checks, then actual Linux/Docker and Electron scenarios. Targets: macOS arm64 App/service, Linux amd64 Gobble. Go module1.26/local1.27.1, Electron44/React19/TS5.9. Initial failures and final tested source/build identities will be retained separately. No full root/macOS, cross-platform or packaging claim.


## Result

The User can check a copied input and installed tools, confirm Start in existing Chat, follow the exact admitted Flow, discuss an observed task, reconnect after App/service restart, and request Stop for the observed owner. Gobble owns execution and settlement. No Agent Start/Stop tools or document editors were added.

[Concepts and module boundaries](architecture.md) · [Verification and known limits](verification.md) · [Actual Run Flow](ui/run-flow.png).

Actual Trim Galore → FastQC succeeds on synthetic reads. A second synthetic test restarts the native service while the controller runs, requests Stop and proves a stopped checkpoint with inactive occupancy. Closed-contract tests, engine/native checks and actual Electron acceptance cover identity, replay and exact task discussion. The two existing pack-test failures reproduce in the P3 baseline and remain outside this milestone.

Next decision: review P4, then sketch and approve P5 feedback/reuse/Resume. No P5 implementation, publication, commit or packaging was performed.
