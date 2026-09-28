# P2B-2 — Create a Pipeline through the shared view

Status: design checkpoint, 2026-09-12. P2B-1 result accepted; the owner authorized
this next design. Production implementation awaits review of this concrete slice.

**Recommended: Project → New pipeline → select data → describe the goal in the
existing Chat → Agent proposal → Gobble check → B review → Adopt as new pipeline.**

[Interactive sketch](sketch.html) · [First-version review](sketch.html?scene=review) ·
[Alternative Chat entry](sketch.html?concept=chat) · [Design and ownership](design.md) ·
[Implementation sequence](plan.md) · [Author review](review.md)

The sketch is an interactive illustration with fixed sample data. It does not
connect to an Agent, process reads, persist drafts, or create real Pipelines. Use
the Preview menu for checking, changed-file, unavailable-engine and adopted states.
The first proposed scope is a single explicitly single-end FASTQ Project file,
Trim Galore and FastQC. Quality 25 / length 40 are example user requests, not defaults
or scientific recommendations. Broader analysis support follows separately.

The accepted B comparison is preserved: numbered additions in the flow, selected
facts below, exact references in the same Chat. For creation, Current explicitly
says **No current version**. A creation draft becomes a registered Pipeline only
when the User adopts its checked first version. Existing pipelines remain separate.

## Decisions for review

1. Start from a small Project action; keep the existing composer as the only place
   to describe the goal and request changes. Chat-first remains a later entry option.
2. Separate creation drafts from registered Pipelines; retain drafts on close and
   support explicit discard without leaving an empty Pipeline.
3. First implement one complete creation path with clear input/settings/output
   facts. Source, dependencies and engine wiring remain behind the UI.
4. Complete creation and adoption before adding execution preparation and Run controls.

No product source was changed for this checkpoint. See the plan for required
schema migration, full-graph qualification and actual integration verification.
