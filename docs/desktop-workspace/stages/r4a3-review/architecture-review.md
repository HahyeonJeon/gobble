# R4a3 architecture review

2026-09-09. Sequential author review in the same task; no independent reviewer claim.
The review covers source organization, observable APIs, state/byte authority and
compatibility. R4a2's parser, registered-file service and capture storage are reused.

| Owner                                          | Owns                                                                               | Delegates                                                                              |
| ---------------------------------------------- | ---------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| `contracts/notebook-viewport`                  | Closed bounded selectors, part/range containment                                   | Loaded raw-source boundary validation to Main                                          |
| `RenderSession`                                | Ephemeral ready acknowledgment and viewport generation                             | Project and foreground authorization to Controller                                     |
| `WorkspaceController.notebookViewport`         | Active Project, current lease and Main source validation before accepting a report | Rendering geometry to the renderer; storage to existing writer                         |
| `shared-context/notebook-observation`          | Pure bounded text/target projection and explicit preview semantics                 | Pixel capture, turn authorization and provider delivery                                |
| `ObservedReads.recordNotebook`                 | Turn-local targets and currentness callback, at most 32 receipts                   | No source bytes or second persistent observation cache                                 |
| `NotebookViews.reference`                      | One exact temporary read/capture and finally-disposal                              | Raw decoding to NotebookHost; reference presentation to the existing reference session |
| `materializeNotebook` / `NotebookCapture`      | Existing text/raw quote or PNG evidence representation                             | Durability, question rules and delivery to existing evidence/question owners           |
| `useNotebookViewport`                          | Fully visible DOM ranges, nested clipping, painted image rectangle                 | Main validates all reported addresses before granting authority                        |
| `NotebookTextHighlights` / NotebookImage       | Authored overlays independent of local User intent                                 | Durable marks to existing shared references                                            |
| `NotebookSurface` / `NotebookReferenceContent` | Retained base reader and one temporary exact excerpt                               | Show resolution and Return layout state to Main                                        |

## Contract and ownership decisions

- A Notebook stays one registered file Resource inside a Pane. Cells/outputs are
  addresses inside its View; no new Pane, conversation, document repository or
  generic plugin registry was introduced. Gobble Run/Pipeline execution is unchanged.
- NotebookTarget v6 remains the common User/Agent address. Workspace v14 permits
  authored Notebook references; bundle v16 and shared-views-v9 expose the new input.
  Frozen Workspace v13 and v8 tools reject future shapes, and prior conversations
  require existing explicit renewal. Historical schema bundles are untouched.
- Visible text authority and available image addresses are deliberately distinct.
  Default observation returns actual text, while an image needs explicit returned
  pixels and model capability. Preview is explicit, single-part and never pointable.
  A Notebook renderer report cannot supply raw source bytes or captured pixels.
- The Controller checks public range/part bounds and NotebookHost checks surrogate
  and raw control-escape boundaries. The receipt's callback rechecks foreground,
  ready identity and viewport generation at use. User Share also validates the
  Main-held source before publishing. Turn release removes receipt authority.
- Explicit observations reuse QuestionService's bounded immutable asset cache.
  Default observation does not consume question-evidence slots. Question creation
  checks the current exact registered source; saved captures remain historical.
- Show loads only the exact requested source revision through one temporary host.
  It returns an excerpt and disposes decoding state. Missing/changed source refuses
  Show rather than searching for equivalent text. Matching existing captures can be
  previewed through the existing offline evidence UI.
- NotebookSurface preserves the mounted base reader. NotebookImage's Blob lifetime
  follows source/part identity, activation and explicit retry, so Show acknowledgments
  do not destroy an unfinished region. Return preserved native scroll, folds,
  selection, image gesture and the existing composer in real Electron scenarios.
- A temporary reference excerpt is not an Agent observation surface. Agent reads
  receive an explicit Return instruction. This avoids falsely treating the hidden
  base Notebook as visible while the User is inspecting a reference.

## Review evidence and limits

320 unit checks exercise current/frozen contracts, source ownership, scope inclusion,
viewport/turn invalidation and reference disposal. Native cases cover visible text,
clipped ranges, painted images, User/Agent marks, question capture, historical
fallback, Show/Return and compact controls. Real GPT-6-Astra independently used the
actual App tools on a synthetic Notebook, produced one image pointer and question,
and preserved User state through Show/Return. This is a live provider verification,
not an independent source-code review or a Jupyter integration claim.

The current profile addresses exact Source and plain-text output. Basic Markdown
requires revealing Source for precise observation. There is no notebook editing,
kernel execution, widget runtime or CSV chart creation in this checkpoint.
