# R4 concept review evidence

Date: 2026-09-08. Subject: saved Notebook reading and its future execution
boundary. Owner/author: Codex in this task. Consumer/decision owner: project owner.
Status: **Recommendation and R4a1 plan ready for owner review; implementation not
started.** This is an author review, not an independent audit or participant study.

## What was reviewed

Current Project/Workspace contracts, file reader, reference session, render receipt
and evidence owners were inspected. The production reader has no Notebook kind;
`.ipynb` currently follows generic text preview. Existing Project registration,
WorkspaceController, captured evidence and provider tool boundaries can be reused
without a second Notebook repository. R4a1 will determine the bounded document
projection and MIME implementation before any production contract is widened.

Primary Jupyter sources and their implications are recorded in
[design.md](../design.md#primary-research-checked-2026-09-08). No dependency was
installed and no Jupyter compatibility or large-data support was inferred from
those documents. Dynamic documentation versions can differ from installed software.

The [generated image](../concept.png), made with the built-in image-generation
tool from [the saved prompt](../concept.prompt.txt), is an illustrative composition.
It is not a screenshot, scientific result, output consistency proof, or approved
implementation. It introduces illustrative content and differs from the exact
sketch in labels and code/result details; the deterministic sketch and design own
the proposed interaction contract. No generated code or table values enter the App.

## Prototype evidence

[Check results](sketch-checks.json) · [Review harness](check-sketch.mjs)

The isolated installed Chrome session used no personal browser profile. The 14
checks cover source attachment, composer focus, frozen evidence while selection
changes, read-only Show, complete Return state, earlier-version fallback, explicit
Refresh, exact demo send, concept B's uncertain connection state, focus/split,
cell-jump focus, keyboard activation and three responsive layouts. No page-script
errors occurred. Tests use explicit preset selection buttons, not a real Notebook
parser or production text/image gesture mapping.

Screenshots were also visually inspected:

- [1440×1000 reading and attachment](reader-1440.png): the Notebook and Chat dominate;
  lower Pane remains useful for a method comparison, with no extra inspector.
- [1280×800](reader-1280.png) and [900×650](reader-900.png): selection and Send remain
  visible; the narrow layout starts with Files collapsed. Chat/Notebook content
  scroll within their areas. Focus can give the Notebook the full central height.
- [640×600 exploratory layout](reader-640.png): included as a prototype stress
  check, not a new production minimum-window promise.
- [Show](show-reference.png) and [earlier-version capture](captured-earlier-version.png):
  state labels and Return stay visible; the reference cannot mutate local selection.
- [Connected alternative](connected-alternative.png): the environment/session
  boundary is explicit. No actual server, execution or save is simulated as success.

Author review found that the first draft left cell-selection buttons active during
temporary Show. They were disabled and the local-selection toolbar hidden while
reviewing a reference; Return restores both. The final harness checks this behavior.
The generated image is broad visual direction; the sketch fixes the synthetic
code/output example and preserves the existing author-mark conventions.

Not checked at this proposal checkpoint: native Electron Notebook rendering,
screen-reader output, production menu/shortcut integration, image region gestures,
150% browser/text scaling, real file decoding, live Agent delivery, kernel execution,
representative user performance and large-file timing/memory. These remain explicit
implementation/qualification work, not successful tests implied by this sketch.

## Seven design activity results

Common record fields for every row: owner/actor is Codex unless specified; subject
is this R4 concept and its bounded handoff; current inputs are the latest user
checkpoint, accepted compact workspace and R3c3 baseline. Trace/route is
design.md → implementation-plan.md → this review and its linked evidence.
Dependencies are existing App selection/evidence/layout owners; no production
implementation is present. A row remains open whenever its stated uncertainty
could change the proposal. Owner acceptance is not representative-user evidence.

| Activity / exact disposition | Method, evidence and current decision | Counterevidence, limitation and reopen condition |
| --- | --- | --- |
| Discovery — **Performed for the current subject** | Source inspection plus six primary Jupyter references; design.md research/ownership sections. Saved reader is a feasible candidate. | No representative research files or installed Jupyter trial. Reopen when real files require active outputs or exceed candidate limits; discovery remains open for R4a1. |
| Problem framing — **Performed for the current subject** | Mapped user requests to exact shared targets, compact comparison and one composer; design.md requirements and plan scenarios. Existing requirements reused, new scope proposed. | Saved-reader-first ordering is not yet owner-accepted. Reopen on owner feedback or evidence that an execution workflow is required first. |
| Concepts — **Performed for the current subject** | A continuous saved reader versus B connected document/session, compared before implementation. design.md matrix and sketch A/B. A recommended. | B may be preferable for editing-heavy research; comparison does not qualify an embedded Jupyter implementation. Decision awaits review. |
| Prototyping — **Performed for the current subject** | Generated direction plus deterministic state/layout sketch; 14 checks and screenshot review above. Reviewable proposal, no production proof. | Preset gestures, synthetic content, untested assistive technology/zoom and no real decoding. Reopen on owner walkthrough, layout failures or R4a1 mechanics that change interaction intent. |
| Representative-user testing — **Performed for the current subject** | Readiness assessment and task protocol only: implementation-plan.md review workflow. Zero participant sessions; evidence is insufficient and this activity remains open. | No usability conclusion or material-choice acceptance is based on the author, automation or project-owner familiarity. Next actor: project owner/research user performs the tasks; record completion, confusion, recovery and conditions before drawing conclusions. |
| Design–implementation collaboration — **Performed for the current subject** | Single-author source/API boundary review; design.md structure and implementation-plan.md staged obligations. Existing state owners retained. | No independent reviewer or runnable Notebook adapter yet. Reopen if qualification needs a different model, source transport, rendering isolation or capture shape. |
| Post-release improvement — **Not applicable with exact reason** | There is no released Notebook implementation or observed use of it, so a post-release decision cannot be made for this artifact. Scope is a pre-implementation proposal. | No release/maintenance claim is made. When a real reader is used, the owner reviews the bounded signals below and explicitly chooses improvement/no change and maintenance disposition. |

The representative-user activity records a failed readiness condition, not a claim
that a participant test was performed. It is intentionally open. The review gate
in this task comes from the user's explicit stage-by-stage instruction.

## Reopenable success measures

At the R4a2/R4a3 review, the project owner and implementer inspect a small scripted
research walkthrough: identify source versus output, attach the intended target,
follow an Agent reference and recover after a source change. Record target
correctness, whether the original draft/view survived, time spent finding the
reference and observed confusion. Faster completion cannot compensate for a wrong
target; fewer clicks cannot justify automatic sending or hidden execution. One
silent retarget, lost draft or false execution claim reopens the design. No
background monitoring or data collection is scheduled by this proposal.

## Product preservation

[Baseline](baseline.json) and [preservation check](preservation.json) cover 554
pre-existing files under app, internal, cmd and .gobbi, excluding dependencies,
build outputs and test output. This checkpoint changes documentation and a local
prototype only. Existing production tests are not rerun or reported as new
Notebook verification. The R3c3 evidence remains the last product verification.
