# Saved Notebook qualification — R4a1

An isolated Electron/React reader for synthetic `.ipynb` bytes. This qualifies
cell/source/output addresses, passive representations, bounded decoding and
capture; it does not register a production View, contact an Agent, save a
Workspace, edit a Notebook or start a Jupyter kernel.

From `app/`, using existing installed App dependencies:

```sh
node qualification/notebook/qualify.cjs
node qualification/notebook/view.cjs
```

The first command runs strict TypeScript, temporary bundling, pure tests and native
scenarios. The second opens the inspection app. Choose a synthetic fixture and
Open notebook. Closing that app removes its temporary build/profile. No packages
or runtimes are installed, and the normal Gobble profile is never read or changed.

## Owners and API

| Owner                                                                              | Responsibility                                                                                                                                                                            |
| ---------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `model.ts`, `worker-protocol.ts`                                                   | Private qualification types and checked worker result shapes. No production schema/version changes.                                                                                       |
| `parser.ts`, `image-source.ts`, `text.ts`                                          | Bounded format projection, MIME selection and encoded image geometry; source/display normalization and raw quote mapping. No Electron/React state.                                        |
| `targets.ts`                                                                       | Closed target/receipt validation, revision and output identity, text/image bounds and observed containment. Object property order is irrelevant.                                          |
| `worker.ts`, `worker-client.ts`                                                    | A parser worker per read; cancellation, timeout, two-read admission and termination ownership.                                                                                            |
| `host.ts`                                                                          | One current snapshot and visible receipt; native image decode/capture. Returned View data is detached from authoritative data. Captures are returned and not retained in a second store.  |
| `bridge.ts`, `preload.ts`, `main.ts`                                               | Four named request/reply operations and invalidation, sender checks, synthetic byte source and isolated native windows/assets. Main-only test observation is absent from preload.         |
| `renderer.tsx`, `reader-parts.tsx`, `visibility.ts`, `text-window.ts`, `style.css` | Inspection composition, local text/image selection, passive Markdown, bounded DOM ranges and geometry. The temporary composer proves draft preservation, not production Chat integration. |
| `fixtures.ts`, `notebook.test.ts`, `test.config.ts`, `qualify.cjs`                 | Deterministic inputs, pure assertions, native cases and evidence.                                                                                                                         |
| `build.cjs`, `view.cjs`, `tsconfig.json`                                           | Temporary construction, review launch and exact strict compilation inclusion.                                                                                                             |

`parseNotebook(bytes)` returns an immutable-by-ownership snapshot after checking
the reader profile's required fields, format/version, IDs and budgets. It is not a
complete replacement for the nbformat reference validator. Unknown metadata is
ignored in the projection; source bytes are never rewritten. Unknown cell/output
kinds remain explicit source-only/unavailable slots.

`NotebookHost.load/view/visible/observe/image/capture/invalidate/close` separates
source lifetime from observation lifetime. Load supersedes the prior request and
does not promote failures. `visible` accepts only newer generations for the current
ready source. Invalidation removes the old receipt immediately. Observed capture
requires its exact generation and contained target. User capture still binds to
the exact current file/part revision. Closing releases the snapshot and cancels a
pending worker; it has no kernel or application-account side effect.

## Qualified representation profile

- nbformat 4.0–4.5 required reader fields; Python/R labels, code/raw source and
  basic Markdown headings/paragraphs. Exact Markdown selections use Source.
- Valid unique cell IDs; missing IDs are allowed only in legacy formats and use
  revision-scoped ordinals. Old references do not follow reordered/new content.
- Source and output ranges address the LF-normalized, control-escaped display in
  **0-based UTF-16 offsets, half-open**. Original source quote bytes/line endings
  remain recoverable. Surrogate/control-escape splits and empty ranges fail.
- Saved streams, errors and plain text; PNG/JPEG outputs; original output index,
  MIME and representation digest remain distinct from cell source. Raw HTML,
  SVG, JavaScript, widgets, external resources and Markdown attachments are not
  active. Unsupported images can choose a plain-text alternative during parsing;
  a later native decode failure is explicit, not silent content substitution.
- PNG animation and JPEG EXIF/orientation are unsupported in this profile. Native
  decoding must match the admitted natural dimensions. Captured image rectangles
  use integer natural pixels, top-left origin and half-open bounds, with no resampling.

The selection metadata is a private qualification shape. R4a2 must map it to the
existing Project/resource/evidence contracts, with explicit coordinate units and
frozen migration inputs. Numeric inspection controls are not a production UX
commitment. Actual Agent pointing and semantic plot/cell-entity selections are not
implemented. CSV chart creation remains excluded.

## Limits and their meaning

Source ≤8 MiB; decoded source/output text ≤1 MiB; ≤1,000 cells; ≤100 outputs/cell
and ≤1,000 outputs/document; JSON preflight depth ≤32 and structural-item budget
100,000. A read worker has a 5-second deadline, 128 MiB old-generation and 16 MiB
young-generation heap limits. Cancellation terminates the worker; this is not a
hard process-RSS ceiling or a proof against every malicious input.

Two reader windows at most. The inspection UI displays 20 cells per page and one
expanded saved image per reader. Text DOM windows contain at most 2,048 UTF-16 units
or 200 lines; continuation keeps absolute source offsets. Basic Markdown is also
bounded and directs continuation to Source. Visible geometry scans at most 4,096
code points and returns at most 4,096 UTF-16 units across 64 parts. It may omit
otherwise visible material when this budget is exhausted; only returned ranges
are pointable. Production integration must expose such limits in observation metadata.

Encoded image ≤4 MiB; natural image ≤4 MP; materialized display PNG ≤8 MiB. Decode
is on demand using Electron nativeImage, not a retained all-output bitmap cache.
Native image decoding is synchronous and has no interruptible 5-second worker
deadline; measured bounded fixtures qualify this candidate, not an adversarial
decoder time guarantee. A renderer restart/crash policy remains R4a2 work.

Captures: text/raw quote ≤64 KiB each; PNG ≤768 KiB and ≤1536 pixels per edge.
Large regions fail with a smaller-region action. No automatic resize, truncation,
chart generation or execution. A held capture is independent of later source loads.

See [stage decision and evidence](../../../docs/desktop-workspace/stages/r4a1-notebook-qualification.md).
