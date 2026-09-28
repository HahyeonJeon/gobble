# Gobble App

Electron/TypeScript/React workspace with an app-owned native Go Project service.
Stages 1–3 provide typed contracts, an isolated desktop host, a native Project
service and a Project-centered shared workspace. Open a folder, browse real files,
compare materials in two panes, and keep selections and an unsent draft with each
Project. Stage 4 adds official ChatGPT sign-in, named peer agents, model selection,
addressed text messages, independent interruption and delivery recovery.
Stage 5.1 adds the right Project chat and vertically stacked Panes. Stage 5.2 adds
explicit shared-view access, scoped Agent tools and authored User/Agent pointers.
Stage 5.3 adds inline draft attachments and immutable addressed text/image evidence.
Stage 5.4 adds evidence-backed Agent questions and explicit replies in the same composer.
Stage 5.6 moves Agent controls into Chat, selection tools next to their source, and
saved evidence into a readable dialog that preserves the draft and reading area.
Actual account/agent verification is recorded separately from test substitutes.
Stage 6 verifies the real Docker runtime, independent controller lifetime,
restoration and a signed-in Agent's Run/image observations, pointers and replies.
See the [integration evidence and limits](../docs/desktop-workspace/stages/06-integration-evidence.md).
R1 adds portable versioned references. R3a adds exact Run/task/attempt/log navigation and
Agent observations. R3b1 retires CSV chart creation: CSV remains a table with exact selections
and attachments. Dependency facts, targets and renderer qualification are introduced separately;
R3b3 adds exact Agent group/pair references and reversible Show/Return to the existing dependency view. CSV chart creation remains removed.

P2A-1 makes a checked pipeline flow the default Pipeline surface: native import,
actual Gobble steps/ports/connections, local details, fit/zoom and a connected step
list. The native service retains declared source/configuration inputs and runs a
bounded pinned Linux inspection; the App does not evaluate Go or launch analysis.
Read the [design and qualification boundaries](../docs/desktop-workspace/stages/p2a1-pipeline-flow/README.md).
P2A-2 adds exact step, port, connection and declared-setting selections, immutable
Chat attachments and independent Agent marks on that checked flow. Gobble inspection
v2 supplies module-authored settings; v1 artifacts remain readable. Trim Galore
publishes Quality threshold and Minimum length, with unknown defaults shown as
Tool default. The existing reference/evidence owners enforce artifact identity,
observed-target authority and stale refusal. Read the [P2A-2 design and review](../docs/desktop-workspace/stages/p2a2-flow-discussion/README.md).
P2B-1 implements B / Change spotlight: a Proposed flow with numbered differences,
paired Current/Proposed detail, exact Chat context, scoped per-message Agent authoring,
Gobble checking and User adoption of retained source. First adoption creates a managed
copy; the imported folder stays separate. Canonical Trim settings and an added FastQC
branch are the qualified scope. Unknown changes cannot be adopted. New-Pipeline creation review is available through P2B-2.3; exact first adoption is available through P2B-2.4. P3 adds saved Run preparation and exact data/settings/environment discussion for App-created single-end Trim Galore → FastQC, including supported refinements. P4 adds exact data staging, User Start/Stop, independent engine admission and retained Run Flow monitoring. P5A links sent Run evidence to a supported visual refinement and a separately authorized fresh analysis. P5B-1 through P5B-4 add continuation history, exact engine admission, durable transport and shared Flow/Chat review. P5B-5 qualifies actual Trim/FastQC continuation, repeated attempts and restart replay using a new pinned local engine. Read the [P5B-5 implementation and verification](../docs/desktop-workspace/stages/p5b5-qualification/README.md).
Saved FastQC reports retain their original text, tables and charts for exact User/Agent discussion. Current-message chart access is recipient-scoped. Pipeline comparison navigation survives maximize/restore and compact Chat transitions within the active Project session.
Workspace storage is v21, contract bundle v30, service catalog v5 and shared toolset
v15. See [the current design and verification](../docs/desktop-workspace/stages/p3-run-preparation/README.md).

## Run and verify

Requires Go 1.26 or newer, a supported Node version from `package.json`, and npm.
From this directory:

```sh
npm ci
npm run dev
npm run check
```

For the native service, from the repository root:

```sh
go test -race -count=1 ./internal/appservice ./cmd/gobble-service ./cmd/gobble-container
go vet ./internal/appservice ./cmd/gobble-service
```

`check` verifies formatting, each process's TypeScript configuration, generated
schema consistency, unit/dependency tests and the built app in real Electron.
Unit tests launch the actual native service and validate its responses against
the shared TypeScript schemas. Electron tests use that service and real temporary
files. Native chooser results and the save-failure dialog choice are simulated.
The Run/log UI test substitutes a test-local Docker CLI while using the real Go
service and Electron UI; it does not execute Docker or an analysis.
Tests create and remove temporary profiles; no account or Docker daemon is
required. Docker adapter fixtures verify arguments and error handling, not live
engine compatibility. macOS close/activate evidence is target-specific.
`npm run build` produces development artifacts under `desktop/out`, not an
installable or signed `.app`. `npm run test:electron` builds and tests that output.

The separate live suite requires a running local Docker daemon and an already
built Linux/amd64 Gobble runtime image:

```sh
GOBBLE_LIVE_RUNTIME_IMAGE=gobble-runtime:local npm run test:live-runtime
```

It resolves that image to its exact local ID, creates a fresh small Project,
executes the real CLI, and compares Run/attempt/log observations through Electron.
It verifies independent completion after app quit, restoration, duplicate
attachment and daemon/Project mismatch refusal. Only its owned containers are
stopped/removed. Audit logs, Project outputs and the isolated profile remain in
`test-results/live-fixtures` for diagnosis and visual review. Missing prerequisites
fail the opt-in suite; it never substitutes a fake engine or silently skips.
Real signed-in Agent review is separately recorded and is not an automatic part
of either command.

## Ownership and dependencies

| Path                         | Responsibility                                                             | Allowed product dependencies                                    |
| ---------------------------- | -------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `contracts/src`              | Portable schemas, inferred types, structural/reference validation          | TypeBox, own files                                              |
| `contracts/schema/v30.json`  | Generated Draft 7 schema for language-neutral consumers                    | None; committed output                                          |
| `desktop/src/main`           | Application/window lifetime, origin and IPC authorization                  | Electron, Node, contracts, own files                            |
| `desktop/src/main/workspace` | Serialized Project state, render lifetime, evidence checks and persistence | Node, contracts, own files; Electron only for IPC/menu adapters |
| `desktop/src/main/service`   | Native process ownership, authenticated client and named handlers          | Electron, Node, contracts, own files                            |
| `desktop/src/preload`        | Named, validated capabilities only                                         | Electron, contracts, own files                                  |
| `desktop/src/renderer`       | React presentation and transient UI state                                  | React, contracts, own files                                     |

The tests enforce this import direction, including type imports and re-exports.
Each process compiles separately. Preload is a bundled CommonJS entry so the
sandbox does not require third-party modules at runtime. Contracts compile
without Node or DOM types. Go's engine and Linux CLI require none of this tooling.

The native entry point is `../cmd/gobble-service`; `../internal/appservice` owns
its catalog, contained file reads and runtime adapter. It imports only the Go
standard library. Files are separated by those responsibilities, with focused
behavioral tests; the service does not import the engine or provider code.

The public `window.gobble` bridge exposes `getAppInfo`, `projects.list`,
`projects.chooseFolder`, `files.list`, `files.read`, `runs.list`, `runs.attach`,
`runs.snapshot` and `runs.logs`. The `workspace` namespace adds `connect`,
`openProject`, `read`, `command`, `updateDraft`, `present`, `interaction`, `loadSurface`, `acknowledge`, `invalidate`,
a validated native-shortcut subscription and Project document updates. `command` accepts a closed action union with
Project ID, expected workspace revision and request ID. The native dialog supplies a registration path;
renderer queries use Project/resource IDs. No arbitrary filesystem path, raw
IPC, credential or command execution capability crosses the bridge. Requests
are checked against the current top-level workspace sender and schema.
Responses are checked again before returning to the renderer.
Static assets use `app://gobble/index.html`, strict CSP and a contained build-only
handler. Native requests, navigation, popups, downloads and webviews default to deny.

The development server uses only `127.0.0.1:5173`, with separate HMR CSP allowances.
Built output does not use those allowances. Use `--gobble-profile=/absolute/path`
when launching Electron directly to choose an isolated local app profile.

## Domain and module map

A Project is the registered work scope. Its App Workspace holds shared work
state; an engine execution workspace is a different filesystem concept. A
Resource is the subject, a Surface is an opened presentation, a Pane contains
its tabs, and the Electron Window hosts the active workspace. Pipeline definition,
Plan, engine Run, app Run registration and runtime task instance are distinct.
See the [domain model and metadata owners](../.gobbi/projects/gobble/memory/design/architecture/workspace-domain.md).

```text
contracts/src/
  project, agent, resource, file, run     domain and native-service data
  surface, workspace                    opened views and pane layout
  workspace-document                    current v9 work state + validation
  workspace-document-v1/v2/v3/v4/v5/v6/v7/v8, workspace-migration frozen storage readers
  selection-v2, reference-v2             frozen historical and v4 Agent tool inputs
  observed-evidence                     portable captured task/log facts and exact text coordinates
  tabular-view, view-links               table settings, legacy plot metadata and linked membership
  tabular-actions                       closed source-validated User intents
  tabular-projection                    table filtering and legacy numeric archive validation
  run-presentation, log-presentation     disposable bounded display models
  service, workspace-bridge, bridge      named transport contracts
  result, identity, context, decision    shared errors, IDs and evidence

desktop/src/main/
  service/client                        private process/HTTP lifetime
  service/project-service               typed routes, parsing and association
  service/run-presentation, log-presentation Monitor v2 facts to display models
  workspace/service                     registered resource resolver
  workspace/controller                  Project authorization + ordered commits
  workspace/model                       pure state transitions
  workspace/render-session              ephemeral loading/ready observations
  workspace/tabular-snapshots            one bounded load per visible link/revision
  workspace/tabular-commands             linked placement/settings/refresh transitions
  workspace/storage                     atomic profile state
  evidence/capture                      pure observed materialization + capture I/O port
  evidence/storage, references          immutable assets and explicit recovery-root traversal
  evidence/service                      recipient-bound preparation and delivery admission
  collaboration/coordinator             recipient admission and delivery recovery
  collaboration/provider, store         application-owned ports
  collaboration/host, ipc               composition and trusted host entry
  codex/transport, account, conversations provider runtime adapters
  codex/protocol, generated, policy      pinned wire types and bounded projection

desktop/src/renderer/workspace/
  WorkspaceApp, ResourceWorkspace, Pane composition and stacked placement
  ProjectBar, Sidebar, useExplorer       compact Project navigation and transient explorer visibility
  PresentedViews, Splitter               visible-region handshake and accessible geometry
  navigation/                           FilesBrowser and RunsBrowser
  views/                                typed content rendering and selection
  PaneActions, EmptyPane                view actions popover and actionable empty state
desktop/src/renderer/selections/
  SelectionToolbar                      current-source attach/share/clear with feedback
  LinkedSelectionToolbar                one linked-group Discuss action and column scope
  SelectionMenu                         locate local selections in hidden/inactive views
desktop/src/renderer/evidence/
  AttachmentList, EvidencePreview        attachment chips and saved-content preview dialog
desktop/src/renderer/agents/
  AccountPanel, AgentEditor, AgentMenu   account/settings and chat-side Project roster
  agent-status                          pure status presentation
  useAgents                             validated account/agent subscriptions
desktop/src/renderer/chat/
  ProjectChat, ChatComposer              one Project chat, shared recipient guard and persistent composer
  ChatTimeline, timeline                public read projection and scroll behavior
desktop/src/renderer/ui/
  ModalDialog                           shared portal, host interaction guard and focus lifecycle
```

`RunView` receives `RunPresentation` and emits `onOpenLogs(LogTarget)`;
its parent Pane chooses placement. React does not decode Monitor fields or
reconstruct pipeline state. `ProjectService` owns response association so both
IPC handlers and workspace resource readers use the same route/validation code.
`RunRegistration` in the Go catalog is an app association, not engine state.

Catalog and Resource contracts remain v1. Workspace documents are v13. A storage-only reader
migrates v1–v12 documents and preserves the exact original in the corresponding `.vN.backup`
before the first write. Published v1–v14 schema bundles remain unchanged; schema generation
targets v18. Reference versions 2 and 3 remain readable, with version 4 for dependency
group/pair selections, version 5 for PDF page/region captures and authored references,
and version 6 for Notebook source/output selections and authored marks. Frozen toolsets v6/v7/v8 retain their original input shapes; current v9 adds scoped Notebook observations and pointing. Only LocalSelection requires an open Surface binding. For a v1 log Resource,
`taskId` historically contains the runtime instance identity: new code uses
`logResource({runRef, instanceId, attempt})` and `logTarget(resource)` at that
compatibility boundary. Do not use an authored `task_id` as an instance.
The bundled workspace load IPC now carries a typed Run display model; main,
preload and renderer must be rebuilt together. Native snapshot HTTP remains
unchanged. A separately versioned remote consumer would need protocol negotiation.

Run/log attachment capture uses retained Main data and the exact visible render receipt.
The asset is published before the draft references it. Preparation and Send validate
the saved bytes and recipient without querying the current attempt. New stream
coordinates are half-open, 1-based lines / 0-based UTF-16 columns in a decoded
preview, never original-file offsets. Legacy combined-log text coordinates remain
unchanged. Captures include only selected log text plus observation identity and
stream bounds; unselected stream text is excluded. The native capability remains
current-attempt reads with up to 4096 bytes per stream, and completeness is unknown.

Run previews contain at most 1000 instances. Retained Run/log JSON payloads are bounded to
1 MiB each and 2 MiB per window session; hidden/replaced loads lose their receipt.
Only capture-owned blobs proven unreferenced by the primary, last backup and retained
migration documents may be reclaimed. Unreadable/unknown recovery roots preserve
the bytes. Reclamation follows reference ownership changes and backup advancement;
ordinary draft keystrokes do not repeatedly scan recovery roots. Existing evidence
quotas and file/table/image behavior remain in force. Historical scatter metadata remains readable.
R3a2 adds compact task search/state filtering and stderr/stdout navigation. Surface
`runView` / `logView` persist navigation; LocalSelection remains the single task/range
owner. `RunView` and `LogView` produce exact selectors, while `Pane` owns companion
placement and reuses an existing exact-attempt tab. `useSurfaceLoad` owns asynchronous
load/acknowledgment and explicit refresh; React owns unfinished search, pointer and scroll.
Filter/stream changes reuse Main's retained observation. Only initial loads and Refresh
read Gobble. A failed refresh keeps the last observation and failure notice across
stream changes; success clears invalid local coordinates while preserving captures.
New receipts bind these settings through the existing v3 presentation counter.
Old combined log views remain labelled and can explicitly switch to separate streams.
R3b1 registers `shared-views-v6`, removing Agent chart creation. The existing explicit renewal flow retires v5 bindings. Current Run/log observations return v3 evidence,
a turn-scoped `observedReadId`, and the exact renderer acknowledgment. Default scope
honors the displayed task filter or stream; explicit `source-preview` can inspect
another task/stream without changing User state. These source previews are not pointable. Point authorization requires a
contained returned target and the still-current receipt. `gobble_run` and
`gobble_logs` expose bounded, structured source observations without visual authority.
The frozen `shared-tools-v4` and `shared-tools-v5` modules preserve their published input shapes.

`ObservedReads` owns turn-local scope receipts. `ObservedReferenceViews` owns at most
one transient 1 MiB observation and its return destination; `RenderSession` binds the
effective presentation to readiness. React keeps the base Run/log presenter mounted
under **Show in view**, preserving its local selection and scroll; **Return to my view**
restores it and source focus. Pointer arrival never changes focus or presentation.
Keyboard log navigation scrolls to the caret. Agent marks are distinct from User selection.

Historical pointer targets never map to new bytes or attempts. **View captured evidence**
opens an existing sent/question capture with a containing explicit task/range target;
pointers create no extra asset store. **Find current task** explicitly reloads the Run
and searches its current bounded preview (the search uses up to 200 identity characters),
without changing the historical reference. An absent task in that preview is not a
whole-Run absence claim. Refresh and base selection edits are unavailable during the
temporary reference presentation. Return, closing its Surface, Project switch and
restart clear transient presentation; reopening a closed source requires a current read.
See [R3a3 implementation and evidence](../docs/desktop-workspace/stages/r3a3-agent-observation.md).

## Contract changes

Edit `contracts/src`, add relevant behavior/negative tests, then run
`npm run schema:generate`. Never hand-edit generated JSON. Use a definition such
as `#/definitions/WorkspaceSchema`; the schema bundle's untyped root rejects all
payloads. Unknown versions, properties and union variants are rejected without
coercion. Revision strings are equality markers, not ordering guarantees.

`parse` checks structural shape. `parseWorkspace` and `parseAddressedContext`
and `parseWorkspaceDocument` check the following cross-field invariants that Draft 7 cannot express:

- Records/evidence match their Project; IDs are unique within each record collection.
- Every open surface appears in exactly one pane and its active tab is a member.
- Opening/requesting agents exist, and view/selection kinds match resource kinds.
- Normalized image rectangles fit within the image; text ranges are nonempty and ordered.

Future consumers must implement applicable semantic checks in addition to JSON
validation. Identity prefixes prevent accidental wire mixups; they grant no
authority. The service/controller must verify authenticated Project/resource
association, current revisions and recipient attachment at use time. A render
acknowledgment is accepted only for a visible surface, current renderer session,
load generation and displayed data revision. Closed views may retain decision evidence.

The versioned service envelopes and inputs are defined in `contracts/src/service.ts`.
Go independently validates ownership, request fields and persistence invariants;
the process integration tests check actual Go responses against the TypeScript
contract. The Go service does not run Node or dynamically load JSON schemas. The App bundles its
TypeScript contracts and frozen historical storage schemas at build time.

## Project service behavior

The service starts lazily. A private stdin pipe carries a fresh token and profile
directory; stdout returns a versioned instance/port handshake without the token.
HTTP binds only `127.0.0.1` on an assigned port, requires authentication and the
exact Host, and rejects browser Origin headers. The renderer never receives the
port or token. Requests, output, concurrency and runtime queries are bounded.
Parent-pipe closure ends the service and its query helpers. A later explicit
request can restart a failed service; catalog mutations are not replayed by the
transport client.

The service alone writes `<profile>/service/catalog.json`, retaining
`catalog.backup.json` and an exclusive file lock. Matching request IDs reuse a
registration; differing payloads conflict. Unknown/corrupt catalogs or a missing
primary with an existing backup require explicit recovery. They are preserved,
not automatically reset. An ambiguous storage failure stops catalog use until
restart. No service state is written into a Project or Gobble checkpoint.

Canonical roots are deduplicated; overlapping roots are rejected. Physical folder
identity detects replacement. Explicitly selecting the same moved folder restores
its registration and resource IDs. Resource mappings survive service restart;
each read checks containment using `os.Root`. Symlinks are not listed as usable
resources, and special files cannot block ordinary preview/discovery reads.

Preview limits are explicit: 500 directory entries per listing; 8 MiB per file;
1 MiB UTF-8 text; CSV up to 100 columns and 500 rows; PNG/JPEG up to 20 megapixels.
Large listings/tables report truncation. Pagination and large dataset viewers are
later work. File revisions are content hashes, and stale requested revisions are
rejected. HTML is text, not executable report content.

Run discovery checks the Project root and immediate `runs/*` directories for a
`.gobble` directory. A candidate is not an inferred execution state. Attachment
requires a Project-local workspace and `.gobble-runtime.json` with the recorded
daemon/image. This adapter supports a local Unix Docker endpoint, the exact
Linux/amd64 image and the default `/gobble/project` mapping. Custom Compose
mapping, external workspaces, remote daemons and other platforms are unqualified.
The adapter never falls back to the current app's engine or pulls another image.

Queries run only the pinned image's `inspect identity/monitor` with bootstrap
disabled, a read-only Project mount, no network and no Docker socket. They check
engine identity, Run ID and the selected current attempt. Identical concurrent
reads share a query; different reads of one Run serialize. Cancellation ends one
reader's wait, while the shared query has its own bounded lifetime. Snapshot
equality markers and observation times are distinct; log tails are limited to
4096 bytes per engine stream. Cleanup targets only uniquely named query helpers.
Live Docker/engine behavior and detached-controller independence still require
stage 6 evidence; Docker was unavailable on this host during stage 2.

Provider credentials/transcripts belong to the later official Codex integration.
There are no Start, Stop, Resume or source-editing endpoints in this slice.

## Shared workspace

The sidebar belongs to the Project: files, attached Runs, agent references and
questions displayed in the chat timeline. Opening an existing resource activates its view; Duplicate
creates a separate view. New files open in the active pane, or explicitly in the
other pane. Each pane has tabs, pin, move, duplicate, refresh, maximize and close.
Switching Projects restores their own layouts and drafts.

CSV rows, text ranges and image regions can be selected as local context.
Selections retain source identity and revision. A changed file or log tail makes
an old selection visibly stale. Images support pointer selection and percentage
fields for keyboard input. Local selections are private until explicitly published with Share selection.
The right-side Project chat keeps drafts, addressed messages, compact activity and
public agent replies. There is one composer through collapse and compact view changes.
Send requires a connected ChatGPT account, configured recipient and available model.
Send submits typed text and explicitly added text/table/image/Run/log attachments to one recipient.
Shared selection pointers and Agent observation tools require separate shared-view access.
Use Reply on an Agent question to quote it in the same composer. Cancel reply keeps text and attachments; Dismiss records no answer.

Central Panes stack vertically. Up/Down resizes their heights; Left/Right resizes
chat width (360–560 pixels, 420 by default). Both separators support pointer input
and Home/End. Images fit within the available view; selection coordinates remain
normalized to the original image, without letterbox offsets. Percentage inputs
provide the keyboard alternative. Enter sends; Shift+Enter inserts a newline;
IME composition cannot submit. New content preserves the user's reading position
unless they are already following the end of the chat.

View tabs retain arrow/Home/End navigation. Native shortcuts are Cmd/Ctrl+O to
open a folder, Cmd/Ctrl+W to close the active view, Cmd/Ctrl+Shift+W to close the
window, Cmd/Ctrl+\ to split, and Cmd/Ctrl+Shift+P to toggle pin. Below 1320px navigation
compacts; below 1120px a Workspace/Chat switch retains the same composer. A resource
region under 520px high shows its active Pane and a switcher, preserving both records.

The main-process controller is the only workspace writer. React owns transient
loading, pointer/focus and draft display. The controller serializes commands,
rejects stale revisions and keeps 256 request receipts; repeats with a conflicting
action fail. Agent commands use the separately authorized Stage 5.2 application port;
the user-command bridge remains renderer-only.

Each Project document lives in `<profile>/workspace/projects/<projectId>.json`.
The window record is `<profile>/workspace/window.json`; each retains its last
valid `.backup`. Writes validate, flush and atomically replace private files.
Corrupt, unknown-version, missing-primary or externally changed files are
preserved with an error. The storage-only migration retains the v1 presentation mapping (equal vertical split, preserved collapsed state/draft/recipient, initial 420px chat) and maps every v1/v2 reference into v3 portable targets and local Surface bindings. Captured asset bytes/hashes remain unchanged. Reads do not write; later saves never replace `.v1.backup`, `.v2.backup` or `.v3.backup`. See [R1](../docs/desktop-workspace/stages/r1-durable-references.md) for contracts, ownership and recovery. Unknown versions are not guessed or reset.
Accepted changes survive normal close/relaunch. Normal close/quit waits for
received saves; an actual save failure offers Keep Open before losing a local
draft. Crash recovery uses the last committed record; no action/message is replayed.

Limits are 32 tabs per pane, 64 surfaces/context selections, 100 activity entries,
16000 draft characters and 4 MiB per document. Hidden views lose their render-ready
lease; the trusted renderer reports its mounted Pane set using its current session before loading, and the host intersects this with the active/maximized layout. Returning to a hidden view requires fresh load/paint acknowledgment. At most four loads run concurrently and only visible view data is retained.
Images use local, revoked blob URLs from validated PNG/JPEG bytes. Project files,
provider stores and engine checkpoints are not workspace persistence locations.

## Toolchain decisions

Exact versions are recorded in `package.json` and `package-lock.json`. Vite 7 and
React plugin 5 satisfy electron-vite 5's supported peer range; TypeScript 5.9 and
Vitest 4 run with the current Node 25 host. Vite's esbuild dependency is overridden
to 0.28.2 because its 0.27 range carries GHSA-g7r4-m6w7-qqqr. Build, development
startup and test evidence are required when changing this override. TypeBox 0.34
is the maintained LTS package; runtime validation interprets schemas without eval.

See [design and approval stages](../docs/desktop-workspace/README.md) and the
[design refinement record](../docs/desktop-workspace/stages/03-design-refinement.md).

## Codex account and agents (stage 4)

On the current macOS target, `npm ci` installs the pinned official
`@openai/codex@0.153.4` and its native runtime dependency. The app uses that local
package by default. Development can explicitly set `GOBBLE_CODEX_EXECUTABLE` to
an absolute compatible executable; its reported version is verified before
startup. It never searches another desktop app's installation. Packaged runtime
resources, native signing and notices remain stage 7.

Open **Account → Sign in with ChatGPT**, then complete the official browser flow.
The provider owns credentials and private history under `<profile>/codex/home`;
Gobble never parses/copies them. `<profile>/codex/session.json` contains only a
local sign-in-session reference. API keys and implicit API billing are unsupported.
Models and reasoning options come from the signed-in runtime's catalog.

Add named agents under a Project, select a recipient, and send typed text.
Each attachment has its own thread and pending submission; two turns can run
concurrently across the app. **Stop** targets the selected thread/turn. **Check
status** reconciles an uncertain record through official provider history and its
client submission ID. No message is automatically resent after timeout/restart.
**Start new conversation** retains shared history; an unresolved earlier delivery
is explicitly marked unconfirmed. It does not cancel or fabricate a provider
outcome. Active turns must be stopped first.

The `collaboration` bridge exposes only `status`, `account`, `configure`, `send`,
`agent` and `onChanged`. It does not expose raw RPC, tokens or tools. Its app-owned
provider port is independent of Codex wire types. Focused request types are
reproducibly generated from the pinned binary: `npm run codex:schema` updates
`codex/generated.ts`; `npm run codex:schema:check` verifies it. Unused provider
fields and internal reasoning are not copied into app contracts.

Discussion runs with no environment access, read-only sandbox and approval
policy `never`; local shell, browser/computer use, hooks, plugins/apps, delegation,
web search and image generation are disabled. Shared-view tools require explicit per-Agent
access, described below. Source writes and analysis execution remain later work. Explicit addressed evidence delivery is available in Stage 5.3.

The optional Project document extension persists at most 40 addressed exchanges,
16000 input characters and 32000 displayed response characters per exchange;
the existing 4 MiB encoded-document limit also applies. Admission refuses input that already exceeds a known limit.
Output-size or disk failures remain explicit and preserve prior durable records. Full provider history is provider-owned. Future paginated
Project history must use an explicit storage revision, not silent truncation.
Two turns, 32 pending RPCs, 8 MiB frames, bounded deadlines, 256 queued events per
active delivery and at most 25 renderer status updates/second bound host work.
Saved terminal results win over a following disconnect event.

Automated account/agent tests use `desktop/tests/fixtures/codex.cjs`, a test-local
protocol peer, with a simulated browser ceremony. They do not authenticate to
OpenAI or establish live model behavior. The pinned official binary separately
passed initialize, signed-out account, catalog and read-only thread initialization
in a fresh isolated profile. After user browser sign-in, two live peer responses,
isolated interruption and normal restart without replay were separately verified
in the [stage 4 record](../docs/desktop-workspace/stages/04-agents.md).

Account and agent settings dialogs are transient renderer UI outside Project
Surfaces/Panes. Their shared Dialog component owns body-portal placement, native
modality and trigger-focus restoration, preventing sidebar CSS from overriding
primary actions. Electron tests measure text contrast before hover, on hover and
with keyboard focus and verify Escape returns focus to the trigger.

## Shared views and communication tools (Stage 5.2)

Existing agents remain **Messages only**. In Agent settings, **Enable shared views and
start new conversation** opts into this Project's tools while retaining earlier public
messages. New agents have the same access choice. **Disable shared views and stop** revokes
calls immediately and requests interruption. Restart never replays a tool or message.

Select table rows and included columns, a text range, or a normalized image region, then
choose **Share selection**. This publishes a labelled Project pointer without sending a
message or transmitting source bytes. An enabled Agent may query published pointers and
observe the ready shared preview. Agent marks and User local selections are separate.
**Reveal** returns to the referenced resource; old source revisions show a warning instead
of a misplaced highlight. **Retract mark** preserves history. A static Plot region denotes
pixels, not semantic points or axis ranges. Explicit immutable attachments are described below.

The main-process `shared-context` owner contains the bounded tool catalog, protective layout
policy, preview materialization and invocation cache. `contracts/shared-tools` holds closed
provider-neutral inputs; `contracts/shared-context` defines access and authored pointers.
`collaboration/coordinator` binds and revokes active caller authority. `codex/conversations`
adapts only the pinned dynamic-tool envelope. `workspace/controller` remains the sole durable
Project writer; `render-session` tracks actual visible acknowledgment identity. Renderer
`shared-context` components draw marks and references without access to provider authority.

Only a visible, focused, non-minimized Project window without a dialog can supply observations.
Content comes from loaded preview data. Text is limited to 64 KiB; images to 1536 pixels on
the long edge and 1 MiB encoded data. Each active submission has 32 tool IDs and an 8 MiB
result cache; the transport admits four simultaneous callbacks with a 15-second deadline.
Observation retries revalidate the original lease. Source bytes are not persisted as shared
references. Explicit addressed evidence uses the separate storage owner described below.

The registered tools are `workspace_list`, `workspace_resources`, `workspace_open`,
`workspace_arrange`, `workspace_observe`, `workspace_point`, `workspace_release`,
`gobble_runs`, `gobble_run` and `gobble_logs`. Project/Agent/turn identity comes from the
bound conversation, never model arguments. User-owned, pinned and claimed views are
protected. Two opens are atomic; loading/hidden state is explicit. The tools expose no
shell, source writes, new analyses, other Agent messages or private drafts.

Pinned Codex 0.153.4 requires its isolated code-mode host to dispatch dynamic tools.
That runner receives dynamic results as strings; image observations include a receipt and
data URL, which the adapter instruction explicitly forwards with the image helper.
OS execution tools remain disabled. Actual text/image consumption and the distinction
from Electron protocol fixtures are recorded in [Stage 5.2](../docs/desktop-workspace/stages/05-2-shared-pointing.md).

## Addressed attachments (Stage 5.3)

Choose **Add to message** on a local text/table/image selection, or **Add view preview**
inside a ready view without a selection. The draft copies that exact resource version and range.
Later local selection changes do not change the attachment. Closing a view preserves its draft
attachment. Select a recipient and expand an attachment inside the single composer to inspect
its actual bounded content. Only one preview expands at a time. Remove uses its own labelled
button. A message can contain attachments without typed text.

Preparation binds the attachment intent revision, recipient configuration, account session and
workspace session. Changing recipient/model or reopening a Project invalidates earlier tokens.
Typing changes the message body independently. **Send** revalidates the sources and binding,
stores immutable assets, and atomically records the addressed Submission plus its manifests and
matching draft consumption before provider input. A stale source, missing image capability,
quota or save failure preserves the draft. There is no fallback to newer source bytes, dropped
attachment, another Agent or an automatic resend. Messages-only Agents may receive these
explicit attachments without acquiring shared tools.

Each sent user message expands its saved content inline, including after the original file
changes or its Surface closes. Missing/corrupt assets display **Evidence unavailable**; source
files never reconstruct historical evidence. Resource versions and image crop/scale details
remain visible. Static Plot rectangles identify pixels, not semantic graph points.

`contracts/evidence` and `evidence-bridge` own portable values and named preparation/preview
calls. `main/evidence/draft` owns pure intent/consumption rules; `materialize` owns exact bounded
content; `storage` owns immutable hashed bytes; `service` owns temporary recipient-bound
preparation. `WorkspaceController` is still the only Project/manifest writer. The collaboration
coordinator accepts the prepared content through an application port; CodexConversations maps
only text/image values to pinned official input types. Renderer `evidence` owns chips, inline
content, stale-request disposal and revoked blob URLs. The Go service and engine are unchanged.

At most 16 attachments and two images are accepted per message, with 64 KiB combined serialized
text content, 1 MiB encoded bytes per image, a 1536-pixel long edge and 4 MiB encoded delivery
payload. Oversized explicit selections are refused rather than truncated. Whole previews
identify any source truncation. Preparation admits two concurrent operations, retains up to
four Project payloads and expires after ten minutes; tokens are never persisted.

Evidence assets live under `<profile>/workspace/evidence/<projectId>/<sha256>.blob`, separately
from the 4 MiB Project document. Each Project has a 64 MiB physical quota including unreferenced
assets left by interrupted acceptance. Exclusive publication never overwrites an existing hash;
reads check size/hash and reject symlinks. Historical assets are not automatically deleted.
Draft attachments and sent manifests are compatible optional v2 fields; the v1 reader accepts
only its legacy message shape and the archived v1 JSON schema is unchanged.

See [Stage 5.3 design and qualification](../docs/desktop-workspace/stages/05-3-addressed-evidence.md).

## Questions and replies (Stage 5.4)

`workspace_question` creates a durable Question variant in the existing Project
Decision collection. It requires one to sixteen observation/received-attachment
IDs belonging to the requesting Agent, with up to six plain-text suggestions.
It returns immediately after durable creation; no provider callback waits for a
human answer. Observed IDs last only for the active turn; sent attachment IDs
identify that Agent's received evidence in the same conversation. New questions
cannot read another Agent's evidence or any draft.

QuestionService (`main/questions/service.ts`) owns bounded observation authority,
question creation and dependency validation. Its pure rules in `model.ts` own
Reply, Cancel, Dismiss and answer acceptance. The ports expose only question
records and received-evidence metadata, not message bodies or provider DTOs.
WorkspaceController remains the sole Project writer; EvidenceStorage retains
immutable text/image bytes. QuestionMessage and ReplyTarget are presentation
components without an input or submit flow. ChatComposer remains the one draft
owner. Question evidence previews reuse the named evidence bridge and validate
Project/question/attachment membership before reading saved assets.

Reply explicitly binds the recipient to the requesting Agent and preserves typed
text and attachments. Cancel reply removes only that association. An arriving
question never changes the draft. The single Send action revalidates pending
state and the original resource revisions, commits the answer's Submission link
and matching draft consumption atomically, then delivers. A plain message is not
inferred to answer a question. Failed admission preserves the draft; uncertain
provider delivery retains the recorded answer and its separate delivery state.
The quote and saved question evidence accompany the reply, including when the
user explicitly starts a new conversation for that same Agent.

A Project supports up to 100 new questions under its existing JSON limit and
64 MiB evidence quota. Each turn may retain 16 observation assets within 2 MiB;
normal tool deadlines/cancellation revoke them. Questions and combined reply
attachments retain the 16-item, two-image, 64 KiB source-text and 4 MiB encoded
payload limits. Source freshness is checked at creation and explicit Send; no
background filesystem watch is claimed. Changed dependencies invalidate pending
questions, with original previews preserved. Missing/corrupt saved assets remain
unavailable. Normal restart restores questions and reply drafts without replay.

The tool set is now `shared-views-v2`. Existing v1 shared bindings are preserved
and visibly require the user's Start new conversation action to enable questions.
Messages-only Agents keep their existing behavior. The v1 storage reader remains
frozen; older foundation Decisions stay readable without fabricated snapshots or
reply actions. [Stage 5.4](../docs/desktop-workspace/stages/05-4-inline-questions.md)
records the design, verification and live-model evidence.

## Integrated Project chat (Stage 5.5)

New user messages and Agent responses have separate timeline positions while
referencing the same Submission. Main saves `responseStartedAt` with the first
nonempty response before publishing its stream. Later chunks and reconciliation
keep that position. A response remains one aggregate for a turn; sentences and
tool activity are not independently reordered. This timestamp is first app
receipt, not provider generation time. Existing records without it retain their
paired display; no missing historical timestamp is inferred.

`chat/timeline.ts` owns the read projection, `SubmissionMessage` owns message and
delivery presentation, and `useChatScroll` owns transient reading state. The
scroll owner follows the end or preserves the first visible item's offset,
including content growth, preview loading, width changes and hidden/show regions.
New messages provides an explicit jump to latest. In reply to returns keyboard
focus to the original user message. These controls never change recipient or draft.

Project errors appear above both workspace and chat, including compact Chat mode.
Recipient and model changes hold Send until their save resolves. Toolset renewal
uses one predicate across Main, adapter and UI: disabling shared access revokes
calls but cannot rewrite an old provider thread's registered tools. Explicit
Start new conversation preserves earlier Project history.

The [5.5 review record](../docs/desktop-workspace/stages/05-5-ui-integration.md)
contains the ownership sketch, automated Electron journeys and live peer/image
qualification. Actual Gobble/Docker qualification and local packaging are later
review checkpoints.

## Source actions and saved-content inspection

Local selections belong to their Surface and source revision. The loaded SurfaceView
shows SelectionToolbar only for that current version. Add to message creates a draft
attachment; Share selection publishes an authored Project pointer. Neither sends a
message. The Project bar's Selections overview finds inactive or hidden selections;
Show in pane activates that Surface and returns keyboard focus to its tab.

AttachmentList is shared by draft, sent and question evidence. A chip opens an
EvidencePreview inside the common guarded ModalDialog. The preview reads only the
prepared/sent/question snapshot through the evidence capability; it never registers a
Surface, rereads a live source or becomes another composer. Close/Escape restores
trigger focus. Missing saved content has an explicit Retry preview action. Existing
stale-source/model/recipient/reply checks still gate Send in the host and composer.

PaneActions owns a temporary native popover for pin, move, duplicate and refresh;
expand and close stay direct. Overlay lifetime blocks shared-view observation, and
outside click, leaving focus, resize and Escape dismiss it. EmptyPane emits browse
and move intents. WorkspaceApp/useExplorer chooses navigation visibility and the
host retains Pane/Surface identity and storage authority. Explorer callbacks emit
active/other destination intent; useWorkspace resolves it after earlier queued Pane
changes, so rapid browsing cannot capture a stale rendered target. Sidebar does not
receive the full WorkspaceDocument.

Timeline grouping is a pure projection of consecutive recognized routine view
activity. Unknown events, submissions/responses, questions and marks interrupt it.
The first event remains the stable scroll anchor; disclosure retains every original
message. Shared marks keep author, source, selection and Reveal visible, with full
notes and secondary retraction in Mark options. No engine or public contract changed.

## Integrated UI review (Stage 5.6c)

The Project title follows its allocated header width so region controls remain
reachable with long names and enlarged text. ChatComposer presents one current
blocking explanation beside the draft, outside scrolling question/attachment
context. Question image compatibility, toolset renewal and evidence preparation
remain owned by their existing contracts and host; presentation adds no saved state.

Electron UX scenarios cover 0–12 Agents, long titles/files/drafts, four attachments,
pending replies and 100–150% native zoom. Complete-window review captures use
BrowserWindow.capturePage because page screenshots crop the physical native surface
at enlarged zoom. [Stage 5.6c](../docs/desktop-workspace/stages/05-6c-integrated-review.md)
records classified findings, native review and the exact verification subject.

## CSV tables and legacy chart compatibility

CSV files open as tables (500 rows / 100 columns at most). Row keys remain scoped to the
exact file revision; sorting and included columns do not change identity. Selection and
**Add to message** use the existing draft and immutable evidence capture. No chart builder,
plot settings, chart renderer or Plotly runtime dependency remains.

`workspace_open` accepts registered resources only. `shared-views-v6` replaces the chart-open
Agent arguments; older provider bindings use the existing explicit renewal action. The host
rejects retired `openScatter`, `openLinkedTable`, `scatterSettings` and `scatterViewport`
requests before I/O or mutation, including old request replays. Their IPC shapes remain only
for a clear compatibility refusal. They are not advertised as current capabilities.

Workspace v8 and historical plot evidence remain readable without rewriting stored bytes.
A saved chart tab displays **Chart view retired** and **Open source table**; Main refuses a
render acknowledgment for that retired view, so an Agent cannot claim it observed a chart.
Historical chart pointers reveal their exact source rows in a table after version validation.
The original immutable metadata stays in the archived capture. Pure legacy projection and
validation helpers remain only where required to interpret those archives.

Existing `ViewLink` records still own their saved row membership once. They are not created
by new chart flows. Legacy linked tables retain filter/selection behavior and capture scope.
Normal new tables require no linked chart or extra application state.

## R3b1 dependency foundation

- `contracts/run-dependencies.ts`: closed read, observation, target and capture schemas with
  explicit preview/completeness facts. Dependency target v4 is allocated but is not yet accepted
  by the live EvidenceRef/Workspace/Agent unions.
- `contracts/dependency-projection.ts`: pure grouping and directed topology; bounded groups,
  edges, diagnostic fallback and iterative cycle detection. No aggregate execution state,
  inferred sample edges, layout coordinates or scheduler behavior.
- `contracts/dependency-resolution.ts`: exact returned IDs and direction, Project/Run identity,
  source revision and historical/unavailable results.
- `main/service/dependency-presentation.ts`: consumes one coherent monitor response; distinguishes
  missing, invalid and reported-empty topology before the old presenter normalizes it.
- `main/evidence/dependency-capture.ts`: independent versioned revision and frozen group/edge
  context. Member records and bytes are bounded without truncating identities; no endpoint logs
  or private runtime paths are implicitly attached. R3b2 connects it to the existing store.
- `qualification/run-dependencies`: isolated renderer comparison under the production asset
  handler/CSP. Minimal HTML/SVG is selected for R3b2; no graph dependency is imported by or installed into the product.

Current bounds: 1,000 inspected edge entries, 80 returned groups, 200 returned edges and the
existing 1,000-task preview. Larger or malformed topology uses a bounded list; omitted scopes
are explicit. Captures return at most 100 member records per selected group/endpoint and at most
64 KiB. These are local capability bounds, not a claim of full-run or large-dataset support.

[Stage record](../docs/desktop-workspace/stages/r3b1-foundation.md) tracks the exact approved scope,
source identity, tests, qualification decision and limits. R3b2 adds User navigation and R3b3
adds Agent graph references only after separate owner approval.

## R3b2 User dependency navigation

The existing Run Surface switches between Tasks and Dependencies. `RunView` owns mode and the
shared freshness/readiness notice. `RunTasksView` owns task filtering. `DependencyView` coordinates
bounded graph/list navigation and observed member detail; `DependencyGraph` alone owns SVG/HTML
geometry and native scrolling. `DependencyEvidence` renders captured facts without a live read.

`runMode`, `dependencyNavigation({query, representation})` and `dependencyCamera({zoom, x, y})`
use the existing serialized Workspace command path. Camera writes cannot overwrite a search,
advance source identity or revoke a semantic render receipt. Mode changes retain one selection,
the Tasks filter and the dependency camera. Explicit Show and Fit move the viewport; refresh
never fits automatically. A task hidden by its filter has a Show selected task action.

A group is an authored Task ID plus returned instance membership, with separate observed counts.
A dependency is a directed authored Task pair, without inferred sample routing or causal meaning.
Only an exact instance/attempt can open logs; templates and attempt zero cannot. Discuss publishes
bounded frozen bytes to the existing evidence store and focuses the same composer. Send remains
explicit. Later source changes never replace saved attachment facts.

Main derives both presentations from one monitor read. Missing/cyclic/over-limit topology has a
labelled list alternative. An optional dependency preview that exceeds the existing 1 MiB view
bound is omitted with an explanation so admissible Tasks remain usable. The 2 MiB retained-session
bound is unchanged. Agent v6 view observation/pointing cannot claim a dependency display;
explicit source-preview reads still use the old task facts. New Agent graph tools are R3b3.

See the [stage design, ownership and evidence](../docs/desktop-workspace/stages/r3b2-user-navigation.md).

## R3b3 Agent dependency references

`workspace_observe` accepts Run `representation: tasks | dependencies`; default uses the displayed
mode. A dependency read returns bounded authored groups, member context and directed pair targets,
with one turn-local read ID and the current render receipt. There is no fabricated whole-graph target.
`workspace_point` requires that exact returned target and receipt in the same turn. A hidden endpoint
returned as pair context does not authorize a group pointer. Source previews can read another Run
representation without changing User settings but cannot authorize visual pointers, including tasks
and logs. Current source identity and display scope are rechecked at the serialized publish boundary.

`dependency-observation.ts` owns bounded semantic projection; `ObservedReads` owns its scope receipts.
`ObservedReferenceViews` remains the sole temporary presentation owner. `current-dependency.ts`
provides explicit current-source search without rebinding historical targets. No new source store,
transport, graph library, polling loop or agent-controlled mode navigation is introduced.

Agent marks label their author and use dashed amber outlines/lines independently of the User's
selection. Only User Show opens a read-only dependency presenter. Its camera is temporary; the base
view stays mounted. Return restores Tasks/Dependencies mode, filters, camera, selection and draft.
Changed-source Show refuses substitution; existing sent/question captures can be opened by exact
v4 target. Find current group/dependency opens a fresh list search and never rewrites the pointer.

The R3b3 baseline introduced Workspace v10, schema bundle v12 and `shared-views-v7`. Existing Agents require the
explicit New conversation renewal flow; old messages and evidence remain. See the
[stage design and verification](../docs/desktop-workspace/stages/r3b3-agent-references.md).

## R3c2: User PDF reading and captured discussion

A PDF is a file Resource displayed by a PDF Surface in an existing Pane. `PdfViews`
under `main/workspace` owns at most two visible decoder hosts. Navigation is durable
Surface state; document handles, worker, page model and raster are transient.
`ServiceResources.readSurface` is the presentation-aware read; ordinary source reads
cannot impersonate a displayed PDF. The Go service transports contained, revision-bound
PDF bytes. Its raw PDF result never crosses the Workspace renderer's file IPC.

`main/pdf/host.ts` validates the fixed sandbox's sender, main frame, job, source hash,
page model and exact raster crop. `desktop/src/pdf` contains only the decoder, fixed
worker wrapper and narrow preload; it has no Workspace, account, path or shell bridge.
PDF.js 6.3.289, worker, CMaps, standard fonts and licenses are bundled under `out/pdf`
by `pdf-build.ts`. The default shell CSP is unchanged. Invalid or partially decoded
pages fail locally; PDF.js warnings do not become silently incomplete successful pages.

`renderer/workspace/views/pdf/PdfView.tsx` owns page/zoom/rotation controls, pointer
and keyboard region editing, image readiness and explicit Refresh. It emits existing
Workspace commands; source access and evidence publication remain in Main.
`contracts/pdf.ts` defines source-coordinate page/region identity and exact PNG geometry;
`main/evidence/pdf-evidence.ts` freezes the held rendition into existing image evidence.
The common composer, preparation, Send and saved history own the rest of the flow.

User selections bind source revision, decoder profile, page index and model hash.
Captures retain the transform and exact pixel crop without resampling. Whole-page or
region captures use the existing limit of two images per message, 1536 pixels per edge
and 768 KiB PNG per image. Oversized captures ask for a smaller region or lower zoom.
Inputs are capped at 8 MiB/200 pages, 4 MP output, 20 MP embedded images and five seconds
per decoder job. These are admission/work limits, not a hard process-memory cap.

Read-only page/region capture is supported; exact text targeting, passwords, OCR,
form/annotation appearance and PDF actions are unsupported. Refresh explicitly reads
a newer source. If the saved page no longer exists, close and reopen at page 1.
Captured draft and sent evidence remains usable after source replacement/deletion or
restart. Hidden/closed Surfaces and Project/window transitions dispose decoder hosts.

R3c2 introduced Workspace v11 and bundle v13. v9/v10 storage schemas are frozen JSON
with restored TypeBox tags for runtime validation, preventing new PDF shapes leaking
into old formats. Original version backups and historical bundles are retained.
R3c2 used `shared-views-v7` for explicitly sent PDF captures; R3c3 below extends shared observation. See [R3c2 design and verification](../docs/desktop-workspace/stages/r3c2-user-pdf.md).

## R3c3: Agent PDF observations and references

R3c3 introduced **Workspace v12** and bundle **v14**. Shared Agents use
**shared-views-v8**. Frozen Workspace v11 and toolset v7 schemas preserve old validation;
storage migration saves the original v11 bytes. Earlier Agent conversations use the existing
explicit renewal flow before accepting new tools.

`usePdfViewport` reports the fully visible raster rectangle through a named, ephemeral bridge.
`RenderSession` binds its generation to the painted page receipt. `pdfObservation` maps the
permitted rectangle into a source/page/model-bound target; `ObservedReads` holds turn-local
addresses and guards. `capturePdfRaster` crops the validated PNG without resampling and returns
both pixels and their PDF transform through existing image delivery. Question evidence reuses
that adapter; `ServiceResources.checkPdfSource` verifies the current exact source bytes without
creating another decoder. The engine's execution and Run ownership stay unchanged.

Default observation scope is the clipped viewport. A smaller region is permitted inside it;
changes of the clipped rectangle (scroll/resize), page/zoom/rotation, hide, replacement and presentation invalidation revoke prior pointing
authority. A source-preview explicitly reads only the current decoded page and cannot authorize
pointing. The image must be within existing delivery bounds. No off-page background reads or
semantic PDF text targeting are exposed. A changed file does not change the retained displayed
page until Refresh; references keep that precise version rather than adopting new bytes.

Authored PDF marks appear in the existing reader and Chat. They do not send messages, change
User selections or navigate. `ObservedReferenceViews` owns User-triggered Show/Return;
`PdfViews.reference` uses at most one short-lived additional decoder and disposes it after an
exact-source page read. The two interactive decoders and base React reader stay retained.
`PdfSurface` keeps the base reader mounted while a read-only `PdfReferenceContent` is visible,
so Return restores its page, zoom, rotation, scroll and local selection. Source replacement or
deletion fails Show closed; `capturedReference` can open an already stored matching PNG.

See [R3c3 design, ownership and verification](../docs/desktop-workspace/stages/r3c3-agent-pdf.md).

## R4a2: saved Notebook reading and User evidence

R4a2 introduced **Workspace v13**, bundle **v15**, and retained shared tool
input **shared-views-v8**. Frozen v12 storage and the v8 target/selection
unions prevent Notebook shapes from leaking into older formats or Agent tools.
No production module imports the retained R4a1 qualification code.

A registered `.ipynb` is a file Resource rendered in an existing Pane. Cells and
saved outputs are addresses inside that View. `main/notebook/` owns the qualified
worker parser, decoded source and exact text/image capture; `workspace/notebook-views`
owns the lifetime of at most two visible readers. RenderSession and
WorkspaceController authorize the current Surface and revision. A focused
`workspace.notebookImage` bridge supplies only an on-demand decoded image. The
public file bridge refuses raw Notebook bytes; the internal registered-file service
uses a named Notebook variant capped at 8 MiB, retaining generic text's 1 MiB limit.

The private worker boundary and target validators use the existing pinned TypeBox
version, explicitly declared by desktop. This dependency is allowed only in those
two Main files by the import-boundary check. Public contracts remain in contracts;
raw Notebook fields do not become public View fields.

`renderer/workspace/views/notebook/` contains the reader, cell presenter, bounded
text and image gestures. It renders 20 cells per page and at most one requested
image per reader. Notebook pages are persisted in the existing Surface. Text
windows preserve absolute UTF-16 offsets; image selection uses original integer
pixels at every display zoom. React emits intent and never authors capture bytes.

`evidence/notebook-evidence` adapts Main captures to the existing immutable asset
store. Text retains displayed text plus the original quote, including line endings.
Images retain original dimensions and an exact crop. A named draft-preview request
is authorized through the active Project and draft attachment, then rechecked after
storage I/O. It requires no account, source reread or recipient. Explicit Send still
uses existing preparation, recipient and image limits. Renderer loss revokes reading
leases before a native Reload workspace choice; saved drafts and captures restore.

Supported profile: saved nbformat 4.0–4.5, code/raw/basic Markdown source, plain
text/error output and PNG/JPEG. Inputs are bounded to 8 MiB, 1,000 cells, 100 outputs
per cell / 1,000 total, 1 MiB decoded text and 4 MP per image. Text evidence including
its metadata is capped at 64 KiB; PNG evidence at 768 KiB and 1536 pixels per edge,
with no resampling. HTML, SVG, widgets and active/remote content are unavailable.
Execution is not verified. R4a3 below adds Agent observations and authored references.
Jupyter integration, editing and execution remain later review checkpoints. CSV chart creation stays removed.

See [R4a2 ownership, verification and next-stage sketch](../docs/desktop-workspace/stages/r4a2-user-notebook.md).

## R4a3: shared Notebook observations and authored references

At R4a3, storage was **Workspace v14**, generated bundle **v16**, and Agent toolset
**shared-views-v9**. Frozen Workspace v13 and toolset v8 remain unchanged readers;
existing conversations use explicit toolset renewal before receiving Notebook inputs.
The Notebook target remains schema version 6, shared by User and Agent communication.

`notebook-viewport` defines bounded visible selectors and exact part containment.
`useNotebookViewport` measures fully visible UTF-16 units through nested clipping and
painted image rectangles in original pixels. It reports at most 64 parts / 4096 text
units through `workspace.notebookViewport`. Main validates document and raw-escape
boundaries, then RenderSession binds the report to a ready generation. Reports and
receipts are transient; scrolling, folding, changing presentation or ending the turn
revokes old pointing authority.

`shared-context/notebook-observation` returns Main-owned public text and addresses.
Default observations advertise visible images without granting image authority.
Explicit image observation delivers an exact PNG crop to an image-capable model.
Explicit single-part observation can preserve question evidence through the same
`materializeNotebook` adapter and EvidenceStorage used for User attachments.
Source-preview requires an explicit part selector and never permits pointing.
Basic Markdown is read through Source for exact addressing; there is no execution.

SharedContextHost and QuestionService own tool routing and question rules. They use
`NotebookCapture` callbacks and existing turn receipts; no separate conversation,
capture store, kernel or generic plugin framework is introduced. User Share mark and
Agent workspace_point publish the same authored reference shape. Neither changes
User selection, navigation, image gestures or the composer draft.

Show reads exact registered bytes in one temporary NotebookHost, freezes a bounded
text/image excerpt, disposes the host, and presents it in ObservedReferencePanel.
NotebookSurface keeps the User reader mounted through Show/Return. The temporary
excerpt grants no Agent observation authority: Return resumes normal shared reading.
Changed/missing source fails closed and only matching existing captured evidence can
supply historical content. Gobble still owns Run/Pipeline execution.

See [R4a3 ownership, test evidence and next-stage sketch](../docs/desktop-workspace/stages/r4a3-agent-notebook.md).

## P1: Pipeline registration and presentation ownership

Current contracts are **bundle v17 / service catalog v2 / Workspace v14 / shared
Agent toolset v9**. Historical bundles and workspace/evidence readers remain unchanged.

A Pipeline definition has its own `pip_` identity, Project, display name, Go package
resource and entry source resource. Register an existing source folder using Files
→ Register pipeline, then use the collapsible Pipelines list to open its source in
an existing text Pane. Registration parses a bounded set of Go source declarations;
it never compiles, executes or edits source and does not establish Plan validity.
Selection, references and Chat use the existing file revision contract. An entry
file binding is not an immutable source revision or an execution authorization.

`contracts/src/pipeline.ts` owns current registration DTOs; `internal/appservice/pipelines.go`
owns recognition and idempotent catalog registration. Named Main/preload operations
validate input and response association. No new Agent tool or pipeline effect endpoint
is enabled in this stage. Existing Runs are not associated by matching labels.

The service accepts v1 catalogs through strict decoding and migrates in memory.
The first mutation archives exact original bytes as `catalog.v1.backup.json` before
publishing v2. Later saves retain that archive and preserve Project/Run receipts.
Conflicting archives, corruption and unknown versions preserve data and fail visibly;
older service binaries cannot open v2 automatically.

`main/workspace/presentation-state.ts` owns reconciliation and cleanup of transient
render/reference/table state. The specialized helpers keep their invariants and
WorkspaceController retains authorization, epochs and the one durable write queue.
No pipeline lifecycle state is added to the workspace document.

See [stage design and evidence](../docs/desktop-workspace/stages/p1-foundation/README.md)
for the exact implementation boundary, tests and the next approval checkpoint.

P2B-2.1 adds native creation-draft storage and per-Pipeline Current ownership. Drafts
are separate from registered Pipelines; their UI, Agent source creation and first
adoption are later parts. Catalog1/2/3 migration preserves original bytes before the
first successful write. See [storage stage](../docs/desktop-workspace/stages/p2b2-1-creation-storage/README.md).

P2B-2.2 adds the native creation-check foundation: Gobble-owned bounded scaffold,
fixed input descriptor preserving declared compression, profile-pinned runtime,
sealed source candidates and complete single-end Trim Galore → FastQC qualification.
The new native routes have no renderer IPC or Agent-tool exposure yet; Part 3 will
connect the accepted UI. First adoption is implemented in Part 4 below. See the
[implementation and evidence](../docs/desktop-workspace/stages/p2b2-2-creation-check/README.md).

P2B-2.3 connects **New pipeline → chosen single-end data → existing Chat → checked
Proposed flow**. Creation drafts are first-class Pane subjects, separate from registered
Pipelines. User setting references and Agent marks share immutable checked facts. The
renderer cannot author source, select raw runtime bindings, adopt or run. Workspace18,
bundle22 and shared-views-v12 preserve frozen workspace17 history; catalog4 is unchanged.
See [creation UI boundaries and verification](../docs/desktop-workspace/stages/p2b2-3-creation-ui/README.md).

## P2B-2.4 — first adoption

The User reviews the checked creation flow, confirms a display name and adopts it
as a new Pipeline. One catalog5 transaction retains Pipeline + first Current +
draft birth mapping + receipt. Exact retries and restart recover the same result.
No Run starts. Creation history keeps its no-before comparison; Open pipeline
shows the registered Current and preserves Chat. Later refinements resolve the
retained creation source through the existing proposal path. Bundle23 adds User
adopt/outcome IPC; Workspace18 and tools12 are unchanged.

See [ownership and API](../docs/desktop-workspace/stages/p2b2-4-first-adoption/implementation.md)
and [test evidence](../docs/desktop-workspace/stages/p2b2-4-first-adoption/verification.md).

## P3 — preparation without execution

Current Flow includes a collapsible Run review with Data, Settings and Environment sections. The User selects an installed preparation engine, prepares the exact Current, and discusses an exact saved section in the existing Chat. Earlier reviews and sent references remain immutable when Current or input observations change. Agent marks retain their originating submission and do not replace User selections.

`internal/engine/prepared.go` owns the complete private Document codec and conservative scope validation; root `preparation.go` and trusted CLI `prepared-review` regenerate scientific review from those bytes without source evaluation. `internal/appservice/preparation*.go` owns retained-source preparation, bounded jobs, private payloads and durable review records. It imports only the mechanism-free preparation contract, never the engine. `main/service/run-preparation.ts` validates the closed native boundary; `main/pipeline-review` owns exact discussion authorization. React's `useRunPreparation` owns pending UI work only, while `RunPreparation` renders safe review and emits intents.

Preparing observes input metadata; it does not copy or validate the research data, create a Run, or establish tool/workspace readiness. Future P4 admission must validate the exact payload, runtime, staged input and workspace without invoking Project Go. Imported Pipelines without the retained creation input binding are explicitly unsupported in P3. Workspace18 storage and bundle23 remain frozen; Workspace19 migration archives original bytes.

See [P3 ownership, qualification and evidence](../docs/desktop-workspace/stages/p3-run-preparation/README.md).

## P4 — Run control

The accepted P4 implementation adds `launches` as a named trusted bridge. A native launch review checks a copied FASTQ input, installed analysis tools and an exclusive output location. The User confirms Start in the existing Project Chat; no Agent execution tools are added. Gobble persists admission before task submission, and a disconnected App reconnects to that exact Run. The Run Flow uses its saved preparation, joins task state by identity, and reuses existing task/log discussion evidence. Stop is addressed to the observed owner lease. See [ownership](../docs/desktop-workspace/stages/p4-run-control/architecture.md) and [verification](../docs/desktop-workspace/stages/p4-run-control/verification.md).

Workspace19/catalog5/toolset13 are unchanged; bundle25 adds closed command/review contracts. Prepared payload v1 and ordinary monitor schema2 remain stable; admitted checkpoints require pointer format2. This is local development qualification for the single-end Trim Galore → FastQC scope, not packaging or arbitrary pipeline execution.
