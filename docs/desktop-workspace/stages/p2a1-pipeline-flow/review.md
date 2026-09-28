# P2A-1 — Implementation and visual review

Date: 2026-09-09. Author implementation and self-review against the approved
P2A-1 contract; no independent reviewer or new delegated task was used.

## Review results

| Finding                                                                                  | Resolution and evidence                                                                                                                                                                                                                      |
| ---------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Default pipeline opening previously showed a source file.                                | Pipeline registration now resolves to its own Resource/Surface and saved Gobble flow. Source is optional Files navigation. Native import requires no package or command input.                                                               |
| Native folder aliases on macOS failed containment against a canonical Project root.      | Resolve the chooser path before the relative-root check; outside paths remain refused. Native import and a focused service alias regression cover this.                                                                                      |
| Two separate reuse checks could disagree when opening the same pipeline in another Pane. | `reusableSurface` is the shared pure User-navigation policy used by the controller and transition. Existing file behavior remains; pipeline reuse is per requested Pane. Unit and actual two-Pane open/close scenarios cover the difference. |
| A late import result could attempt navigation after its Project browser was replaced.    | Scope completion to the mounted Project before updating state or opening a view. Main still validates active Project ownership.                                                                                                              |
| The first native flow screenshot clipped the far end of the graph.                       | Add Fit and bounded zoom controls. Representative parallel paths now fit beside Chat; the connected list remains available. No graph facts change with layout.                                                                               |
| Short split Panes left too little visible detail space.                                  | At short heights, local inspection prioritizes details and Close returns to the flow. Native compact screenshots verify readable selected input/output fields. Consolidated the final CSS rules after review.                                |
| Inspection compilation needs executable disposable scratch.                              | The retained source/root stay read-only; only the bounded scratch mount permits the temporary compiled driver. Actual Linux evaluation qualifies this requirement.                                                                           |
| A host-client deadline alone did not bound orphaned container work.                      | Add a container-local timeout as well as the service deadline and owned-container cleanup. The successful native path uses this exact command. Full deadline expiry is not represented as a tested scenario.                                 |
| Adding pipeline resources could silently broaden historical reference schemas.           | Keep original discussion Resource branches separate, freeze Workspace v14 and write v15/bundle v18. Historical exported discussion/tool/storage definitions remain equal; existing bundles remain byte-identical.                            |

## Boundary review

- Gobble public inspection carries actual validated ports, boundary connections,
  explicit controls and requested resources. Plan serialization is unchanged;
  renderer layout supplies geometry only. No path/name heuristics infer dependencies.
- Source retention, runtime dispatch, job lifecycle and UI presentation are separate
  owners. The native service imports no Gobble root/engine package and executes no
  Project Go process on the host. No source application or Run admission is added.
- Main checks the closed artifact schema and Project/Pipeline/job associations,
  endpoint existence, uniqueness and acyclicity before presenting it. The service
  owns protocol/size/count checks and exact source/runtime identity. An incompatible
  runtime payload is not displayed as a valid flow.
- Current canceled/failed/stale candidates do not replace the service's earlier
  successful artifact. Files/configuration are retained with independent hashes;
  the current source is not substituted into an old record.
- Existing single Workspace writer, Agent Chat and file/Run reference machinery
  remain the App foundation. New flow details are local inspection, deliberately
  distinct from a claimed Agent-visible selection until P2A-2.

## Native visual checklist

- [x] Import an owned existing analysis through the native folder chooser.
- [x] Open the flow without a source textbox or a newly created Run.
- [x] See actual input boundaries, five steps and seven connections, including
      parallel alignment/quality paths.
- [x] Use keyboard step activation, inspect declared ports/resources and navigate
      an exact connection.
- [x] Switch Flow and Step list without changing the data.
- [x] Fit the representative flow beside the right Chat and an image in the other Pane.
- [x] Keep draft and other data through a failed check and normal restart.
- [x] Inspect details at a compact native window size.

The final native regression also opens the same pipeline in both Panes, closes
only the duplicate and returns to the prior image. Its result is recorded in
[verification](verification.md).

## Remaining checkpoint boundaries

P2A-2 adds exact artifact/step/port/connection/field references, User attachment,
Agent observations and independent Agent marks. It must connect readable UI
subjects to authoritative artifact metadata without taking User focus/selection.
P2B adds Agent-authored candidates and visual change acceptance. P3–P5 add prepared
execution, Start/Stop and refinement/Resume. The current flow is not execution
authority. General module settings, automatic setup/creation and runtime delivery
are not complete in P2A-1.
