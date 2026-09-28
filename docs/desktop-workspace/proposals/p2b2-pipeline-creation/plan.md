# P2B-2 implementation sequence

Proposed 2026-09-12; implementation begins after the owner reviews this sketch and
design. Goal: create one new, checked Pipeline through Project data and Agent Chat,
review its complete first version in B, adopt it, then reopen it after restart.
The flow must work in a Project that already contains another unrelated Pipeline.

## Part 1 — model, ownership and compatibility

Add explicit creation-draft/candidate records and a draft View subject. Introduce
imported/managed Pipeline origins and migrate catalog v3 Project revision pointers
to per-Pipeline ownership without changing existing adopted source or receipts.
Extract shared bundle sealing/receipt helpers only where creation and refinement
have the same invariants. Keep commands for creating, updating, submitting and
adopting a draft distinct from existing refinement commands.

Proposed API shapes: createDraft(project, intent), updateDraft(draft, expectedGeneration,
inputObservation), submitCandidate(draft, expectedGeneration, allowedEdits),
getCandidate(candidate), discardDraft(draft, expectedGeneration),
adoptCreation(draft, candidate, artifact, requestId). Service validates Project and
identity on every call. User adoption remains separate from Agent authoring tools.
Exact wire names are frozen during implementation; do not expose arbitrary paths
or generic shell commands as a shortcut.

Tests: catalog migration with old receipts; two Pipelines in one Project; wrong
Project/draft/candidate rejection; imported shared-source restrictions preserved;
old Workspace/backup restoration. Version changes travel together across contracts,
service, IPC, persistence and shared tools. Tentative next versions are workspace18,
bundle21, catalog4 and shared toolset12; confirm current baseline before assigning.

## Part 2 — Gobble-owned creation check and service storage

Provide the read-only bounded scaffold export, service-profile runtime binding and
managed bundle construction. Implement a dedicated whole-creation qualifier for the
single-end Trim Galore → FastQC scope. Reconstruct/compare complete supported tasks,
reject hidden or unrepresented behavior, and bind input descriptors and all topology
facts to the resulting artifact. Reuse the retained-source evaluator and job limits.

Tests: full canonical graph accepted; changed command/env/control/resource or hidden
step rejected; wrong input/edge/tool, extra file, path escape and setup modification
rejected; unavailable engine; metadata drift; cancellation/generation race; source
sealing; evaluator cannot access original research data. Exercise the actual pinned
Gobble runtime, not only a stub producing expected JSON.

## Part 3 — shared UI and Agent integration

Add the small Project action, separate draft navigation and central data chooser.
Connect to existing Chat context and per-message authoring policy. Extend B details
with a no-current creation base and exact addition references. Preserve input binding
when removing discussion attachments, pinned Views, unsent text and independent
User/Agent attention. Expose readable unsupported/check-failed/reconnect states.

Tests: new-versus-refine scope, addressed Agent and opt-in; data selection through
Chat; selected addition remains fixed in sent evidence despite later UI selection;
Agent points to the same checked addition; keyboard/focus; compact workspace/chat
switching; failed/changed/closed drafts retain correct state. Use actual Electron
screens with the implementation, not prototype screenshots, for result evidence.

## Part 4 — exact adoption and restart

Atomically register Pipeline + first Current + receipt + draft mapping. Preserve
history and imported source. Reconcile retries and interrupted saves through the
same authoritative catalog path as P2B-1. Adoption must not start a Run.

Tests: repeat request creates one Pipeline; simultaneous adoption chooses one result;
data/runtime drift refuses adoption; crash before/after durable commit; restart after
discard/check cancellation/adoption; historical reference resolution; an existing
unrelated Pipeline's Current remains identical. Finish with an actual signed-in
Agent proposing source, Gobble checking it, User-equivalent explicit adoption in a
disposable test Project and restart verification.

## File responsibility plan

| Owner | Existing code to extend / proposed focused files |
| --- | --- |
| Gobble | `internal/engine/pipeline_review.go`, `internal/pipelinereview/`; focused creation qualifier and scaffold export; retain refinement API semantics |
| App service | `pipelines.go`, catalog migration, `pipeline_source.go`/runtime helpers; focused `pipeline_creation.go`, creation source/check and adoption files rather than growing HTTP routing into lifecycle logic |
| Contracts | `app/contracts/src/` draft types, origin union, no-current reference and exact command/result validation |
| Electron Main | Existing Pipeline review host/shared tool adapter and workspace writer; a small creation coordinator delegates storage/checking to the service |
| React | Feature-owned creation chooser/status components; reuse flow, B comparison and Chat components; presentation contains no source generation or semantic qualification |

Do not mirror one large controller with dozens of forwarding classes. Separate a
file when it owns a distinct invariant or rendering concern; keep trivial helpers
near their callers. Review dependency direction and failure handling alongside each
part, then run focused checks. At the full slice result, run appropriate service
race/vet checks, App type/build/unit and affected Electron scenarios. Broaden only
for schema migration and shared-host regressions that this work actually touches.

Each part starts with a brief diagram/scope update; results and any material design
change are shown before proceeding, following the owner's staged review request.
The accepted P2B-1 artifacts remain historical and are not rewritten to include this
new stage. Record a fresh implementation baseline before product edits.

## Completion and later work

Done means the User can choose a Project file, describe an analysis without seeing
Go configuration, review actual Gobble-qualified UI facts, discuss an exact addition,
adopt once and recover the same Pipeline after restart. Limitations must be visible.

Next is P3: engine-owned executable preparation for the adopted source plus original
data and runtime identity. Adoption of this design is not permission to implement
P3, Run controls, packaging or wider assay authoring.
