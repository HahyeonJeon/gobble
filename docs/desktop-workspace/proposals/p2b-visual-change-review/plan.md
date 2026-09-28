# P2B implementation plan

2026-09-09. The owner accepted B in alternatives-v2 and authorized P2B-1.
[The implementation record](../../stages/p2b1-visual-proposals/README.md) is current. Each part ends with an actual result,
tests and an owner checkpoint before the next part starts.

## P2B-1 — Refine an existing Pipeline through Chat

**User outcome:** ask the Agent to adjust trimming and add a quality-report branch,
inspect checked changes in the shared flow, discuss an exact change and adopt the
reviewed version. The User does not edit or review Go. The existing current
Pipeline remains usable through failure/restart. No analysis starts.

This first part supports one participating Pipeline per managed Project source
revision. Qualify Trim Galore's supported settings and a FastQC branch; preserve
unchanged other steps by complete checked identity. Changed/added unsupported
behavior, unqualified configuration/runtime changes and shared-Pipeline source
publication remain visible refusals. First adoption from an import includes the
managed-copy handoff described in [design.md](design.md).

Implementation order follows dependencies, without treating intermediate APIs as
a completed feature:

1. **Gobble facts and review contract.** Inventory all execution-bearing fields;
   implement versioned normalized review facts, exact differences and explicit
   coverage gaps. Prove supported recipe mappings against actual constructors.
   Version validators/fixtures together. If complete mapping cannot be proved,
   keep adoption disabled and return the concrete limitation for review.
2. **Service revisions and adoption.** Retain bounded scoped source submissions;
   check immutable candidate bytes with the existing runtime owner; keep proposal
   and check-attempt identities separate. Add expected-head publication and durable
   receipt under the catalog writer. Qualify first-import handoff and recoverable
   storage before exposing Use.
3. **Main policy and Agent tools.** Add a turn-scoped proposal capability using the
   existing Agent transport. Reuse the workspace/evidence owner for exact comparison
   references and independent Agent marks. No generic shell, host filesystem or
   Agent adoption command.
4. **React comparison View.** Reuse the flow, routing, fit/list and reference
   primitives. Implement B: one Proposed flow with numbered changes and paired
   Current/Proposed detail below, plus Discuss this change and Adopt proposal actions. Preserve existing Chat, draft, pinned Panes and
   historical views. Keep authored values in English product UI.
5. **End-to-end qualification and result review.** Use an owned disposable Project
   and profile, run the actual checker and signed-in Agent against that fixture,
   capture the native UI and demonstrate restart. Report the tested scope and any
   refusal. Present the result before moving to P2B-2.

### Evidence required

| Layer              | Meaningful acceptance cases                                                                                                                                                                                                                    |
| ------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Gobble             | Real quality/length changes and added branch; same visual metadata but altered command/script/env/params; unknown fields; input/control/resource changes; edge reorder/rename/multiplicity; deterministic identity; no analysis task execution |
| Service            | Exact manifest/hash/base checks, containment and size bounds, cancelled/late checks, incomplete comparison refusal, cross-Project/turn capability refusal, shared-source refusal, moved head, repeated operation and changed-body rejection    |
| Persistence        | Crash before candidate seal, after seal/before publication, after catalog publication/before response; corruption/missing bytes; disk write failure; restart reads a recorded outcome without repeating adoption; referenced history retained  |
| Main/contracts     | Legacy references/evidence remain readable; candidate/base/side mismatch rejected; old capture immutable; Agent cannot adopt; scope revocation; stale observation and draft preservation                                                       |
| Native UI          | Pointer and keyboard/list routes; visible default buttons; exact changed setting and branch; old proposal reference after revision; incomplete/stale/offline outcomes; compact view and pinned Pane preservation; no Run created               |
| Actual integration | Agent receives exact UI evidence, authors allowed source, checker produces real comparison, User action adopts same bytes, restart restores identity; fixture source/profile ownership recorded                                                |

Run focused tests while implementing, then the applicable App contract/type/schema,
native service and Electron suites once the slice is integrated. Run affected Gobble
tests in the repository's supported Linux runtime; separately report the known
native macOS engine constraint. Do not convert historical P2A-2 counts into new
evidence or claim that the existing packed-runner failures are resolved.

## P2B-2 — Create a Pipeline from an analysis goal

After P2B-1 result approval, show a concrete creation sketch. The existing composer
collects the analysis goal and selected Project data; the Agent creates the bounded
source bundle and Pipeline binding in a draft. Gobble checks it; all added steps,
inputs and settings appear in the same review. The User's Use action atomically
registers the Pipeline and publishes its first managed version. Cancellation must
not leave a phantom current Pipeline. No Go template preparation is required from
the User. Reuse P2B-1 contracts instead of building a second proposal system.

Start with the qualified module scope from P2B-1 and exact declared sample/data
references. Broader recipes or shared-source adoption need their own demonstrated
coverage. User validation of data picking and first-version ownership language is
still required; the current sketch is an initial entry-flow hypothesis only.

## Following checkpoints

P3 prepares exact engine-owned execution input and readable launch review; P4 adds
Start/Stop with durable execution reconciliation; P5 connects failure evidence,
Agent refinement and engine-owned reuse/Resume. Managed source and Project data
roots must be qualified together before P3 can launch from an accepted version.

Document editing, Notebook kernels, CSV chart generation, generic plugin/DSL
infrastructure, direct source synchronization, Git integration and packaging are
outside these P2B parts. This plan does not authorize commits, publication or real
research Project mutation.
