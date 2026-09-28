# P2A-2 — Implementation review

Reviewer: implementing assistant. The owner reviews the working result before
P2B. This is an in-session source and native interaction review, not an independent
review or release approval.

## Ownership and organization

- `inspection_settings.go` holds the small Gobble metadata type, cloning and
  validation. TaskSpec/Graph/Compose preserve ownership; inspection emits v2.
  Trim Galore derives the two declared fields from the same validated Options as
  its command. No Plan or engine execution serialization changes.
- `contracts/src/pipeline-reference.ts` defines exact addresses, subject projection
  and capture validation. Active bundle v19/Workspace v16/shared toolset v10 add
  pipeline support; v15 storage and v9 tool inputs are frozen for old readers.
- Main's `evidence/pipeline-evidence.ts` and
  `shared-context/pipeline-observation.ts` adapt pipeline subjects into the existing
  evidence and observed-read owners. They do not own a second store, writer or IPC.
- `PipelineSurface` coordinates local selection, attachment and independently
  authored references. `PipelineDetails` presents semantic facts; `PipelinePorts`
  renders directional selection buttons. `flow-selection` translates between
  renderer and contract addresses; layout/routing remain presentation-only.
- Surface-local `flow-navigation` preserves graph/list/zoom/scroll across temporary
  reference views and loads. It is neither durable Project state nor an Agent API.
  Selection remains in the existing Workspace writer, not in this navigation object.
- `PipelineEvidence` presents retained facts in existing evidence and reference UI.
  Agent policy and tool descriptions use the exact observed target/receipt rules.

## Review resolutions

The observation revision and capture identity serve different lifetimes: the
former hashes the loaded inspection state and expires on refresh, while the latter
uses the immutable artifact plus source identity. No stale receipt becomes current
merely because a previous checked artifact is still displayed during a re-check.

Port addresses include direction; edges include their own ID and both endpoints.
Parallel connections and same-named input/output ports cannot be conflated. Setting
keys are step-local, and a step's metadata does not authorize a mark on an omitted
setting. Observation output is bounded and explicitly reports omissions. A smaller
subject can be requested when a whole step exceeds the capture/read limit.

Agent marks and User selection have separate visual state. Show/Return previously
risked losing the graph camera through a temporary component unmount; a small
Surface-local navigation owner now preserves it without adding workspace state.
The compact native scenario explicitly switches Workspace/Chat, matching the
existing responsive interaction rather than targeting a hidden button.

A final actual-Agent screen exposed a selection timing problem: selection clicks
were silently ignored while an attachment was pending. Selection now uses the
existing ordered command queue during capture; capture retains its original target.
The completion label names that captured subject, so it cannot be mistaken for a
newly selected port. The native scenario changes selection immediately after Add
to message and still expects the original 25 Phred evidence. The actual-Agent
script now asserts the visible output-port details before taking its state baseline.

Saved schema/storage contracts and earlier stage evidence are verified against
the entry inventory. v1 checked artifacts remain readable without invented settings.
The new module example uses Project-relative declared inputs, a valid internal
pipeline name and an owned runtime lock. Qualification creates no Run.

## Deliberate limits

Only integer metadata required by the first supported module is added. There is
no speculative generic form language, command parser, schema-to-editor framework,
brand inference, automatic source remapping or runtime status on a design card.
Module metadata is an author's assertion; arbitrary Project code may describe
itself incorrectly. Full scientific validation and source adoption belong to later
work. Exact native/Agent and whole-App evidence is listed in verification.md.

The final visual pass also removes an unused hidden-base CSS rule and bounds the
Agent badge to its card for long/multiple names. Full author names remain in Chat
and the reference strip. This is a presentation-only correction; its final native
screenshot/build follows the full functional suite. The existing Chat text renderer
can display Markdown punctuation in Agent responses; that earlier presentation
limitation is recorded for a future focused Chat polish, outside this part.
