# P2B-2 author review and handoff

2026-09-12. Author walkthrough of a local HTML prototype, not representative-user
usability testing, actual Agent integration or production release verification.

## Evidence and decisions

The subject is one new-Pipeline creation path in the accepted Project shell. Inputs
are the owner's staged/UI-centered requirements, P2B-1 source, and accepted B design.
The implementing assistant performed the code read and pointer/keyboard walkthrough
through the in-app browser. Source boundaries and decisions are in [design](design.md),
implementation dependencies and verification targets in [plan](plan.md).

| Scenario | Observed outcome / limitation |
| --- | --- |
| Select S02 instead of S01, then use selected file | Existing composer receives S02 context, file size remains associated with the selected file; unsent text retained |
| Send an opted-in illustrative goal | Checking appears with no new Pipeline, then the mock first-version review |
| Select and discuss an addition, then select another node | Attached Trim reference remains Trim while the selected detail changes; removing it preserves data binding |
| Confirm mock adoption | Pipeline count becomes two, separate from the existing example analysis; no Run starts |
| Preview changed input | Adopt button disabled; Review selected file returns to data choice |
| Discard draft | No active draft or new Pipeline; unsent Chat text remains |
| Engine unavailable → Reconnect | Mock readiness returns without source/configuration fields |
| Chat-first alternative → Start pipeline draft | Reaches the same data chooser, not a second wizard or conversation |
| 850 × 900 compact viewport, Enter on Use selected file | Chat becomes visible and focuses the same composer; no document horizontal overflow |
| Keyboard addition → Discuss | Composer focused, exact Check read quality reference shown |
| Default 1280 × 720 visual review | Compact graph and paired detail heading remain together; longer facts scroll inside the detail region |

Corrected during walkthrough: unreadable preview-select text; oversized graph hiding
the paired comparison; inconsistent sample filename/size; attachment removal clearing
data binding; permission event paths bypassing checking-state Send guards; premature
Chat links opening an unchecked review. The prototype still has illustrative fixed
Agent replies, local-only state, and simplified selection/rerender behavior. It is
not evidence of persistence, exact backend identity, keyboard focus continuity after
every rerender, or complete assistive-technology usability.

Screenshots: [Data choice](evidence/data-choice.png),
[First-version B review](evidence/first-version-review.png).
Use the [live sketch](sketch.html?scene=review) for all reachable scene controls.
Some app navigation outside this slice is illustrative or explanatory only.

## Interface lifecycle record

Shared context for every row: actor = implementing assistant; scope = this creation
checkpoint; trace = owner continuation → P2B-1 → this design/plan; no production
release claimed. Owner review is the next route. Reopen when that review changes
scope, actual engine qualification contradicts a depicted state, or implementation
cannot preserve the single composer/reference/storage contracts.

| Activity | Disposition | Inputs, method, evidence and decision | Counterevidence, limits, dependency and route |
| --- | --- | --- | --- |
| Discovery | Performed for the current subject | Read service registration/revisions, engine qualification and App surface assumptions; findings in design | Existing refinement is not whole-creation qualification; route to model/engine parts |
| Requirements | Performed for the current subject | Translate owner direction into explicit draft/adoption/data/reference rules in design | First scope deliberately narrower than general pipeline design; owner reviews scope |
| Alternatives | Performed for the current subject | Compare Project action and Chat-first with different intent/data timing; interactive concept switch | Chat-first is conversationally natural but has new-versus-refine ambiguity; recommend Project action pending owner review |
| Prototyping | Performed for the current subject | Pointer/keyboard walkthrough and default/compact visual inspection above; fix concrete inconsistencies | Mock data and local transitions, no actual engine; route to production integration plan |
| Representative-user testing | Not applicable with exact reason | This checkpoint delivers a proposal for owner review before implementation; no representative participant session was conducted | No usability success/generalization claimed. Arrange user scenario review on implemented slice before treating it as validated |
| Design–implementation handoff | Performed for the current subject | Named owners, proposed commands, failure cases and staged tests in plan | Handoff ready for owner review, production untouched; schema/protocol names frozen in implementation |
| Post-release improvement | Not applicable with exact reason | This creation slice has not been implemented or released | Retain later feedback on discoverability, reference clarity, recovery and input expansion; no telemetry/release conclusions |

Tray/global shortcuts/new windows are not applicable because no such surface is
introduced. Existing Project visual language is reused; no global design-system
replacement or new theme is proposed. Native keyboard/region labels were inspected,
but no screen-reader session or generalized accessibility conformance is claimed.

## Source and verification boundary

Before index updates, all 724 files in the accepted P2B-1 `after.json` still matched
their recorded hashes, with zero drift. Baseline aggregate:
`ec7ad31e21c83e2b8b1919522e80b3f4a3179e5df21ad7d9527ad0a57062130a`.
This checkpoint adds the proposal/sketch/evidence and updates the current navigation
and remaining-work summaries. No production implementation, schema migration,
commit, push or real Project mutation is included. Prototype JavaScript syntax and
local documentation links are checked; P2B-1 production suites are not rerun for a
design-only change. Actual creation verification is explicitly deferred to the plan.
