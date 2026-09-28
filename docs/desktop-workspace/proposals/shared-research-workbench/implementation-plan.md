# Shared research workbench — Proposed implementation plan

Status: **R1, retained R2 table communication, R3a Run/log navigation, R3b dependency navigation, R3c PDF reading and R4a saved Notebook shared reading are implemented and owner-accepted. R4b investigation is complete. The owner deferred immediate Notebook editing on 2026-09-09.** Historical CSV chart features were retired. Production is Workspace v14 / bundle v16 / shared tool input v9. R4c editing/save-copy is deferred; kernel integration, remaining R3 viewers and R5–R6 are proposals requiring separate scope decisions. Stage 7 local Mac packaging is unstarted. [Remaining work and current decision](../../remaining-work.md).

## Goal and success condition

Make it possible for a User to select an exact research target in one central view, discuss it with an Agent, and follow the Agent's response back to the same or a related view without losing identity, version, local work or context.

The implemented shared-reference loop remains the foundation. As of 2026-09-08, the user excludes App-generated CSV charts. New work follows table → chat → exact reference and Run → task/attempt/log → chat; viewing scientific results does not authorize a chart builder.

## Checkpoints

| Slice                               | Implement / investigate                                                                                                                                                                                                                   | Ownership and design material before code                                                                                             | Completion evidence                                                                                                                                                                           | Next-step gate                                                         |
| ----------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| R1: Durable targets and evidence    | Extract selector/target contracts from `context.ts`; optional origin Surface; explicit units/version; resolver states; migrate existing saved selections, marks and attachments. Retain current text/table/image behavior.                | Target vs local selection vs capture diagram; schema/migration examples; decide source/view revision semantics.                       | Old workspace reopens unchanged in meaning; closed/moved/duplicated Surface does not break a captured reference; stale target never points to different data.                                 | Demonstrate migration/reveal/error states and receive user approval.   |
| R2: CSV table communication         | Preserve tables, revision-scoped row/column selection and immutable discussion evidence. CSV chart creation/rendering is retired by the user on 2026-09-08; archived R2 material is compatibility evidence only.                          | Existing selection and capture owners; no new chart workflow.                                                                         | Table selection/attachment, chart-request rejection and historical evidence preservation.                                                                                                     | R3b1 review.                                                           |
| R3: Real research navigation        | Better Run/task/attempt selection, graph/Plan distinction, bounded logs, file/PDF/report reference; progressively add missing file formats and MultiQC display/data bridge. Split into bounded per-view deliveries, not one large change. | Per-view capability matrix and prerequisite definitions; graph/data owners; HTML isolation diagram.                                   | A failed task can be discussed with the exact attempt/log reference; report capability limitations are visible; source provenance is retained.                                                | Review each new view family before continuing.                         |
| R4: Notebook working session        | R4a file/cell/output view and references; R4b Jupyter integration spike choosing component reuse or trimmed external view; R4c editing/execute/interrupt/reconnect with authoritative document/session bridge.                            | Notebook document ↔ Surface ↔ session ↔ kernel diagram; edit/conflict and uncertain-execution policies; actual environment selection. | Cell identity survives movement; outputs remain versioned; interrupted/disconnected execution does not duplicate; tab close is independent of kernel lifetime; source changes are reviewable. | Approve each sub-slice and its live demonstration.                     |
| R5: Scientific domain adapters      | Choose **one** initial track: genome browser or single-cell/spatial. Genomics: exact assembly/track/locus/feature reference. Single-cell: dataset/cell set/feature/spatial transform and subset export.                                   | Real dataset and preparation contract; identity/coordinate mapping diagram; compare candidate libraries on one fixture.               | UI/Agent agree on exact locus or cell membership; missing indices/conversion requirements are explicit; large-data limits measured.                                                           | User accepts the domain workflow before broadening coverage.           |
| R6: Interoperability and extensions | Optional Workspace MCP server, independently MCP Apps host; version negotiation, per-Project/Agent capability scopes; extension manifest and lifecycle.                                                                                   | Two-direction transport diagram; supported protocol/adapter tuple; failure/recovery and migration contracts.                          | Another supported client can read/reveal/point using the same semantics; unsupported tools/views fail explicitly; no cross-Project access.                                                    | Review compatibility/security evidence before distributing extensions. |

The default first development session should contain **R1 only**, followed by a review. R2 is the first functional proof and starts only after that checkpoint. R3/R4 can be reordered by the user's research needs; R5's two domain tracks are alternatives, not a commitment to build both at once. No estimate assumes one slice equals one conversation turn.

## R1 change boundary

Expected owners: `app/contracts/src/context.ts`, `resource.ts`, `surface.ts`, evidence/workspace persistence schemas and migrations; `main/workspace/selection.ts`, existing reference/evidence owners; renderer reference chips/reveal states and their meaningful tests. The precise file list is finalized after accepting the contract.

Do not change engine execution, provider account behavior or user datasets for this migration. Keep WorkspaceController as the aggregate writer. Do not introduce a plugin marketplace, arbitrary metadata bag, event bus or parallel reference store. Extract reusable parts because responsibilities differ, not to reach a directory count.

Migration procedure: identify existing document/evidence versions → write an immutable backup → validate transformed document and references → atomic write → reopen and verify. Existing ordinal CSV keys remain revision-scoped; a migration cannot manufacture cross-revision biological identities. Historical assets retain their hashes/content. Unknown future shapes must be preserved/rejected safely according to the storage version policy, never silently stripped.

## User scenarios and checklist

The following are acceptance criteria, not results. Use a small synthetic fixture first, then a user-approved representative dataset without putting private research data into the public repository.

| Scenario                                                    | Observable pass condition                                                                                                                                        |
| ----------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Select two CSV rows, then attach them to Chat.              | The attachment keeps exact row and column IDs; sorting does not change membership.                                                                               |
| Drag a lasso or select by keyboard/table.                   | The selected count and concise identity summary match. Navigation and selection modes are visible. A screenshot region is never labelled as exact data points.   |
| Choose Discuss selection, edit the message, then cancel.    | Composer gets one frozen attachment; no turn is sent; removing it leaves data unchanged.                                                                         |
| Send selection to Agent A while Agent B is attached.        | A receives the intended evidence and recipient association; B gets no addressed input through this action. Existing shared-view grants remain separately scoped. |
| Agent responds with a reference to another item.            | It has the Agent's author mark; User's local selection survives. Clicking Show in views reveals the exact item without stealing composer text.                   |
| Close original tab and click a sent reference.              | Reopen from durable target or show captured evidence; no requirement for a live origin Surface.                                                                  |
| Dataset changes after sharing.                              | Sent reference stays on its original revision. UI shows Earlier version; no silent relocation to a similarly labelled row.                                       |
| Two equally matching quotes or duplicate biological labels. | Resolver reports ambiguity; user sees a clear resolution action.                                                                                                 |
| Notebook cell moves, output changes, or kernel restarts.    | Reference identity is preserved or explicitly historical; kernel epoch changes invalidate live-state claims.                                                     |
| Genome display changes coordinate convention or assembly.   | Conversion is explicit; assembly mismatch cannot become a successful reference.                                                                                  |
| Session disconnects during execute request.                 | Show uncertain execution and reconciliation; do not blindly resend.                                                                                              |
| View is hidden, slow or unsupported.                        | Agent cannot claim a current visual observation; retries/cancellation preserve the last valid reference.                                                         |
| Working at 1280×800, narrower window and enlarged text.     | Main work and chat remain usable, controls visible without hover, focus returns predictably. Keyboard access exists for canvas selections.                       |
| Large data or too many selected members.                    | Read/capture is bounded and cancelable; evidence shows exact count, returned count and sampling/truncation.                                                      |

## Verification strategy

- Contract/property tests for coordinate conversion, Unicode ranges, selector validity, exact/ambiguous resolution and stable membership under reorder/filter. Include negative cases that could otherwise point to the wrong scientific target.
- Migration fixtures from real existing schema versions, containing user/Agent marks, attachments and closed views. Compare meaning and captured asset hashes, not only JSON shape.
- Adapter contract tests cover declared capabilities, source revision changes during capture, cancellation and unsupported selectors. No tests that merely restate a trivial mapper.
- Electron interaction tests cover the complete cross-view/composer loop, focus/pin behavior, visual reference labels and disconnect/restart boundaries.
- A real Agent integration at R2 completion verifies what was actually delivered and returned. A simulated response in the concept sketch is not evidence of provider integration.
- Data/scale spikes record dataset shape/size, machine, pinned library versions, load/selection/capture latency and memory. Set numerical budgets from representative workloads before claiming large-data support.
- Run the repository checks appropriate to the changed scope. Existing Stage 6 engine baseline failures remain documented; do not claim new UI checks resolve them or broaden engine changes into this plan.
- End every slice with files/owners changed, behavior demonstrated, tests and limitations, then the user's approval before the next slice.

## Design lifecycle disposition

| Activity                      | Current disposition                                                                                                           |
| ----------------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| Problem discovery             | Performed from the user's shared-workbench request and current source review.                                                 |
| Existing visual identity      | Reused from the accepted compact central split/right-chat direction.                                                          |
| Concept alternatives          | Compared app container, all-native workbench and common workbench with bridged tools in research.                             |
| Prototype                     | A local interactive concept sketch with synthetic data, selection, attachment and Agent-authored reveal. Not a real analysis. |
| Expert review                 | Source/ownership analysis performed; interactive sketch checks recorded separately below.                                     |
| Representative-user usability | Open. The user must try/approve the proposed flow; library docs and automated checks cannot replace this.                     |
| Implementation handoff        | This plan and the design are drafts until accepted.                                                                           |
| Post-release evaluation       | Future: only meaningful after a real implementation is available.                                                             |

## Original proposal deliverables

Research, domain/ownership design, staged plan and the interactive sketch. The original proposal changed no production code; subsequent R1 implementation is recorded separately. Verification of the sketch concerns its local UI behavior only.

## Historical pre-retirement sketch verification record

Checked on 2026-09-07 in the in-app browser and an isolated headless session of installed Google Chrome. The packaged Playwright browser was absent, so installed Chrome was used without installing dependencies or using a personal browser profile.

- Light/dark appearance at outer sketch widths of 1024, 736 and 360 px: six plotted points, no horizontal control/table overflow and no page-script exceptions. Evidence: [record](review/sketch-checks.json), [light desktop](review/light-1024.png), [light medium](review/light-736.png), [dark narrow](review/dark-360-full.png).
- Select A/C → attach → additionally select F → send: the sent attachment still contains only A/C. The demo Agent can point to B while the local selection remains A/C/F.
- Plot click identifies its source row; keyboard checkbox selection retains focus; removing a draft attachment sends nothing; Focus view/Restore split preserves selection.
- The initial demo used a form submit, which the sandboxed preview did not dispatch. It was changed to a local button handler and the complete flow was verified again. This correction affects only the sketch.
- No live model, Notebook kernel, scientific analysis, large-data performance, source-editing or migration implementation was exercised. Representative-user usability remains open.

The inline sketch is the editable conversation artifact `shared-research-workbench.html` in the thread's visualization directory. It uses synthetic data and labels the Agent response as simulated.

## Next approved design checkpoint

The user accepted R2c completion and requested the next scope/ownership sketch on
2026-09-08. [R3a Run navigation](../r3a-run-navigation/design.md) now proposes the
first bounded R3 family. [R3a checkpoints](../r3a-run-navigation/implementation-plan.md)
start with contracts and observed evidence, subject to the new design approval.

On 2026-09-08 the user accepted R3a3 and requested the next design checkpoint.
[R3b Run dependency navigation](../r3b-run-dependencies/design.md) proposes the next bounded family.

## Current scope decision — 2026-09-09

R4a3 saved Notebook User/Agent communication is accepted and R4b investigation is
complete. The owner declined immediate document editing and requested an inventory
of remaining work. R4c1 edit/save-copy is deferred; R4c2 kernel integration is not an
automatic successor. Historical R4 entries above describe the original proposal,
not a mandatory dependency chain. See [remaining work](../../remaining-work.md).
