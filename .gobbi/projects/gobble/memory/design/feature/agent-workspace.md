# Gobble App — Project-Centered Shared Workspace

Updated: 2026-09-07. The user approved Project as the primary workspace, with
multiple attached agents, runs and other resources; user/agent-controlled shared
panels; and discussion of selected evidence. The approved image is a concept,
not evidence of implemented behavior. Detailed behavior below is the first
implementation design for review. Stage 3 provides the tested local user workspace. Stage 4 adds account/peer
agent text discussion with automated protocol/Electron checks and separately
observed live ChatGPT sign-in, peer replies, isolated interruption and restart. Shared-view tools remain
stage 5. See the
[workspace record](../../../../../../docs/desktop-workspace/stages/03-workspace.md) and
[account/agent record](../../../../../../docs/desktop-workspace/stages/04-agents.md).

Related: [Application architecture](../architecture/application.md),
[Project workspace contract](../architecture/project-workspace-contract.md),
[Roadmap](../roadmap/project.md),
[Review and approved concept](../../../../../../docs/desktop-workspace/README.md).

The [domain model](../architecture/workspace-domain.md) defines exact Project,
App Workspace, Resource, Surface, Pane and Window meanings and maps file/Run/Plan
subjects to supported or future presentations. Pipeline definitions and Plans
are not inferred from a Run's observed Pipeline label.

## Accepted product structure

A person opens a Project and works with its agents around the central shared
work area. The Project contains files, plans, runs, results, agents, decisions
and saved view references. It exists independently of an agent conversation or
pipeline run. The user may have several registered Projects; the primary window
shows one active Project at a time.

The approved concept replaces the earlier conversation-led and run-led layout
candidates. Agent-centered collaboration remains a principle: agents can show,
observe and discuss the same evidence that the user manipulates. Project is the
navigation and continuity boundary. The file path remains `agent-workspace.md`
so existing design links continue to resolve.

The user revised the target layout to stacked central Panes and a right-side
Project chat on 2026-09-07. The earlier image remains a historical concept; the
[revised stage 5 sketch and interaction contract](../../../../../../docs/desktop-workspace/stages/05-shared-context.md)
are under visual review. The current built app still has the stage 4 layout.

## View hierarchy

| Location | Content and navigation |
|---|---|
| Project chooser | Create an app Project or open a local folder; return here if no safe Project can be restored |
| Project / Workspace | Central resource Panes stacked top/bottom; right-side Project chat with one timeline and one composer |
| Project / Files, Plans, Runs, Results | Resource collections inside that Project; opening an item adds or reveals a shared surface |
| Project / Agents | Attached peer agents, their state and settings; selecting an agent changes the discussion recipient |
| Project chat / Question message | Pending and resolved question metadata attached to an Agent message; Reply uses the same composer, without a separate Questions route or form |
| Project / Settings | Root/runtime information and Project preferences |
| App / Account and Settings | Account connection, app preferences and local dependencies; returns to the same Project |

The left rail groups resources under their Project. Runs are one resource type,
not the top-level app organizing unit. Agent names and roles are configurable;
Analysis, QC and Methods in the image are examples, not built-in mandatory roles.

The center is the shared work area. The first layout supports a single panel or
two vertically stacked panes, each with tabs; the user can resize the split, maximize a pane and
restore the arrangement. Save the Project's arrangement. Use no free-position
canvas, arbitrary popups or detached OS windows in this first slice.

The right column provides one chat-style Project timeline and one composer with
a named recipient, optional quoted reply target and evidence chips. User and
Agent messages have clear alignment and identity; view activity is a compact
inline event. Chat width can resize, and collapse preserves the draft. Question
messages use the same input and Send action as ordinary messages. Evidence
previews expand inline rather than creating a third interaction window. Switching
recipient never clears or changes the central panels or exposes private peer
history to the selected Agent.

## Interaction contract

User and agent actions enter the same workspace controller. A panel belongs to
the Project even when an agent opened it. Opening a surface, selecting a row or
answering a question does not authorize pipeline execution.

| Action | Required result |
|---|---|
| User opens a file, run or result | Open/reuse a supported Project surface, with a stable ID and explicit loading/error state |
| Agent presents one or two views | Open/reuse and arrange those references; disclose who opened them and why |
| Agent inspects privately | Use a Project-owned background view tagged to the requesting agent; do not change user focus |
| User asks “look at this” | Address a specific agent and attach the selected evidence reference, version and bounded content |
| Agent observes | Return what the current renderer actually displayed, including selection and freshness; a backend query alone is not a rendered observation |
| User pins a view | Agent requests preserve its contents and placement until the user unpins it |
| Agent releases a temporary view | Close only an unpinned, unmodified temporary view it requested; shared/user-adopted views remain |
| User closes a view | Keep its underlying resource, run and pending decisions; mark the view dismissed so the agent cannot immediately reopen it |
| Agent is interrupted or detached | End that attachment's active work according to provider outcome; preserve Project surfaces, other agents and runs |

Explicit user requests to show or compare views authorize the corresponding
presentation. Preserve keyboard focus in unfinished input; mark the new pane
with a visible attention cue. Unsolicited presentations open in the background.
If both panes are pinned, queue an offer to show the new view. Do not evict work.

Repeated open requests reuse the intended view and request identity. A user can
explicitly duplicate a resource into two panes for different selections. Two
agents requesting layout changes at once are serialized against the layout
revision; a stale request receives a conflict/current layout, not silent
last-writer replacement. A queue entry can expire as unavailable; expiration
is never treated as user acceptance.

## Initial surface set

| Surface | Initial behavior and source |
|---|---|
| Project overview | Registered root, attached agents, resource availability and recent shared activity |
| File / CSV table | Contained bounded text or tabular read; arbitrary CSV is not interpreted as a universal assay sample schema |
| Plan / run | Read-only Gobble-derived graph and state from validated outputs or compatible Inspect projections |
| Task / log | Task identity, attempt, recorded error and supported bounded log tail; missing logs remain explicit |
| Image result | A registered image artifact with scoped selection/capture; never execute image metadata or artifact content |
| Decision | Evidence-linked question, named recipient, effects and response state |

The concept's QC chart demonstrates spatial pointing. Initial visual proof uses
an actual registered static image artifact; a full interactive MultiQC/HTML
viewer is a later surface. Do not invent report data when a file is unavailable.
Arbitrary HTML, websites, executable agent-created UI and bidirectional visual
pipeline authoring are outside this slice. Source display does not imply a
source editor; source changes and run controls arrive with their write contracts.

## Pointing and context sharing

Every shared context item carries `projectId`, resource reference, resource
revision, surface ID and selection. Selection kinds are explicit: table row
keys and columns, text/log range with attempt, or normalized image rectangle
with original image dimensions and content hash. Pixel coordinates alone do
not identify data after zoom, scrolling or resize.

The composer visibly shows recipient and removable context chips. The first
slice addresses one agent per message. It does not broadcast to all attached
agents. The attachment itself does not upload the Project; explicit context and
permitted agent reads are visible in activity. Raw sequencing data and unrelated
files are not automatically included. A tool read reports the bounded content,
source and truncation it actually delivered.

Before dispatch, revalidate the reference and version. If meaning changed, show
“Selection changed” and offer to refresh the context. Preserve the original
reference in history. A response links to the evidence it used and records its
observation time. Do not treat a screenshot as scientific validation or proof
that a model consumed it; integration evidence must show the actual tool result
reached the provider and informed the response.

A hidden Project can satisfy a structured data query without changing focus.
A rendered observation or capture requires a live supported surface. If the
renderer is absent, return `ui_unavailable`; do not fabricate a screenshot or
claim the view was seen. Unsupported background capture can wait for a visible
surface under an explicit request rather than stealing attention.

## Questions and decisions

A contextual question has `decisionId`, requesting agent, Project, evidence
references, dependency revisions, proposed effects and options/free text.
States are `pending`, `answered`, `dismissed` and `invalidated`. It appears as an
Agent chat message with evidence links and an inline Reply action. There is no
separate Questions panel or answer form. Reply sets the question reference and
recipient in the same composer; only explicit Send records an answer. Draft
autosave, including an offline reply draft, leaves the question pending.

Only an explicit user answer changes pending to answered. Closing a pane,
timeout, sign-out and agent interruption do not answer it. Reopening the app
restores pending questions independently of provider connection. An answer
records the selected value and current dependency revisions before delivery.
Send explicitly starts the normal addressed follow-up to the requesting Agent.
If admission is unavailable, retain an unsent draft rather than recording or
automatically delivering an answer. After accepted Send, provider uncertainty
remains separate from the durable recorded answer.

An input/source change that alters the question's meaning invalidates it and
requires a renewed question. Ordinary run progress does not invalidate an
unrelated configuration question. Retry delivery must not create a second
question or submit an answer twice. Resolving scientific intent and authorizing
an operation remain separate when their effects differ.

## Concrete first-session storyboard

| Moment | User and agent experience | Evidence |
|---|---|---|
| Open Project | User selects a folder; Files, Agents and Runs are scoped to it | Stable Project/root registration; Docker may be unavailable |
| Attach two agents | User names two peer attachments and connects the account | Distinct agent/thread bindings and explicit selected recipient |
| Show two views | User asks one agent to open a CSV and an image result side by side | Two real resource references and renderer acknowledgments |
| Point and ask | User selects a row and an image region, then addresses the other agent | Versioned selections and actual bounded text/image delivery |
| Inspect and answer | Agent observes the displayed evidence and asks a contextual question | Observation, question and explicit response remain linked |
| Inspect a run | User or agent opens a compatible CLI-started workspace | Desktop facts match the engine projection, with freshness |
| Reopen | User quits and reopens the app | Project, attachments, layout and pending question restore without replay |

The complete later analysis journey adds source review, plan generation, Run,
monitoring, Stop and Resume impact review. Those operations use the same Project
and surfaces; adding execution must not make the Project subordinate to a run.

## Desktop lifetime, adaptation and accessibility

One primary window is authoritative for the visible Project in this slice.
On the initial macOS target, closing the last window destroys its renderer but
keeps the main process and active app helpers until explicit Quit; later Dock
activation restores the window. Pending rendered observations return unavailable
while there is no renderer, and agent activity cannot reopen the window by
itself. Normal Quit requests interruption of active agent turns, records any
uncertain submissions, and ends app-owned processes with bounded cleanup. Accepted Docker controllers continue independently.
Later launch restores the last valid Project and layout; after a crash it also
marks interrupted agent work and rechecks pending requests before any retry.
Unavailable roots open a recoverable Project view with a locate-folder action;
relinking never silently adopts a different run identity.

A second app launch reveals the existing window. File/protocol deep links,
notifications and detached windows are deferred; unsupported activation inputs
are rejected without navigation or source changes. macOS Dock activation can
recreate the primary window from safe stored state. Off-screen bounds are
clamped to a visible screen.

All open/select/split/pin/close/address/answer actions need keyboard and accessible
names, not only dragging or hover controls. Provide a focus order and return
focus when closing a panel. Keep states readable without color. Tables support
long names, horizontal scrolling and stable row selection. Resize split panes
with pointer or keyboard. At narrow widths use a single visible pane with
retained tabs; do not squeeze two unreadable tables. Motion only explains a
panel/layout change and respects reduced-motion preferences.

Neutral surfaces, teal selection and clear Agent identities remain the visual
language. The revised composition uses central stacked Panes and a right-side
chat with one composer, as requested by the user. Exact tokens, minimum dimensions
and typography remain prototype refinements, not usability claims. The user explicitly requested English for all app-owned copy, menus, errors and
accessibility labels. Preserve the original language of user data and filenames.
Keep strings separable for later localization; the first-slice locale is resolved.

## Design evidence and review obligations

The product owner supplied the workflow and approved the generated Project
concept in this task on 2026-09-06. The screenshot contains illustrative data,
not measured analysis. Stage 3 adds actual Electron and native-service proof for
user workspace interactions. No representative-user usability study has been
completed. Stage 4 verified actual provider login, peer text replies, independent
interruption and normal restart. Agent shared-view/image behavior and live Docker
integration remain later evidence; retain those limits.

| Design activity | Disposition and current result | Open obligation / reopen condition |
|---|---|---|
| Discovery | Reused current evidence: owner workflow and repository design at `7fe3d5c` | Other users' tasks/modalities may differ; broaden before release claims |
| Framing | Performed for the current subject: Project as shared working context | Reopen if navigation or ownership becomes agent/run-dependent |
| Concepts | Project-centered ownership approved; owner then requested central stacked Panes, right chat and one composer on 2026-09-07 | Revised visual sketch is awaiting review before stage 5 implementation |
| Prototyping | Performed for the current subject: generated English refinement and stage 3 Electron implementation | Automated keyboard, rendering and recovery evidence is recorded; representative-user testing remains open |
| Representative-user testing | Reused current evidence: owner feedback informs test questions; no representative-user test performed | General usability acceptance remains open; test actual target tasks/modalities before release |
| Design–implementation collaboration | Contract, controller/storage, local views and stage 4 live provider behavior have implementation evidence | Revised chat layout and Agent-controlled views require stage 5 evidence |
| Post-release improvement | Not applicable with exact reason: no desktop release or users exist yet | Activate after a tested release; no outcome measurement claimed now |

The owner of this design record is the implementing maintainer. Its evidence
locations are this document, the approved image, the contract and desktop review.
Dispositions include the initial concept review and stage 3 local workspace evidence. Prototype, usability
and implementation obligations remain open where the table says so. Success
means users can identify the Project, recipient and shared evidence, recover a
view, and distinguish agent activity from analysis state. More messages, more
panels or faster clicks alone are not success measures. A misdirected context
share, displaced pinned work, or uncertain retry reopens the affected design.

## 2026-09-07 · Stage 5.1 implementation checkpoint

The user approved stacked central resource Panes and a right Project chat with one
composer, and requested communication tools usable by both User and Agent. The first
implementation slice introduces WorkspaceDocument v2 (`paneOrientation: vertical`,
chat width), immutable v1 migration backup, actual mounted-Pane reporting and a dedicated
renderer/chat owner. The Workspace Pane graph and native service contracts remain v1.
User/Agent pointing shares `EvidenceRef` + `Selection` with author-separated shared references;
Agent marks must not overwrite local selection. Text ranges, table row/column IDs and
normalized image regions refer to exact source revisions. Semantic Plot point/axis selection
needs a future interactive Plot resource contract. The accepted initial Agent tool transport
is official Codex dynamic tools through the application-owned port; MCP remains a future
adapter. These tools/overlays/attachments/replies are the subsequent implementation slices.

Current evidence and next checkpoint: [Stage 5.1](../../../../../../docs/desktop-workspace/stages/05-1-chat-foundation.md).
