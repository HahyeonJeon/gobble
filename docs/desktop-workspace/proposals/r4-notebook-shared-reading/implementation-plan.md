# R4 Notebook — bounded implementation plan

Status: **R4a1–R4a3 implemented and owner-accepted; R4b investigation complete. Immediate document editing was deferred by the owner on 2026-09-09.** The R4c edit/save-copy proposal is not approved work; kernel integration remains separately proposed. The original stages below are historical decomposition. [Current remaining work](../../remaining-work.md).
[Concept and ownership](design.md) · [Sketch](sketch.html)

## Stages and review boundaries

| Slice                                           | Concrete outcome                                                                                                                                   | Verification and approval checkpoint                                                                                                                              |
| ----------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **R4a1 — Qualification (accepted)**             | Isolated saved Notebook parser/reader prototype proves document limits, cell/output identity, safe representations and exact range/region mapping. | Native fixture matrix, measured budgets, source/reorder/invalid-ID failures and architecture review. Demonstrate results before approving production integration. |
| **R4a2 — User reading and evidence (accepted)** | Registered `.ipynb` opens in a Pane; cell/source/output selection attaches to the existing composer and survives restart as immutable evidence.    | Contract migration, exact capture, external changes, output fallback, keyboard and compact-window tests. Owner review before Agent integration.                   |
| R4a3 — Agent shared reading (current review)    | Scoped observations, authored marks, questions, exact Show/Return and captured fallback.                                                           | Negative visibility/revision/turn tests, native User-state preservation, real Agent delivery and independent architecture review if available. Owner review.      |
| R4b — Live integration investigation            | Compare reusable Jupyter model/editor versus trimmed connected surface with a semantic bridge on one pinned environment.                           | Demonstrate document authority, selection mapping, save conflicts, kernel ownership and disconnect behavior. Select the integration only from observed evidence.  |
| R4c — Editing and execution                     | Explicit edit/save/execute/interrupt/reconnect, against the approved document and environment model.                                               | Source conflict and uncertain-execution recovery, no blind replay, live output revisions and independent tab/kernel lifetimes. Separately approved scope.         |

The owner accepted R4a2 and authorized **R4a3 Agent shared reading**. R4b and later
slices retain their sketch → implementation/test → review → approval boundary.
R4a3 uses Workspace v14, bundle v16 and shared tool input v9; older formats stay frozen.

## R4a1 acceptance subject

Use synthetic Python and R examples, including existing static output, with no
runtime dependency. Record exact fixture bytes/digests and the app/runtime tuple.
No user datasets, production migrations, provider tool changes or Jupyter server.
The qualification may add focused code and a test command under
`app/qualification/notebook/`; it must not widen production APIs by accident.

1. Read the existing 8 MiB registered-file transport and 1 MiB generic-text limit.
   Compare retaining a deliberately smaller first reader with a named Notebook
   source transport. Measure before choosing; do not globally raise text limits.
2. Validate nbformat 4.0–4.5 candidates and project the required fields. Preserve
   source text faithfully; never rewrite the original file or auto-generate IDs.
3. Prove exact targets for cell source, source range, output text range and saved
   image region. Distinguish local ordinal from validated declared cell ID.
4. Qualify restricted Markdown, text/error projection, PNG/JPEG and alternative
   MIME choice. Disable raw HTML, SVG, JavaScript, widgets, remote resources and
   local-file links in the reader. Unsupported output slots remain understandable.
5. Exercise native scrolling, collapsed outputs and image scaling so a ready/visible
   observation cannot include hidden content. Define capture bounds and failure
   states before wiring any Agent tools.
6. Review module/API boundaries and return the smallest supported profile plus
   measurements, remaining limitations and a concrete R4a2 contract sketch.

Candidate budgets for measurement, **not support claims**: source at most 8 MiB;
1,000 cells; 100 outputs per cell; 1 MiB total decoded source/plain text; 4 MP per
decoded output image; at most two interactive readers with bounded on-demand
output decoding. Test limits at, below and above boundaries and measure combined
resident data. Reject or explicitly omit according to the qualified profile.
Do not truncate a target while reporting an exact complete capture. Existing
provider evidence limits stay authoritative (including 768 KiB PNG, 1536-pixel
maximum edge and two images per message); an oversized image capture reports its
limit instead of silently resampling. Decide JSON depth, output counts, decode
timeouts and aggregate memory from the qualification rather than leaving them
unbounded. No generic caching/virtualization framework before measurements.

## Meaningful fixture and scenario matrix

| Risk                                 | Required evidence                                                                                                                                                      |
| ------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Wrong cell after reorder             | Reordered file with valid IDs retains identity but old target remains tied to old revision; no automatic retarget. Missing legacy IDs use revision + ordinal.          |
| Misleading identity                  | Duplicate/invalid IDs, missing required 4.5 IDs, unknown major/minor and malformed JSON return explicit states; file bytes unchanged.                                  |
| Wrong text location                  | String versus string-array source, CRLF, astral Unicode, empty/end-of-line ranges, repeated text in different cells. Verify host and native display agree.             |
| Wrong output                         | Repeated outputs, output replacement, MIME bundles with different fallback types, streams/errors and ANSI controls. Address original index plus chosen representation. |
| Unsafe content / hidden network work | HTML/script/SVG/widget payloads, remote Markdown resources, local-file links and hostile metadata cannot gain host access or trigger network loads.                    |
| Large notebook stalls                | At/above-limit source, deep JSON, many cells/outputs, decoded image dimensions, cancellation and two readers. Record load/capture latency and peak memory.             |
| False visual observation             | Offscreen/folded cell, partially visible source/output, zoom and scroll during capture. Scope and receipt invalidation must match what was actually shown.             |
| Broken User workflow                 | Select → attach → change local selection; inspect reference → Return; source changes; compact window; keyboard and enlarged text. Preserve draft and origin state.     |

R4a2 adds version-frozen schema/migration fixtures and old evidence digest checks.
R4a3 adds same-turn receipt containment, model representation support, late calls,
Project isolation and actual signed-in Agent delivery. Simulated sketch responses
cannot satisfy either production or real-provider acceptance.

## Original concept review workflow

In the sketch, compare A/B, select a source line, attach it, change the selection,
simulate a reply, use Show/Return, then change the source and reopen captured
evidence. Also inspect the unsupported widget and the narrow layout. Prototype
controls are explicitly labelled outside the App frame and use synthetic data.

Owner feedback should resolve whether saved reading is the useful first milestone,
whether source/output labels make the target clear and whether the lower Pane
should stay available for comparison. The current recommendation keeps it flexible.
Approval is requested because the user explicitly asked for a checkpoint before
each new implementation stage, not because of an added permission policy.

## Historical R4a2 result review

R4a1's 27 pure tests and 15 qualification scenarios (13 native Electron) are accepted;
its source and evidence remain frozen. Review [R4a2 production evidence and the next
ownership sketch](../../stages/r4a2-user-notebook.md) before authorizing R4a3.
Notebook tools and real Agent Notebook observations are not part of R4a2.
