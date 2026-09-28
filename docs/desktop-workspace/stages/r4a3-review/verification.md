# R4a3 verification record

Date: 2026-09-09. Request: R4a3-2026-09-09. Sequential Development → Testing in the
same task. No independent source-review claim. Authoritative source and built output
hashes are in `source-subject.json` and `tested-build.json`; environment in `runtime.json`.

## Results

| Check                                  | Evidence / result                                                                                                          |
| -------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| All TypeScript consumer configurations | `typecheck-final.log` — passed                                                                                             |
| Repository formatting                  | `format-final.log` — passed                                                                                                |
| Generated current schema v16           | `schema-final.log` — passed; published v1–v15 unchanged                                                                    |
| Unit / contract / host suites          | `unit-final.log` — **320 tests, 28 files passed**                                                                          |
| Real Electron regression matrix        | **65 distinct scenarios passed**, combining the full regression run and focused corrections/additions below                |
| Registered Go app service              | `go-final.log` — passed                                                                                                    |
| Live Agent                             | Real **GPT-6-Astra** in a temporary App profile and synthetic Notebook; one exact image pointer and one immutable question |
| Source preservation                    | 269 protected engine/command/memory/qualification/published-schema files unchanged; no missing baseline files              |

The full regression invocation (`native-final.log`) passed 62 scenarios and failed
one newly authored test because it requested “Share mark” instead of the existing
accessible name “Share mark from analysis.ipynb”. The corrected test passed in
`native-user-mark-correction.log`. Two additional focused native negative scenarios
passed in `native-viewport-mutations.log`: output folding and native zoom each
revoked an earlier receipt. These are 65 distinct passing scenarios, not a claim
that the initial full invocation had no failures. No product behavior was changed
for that final locator correction or the two added negative tests.

The earlier focused Notebook run (`native-notebook-second.log`) also passed all 11
then-existing Notebook scenarios, including text drag, original image pixels,
malformed input recovery, bounded text, dual Panes, offline preview and addressed Send.

## Concrete scenarios

- Default observation returns visible text with exact addresses, advertises images
  without delivering pixels and does not consume question-capture slots.
- Explicit saved-image observation returns PNG pixels and an image receipt. Native
  source/text and image marks preserve User selection, focus and unsent next draft.
- Scrolling, folded outputs and native zoom each revoke old pointing authority.
  Preview and missing receipts cannot publish a mark. Clipped long lines remain
  bounded and are addressed in absolute displayed UTF-16 units.
- User output marks use the same exact reference contract. Compact enlarged UI keeps
  Return reachable and preserves selection and draft.
- Show/Return preserves base scroll, folded outputs, text selection and an unfinished
  image region. Missing source refuses Show while a matching saved question capture
  remains previewable offline.
- Pure/host checks cover Project/revision/part containment, bounded report size,
  frozen v8 rejection, v13 migration, explicit v8 renewal, turn release, exact
  temporary reference resolution and cleanup after Project disposal.

## Live provider evidence

`live-request.json` records the synthetic prompt and prior User state.
`live-delivery.json` records the actual completed submission, one NotebookTarget v6
image pointer and one question. The Agent read the visible sample-quality filter
and saved text output, then explicitly observed the image crop at x=40, y=30,
160×80 original pixels. Its description of the teal and red-orange regions matched
the displayed PNG. It did not claim scientific information from the placeholder.

`live-review.json` confirms successful image Show, Return and unchanged User state.
`live-agent-show.png` and `live-agent-return.png` are actual live review screenshots.
`live-isolation.json` records cleanup. Credentials were copied read-only into a
restricted temporary profile, never printed or written back. Only that temporary
window/background process was closed. No second Agent submission was needed when
the original review process's input pipe closed: inspection resumed through its
existing local debugging endpoint. This proves App/provider communication on the
recorded environment; it does not prove Jupyter integration or execution.

## First-failure handling

Original logs and first native screenshots are retained. The skeleton's unused
frozen-schema import and the newly widened selector union's legacy branch were
corrected in their owning contract/integration layer. New text highlight wrappers
were simplified to fragments to retain exact native text-node behavior. First test
assumptions about accessible button labels, which adjacent text stayed visible at
the bottom of a short Notebook, and a synthetic fixture's initial text were corrected
without relaxing visibility or source checks. Focused reruns and the regression
matrix above verify those corrections.

No dependency upgrades, CSV chart creation, notebook editing, kernel execution,
engine changes, commits, pushes or release actions are part of this result.
