# Gobble App — Remaining work

As of 2026-09-29. This is a scope/status inventory, not authorization to implement
all listed extensions. The owner's latest pipeline-collaboration direction supersedes
the earlier packaging-first recommendation and older “next R4c1” checkpoint labels.

## Owner decision

Prioritize **User/Agent discussion through the pipeline flow, data and settings →
visual change review → authorized execution → monitoring and refinement**. The
User must not need Go/Gobble knowledge or source-diff review; the Agent handles
code behind the UI. The owner accepted [P1](stages/p1-foundation/README.md) and
approved continuing with this correction. The [revised design and sequence](proposals/visual-pipeline-collaboration/README.md)
bring pipeline inspection ahead of visual change acceptance. The owner approved its concrete screen and requested implementation.
[P2A-1](stages/p2a1-pipeline-flow/README.md) now provides the actual checked flow;
the owner accepted its [flow visual refinement](stages/p2a1-flow-polish/README.md) and authorized P2A-2. [Checked-flow discussion](stages/p2a2-flow-discussion/README.md) was accepted by the owner. The owner selected B / Change spotlight from the [revised comparison concepts](proposals/p2b-visual-change-review/alternatives-v2/README.md). [P2B-1](stages/p2b1-visual-proposals/README.md) implements scoped refinement, checked visual review and User adoption; the owner accepted its result. The owner accepted the [P2B-2 creation design](proposals/p2b2-pipeline-creation/README.md); [P2B-2.1 storage foundation](stages/p2b2-1-creation-storage/README.md) is complete. The owner authorized [P2B-2.2 creation checking](stages/p2b2-2-creation-check/README.md), and accepted its result. The owner authorized [P2B-2.3 creation UI and Agent discussion](stages/p2b2-3-creation-ui/README.md), accepted by the owner. [P2B-2.4 first adoption](stages/p2b2-4-first-adoption/README.md) is accepted. The owner authorized continuous completion of [P3 Run preparation](stages/p3-run-preparation/README.md), including design, implementation and verification, without intermediate approval stops. The owner then approved the concrete P4 design and implementation. [P4 Start/Stop and retained Run Flow](stages/p4-run-control/README.md) is complete. The owner approved P5A; [linked feedback and fresh analysis](stages/p5a-feedback/README.md) is now implemented and verified; subsequent stages retain the owner’s sketch/result approval rhythm. P5B-1–3 add engine history/admission and durable continuation transport. The owner authorized P5B-4; [shared continuation review](stages/p5b4-visual-review/README.md) was accepted. The owner authorized [P5B-5 actual qualification](stages/p5b5-qualification/README.md), which now provides real-tool continuation evidence for the bounded workflow. Earlier source-oriented plans and reviews stay historical.

The owner does not need document editing now. Defer the proposed Notebook editor,
live unsaved-document references and save-copy controls as one dependent slice.
The existing saved Notebook reader and User/Agent reference flow remain the current
Notebook capability. Preserve R4b findings for future decisions; do not install its
qualification dependencies into production. Kernel connection/execution is a separate
proposal and is not automatically authorized by postponing editing.

CSV chart creation remains excluded. Do not restore it through a general dashboard
or scientific viewer proposal.

## Implemented foundation

Project-centered Panes and right Chat, Agent participation, file/table/image shared
references and captured evidence, Run/task/log navigation, runtime dependency graph
navigation, bounded PDF page/region reading, and saved Notebook source/output
reading with User/Agent references are implemented. Stage 6 demonstrated actual
Gobble/Docker and signed-in Agent integration within its recorded scope. This does
not establish Resume or a packaged desktop application. P4 now adds scoped App Start/Stop.

## Core pipeline collaboration — checkpoint status

| Checkpoint               | Remaining outcome                                                                                                                                                                                     | Owner boundary                                                                                            |
| ------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| P1 foundation            | Implemented and accepted: Pipeline identity and presentation lifetime.                                                                                                                                | Service owns definitions; Main retains one workspace writer.                                              |
| P2A-1 checked flow       | Implemented: native import, actual composed flow, declared ports/control/resource details, fit/list navigation and retained inspection history. Accepted, including the subsequent visual refinement. | Gobble supplies facts; service owns retained inputs, jobs and artifacts; App owns views.                  |
| P2A-2 discuss the flow   | Implemented: exact User/Agent step/port/connection/declared-setting references, immutable Chat attachment, observation and independent Agent marks. Accepted by the owner.                     | Main owns reference/evidence policy; typed settings must come from explicit supported module metadata.    |
| P2B visual changes       | P2B-1 implemented: existing-Pipeline refinement with checked B comparison and exact adoption. P2B-2.1 implements draft storage, origin and per-Pipeline revision migration. P2B-2.2 adds Gobble scaffold/input contracts, profile runtime and whole-creation candidate checking. P2B-2.3 connects creation UI, scoped Agent authoring and exact checked addition discussion. P2B-2.4 adds exact first adoption, atomic recovery and retained birth/Current source ownership.                                                                                            | Agent authors; service owns source lifecycle; User reviews scientific objects in the UI.                  |
| P3 execution preparation | Implemented for App-created single-end Trim Galore → FastQC and supported refinements: exact private engine payload, independent safe review, source/input metadata/runtime identity, saved history and exact User/Agent discussion.                                                                                                               | Gobble owns executable semantics; App never treats a picture or current Plan JSON as execution authority. |
| P4 Start and Stop        | Implemented: content-checked copied data, typed User Start in Chat, durable exact admission/reconnection, retained Run Flow/task discussion and expected-owner Stop.                                                                                                                  | Gobble admits/executes; service records operations; UI/Agent share policy.                                |
| P5 Feedback and Resume   | P5A implemented: exact sent Run evidence → supported Agent refinement → visual review/adoption → linked fresh analysis. P5B-1–5 implement same-design review, exact continuation admission, transport, shared Flow/Chat UI and actual Trim/FastQC Stop/Resume qualification.                                                                                                                     | Gobble owns reuse and recovery; shared references connect earlier Run evidence and new design.            |

The [revised checkpoint plan](proposals/visual-pipeline-collaboration/plan.md)
supersedes the old source-first order, not its source/engine ownership constraints.
Existing Run monitoring does not establish
execution controls. P1 preserves the native macOS root-library support limit and
adds actual focused Linux/amd64 Plan test evidence alongside native service/App
checks. See the [current P2A-2 verification boundaries](stages/p2a2-flow-discussion/verification.md).

## Remaining original first-slice delivery

| Item                           | Outcome                                                                                                                   | Status and boundary                                                                                                                                                                                                                        |
| ------------------------------ | ------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Stage 7: local Mac application | A self-contained local `.app`; verify Project opening, login, runtime paths, shared views and restart from that artifact. | Not started; lower priority than the core collaboration loop under the latest owner direction. Requires its own design/review checkpoint. Signing, notarization, public distribution and automatic updates are outside this local package. |
| Known engine qualification gap | Diagnose the two baseline packed-executable failures recorded in Stage 6.                                                 | Separate unresolved engine evidence: `TestPackPrintpipeArtifact` and `TestPackHostpipeEmptyInspectProtocol`. App generic-runtime verification passed; that does not resolve these tests.                                                   |

## Proposed product extensions, not mandatory prerequisites

| Area                           | Remaining outcome                                                                                                                       | Ownership / scope                                                                                                                 |
| ------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| R3 reports                     | Open existing HTML/MultiQC reports; discuss an exact section, sample or metric with source provenance.                                  | First qualify isolated report display; add semantic references only where an adapter proves them. No report/chart builder.        |
| R3 navigation depth            | Historical attempt access, live log streaming, larger-run pagination; PDF exact text where reliable; additional file formats as needed. | Per-view additions. Existing bounded previews must not be described as complete large-data support.                               |
| R5 one scientific adapter      | Choose genome browsing or single-cell/spatial exploration against an actual research need.                                              | Alternative initial tracks, not a commitment to implement both. Require explicit dataset/coordinate identity and measured limits. |
| R6 interoperability            | Optional external Workspace MCP endpoint and separately considered MCP Apps/extension hosting.                                          | Existing in-App Agent tools do not constitute an external MCP server. These are optional later integrations.                      |
| Later distribution/reliability | Signing, installation/update recovery, additional OS targets, remote environments and broader operational monitoring.                   | Separate roadmap phases; not included in the initial local Mac package.                                                           |
| Deferred R4                    | Notebook editing/save and optional live kernel integration.                                                                             | Editing is deferred by the owner. No immediate implementation of either capability.                                               |

## Recommended next decision

The owner accepted [P5B-5](stages/p5b5-qualification/README.md) and selected the existing single-FASTQ workflow for evaluation. The [actual App walkthrough and improvement concept](stages/single-fastq-walkthrough/README.md) now identify the main gap: completed execution does not lead directly to a readable, shared FastQC report. A compact two-Pane layout also hides the initial Flow below recovery guidance.

The owner approved the **completed step → shared report → exact report attachment → discussion** direction. The first [completed Run presentation slice](stages/completed-run-presentation/README.md) is implemented and verified: Flow-first layout, settled completion controls, preparation wording and admitted Run-list refresh. The owner accepted this result and authorized the [FastQC report-sharing design checkpoint](proposals/fastqc-report-sharing/draft/README.md). The owner approved the [final design](proposals/fastqc-report-sharing/accepted/ideation-index.md) on 2026-09-28. Its first [output-evidence slice](stages/report-output-evidence/README.md) adds exact producer attribution and verified source acquisition; implementation and verification are complete for this bounded foundation. The owner authorized the [saved report slice](stages/saved-fastqc-report/README.md), now implemented and verified: original reading model, isolated decoder, lower-Pane Report View, retained versions, section navigation and close/restart/source-loss reopening. The owner accepted that result and authorized the [Agent sharing slice](stages/report-agent-sharing/README.md), now implemented: whole-report attachment and preview, complete text/tables plus chart inventory at send, current-message recipient-scoped original chart reads, and whole-report User/Agent pointers. Automated Electron qualification verifies all eight original chart wire payloads after restart/source loss. The owner approved the [actual-engine/live-Agent walkthrough](stages/report-live-walkthrough/verification.md), now completed with a fresh successful Run, all eight original charts delivered/displayed, shared report pointing and one checked 20 → 25 setting proposal. Current was not changed and no second Run started. Data and all three completed Agent submissions survive restart. The reproduced maximize/compact navigation defect is now fixed by the [session navigation repair](stages/pipeline-review-navigation/verification.md): each View retains Current/Changes, proposal and change selection independently of observation receipts. Comparison-location restoration after restart remains a separate persistence decision. Long duplicated notes, raw Markdown and parameter-bearing step names are subsequent reading-UX findings. No product source changed during the live qualification itself; the subsequent navigation repair changes renderer state ownership. Report-linked durable questions, earlier-message chart reads and chart-region semantics remain unsupported. The walkthrough is expert evaluation with synthetic data, not representative-user or scientific validation.

The bounded P5B implementation sequence is complete and accepted. Packaging, broader recovery, arbitrary workflows, resource scheduling, additional workflow scopes, Notebook editing and additional formats remain separate decisions.

## Sources of status

- [P5B-5 actual continuation qualification](stages/p5b5-qualification/README.md)
- [P5B-4 shared continuation result](stages/p5b4-visual-review/README.md)
- [P5B approved implementation sequence](proposals/p5b-continuation/plan.md)

- [P4 Start/Stop result](stages/p4-run-control/README.md)

- [P3 Run preparation result](stages/p3-run-preparation/README.md)

- [P2B-2.3 creation UI result](stages/p2b2-3-creation-ui/README.md)

- [P2B-2.2 creation checking result](stages/p2b2-2-creation-check/README.md)

- [P2B-2.4 first adoption result](stages/p2b2-4-first-adoption/README.md)
- [P2B-2.1 creation storage result](stages/p2b2-1-creation-storage/README.md)

- [P2B-2 creation design checkpoint](proposals/p2b2-pipeline-creation/README.md)

- [Implemented P2B-1](stages/p2b1-visual-proposals/README.md)
- [P2B revised comparison checkpoint](proposals/p2b-visual-change-review/alternatives-v2/README.md)
- [Original P2B ownership proposal](proposals/p2b-visual-change-review/README.md)
- [Accepted P2A-2 result](stages/p2a2-flow-discussion/README.md)
- [Implemented P2A-1 result](stages/p2a1-pipeline-flow/README.md)
- [Accepted visual pipeline design](proposals/visual-pipeline-collaboration/README.md)
- [Accepted P1 foundation](stages/p1-foundation/README.md)
- [Approved-plan baseline](proposals/pipeline-collaboration/README.md)
- [First-slice package contract](session-plan/plan-01.md#task-07-native-package--로컬-mac-앱-패키지와-인수)
- [Stage 6 actual integration and retained failures](stages/06-integration-evidence.md)
- [Research workbench extension plan](proposals/shared-research-workbench/implementation-plan.md)
- [R4a3 implemented Notebook sharing](stages/r4a3-agent-notebook.md)
- [R4b investigation and deferred editor proposal](stages/r4b-live-integration.md)
- [Product roadmap: execution and distribution](../../.gobbi/projects/gobble/memory/design/roadmap/project.md)

## Publication checkpoint

The owner requested wrapping up, committing and pushing the accumulated work on 2026-09-29. Publish the current `codex/project-workspace-design` branch; no merge to develop, release, package, or additional feature is part of this checkpoint. See [publication record](publication-checkpoint.md). Follow-up reading-UX candidates require their own concrete design review.
