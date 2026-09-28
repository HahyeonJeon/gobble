# P5B implementation sequence

Status: design approved on 2026-09-13. History foundations and engine review/admission are implemented through [P5B-2](../../stages/p5b2-continuation/README.md), with bounded synthetic execution evidence. [P5B-3](../../stages/p5b3-transport/README.md) adds capability negotiation, native/Main transport and durable intent restoration. [P5B-4](../../stages/p5b4-visual-review/README.md) implements the shared Flow/Chat review and exact observed-task context with deterministic Electron evidence. [P5B-5](../../stages/p5b5-qualification/README.md) now records actual tool qualification after owner approval. The five implementation stages are complete subject to the owner's P5B-5 result review. Preserve inherited modifications and use focused file ownership; no commits, publishing or image replacement implied.

## 1. Freeze concepts and contracts

Record owner feedback on the two concepts and first-slice compatibility. Add versioned continuation review/intent/receipt types, capability negotiation, format-3 origin/epoch history and exact reference selectors. Define bounds and errors. Show the amended ownership/state diagram before changes if this alters the approved flow.

Verification: canonical digest stability; malformed/oversized/cross-Run references; wrong history head; request/body mismatch; old format inspection; new format downgrade refusal; explicit continuation limit. Keep original prepared Start replay and normal CLI prepared Resume rejection.

Exit: schema/runtime validation agree; no User or Agent path can fabricate reuse or execution permission.

## 2. Implement engine review and admission

Separate read-only review from mutation. Qualify the sealed linear workflow, stopped ownership, exact installed tools/data and output content. Under the mutation lock, verify the review again, atomically publish epoch/receipt/attempt plan, then schedule through existing mechanisms. Preserve original Start and every prior attempt. Unsupported/missing/ambiguous inputs fail closed without hidden cleanup.

Verification: review makes zero workspace/container changes; successful Trim keeps attempt/log/output content; stopped FastQC gets attempt 2; stale output/input/tool rejects with no admission; racing confirmations admit once; same request replay; delayed old Stop cannot stop the new epoch; new Stop addresses only the new epoch; crash before/after commit; receipt lookup after checkpoint pruning; repeated stop/continue within bounds. Run focused Go tests, race and vet for affected packages.

Exit: engine invariants pass before enabling any App action. Host tests alone do not qualify Linux tool execution.

## 3. Wire native service and Electron Main

Add a dedicated continuation operation and supervised request/status transport. Restore pending intents after restart, maintain exact project binding, and derive current epoch observation separately from original Start. Capability handshake controls availability. Route Agent read-only references through existing shared capture authority; no Agent Resume tool.

Verification: malformed/untrusted request rejection, process failure and uncertain acknowledgement, exact replay after restart, old Start/Stop history preservation, capability absence, cross-project reference rejection, teardown/cancellation boundaries. Types, schema checks and affected unit tests.

Exit: one typed User action maps to one durable intent and engine receipt; restoration never creates work automatically.

## 4. Add the shared visual review

Use the approved Run Flow with kept/restarted/blocked labels and selected details. Keep the single Chat composer and typed confirmation. Add exact review references and saved-reference replay. Current-newer, older-engine, stale, stopping and unknown-outcome states must explain the next action. Preserve existing B / Change spotlight and new-analysis path.

Verification: pointer/keyboard selection, attach-without-send, Agent pointing, history reference after recheck, draft persistence across compact regions, controls visible before hover, disabled action explanation, focus after navigation/dialog close, visible status during slow check, zero implicit Resume on discussion or reload. Check real Electron layout and accessibility, not only HTML.

Exit: actual renderer follows the reviewed state model. Representative participants must be able to explain saved design, whole-task restart and missing-output refusal; any material misunderstanding reopens copy/structure.

## 5. Qualify the actual engine and end-to-end path

Build a new capability-bearing engine with an exact recorded identity; create fresh synthetic fixture Runs from initial Start using that same image. Do not retrofit the old P4/P5A image/workspaces. Exercise real Trim/FastQC completion/Stop/Resume through Electron and native service, with retained attempt evidence and restart replay.

Verification matrix: successful continuation; Stop still settling; repeat Stop/Resume; output deletion/change; input/tool drift; Current advanced; unavailable capability; lost receipt; process interruption; application restart; old captured reference; compact and wide layouts. Record actual environment/image/command evidence separately from simulated Agent messages. No private research data or real provider call is necessary for the deterministic fixture.

Exit: exact environment report, production hash manifest, limitations and owner-facing summary/screenshots. Old inspection and P5A new-analysis regression checks pass. Packaging and broad recovery are separate stages.

## Review tasks and success criteria

- Explain which design/data Resume uses when Current has changed. A faster click rate is not success if the answer is wrong.
- Point to one kept result and one restarted step; explain that the stopped task starts from its beginning.
- Attach a review to Chat, recheck after output loss, then open the earlier reference. It must still show the earlier meaning and cannot authorize the changed plan.
- Recover from lost acknowledgement without submitting a replacement command.
- At compact width, move between Flow and Chat without losing the draft or selected reference.

Owner: project maintainer with implementation agent. Consumer: design/engine review checkpoint. Before broad release, run bounded task-based sessions with representative research users and keyboard/assistive-input users; record errors and misunderstanding, not only time-to-click. Post-release review is bounded to the first opted-in pilot and a two-week review window, with no research content collected. Unintended work, wrong-design expectations, duplicated intent or reference retargeting reopens the design immediately. Explicit improvement/no-change and Maintenance decisions belong to that future review.
