# Report Agent sharing — execution

Accepted slice 3: saved whole-report attachment, current-message recipient-scoped reads and foreground whole-report pointing. Existing worktree `codex/project-workspace-design`; preserve inherited changes, no stage/commit/publication. Source/engine reader remains unchanged. UI stays English.

## Intent and ownership

User discusses one saved quality result with an addressed Agent from the existing Report View and composer. Create a draft reference, preparation and sent manifest pointing at the existing saved blob. Read original text/tables/inventory at send and original PNGs via `read_report` in that active submission. Update existing attachment, reference and View state. Detach removes draft ownership only; do not delete the saved report, Run or source. Shared foreground observation grants only whole-report pointing; chart reads never grant viewport authority.

## Reach / ordered thinking guide

1. Contracts: add whole-report target and captured representation, delivery-text accounting, read_report arguments; freeze Workspace 20, migrate to 21, regenerate bundle 30/toolset 15 from canonical contracts. EvidenceRef consumers require explicit report handling. Stop when identities, budgets and legacy migration prove consistent.
2. Evidence: reuse one saved blob for attach/preview/prepare/accept. Materialize complete text/tables and image inventory separately from retained JSON size. Require image-capable sharedViews recipient; preserve 2 direct-image and 64 KiB text limits. Existing writer and storage own publication/recovery/GC. Prove saved identity, corruption and recipient/config changes.
3. Agent: existing Coordinator context owns active account/recipient/submission. read_report resolves only an attachment of that submission and revalidates after I/O; return one unmodified PNG with identity and coverage. Reuse existing tool budgets/caching. Foreground observation/pointing requires a current receipt, separate from addressed reads. Prove wrong message/recipient/project, cancellation, source loss and observation staleness.
4. UI: add Attach report to existing toolbar; existing composer preview uses the same saved rendition; show whole-report marks without invented chart-region semantics. Exercise Flow→report→attach→prepare→send, Agent reads, closed/reopened report, compact/focused panes. Stop at completed implementation summary for owner review before the actual live walkthrough slice.

No new inheritance/plugin registry/cache or provider protocol. Report contract/representation is plain data; evidence/report owns pure rendition materialization; shared-context/report owns current-message authorization. Existing operations retain their single owner. Subagent consultation checks the actual authorization and delivery paths; implementation remains here.

## Findings / repairs

- Existing readSentEvidence is historical UI lookup, not recipient authorization. Bind Agent reads to Coordinator's active ToolContext and exact current submission manifest, then recheck after saved data I/O.
- Existing image renderer resizes/crops; original report charts must bypass it and use existing tool image delivery directly.
- A report's retained JSON includes images and must not be charged wholesale to the text delivery budget. Store explicit verified delivered text-byte count in its report representation; sender and persisted validation share accounting.

- New tool registration first failed against the test peer's old workspace_/gobble_ allowlist. Added read_report explicitly to that peer and the product Agent instructions; no general tool allowance.
- Initial attachment UI locator matched both Preview and Remove; narrowed it to the attachment name prefix. The complete Electron scenario then passed, including all eight original image wire payloads.
- New stale-pointer test exposed that Main's presentation identity omitted Report section state. Renderer remounts already noticed the change, but Main could briefly reuse an old receipt. Added reportModuleId to the existing render-session signature; no new lease mechanism.
- Question evidence permits earlier conversational attachments, which conflicts with this slice's current-message report read scope. Report-linked durable questions are explicitly unsupported; use the ordinary Chat attachment flow. No expansion to prior-message read authority.

- Cached chart reads also need Project-session validation: historical lookup is skipped on replay. Added a synchronous sentEvidenceGuard using the existing evidence epoch; Pane closure remains valid, Project switch/disconnect expires the read, including replay.
- Visual inspection found a generic preview footer labelling reports as Run facts and dialog header CSS affecting the nested report. Corrected report copy and scoped its header styles; reused the same ReportView for preview.

## Completion

Implementation and scoped verification are complete. See [verification](verification.md), [affected reach](reach.json), and the [shared discussion screenshot](screenshots/report-discussion.png). Workspace 21 / bundle 30 / catalog 5 / shared tools 15. Retain the exact uncommitted tree for owner review; next is actual-engine/live-Agent qualification.

- Electron batch qualification exposed native focus denials in three sharing scenarios. Repeated runs were inconsistent; preserved all logs and error snapshots. The test harness now reuses its existing native focus operation immediately before sharing and verifies no dialog is open. No production permission gate changed.
- One focused test then received an extra `t` in its draft that the test did not author. Preserved that mismatch; its isolated rerun passed. All 23 affected scenarios have passing evidence across runs, but this is not an all-green single batch. Interactive desktop focus/keyboard interference remains a qualification limit.
