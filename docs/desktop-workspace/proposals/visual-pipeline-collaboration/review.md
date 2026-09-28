# Visual design review record

Date: 2026-09-09. Subject: [the proposed flow UI](design.md) and isolated
[interactive sketch](sketch.html). Reviewer: implementation author, Codex. This
is a design/prototype review, not independent product testing or new Electron
implementation evidence. The visual decision remains open for the owner's review.

## Evidence and actual checks

The author read P1 registration/navigation, Gobble Plan/CLI encoding, module display
metadata, Trim Galore option handling and existing Run-reference boundaries.
The [design](design.md#inspection-contract-gaps-found-in-the-code) records concrete
gaps. The public sources and their date/relevance limits are linked in that same
document. A generated bitmap and a local browser prototype were created; no
production dependency was installed and no live Agent/analysis was invoked.

The sketch was operated through the browser UI at its default 1280 × 720 viewport
and at 900 × 700. The temporary viewport override was reset. Screenshots were
visually inspected in the tool output; the retained bitmap is the separate generated
concept, not a screenshot of a completed product.

| Prototype action                                      | Observed result                                                                                                                                                |
| ----------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Select the Trim → Quality check connection            | Details identify both endpoints and trimmed reads; no claim that alignment consumes a quality report.                                                          |
| Discuss the connection, then switch to Current        | The original Proposed reference remains attached and the typed draft remains unchanged. Current displays quality 20 while Proposed shows 20 → 25.              |
| Show the Agent's proposed Quality check               | The proposed flow opens with that step selected; existing draft/reference remain. This is a scripted reference link, not a live Agent call.                    |
| Select the Pipeline changed state                     | Accept changes is disabled and the stale-base explanation is visible.                                                                                          |
| Discuss a retained earlier reference in that state    | A local preview message carries Earlier version. Staleness prevents applying the old proposal, not talking about retained history. No network request is sent. |
| Inspect Accept changes                                | Confirmation names the two changes and says adopting a pipeline does not start an analysis. The author closed this dialog without testing source application.  |
| Press Enter on Quality threshold, then Discuss        | The input field gets a specific Quality threshold · Current reference; one composer remains.                                                                   |
| Change to a narrow viewport and open Chat             | The same reference and composer remain available through Workspace/Chat switching, without a third permanent detail column.                                    |
| Return to default size and reload the isolated sketch | The default proposed-flow example is ready for owner review; the tab is retained as a deliverable.                                                             |

Visual review found the initial minimum-height layout could overlap the detail area
and preview footer at the default window height. The work area now permits its
internal graph to scroll, and the example's vertical spacing fits the visible
canvas. The header, detail fields and composer were checked again after that fix.

The prototype initially blocked discussion when the pipeline changed. Review
corrected that behavior: retained earlier facts remain discussable with an explicit
version label, while source adoption needs an updated comparison. The final UI
path above was checked after this correction.

### Scope limits

Only one example connection and a small set of fields are interactive. The sketch
does not implement full graph keyboard navigation, actual render receipts, engine
checks, semantic difference coverage, persistence, source application, multi-select,
group expansion, Run execution or a full accessibility audit. Some surrounding
navigation is illustrative. It is adequate to discuss hierarchy and the chosen
selection/Chat flow, not to prove production correctness or representative-user
understanding. No full App test rerun was needed because App/engine source did not
change; P1's test counts stay historical.

## Design activity dispositions

Common owner/trace for the following seven results: Codex authors this proposal,
the User owns concept acceptance, implementation owners follow [plan.md](plan.md).
Inputs are the 2026-09-09 user correction, the exact P1 baseline, sources cited in
design.md, and the current generated concept/sketch. Every decision is limited to
this proposal. Missing usability evidence is not inferred from test automation.

| Activity and named result                                                                | Disposition                       | Method, decision state, limitation and reopen route                                                                                                                                                                                                                                                           |
| ---------------------------------------------------------------------------------------- | --------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Discovery — Discovery evidence reviewed                                                  | Performed for the current subject | Traced local Plan/display/source code and primary workflow prior art. Exact metadata gaps are recorded in design.md. Scientific interpretation and user comprehension remain uncertain; reopen when a real pipeline cannot be faithfully projected.                                                           |
| Framing — Design requirements accepted for the current subject                           | Performed for the current subject | User's no-Go, visual-collaboration requirement is explicit. Draft/code review is no longer a required User step. Screen details remain proposed; send contradictions back to the same owner review.                                                                                                           |
| Concepts — Concept decision recorded                                                     | Performed for the current subject | Compared a branching flow with inline details against a guided step sequence. Flow is recommended, with an equivalent list for access/density. It is not a measured usability conclusion; reopen if branch or change comprehension fails.                                                                     |
| Prototyping — Prototype evidence reviewed                                                | Performed for the current subject | Generated hierarchy concept plus UI-operated selection/version/state sketch. Layout overlap and stale-history discussion were corrected. Engine/Agent integration and full modalities are absent; the prototype is not a finished-product proof.                                                              |
| Representative-user testing — Test evidence reviewed                                     | Reused current evidence           | Only the owner's current product expectation and earlier compact-UI feedback are reused, as requirements evidence. They do not establish ability to use this new screen. This result remains open: owner walkthrough and later unfamiliar-user scenarios are required before claiming no-code usability.      |
| Collaboration — Design and implementation obligations reconciled for the current subject | Performed for the current subject | Read the actual owners, moved inspection earlier, and assigned complete relationships, typed settings, comparison coverage and reference identity to appropriate code boundaries. Implementation qualification remains open until P2A/P2B; no contradictory prototype behavior becomes an implicit contract.  |
| Post-release — Post-release design review closed                                         | Not applicable with exact reason  | This capability has not shipped, and there is no population or release telemetry to review. No measured outcome or maintenance conclusion can be produced for it. The future bounded review measures below are handed to the implementation owner; production failure or a first pilot reopens this activity. |

Identity follows the latest explicit user requirement and existing project concept
documents, then the actual P1 layout and current App color/type tokens. The new
brief does not create a project-wide DESIGN.md or replace the existing design system.

## User scenarios and implementation checklist

These are unchecked evaluation tasks for a later working slice; the browser checks
above do not complete them. Use a researcher unfamiliar with Go/Gobble for the
source-free journey. Record the actual subject, participant, task and input modality.

- [ ] Open/import an analysis and explain the main flow without opening source or
      entering a programming term. If setup fails, identify the needed input/action.
- [ ] Point to an input, an exact directed connection and a setting; the recipient
      identifies the same version and object. A repeated display name cannot confuse it.
- [ ] Ask for a new quality-check branch and identify what remains unchanged and
      what data feeds alignment. Extra geometry must not imply extra dependencies.
- [ ] Explain a proposed numeric/tool/input change from its visual review. Identify
      any effects the comparison cannot summarize; do not require Go review as fallback.
- [ ] Read an Agent mark, show it and return while preserving a draft and the other
      Pane. Repeat in a compact window and using keyboard/assistive navigation.
- [ ] Reopen a previous reference after another proposal. Discuss the earlier
      version, while an attempt to apply stale effects requires an updated review.
- [ ] Distinguish accepting a pipeline from running it. No generic reply or Agent
      content grants execution authority.
- [ ] Inspect the same logical step in a Run while recognizing its execution
      identity, actual sample/attempt and historical status.

Measure correct target identification, comprehension of changed behavior and
recovery without code. A faster click-through or more accepted proposals is not a
success metric by itself. Guardrails are zero silently misbound references, no
unreviewed effects, no hidden incomplete comparisons and preserved User drafts.
The implementation owner reviews bounded pilot observations with the User at each
checkpoint; a wrong target, mistaken execution intent or required source reading
reopens the responsible design choice. No background telemetry is enabled here.
