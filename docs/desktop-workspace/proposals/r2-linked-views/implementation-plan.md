# R2 — Implementation checkpoints and acceptance scenarios

Status: **R2a–R2c implemented, 2026-09-08; R2c completion accepted by the user.** [R2c result and verification](../../stages/r2c-evidence-and-agent.md) includes the actual signed-in provider trial. R3 remains a separate checkpoint.

## Goal

Complete a real CSV table → scatter selection → Project chat → Agent reference → reveal loop with exact source membership, independent User/Agent marks and clear version boundaries. See [design and ownership](design.md).

## Sequential delivery

| Checkpoint                            | Implement                                                                                                                                                                                                                                        | Required evidence before moving on                                                                                                                                                                                                                                                                                   |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| R2a — Contracts and chart integration | Freeze table-to-scatter numeric semantics; explicit ViewLink/single membership owner; typed reference presentation; versioned render acknowledgment; v4 migration. Prove a locally bundled Plotly SVG scatter in the existing Electron renderer. | Contract/migration negative cases, existing v3 compatibility, 500-row chart events with exact row keys, stale render rejection, CSP and lifecycle checks, measured dependency cost. Summarize selected implementation and obtain approval for R2b.                                                                   |
| R2b — Linked User interaction         | Real CSV open-linked-plot flow; compatible numeric columns; controlled table/plot selection; shared filter; local sort, axes, pan/zoom; one Discuss action; close/reopen behavior.                                                               | Electron scenarios show actual source rows, finite-value exclusions and truncated preview scope; sorting/filtering/zooming cannot change membership; focus and pane protections hold. User reviews interaction before R2c.                                                                                           |
| R2c — Evidence and Agent round trip   | Freeze plot context with draft targets; semantic selected-row evidence; shared tools list/open/observe/point with presentation revisions; explicit reference view/return; historical state.                                                      | The exact selected IDs/values reach the intended Agent; an actual provider response points to a related exact row and preserves User selection. No automatic turn from selection/attachment; source change blocks stale draft Send. Full affected checks and clean source/owner review before the next viewer slice. |

Do not begin Notebook, genome, single-cell or external MCP work inside R2. A successful chart-library spike is not the completed collaboration loop. Reuse the existing desktop profile isolation and avoid running demonstrations against user research files.

## Meaningful acceptance scenarios

| Scenario                                                              | Required result                                                                                                               | Primary verification                                             |
| --------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- |
| Duplicate sample labels and reordered rows                            | Source row keys select distinct members; labels and visual order never become identity.                                       | Pure projection/contract tests + Electron interaction.           |
| Empty, non-finite, malformed and zero-valued coordinates              | Empty/malformed values are excluded from plotting, zero is valid; raw values preserved; visible count explains exclusions.    | Shared numeric parser fixtures.                                  |
| Single-point, empty and constant-axis data                            | Useful empty state or padded valid domain; no NaN range, exception or fabricated point.                                       | Projection tests and renderer scenarios.                         |
| Select S03/S05, sort, filter them out, pan and zoom                   | Selection remains S03/S05; hidden count is honest; returning to the view reveals the same members.                            | Actual pointer/checkbox Electron scenarios.                      |
| Complete a box, then change data                                      | Exact resolved member list stays bound to its original revision; box does not become a live predicate.                        | Revision race tests.                                             |
| Old plot selection event arrives after axes/source switch             | Host rejects old render/spec/view-state acknowledgment; no mutation of current selection.                                     | Negative contract/host tests.                                    |
| Source changes between linked loads                                   | Views are not reported as coherently linked until they use one snapshot/revision.                                             | Controlled reader race test.                                     |
| Discuss, then change selection/axes/filter                            | First attachment keeps its original target and context; a second attachment is independent.                                   | Evidence tests + composer scenario.                              |
| Source changes after attaching or preparing evidence                  | Outdated state is shown and acceptance is blocked; no silently refreshed selection.                                           | Existing prepare/accept race tests extended for scatter context. |
| Selected evidence exceeds the bounded payload                         | Exact requested/returned scope is explicit; full-set claim is blocked until the selection fits.                               | Evidence size boundary tests.                                    |
| Agent points to a filtered-out sample                                 | User selection and view stay unchanged until explicit Show; reference view reveals the target; Return restores base settings. | Shared-tools tests + Electron scenario.                          |
| New Agent message while an earlier reference view is open             | New message cannot replace the active reference view; each Show button retains its own target/context.                        | Timeline/reference lifecycle scenario.                           |
| Close/move/duplicate the original Surface                             | Durable references resolve independently of origin; no accidental links to unrelated duplicate Surfaces.                      | Workspace lifecycle and resolver tests.                          |
| Reopen v1/v2/v3/v4 workspace with old selections and sent evidence    | Preserve meaning and evidence hashes; old Surfaces stay unlinked; original file backup is retained.                           | Migration fixtures and restart test.                             |
| Agent tries foreign Project, unknown row or incompatible presentation | Host rejects it through the same policy used by User actions.                                                                 | Negative host/tool tests.                                        |
| Narrow window, keyboard-only operation, light/dark theme              | Essential controls visible without hover; table provides selection alternative; composer and pane focus preserved.            | Visual review at desktop sizes and narrow fallback.              |
| Renderer failure/unmount during chart work                            | Other Surface/chat stays usable; callback cannot update a disposed instance or acknowledge a stale render.                    | Lifecycle/error scenario.                                        |

## Review and test policy

Keep each implementation change within a named owner. Review exported APIs and state placement before adding new modules. Avoid tests that merely repeat a mapper: test meaningful identity, lifecycle, numeric, size and version failure cases instead.

Run formatting, type/schema validation and focused tests as scope grows. Run the full relevant app checks at each completed implementation checkpoint. Repeat broader tests only after changes or unresolved failures justify it. Existing engine issues remain outside this UI task.

The design sketch is tested separately and has no real IPC/provider/storage. Its synthetic Agent is explicitly labelled. A live R2c trial must record what was delivered and returned, and be performed within the user's authorized provider scope; do not count a scripted answer as live integration evidence.

Each checkpoint ends with the concrete behavior demonstrated, changed owners/files, tests, limits and a user approval request for the next checkpoint, following the user's requested workflow.

## Current checkpoint after R2c

The full R2 loop now includes frozen semantic evidence, explicit Agent observation
scope, current-receipt pointers, protected linked plot opening, temporary Show/Return,
separate authored marks and exact-source failure behavior. The actual signed-in
provider trial returned correct selected values and a distinct related row without
changing User membership. See the R2c record for construction, visual and runtime
proof and remaining preview limits.

R3 implementation has not begun. First choose one bounded research-navigation view family and define
its resource, presentation, selector, provenance and ownership contracts. Show that
sketch before implementation; retain a separate approval checkpoint for each family.

## Next approved design checkpoint

The user accepted R2c completion and requested the next scope/ownership sketch on
2026-09-08. [R3a Run navigation](../r3a-run-navigation/design.md) now proposes the
first bounded R3 family. [R3a checkpoints](../r3a-run-navigation/implementation-plan.md)
start with contracts and observed evidence, subject to the new design approval.
