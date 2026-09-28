# Stage 4 — Author design review

Reviewed 2026-09-06 as part of the authorized implementation. This is the
implementation author's review, not an independent-agent evaluation.

Subject: `0fec48e8c57d298c36ea1d90c5316cdbac2c756cc4f92674d5ce0fa636dfee81`,
136 files across app/native service, using the digest convention in the
[stage record](../stages/04-agents.md). Scope: account/provider integration,
Project attachments, addressed text delivery, persistence/recovery, IPC and UI.

## Ownership and API findings resolved

| Finding | Resolution and evidence |
|---|---|
| Provider configuration and Project authority could become coupled | The provider working directory is app-owned; registered Project roots are not passed implicitly. The runtime owns an app-only discussion directory and empty environment list. Files/Run metadata still come from the Go service through Workspace resources. |
| Multiple provider processes could race shared credential refresh | One official app-server owns the app-specific account; independent thread/turn IDs retain attachment separation. A shared process crash is explicitly a shared connectivity failure. |
| Application delivery logic initially depended on adapter types | `collaboration/provider.ts` owns AccountAccess, ConversationProvider, ProviderTurn and ProviderEvent. `codex/` implements those ports and contains wire decoding; the coordinator imports no Codex implementation. |
| Attachment binding and input acceptance must commit consistently | `CollaborationStore` narrows the existing WorkspaceController writer to roster/history. Binding, submission and matching draft consumption share one atomic Project commit before provider input. Storage-failure tests prove no unrecorded new input. |
| A dropped acknowledgment could cause duplicate input | A persisted request ID becomes `clientUserMessageId`. Duplicate requests return the existing record; Check status matches official history by that ID. Missing history stays uncertain. The real Electron restart test sees one provider turn, not two. |
| A following disconnect could overwrite a queued terminal result | Disconnect handling rechecks the active delivery identity after preceding events drain. A regression test preserves the completed response. |
| Save failure after turn acceptance could lose cancellation ownership | The transient record retains the acknowledged turn ID even if its next save fails. Quit still targets that turn; storage failure remains explicit, with Keep Open before discarding new unsaved state. |
| Dead browser ceremony could leave sign-in stuck | Transport loss clears the pending login ID and returns incomplete login to signed out. Reconnect can start a fresh ceremony; a protocol test covers it. |
| Background collaboration commits can race a layout revision | Host document events use monotonic revisions. Only an explicit stale-revision rejection allows one retry of the same layout request ID after a read that preserves render leases. Provider submissions never use that retry path. |
| Sidebar button CSS made the default sign-in action unreadable | Dialogs now render through a body portal owned by AgentDialog, retaining native modality and restoring trigger focus on close. A real Electron regression failed at 1.04:1 before the fix and passes default/hover/focus contrast and keyboard-close checks afterward. |
| Growing UI responsibilities could mix local selections with sending | `SelectionChips` owns local evidence controls; `agents/` owns account, agent forms and public message presentation. Discussion composes these and keeps its unsaved draft lifetime across tab/collapse changes. Send carries only typed text. |

## Directory and design assessment

Contracts are closed portable schemas and semantic validators, with optional
v1 persisted extensions. Main owns privileged I/O and single-writer Project
state. Preload exposes named methods and validated events only. React owns
presentation and transient input/focus. The existing architecture test checks
imports, re-exports and type imports across these process layers.

Classes correspond to actual lifetime/state owners: transport child and pending
RPCs; provider account session; delivery admission/order; Project persistence.
Provider projections, policies and contract validation are functions/data.
There is no generic event bus, global command executor or container. The host
composition is the only place where concrete Codex adapters are assembled.
The coordinator has one reason to change: addressed delivery and its lifecycle.
Provider wire evolution is confined to generated types and the adapter.

Bounds are explicit: two active turns, one pending submission per agent,
32 transport requests, 8 MiB frames, request deadlines, 256 queued events per
delivery, bounded displayed text, and status-event coalescing. Overflow disconnects
the owned provider and exposes uncertainty. Completion/cancellation and merely
discarding stale UI events remain different operations.

The official version's SandboxMode is `read-only`, while its returned
SandboxPolicy is `readOnly`. An initial live smoke caught this distinction;
focused generated request types now prevent the misspelling from compiling.
They reproduce with `npm run codex:schema:check` from the pinned official binary.

## Verification classification and open items

`npm run check` passed with 93 Vitest tests and 15 actual Electron tests.
The provider peer and browser ceremony in those integration tests are explicit
substitutes. The official runtime separately passed unauthenticated initialization,
account/catalog reads and read-only thread creation. After user browser login,
separate live evidence verified two actual peer responses, two distinct thread
bindings, isolated interruption and normal restart without replay. The stage
record distinguishes those observations from substituted-provider tests.

The source is locally construction/behavior verified for the tested boundaries.
Live ChatGPT sign-in, two actual agent responses, interrupt isolation and
reconnection after normal quit/relaunch passed. The 40-exchange/4 MiB Project history limit is
visible and intentionally fails without evicting history; paginated storage is a
future explicit revision. Thread-scoped workspace tools, image/evidence delivery,
live Docker integration and packaging require their later stages. Stage 5 has
not begun, and no next-stage approval is inferred from passing these tests.
