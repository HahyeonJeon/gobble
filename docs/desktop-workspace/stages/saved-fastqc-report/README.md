# Saved FastQC report — execution

Accepted: ../../proposals/fastqc-report-sharing/accepted/ideation-01.md (slice 2).
Worktree: existing codex/project-workspace-design; preserve inherited changes; no staging/commit.

## Reach and thinking guide

1. Contracts and parser: freeze Workspace v19, add source/rendition/report resource and saved-report metadata, migrate to v20; canonical contracts/src sources, export-schema.ts generates bundle v29, schema:check proves consistency. Main FastqcReader owns sandbox lifecycle; report-reader/reader.ts owns the qualified 0.12.1 decoder. Prove full real fixture equality and whole-report rejection on unsupported content/bounds.
2. Retention: existing EvidenceStorage accepts a minimal blob descriptor (current force: attachment and report both own immutable bytes); no fake attachment and no second cache. Report store verifies attribution and saved bytes. Workspace single writer publishes capture plus metadata/View together. Small Project savedReports metadata roots permit close/restart/reopen; blobs stay in existing quota/GC. Failed publication remains orphan-managed. Prove corrupted saved content never falls back to source and prior profiles migrate.
3. UI: completed quality step opens saved report below Flow; report has source headings/status labels/tables/original PNGs, TOC, exact producer logs. Existing Pane controls provide focus/move/close. Render no source HTML. Existing workspace command is the only new user entry point. Explicitly refuse Agent report open/observe/select/share/attach until slice 3. Prove English UI, compact/focused rendering, original charts and saved reopen.

No Go/engine changes: prior output-evidence slice already proves provenance. No charts/editor/plugins/general HTML support. No design pattern; concrete lifecycle owner, plain immutable records, existing writer.

Completion: implementation is retained uncommitted. See verification.md for final checks, screenshots, limits and tree identity. Next slice remains Agent attachment/tools.

## Findings and repairs

- Surface close removes the View. Project-level saved metadata must retain the same blob to support explicit saved reopen after close; payload remains outside Workspace document.
- Report payload must fit the existing SurfaceData envelope as well as the 1 MiB blob bound; no global budget increase.
- The actual 0.12.1 HTML is also well-formed XML. Decode it strictly as XML in a network-denied sandbox to reject browser HTML error-repair and unknown markup.
- Initial baseline command used app cwd as repository root; directory creation failed before any edits. Corrected the root and reran.

- First parser qualification failed in test setup: Playwright main evaluation does not expose CommonJS require. Moved the test-only class import to the temporary Electron entry and require success of the real fixture before interpreting rejection checks.
- Adding Report exposed legacy fallthroughs that assumed every non-file was a Run/log. Added explicit unsupported guards to observation, capture and reference paths; no fake Report attachment semantics.

- First real sandbox job blocked its own data-URL bootstrap (ERR_BLOCKED_BY_CLIENT). The host now allows only the exact encoded fixed main-frame bootstrap, then denies every request. Report nodes remain unmounted. No report-provided URL is allowed.

- Qualified FastQC FAIL decorative PNG is missing one byte of its IEND chunk. Strict chart PNG validation correctly exposed this, but source decorative icons are rendered as literal labels by the accepted profile. Freeze the exact qualified decorative bytes per label; only these known assets may become labels. Original substantive graphs still require complete PNG chunk/CRC/dimension/decode validation.

- The real fixture also has three substantive PNGs with the final 1–2 IEND CRC bytes absent. The reading profile recognizes only this exact 10/11-byte zero-length IEND suffix, checks every complete chunk CRC, and requires successful bounded bitmap decode. It retains the unmodified original byte sequence; no repair/resampling/re-encoding. Other truncation remains unsupported.

- UI capture/open/section navigation passed initially, but the test guessed a focus-menu label. Corrected it to the existing accessible Maximize secondary pane control and actual work-pane class; no UI feature was added for a test assumption.
- Parser rejection test initially used an absent base64 substring. Every mutation now asserts it changed the source before expecting rejection.
- Checked the pinned upstream 0.12.1 OverRepresentedSeqs.makeReport: its no-hit branch emits a plain paragraph, already supported by the reading profile (https://raw.githubusercontent.com/s-andrews/FastQC/v0.12.1/uk/ac/babraham/FastQC/Modules/OverRepresentedSeqs.java).

- Visual check found that maximizing remounts a View and lost the selected report section. Added one Report View navigation field/intent following the existing PDF/Notebook host-validation path. It addresses an existing module only, changes presentation rather than saved identity, and survives focus/move/restart. No report semantic selection was enabled.

## Result and ownership

- Gobble and native service retain execution/source authority unchanged. The only new service client route is readReport with exact Project/Run/instance/attempt association.
- FastqcReader owns a 10-second hidden sandbox job. The reading profile owns strict XML grammar, qualified decoration and original chart bounds; no network, preload, injected HTML or scientific recalculation.
- ReportEvidence owns source/content validation and one immutable JSON capture. EvidenceStorage now accepts minimal blob metadata used by both report and attachment owners, preserving existing file/quota/GC behavior.
- Workspace is the sole writer for Project savedReports metadata and Report Views. Close removes a View, not its saved report. Failed captures are cleanup candidates on the next commit/restart. A View stores only its section address; navigation does not change captured identity.
- React renders English navigation plus source text/tables/original PNGs. Existing Pane controls provide focus, move and close; selected section survives a focus remount. Exact old-attempt log failure never changes report identity or jumps to newer logs.
- Agent report opening/observation and User sharing/attachment are explicitly unavailable in this slice. No temporary fake attachment representation was added.

## Concrete reading limits

Source <= 1 MiB; complete retained JSON and displayed SurfaceData envelope <= 1 MiB; text/table/inventory <= 64 KiB; Project evidence quota 64 MiB; saved metadata <= 128 reports. Decoder <= 12000 nodes, depth 16, 16 modules, 16 charts, 32 blocks/module, 32 columns, 2000 rows/table, 1536 pixels per chart edge; 10-second deadline. Whole-report rejection on an exceeded bound. Source IEND quirk is recorded above; original PNG bytes remain untouched.

## Current visual result

[Stacked Flow/report](screenshots/report-stacked.png), [original chart](screenshots/report-chart.png), [focused report](screenshots/report-focused.png), [compact report](screenshots/report-compact.png), [saved reopen](screenshots/report-restored.png).

## Next bounded slice

Whole-report attachment to the existing composer; complete source text/tables and chart inventory for the addressed Agent; recipient/current-message-scoped read_report image retrieval; whole-report observation/pointing. Use these same retained bytes. Do not add semantic chart/metric regions, document editing, CSV chart generation or a report builder. Obtain the owner's result review before advancing.

- Full format check found pre-existing drift in workspace-v17.json and creation-candidate.json; both byte hashes match before.json. Those inherited files remain untouched. Newly frozen v19 formatting was corrected; all task-modified authored files pass their targeted formatting check.

- Final shipping build identity differed only for the decoder because the parser spec rebuilt it. Removed that redundant build; parser and UI qualification then passed (5 tests) against the unchanged shipping decoder. See verification.md for hashes and limits.
