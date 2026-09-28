# R4a1 author architecture review

2026-09-09. Subject: the isolated Notebook qualification, reviewed by its author.
No independent reviewer, product integration or release verdict is claimed.

The current source keeps identity and effects out of React: parsing and target
resolution are plain functions; a Host owns a single current document/receipt;
the worker client owns admission, cancellation and termination; React owns local
gestures and inspection state. No Notebook repository, global event bus, generic
plugin interface, provider adapter or engine state model was introduced.

Reviewed API corrections:

- Equivalent JSON cell/part addresses are compared by values, not property order.
- `view()` returns detached data and image metadata without pretending empty
  base64 is a decoded image. Raw source and encoded image ownership stay in Host.
- Worker result validation covers every projected field; runtime input starts as
  unknown. Closed target/receipt schemas reject extra fields and mismatched parts.
- Visibility invalidation clears the prior receipt; only accepted reports advance
  its generation. Loading cannot accept a new ready observation prematurely.
- Display LF/control escaping has a separate profile and raw-quote mapping. Source
  bytes/IDs are not normalized by writing back to the Notebook.
- One expanded image per reader and text windows bound browser work separately
  from the whole-file transport. Text observation is a bounded, explicitly partial
  set of ranges, not an assertion that the whole Notebook was read.

The React inspection composition has its own temporary draft/capture state solely
to exercise reading/return behavior. R4a2 must reuse production WorkspaceController,
RenderSession, evidence and composer owners rather than importing that composition.
Qualification source is not imported by the production application.

## Corrections and rejected evidence

Earlier reports are retained as `previous-*.json` beside qualification.json.
The original classifications are preserved; these are the reviewed causes:

1. **Product defect:** invalidate and publish both incremented the same generation,
   rejecting the next valid visible report. Fixed by advancing only on accepted
   reports and rejecting reports while loading. The complete suite was rerun.
2. **Test defect:** the scroll test set an already-zero scroll position to zero, so
   no change occurred. It now forces movement and asserts the fixture moved before
   checking invalidation. This did not authorize weakening the expiry rule.
3. **Test evidence defect:** Playwright's screenshot at native 150% zoom captured a
   clipped area, despite correct layout coordinates. The rejected artifact remains
   `zoom-150-playwright-crop.png`. Final evidence uses the exact reader window's
   `webContents.capturePage()` and records real content-size/zoom conditions.
4. **Performance defect:** a 16,384-unit text window passed correctness checks but
   a long-line walkthrough took about 6 seconds and the first Tab reached about
   704 MiB peak working set. That candidate was rejected. The final 2,048-unit
   window/4,096-point geometry budget is checked against a 2-second combined
   open/next/previous scenario; final measurements are in qualification.json.

Small construction corrections (syntax/narrowing/DOM iterable inclusion) preceded
behavior tests. None changed production files or existing compiler settings.

## R4a2 integration handoff

Recommended next slice, requiring owner acceptance: add a named bounded Notebook
source path to the existing registered-file service; introduce one Notebook
contract owner for source/output targets and typed read state; promote the
qualified parser/target logic into Main-owned modules; render through existing
Pane/Surface composition and attach through current evidence/composer owners.

The saved file remains `ResourceRef.kind=file`. Go appservice owns registered
access and exact byte revision, Electron owns document projection and capture,
React owns presentation. Gobble engine Run/Pipeline authority remains unchanged.
Do not globally raise generic text preview limits. The >3 MiB embedded-image
fixture establishes a concrete reason for a named ≤8 MiB Notebook transport.

Before public schemas change: freeze current Workspace v12 / bundle v14 / toolset
v8 inputs, choose explicit text units and output representation identity, and add
migration fixtures that preserve all existing captured evidence. No new public
version number is allocated by this qualification.

Keep malformed/unsupported output states visible, disclose partial text/Markdown
and image support, preserve local reading state while showing captured evidence,
and implement durable reopen/source-change behavior through existing owners.
R4a3 supplies actual Agent observations, marks, questions and exact in-document
Show/Return; R4b/c supply Jupyter connection, editing and execution. No automatic
kernel start or execution from a selection, file open or decoded output.
