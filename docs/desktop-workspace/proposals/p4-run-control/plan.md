# P4 implementation and acceptance plan

Status: proposed, awaiting owner review of the linked screen and choices. Parent: [P4 design](README.md).

## P4.1 — engine admission and expected-owner Stop

Create focused `internal/engine/admission.go` and public prepared execution APIs, with dedicated no-source CLI entries. Reuse DecodePrepared, the existing scheduler, occupancy and checkpoint writer. Retain the launch correlation in the admission checkpoint before tasks can start. Expected-owner Stop extends the existing lease-specific mechanism; do not add App process killing. Define exact receipt lookup and explicit outcomes for admitted, conflicting, not admitted and unconfirmed. An existing admitted identity is never scheduled again, even when its controller has exited.

Co-touch `run.go`, `stop.go`, `state.go`, persistence/monitor readers and CLI parse/help only as required. A versioned engine compatibility decision must cover all readers, exported identity/monitor contracts and fixtures together; no silent optional-field interpretation in an old strict reader. Keep the private prepared-v1 representation unchanged unless semantic necessity is proved. The entry takes bounded host-created intent/payload files, never Project Go or an arbitrary shell string.

Verify private-payload tamper, wrong runtime/input/workspace binding, duplicate admission, interruption around checkpoint publication, no task before admission, same-request lookup after restart, late Stop and newer-owner races. Use engine executor seams for precise fault injection plus actual isolated Linux CLI evidence. Do not claim the old native-macOS root-library gap is resolved.

## P4.2 — native launch review and operations

Create separate owners for `launch_review` (bounded copy/format checks, installed tool set, target allocation), `launch_operations` (intent journal and reconciliation) and `launch_runtime` (fixed create/inspect/start/controller protocol). Keep opaque payload retention in the P3 owner. Share only mechanism-free DTOs with native service; never import the engine. Extend Run registration with exact launch/preparation association rather than replicating monitor facts in the operation journal.

One active data-copy check per Project, streamed in bounded buffers; cancellation is cooperative and removes only task-owned partials after close. Copying may take longer than an inspection and needs observable byte progress rather than the existing short inspection timeout. Define byte/disk and record limits before source implementation. Proposed first input dialect is nonempty four-line FASTQ / gzip FASTQ with sequence/quality length and complete stream checks; explicitly reject unsupported multiline records. This is format validation, not assay-quality validation.

Closed contracts: create/read/cancel launch review, authorize Start with exact review/request, get operation, request expected-owner Stop. Main obtains actor authority from the trusted window; renderer cannot provide paths, endpoints, argv, credentials or a forged owner receipt. New internal API paths may be `/pipelines/:id/launch-reviews`, `/pipelines/:id/starts`, `/operations/:id`, `/runs/:id/stop`; freeze names with schemas at implementation, not as a promised external SDK now.

Before dispatch revalidate target reservation, staged content, installed tool set, exact preparation and runtime. Mount only the selected result allocation and required retained artifacts; never mount arbitrary Project source with execution authority. Preserve ambiguity after transport loss. Service close cancels local waits/checks, not an admitted controller. Tests cover every crash interval from intent write through controller create/start, engine admission, Run registration and acknowledgement.

## P4.3 — shared Flow, action card and actual acceptance

Compose a typed Run-control host, frozen preload methods, a compact launch-review component, an inline execution-action card in existing Chat, and Run-Flow projection from exact task IDs. Reuse existing Run/task/log views and exact evidence owners. A stable engine Run identity/attempt snapshot must accompany anything User or Agent discusses. Agent marks never move User selection. Authoring remains Agent-owned and independent from execution permission.

Verify these User scenarios:

1. Check a P3 Current; observe copy progress and a concrete data/target/tool review. Cancel safely before Start.
2. Change data, Current, selected runtime or reserved target before Start: no execution; preserve Chat and explain which review changed.
3. Explicitly Start a valid review; repeated clicks/request replay create one Run and one admission.
4. Click a running step, read task/log details, and discuss its exact snapshot in the existing Chat.
5. Close/reopen App during execution: same controller and Run; no new scheduling or automatic Stop.
6. Interrupt dispatch acknowledgement, Docker availability or service recording: truthful reconciling state, never blind retry.
7. Stop the observed Run; show request/settlement separately. A delayed old-owner Stop cannot affect a replacement owner.
8. Actual synthetic two-step analysis reaches expected outputs; an injected long-running test workload proves Stop settlement. Test substitutes must remain separate from real scientific-tool acceptance.
9. Existing P3 history/discussion, B review/adoption and Workspace migration/pane/Chat regression checks pass.

Construction evidence: changed-source format, TypeScript, closed-schema consistency, Go vet, and development build. Behavior evidence: focused engine/native race tests, App unit/contract tests, actual Linux Gobble/Docker and Electron scenario/visual checks. Preserve initial failures, final source/build identity and limits. No test uses live research data or a paid/live-model response as the only oracle.

Completion means all three parts are implemented and verified within the agreed first scope. This design checkpoint alone is not P4 completion. P5 failure-driven refinement and Resume remains later.
