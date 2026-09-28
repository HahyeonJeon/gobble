# FastQC report sharing — requirements and three-level design

Index: [Accepted design](ideation-index.md). Confirmed by the owner on 2026-09-28.

## Requirements

**Goal:** a researcher opens the output of a completed quality step and discusses
that exact result with an Agent without seeing Go, HTML source or storage paths.
**Why now:** the actual workflow walkthrough found the completion-to-result gap.

| Actor | Need | Observable outcome |
| --- | --- | --- |
| User | Read the completed step's result and ask about it | Flow → readable report → one existing draft attachment. |
| Agent | Receive the same evidence with useful content | Complete source text/tables plus enumerated retrievable original charts from the captured version. |
| Gobble | Own executable facts | Proves exact Run/task attempt/output declaration and recorded checksum. |
| App | Own shared presentation and evidence retention | Displays/sends a verified rendition; never changes execution or recomputes quality judgments. |

Read registered execution evidence and one declared report file. Create an immutable
report capture when opening it, bounded by existing Project quota. Update only View,
navigation, draft and sent-reference state. Delete no source, Run or history. Reuse
existing evidence reachability/cleanup policy; closing a Pane does not delete evidence
still referenced by a draft or message. Unreferenced failed capture uses existing
orphan handling rather than a second cache lifecycle.

Included: App-created single-end Trim Galore → FastQC; supported FastQC 0.12.1 HTML
dialect first; static original text/tables/PNG; whole-report reference; exact selected
latest successful task attempt at capture; one immutable blob; current-message recipient chart reads.
Deferred: arbitrary HTML, ZIP/multiple-resource reports, external assets, other FastQC
dialects until qualified, MultiQC, metrics/region selection, charts, editing, live
report updates, general artifact plugins and old-engine migration.

## Design

### Conceptual Definition — Material change

- **Output evidence:** engine-owned attribution of one declared output to a successful
  task attempt. Contains Run identity, task identity, attempt, port, recorded digest
  and byte size. A pathname or a successful process alone does not establish it.
- **Report source:** the exact contained file bytes whose digest/size match that output
  evidence. Current source drift prevents a new attributed capture.
- **Saved report:** the complete supported reading content captured from that source:
  ordered source headings, text, table cells, literal source status/icon labels and original chart PNGs. Includes source
  digest, producer attribution and reading-profile version. It is a rendition, not an
  archive/download copy of original HTML/CSS. Its separate content hash protects the
  retained representation. No inferred scores, regenerated plots or scientific verdicts.
- **Report View:** a presentation of a saved report inside a Pane. Pane position, zoom,
  focus and table-of-contents navigation do not change report identity.
- **Report reference:** a whole-report evidence target carrying source and saved-content
  identity. Images have local identifiers for reading, not chart-region/metric semantics.
- **Read coverage:** charts actually returned to an Agent, distinct from report attachment
  availability. Do not claim a chart was observed merely because its name was delivered.

Accepted identity:

```text
Project + registered Run + engine Run ID
  + task instance + producing attempt + output port
  + source SHA-256 + source byte size
  + reading profile/version + saved report SHA-256
```

Output-report support uses a separate, independently versioned `output-capabilities` query. Preserve
the existing strict prepared-capabilities response and its launch/continuation callers;
adding unrelated fields there is not backward-compatible by assumption. An older
engine's unsupported-query result is an explicit unavailable-report capability.

Checkpoint ID is capture-time provenance. It is not a durable address required for
reopening: engine checkpoint generations are pruned. A later identical filename,
latest task attempt or updated Current design must never replace this identity.

FastQC eligibility is an engine-recognized recipe, not a declared module name or
filename. The output query returns optional `recipe: fastqc-v1` only after validating
the canonical task shape against the retained admitted declaration. The existing
`ReviewStep`/FastQC recognizer and `QualifyCreation` provide the relevant policy.
Runtime defaults in checkpoint tasks must not be mistaken for authored declarations;
use validated reconstruction or the retained sealed prepared document tied back to
the checkpoint, following existing continuation review ownership. Native service
requires that recognized recipe and declared `html` port for the supported workflow.
Unrecognized output may still have generic provenance but is not a supported report.

The first slice captures only the requested **latest successful task attempt**. If
that attempt is no longer latest, return an explicit unavailable result rather than
reading another attempt's output. Historical captured reports remain readable from
saved bytes. Their exact producer logs may later be unavailable under the current
latest-attempt log API; never route those controls to newer logs.

Confirmed policy choices in [discussion](discussion-01.md): reading-model option wins on
one content owner; on-demand chart reads preserve current message limits; unsupported
legacy engines preserve exact runtime binding. Reopen these choices if a real workflow
needs other formats, multi-file assets, messages-only Agent access or historical-engine
reading. Do not generalize in anticipation of them.

### Class and Function Design — Material change

These are accepted public boundaries, not implementation recipes. Use plain records
and concrete existing owners; no inheritance or plugin registry. Class state is only
justified for the isolated parser's lifecycle and the existing Workspace/evidence owners.

| Unit | Conceptual definition | Responsibility | Boundary | Relationship | Caller contract |
| --- | --- | --- | --- | --- | --- |
| Gobble `ReadOutputEvidence` | Attribution of one output | Resolve the declared port and recorded evidence of the requested latest successful attempt from a coherent retained engine state | No HTML parsing, App IDs, file mutation or scheduling | Existing engine state reader → CLI query → service | Input workspace, engine Run ID, instance, attempt, port; output versioned attribution record with optional engine-recognized recipe; unsupported/missing/ambiguous/non-successful evidence distinguished. |
| Native `ReadRunReport` | Verified report source | Resolve registered runtime/Project containment, require engine-recognized fastqc-v1/html output in the supported App-created workflow, return exactly the bytes checked against engine digest/size | No caller path, checkpoint parsing, rendering or schema inference | Runtime query + contained file reader → Main | Input Project, Run ref, instance, attempt; output attributed bytes; unsupported runtime/attempt, unavailable, changed, unstable read or oversize. Port is fixed by this bounded operation, not a generic user option. |
| `FastqcReader.read` | Supported report rendition | Validate complete supported dialect into ordered plain reading data under bounded isolated parsing | No privileged bridge, external network/assets, script execution or scientific inference | Main verified source → sandboxed parser → Main validated model | Input verified source bytes; bounded parsing time, structure depth/count and decoded image dimensions/bytes; output profile, original content and PNG inventory; unknown/lost content or exceeded bounds is explicit unsupported, never partial success. `close` releases parser resources. |
| Existing Workspace `open` / report branch | Shared report presentation | Publish one verified saved report and its View through the existing single writer | Does not infer output paths or expose live arbitrary HTML | Native read + reader + EvidenceStorage → Surface | Input selected Run/instance/attempt; captures before publishing View. Saved-ref reopen reads capture; explicit source opening is a separate action. |
| Existing evidence `prepare` / `accept` / `preview` / report branch | Addressed saved evidence | Validate report identity, model capability and send-time recipient; preview the same saved content | No live source reread, execution or silent chart omission | Workspace references + storage → provider/coordinator | One whole report attachment; text/tables/inventory delivered at send; image-capable sharedViews recipient required. Existing prepared-draft identity/consumption rules apply. |
| `read_report` Agent tool | Reading an addressed report | Resolve one attachment only from the active message's sent evidence and matching recipient | No arbitrary path/hash/request browsing; no new pointing authority | Existing turn/submission authorization → retained evidence | Input attachment ID and optional returned image ID; no image ID returns complete text/tables/inventory; image ID returns one original PNG plus context. Caller cannot supply a different submission/Agent. |
| Existing `workspace_observe` / `workspace_point` report branch | Foreground shared report reference | Issue and validate whole-report observation receipts and author marks | Existing ready/foreground/view rules retained; saved chart reads alone do not grant pointing authority | Saved View → shared reference policy → Agent/User marks | Observe ready displayed report returns identity, bounded visible reading content and whole-report receipt; point targets whole report only. If no view is ready, existing view-opening/reference-preview path must precede a visual pointer. |
| `ReportView` | Saved report reading surface | Present source content in order with internal navigation | No HTML execution, authority decisions or source refresh | Validated reading model → React text/table/image presentation | Input saved report and existing attach/focus/log callbacks; output UI events only. No chart extraction/recalculation in Renderer. |

**Representation and delivery bounds**

- Retain one versioned report JSON blob no larger than the existing 1 MiB blob limit,
  under the existing 64 MiB Project quota. It contains original PNG bytes (encoded),
  ordered text/tables, literal source status labels, complete chart inventory and attribution. No child-blob graph. Parsing has a finite job deadline and bounded node/depth/image counts in addition to input bytes; timeout or decoded-content excess returns unsupported and releases the isolated reader. Concrete limits must be qualified against the supported fixture during the reader slice.
- Bound source HTML at 1 MiB for this slice; profile text/table delivery at 64 KiB;
  each chart must satisfy existing image delivery dimensions/byte checks. A report
  exceeding any bound is unsupported as a whole; do not drop sections or squeeze
  multiple charts into an unreadable image. Existing file preview's broader 8 MiB
  bound is not automatically a report capture allowance.
- Storage bytes and delivered text bytes are distinct budgets. Report blobs must not
  be serialized wholesale into a model message or counted as ordinary 64 KiB JSON
  text. Evidence service and coordinator must agree on the report-specific delivery.
- Keep the existing two-image-per-message limit. Initial report delivery includes
  complete original text/tables and chart inventory. `read_report` returns one chart
  per call under existing tool response/turn limits. No increase to generic limits.
- PNGs remain original source graphics. Unsupported image dimensions fail explicitly;
  no hidden resampling to simulate whole-report coverage. A non-image-capable model
  cannot receive a chart-containing report silently as text-only evidence.

**Trust and retention**

Appservice verifies bytes from the registered Run's returned contained regular file
against recorded engine checksum/size, including changes during the read. Main only
parses bounded verified content in a sandboxed worker/renderer with no network or
privileged application bridge. Known structured content is rendered as text, tables
and validated image data, never injected source HTML/CSS. Unexpected report content
must cause an explicit unsupported result, not silently disappear during extraction.
Preserve original status/icon alt labels, including PASS/WARN/FAIL variants, as source
content; never recompute those judgments. Known decorative icons may become their
literal source labels. Preserve footer/link text as inert text; no source anchor can
navigate or fetch. External assets and executable content are unsupported. An inert
external footer link does not by itself make the known FastQC fixture unsupported.

Opening captures into existing content-addressed storage before publishing the View.
Draft, sent-message, shared-reference and View roots retain that same saved identity.
Restart/reopen uses saved content even if source is gone. Missing/corrupt retained
content remains unavailable; never reacquire current source as a substitute.

Agent chart reads are **addressed evidence**, not foreground observation. They can
work after a Pane closes because the User sent the whole saved report to that Agent.
Authorize through active Project/account/recipient/submission and attached report
identity. Another Agent or unrelated message cannot read it by guessing its hash.
This first slice only authorizes report chart reads for attachments on the active
message. Opening a report in the Workspace or mentioning an earlier attachment does
not authorize a full report read. Prompt “Attach this report to your message” when
needed; the existing report attachment action can reattach the same saved version.
Foreground observation identifies the report and bounded visible content; it does not
claim delivery of unseen charts. Broader earlier-turn reading is deferred.
Conversely, reading saved bytes cannot mark an unseen current View; whole-report
pointing still uses the existing shared observation receipt.

### Codebase Structure — Material change

Accepted new files stay flat within existing owners. Exact integration signatures may
follow existing naming, but any changed ownership or data promise returns to this design.

```text
internal/engine/output_evidence.go       ← CLI query routes
internal/appservice/run_report.go        → runtime reader + engine query
app/contracts/src/run-report.ts          ← schema consumers / generator
app/desktop/src/main/fastqc.ts           → isolated FastQC reader lifecycle
app/desktop/src/report-reader/reader.ts  ← isolated entry (DOM/data decoding only)
app/desktop/src/main/evidence/report.ts  → existing EvidenceStorage
app/desktop/src/main/shared-context/report.ts → sent evidence authorization
app/desktop/src/renderer/workspace/views/ReportView.tsx → validated contracts
```

| File / affected owner | Conceptual definition | Responsibility | Boundary | Relationship |
| --- | --- | --- | --- | --- |
| `output_evidence.go` | Engine output attribution | Exact attempt/declared output proof | No App/content decoding | Existing engine state; CLI consumes. |
| `run_report.go` | App's verified Run report source | Registered-runtime and file verification | No private checkpoint interpretation | Runtime/files; Main consumes. |
| `run-report.ts` | Report interchange vocabulary | Public source/rendition/ref validation | No I/O or UI policy | Service, Main, Renderer and generated schemas depend on it. |
| Main `fastqc.ts` | Bounded FastQC parsing session | Isolated reader process/job lifetime | No evidence authorization or general HTML host | Uses parser; evidence capture calls it. |
| `report-reader/` + `reader.ts` | FastQC reading-content decoder | One isolated build entry with its decoding code | Never imports Main/service/workspace | Receives bounded source; returns supported model. Separate entry is required by sandbox isolation, not hypothetical extensibility. |
| `evidence/report.ts` | Saved report evidence | Capture, preview and provider-text materialization | No runtime resolution or Agent identity lookup | Reader + storage; evidence service and workspace call it. |
| `shared-context/report.ts` | Recipient-scoped report reading | Matching current submission/attachment access | No live source/foreground bypass for pointer creation | Existing authorization and stored evidence; tool broker calls it. |
| `ReportView.tsx` | Report reading UI | Present validated content and local navigation | No execution/data authority | Contracts and callbacks; existing SurfaceView calls it. |
| Existing CLI/service/IPC/preload registries | Their existing typed routes | Register only the narrow report operations | No new forwarding service/class | Call the owners above. |
| Existing resource/surface/reference/workspace schema owners | Existing workspace state | Add report discriminants and preserve old records | Do not edit frozen historical definitions | Current schema generator + migration/bundle readers. |
| Existing evidence service/coordinator/storage roots | Existing send and retention lifecycle | Account for report delivery separately from retained bytes; preserve quotas/GC roots | No larger global image budget | Report materialization; current provider input shapes remain text/image. |
| Existing shared-context catalog/dispatch/observation and toolset version | Existing shared communication | Register report reads and whole-report observation/pointing | No external MCP server or broad filesystem tool | Same Main policy for User/Agent operations. |
| Existing RunTasksView/RetainedRunFlow/SurfaceView | Existing Run navigation | Add explicit quality-report action from supported exact task | Keep selected-attempt logs and User confirmation unchanged | Workspace open branch and ReportView. |

Current schemas are canonical. Implement a current-version migration and regenerate
through existing contract tooling; preserve historical captures verbatim. Bundle and
shared-toolset versions must change when new report resources/tools land. No numeric
version is reserved by this design.

## Acceptance and stage boundaries

1. Engine recognized recipe/declared-port evidence (reject name-only lookalikes and unchecked runtime defaults), latest-exact-attempt/port evidence, wrong Run/attempt, superseded attempt, failed/stopped attempt,
   absent/duplicate recorded checksum, legacy capability and unchanged launch/continuation capability decoding; service missing/changed/
   replaced-during-read file, containment/symlink and hash-of-returned-bytes checks.
2. Supported real FastQC 0.12.1 fixture content equality: all ten modules, original
   table/text, source status labels, inert footer text and eight original chart PNGs; malformed/unknown markup, scripts/external
   assets, oversize and image bounds rejected without partial-whole claims.
3. Save/open/move/close/restart/source deletion preserves report content and attribution;
   retention covers View, draft, sent message and marks. Old workspace/bundles still load.
4. Agent receives actual report text/tables and can read all enumerated charts across
   bounded calls; each PNG equals saved source bytes. Wrong recipient/message/project,
   unsupported model and detached read contexts fail. No chart read grants execution
   or unsupported visual pointing authority.
5. Actual Electron Flow→report→draft→send, User/Agent whole-report preview/pointer,
   focused/stacked/compact panes, explicit logs route, original evidence after newer
   Run/Current. New capable engine/new Run used for end-to-end qualification; old
   engine Run stays monitorable and clearly lacks report capability.

Begin execution with output evidence; return a completed slice summary before advancing. Implementation records live outside this frozen design result.
