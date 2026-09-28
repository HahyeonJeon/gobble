# Revised implementation checkpoints

Status: proposed concrete response to the owner's visual-first correction,
2026-09-09. P1 result accepted; the [new screen and contract](design.md) are being
presented before implementing the revised P2. Each part ends with a working
demonstration, source/design review and User approval before the next part.

The old five-group plan is retained as a historical decomposition. The correction
changes dependencies: inspection and shared Plan references must precede visual
change acceptance. It does not authorize an App editor, a second workflow engine,
source writes to real research Projects, or analysis execution during this review.

## Sequence and scope

| Part                               | Observable result                                                                                                                                                                     | Responsibility and stop boundary                                                                                                                                                                           |
| ---------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| P2A-1: real pipeline view          | Open a Project pipeline by its name; see its actual checked steps, pipeline inputs, outputs and connections, with useful empty/error states. Default opening no longer lands on code. | Gobble owns complete inspection facts; service owns retained source/input identity and bounded jobs; App renders. No source editing or Run launch.                                                         |
| P2A-2: discuss the flow            | Select a step, exact connection, port or supported setting; attach it to Chat. Agent observes and points to the same version without taking User selection or focus.                  | Add versioned pipeline references through existing Main policy. Friendly labels and the first supported module settings are checked at their owners. No source application.                                |
| P2B: visual change collaboration   | Ask for a change or describe a new pipeline; Agent prepares bounded source/config, Gobble checks it, and the User reviews the change on the flow before accepting.                    | Service owns candidate/base manifests, comparisons and one managed source writer. App owns review. Includes source-free onboarding/managed registration and restart/conflict recovery. No analysis launch. |
| P3: execution preparation review   | Qualify the exact engine-owned prepared payload and source/input/runtime binding, presented as readable analysis details.                                                             | Completes the execution-identity portion of old stage 3; ordinary Plan JSON is not executable authority. No automatic Start from an accepted change.                                                       |
| P4: Run and Stop                   | Start the exact reviewed preparation with visible target/effects; monitor true execution; Stop targets its actual controller ownership.                                               | Original stage 4 admission, durable operation and reconciliation boundaries remain. No App scheduler or blind retry.                                                                                       |
| P5: failure, refinement and Resume | Select a failed Run step/result, discuss it, review the proposed pipeline change and Gobble-owned reuse result, then explicitly Resume.                                               | Historical Run evidence remains distinct from the new design. Gobble owns recovery/reuse; Project and Chat connect the loop.                                                                               |

P2A-1 and P2A-2 are separate reviewable slices so an incomplete Agent reference path
cannot be advertised as working shared modification. P2B couples the source
lifecycle and visual comparison in one end-to-end authoring outcome. There is no
intermediate milestone whose required acceptance UI is a Go source diff.

## First implementation slice: P2A-1

**Goal:** from an existing supported pipeline, show a truthful, navigable flow in
the central Pane without requiring a Run or showing source as the primary surface.

### Before source changes

- Bind the actual working source to the completed P1 baseline and any subsequent
  approved edits. Preserve existing dirty work and historical schemas.
- Show the final compact flow + selected-details sketch. Record accepted UI
  changes. The User owns the interaction decision, not Go transport details.
- Define one bounded native development scenario: Project, example pipeline,
  declared read/reference inputs and cached supported Gobble runtime. Missing
  setup is a visible outcome, not an ambient install.

### Design and implementation obligations

| Boundary                      | Concrete work                                                                                                                                                                                                                                                       |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Gobble public inspection      | Versioned read result with complete step/port/boundary connections and declared control semantics. Keep public Plan serialization compatible; do not obtain missing edges through path/name heuristics. Add focused engine/CLI qualification for the real consumer. |
| Service source and inspection | Retain exact source/config/input manifest, dispatch a bounded pinned Linux inspection job and store its artifact or structured failure. Go executes only in the evaluator. Handle cancellation/timeout/late results without replacing current identity.             |
| App contracts and Main        | Add closed pipeline resource/view and inspection DTOs; named APIs and response association checks. Keep service process lifetime, artifact resolution and presentation lifetime separate. Main retains its existing durable writer and render acknowledgment rules. |
| Renderer                      | Pipeline view with step/connection navigation, selected details, graph/list adaptation, version/status and source-free error recovery. Default Pipeline navigation opens that view. Existing source reader remains optional.                                        |
| Friendly import               | A chosen supported analysis can be registered/resolved behind “Import pipeline.” If several candidates exist, show meaningful analysis choices. Do not ask the User for a package name, entry function or command. General automatic discovery is not required.     |

Suggested organization follows existing owners: service inspection job/artifact
modules; an active pipeline inspection contract; focused Main resolver; renderer
`views/pipeline` with Flow and Details components. These are responsibility seams,
not instructions to pre-create a plugin framework or an unused service hierarchy.

### Acceptance evidence

- A non-empty supported example produces a real Gobble inspection with no Run.
  Pipeline input edges, branch/parallel dependencies and port identities match its
  engine facts; the App never fabricates them from file names.
- Invalid source, missing inputs/runtime, cancellation and stale/late results have
  explicit states. Existing valid views, selections and draft remain available.
- Changed source creates a new artifact; reopening or restarting does not silently
  relabel old facts. Both Pane slots can show the pipeline without changing other
  data or draft ownership.
- The native service does not import the Gobble engine or evaluate Project Go.
  Inspection resources, output limits and process lifetime are qualified on the
  declared supported runtime.
- Native visual review covers default navigation, representative branching flow,
  readable details, keyboard and compact layout. Tests check identities and effects,
  not pixel positions or implementation-mirroring snapshots.

End by reporting source organization, API boundaries, tests and actual screenshots.
The next owner decision is P2A-2 shared references, not an automatic larger scope.

## Following slice obligations

P2A-2 must carry exact artifact/node/edge/field identity in both User and Agent
observations. Agent marks cannot overwrite User input, camera or local selection.
Old artifact references stay readable; a stale live observation cannot authorize
a new mark on changed facts. Keyboard and the alternate step list expose the same
subjects. Typed settings start with only supported module metadata; unsupported
fields must remain explicit.

P2B adds immutable base/candidate manifests and source application recovery from
the old stage 2, but acceptance is a semantic visual review. Cover added/removed
steps, rewiring, inputs, tool/settings/resource changes, incomplete comparison,
concurrent changes and refreshed proposals. Agent-generated descriptions are
labeled explanations; authoritative facts come from the checked artifacts. New
pipeline creation includes scoped code creation and registration behind the User's
analysis goal and data chooser. Do not add a generic shell or demand user-written Go.

P3–P5 retain the engine preparation/admission, durable command, exact Stop and
candidate-aware Resume boundaries of the old plan. P2A's inspection result does
not by itself qualify privileged execution or whole-engine support.

## Unchanged scope and verification discipline

The current macOS Electron development App and pinned Linux Gobble runtime remain
the supported qualification context. This proposal does not repair the known
native root-engine target limit, the retained packed-runner failures or packaging.
Notebook editing, CSV chart creation, arbitrary graph dragging/rewiring as direct
source mutation, a new authored workflow language and generic extension hosting
remain out of scope.

Production changes require tests at the changed owner boundaries and existing
cross-feature regression checks. This design/prototype turn runs only relevant
prototype/source-preservation checks; P1's 323 unit and 67 Electron results are
historical evidence, not a new test run or qualification of the proposed flow.
