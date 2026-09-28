# R3c — PDF investigation

Date: 2026-09-08. Status: proposal research, not runtime qualification.

## Existing code and constraints

The current service reads registered Project resources through contained file access, checks modification during the read, and hashes the actual bytes. The outer preview limit is 8 MiB; current formats are UTF-8 text, CSV, PNG and JPEG. PDF needs a binary representation, not a new Resource kind. See [files.go](../../../../internal/appservice/files.go) and [file contract](../../../../app/contracts/src/file.ts).

The shell CSP currently disallows workers, frames, objects and network connections. Its asset protocol serves a small set of hashed shell files. A PDF dependency cannot simply be dropped into React and assumed compatible. See [content.ts](../../../../app/desktop/src/main/security/content.ts).

Main materializes existing file evidence from contained source content rather than renderer-supplied text or a desktop screenshot. RenderSession owns current render receipts; ObservedReads owns turn-local Agent scope. PDF must preserve those guarantees. See [materialize.ts](../../../../app/desktop/src/main/evidence/materialize.ts) and [render-session.ts](../../../../app/desktop/src/main/workspace/render-session.ts).

## Primary-source findings

| Source, checked 2026-09-08 | Relevant fact | Consequence for this proposal |
| --- | --- | --- |
| [PDF.js layers](https://mozilla.github.io/pdf.js/getting_started/) | The display API sits above the core parser; a complete viewer is a separate layer. | Evaluate the display API behind a small App adapter. Reusing the whole viewer is not required. |
| [PDF.js examples](https://mozilla.github.io/pdf.js/examples/) | Rendering uses a page viewport incorporating scale and rotation; PDF and canvas axes differ. A canvas cannot perform two concurrent page renders. | Store source-page coordinates; separate render jobs, cancel obsolete work, qualify HiDPI and rotation. |
| [PDF.js public API](https://mozilla.github.io/pdf.js/api/draft/module-pdfjsLib.html) | Loading accepts typed bytes, with worker transfer ownership. Text items expose strings, transforms and direction; text normalization is configurable. Bundled fonts/CMaps/WASM can require additional loading. | Retain authoritative bytes outside transferable buffers. Pin a decoder profile. Qualify offline auxiliary assets; never interpret extracted text as a guaranteed semantic reading order. |
| [PDFPageProxy API](https://mozilla.github.io/pdf.js/api/draft/module-pdfjsLib-PDFPageProxy.html) | Page APIs include a view box, user-unit scale, rotation, text content, rendering and cleanup. | Document identity, page geometry and presentation identity are separate. Do not assume an A4 origin at zero or userUnit=1. |
| [PageViewport implementation](https://github.com/mozilla/pdf.js/blob/master/src/display/page_viewport.js) | The viewport implements conversions between PDF coordinates and viewport coordinates, accounting for its transformation. | Use the qualified inverse transform, not a width-percentage shortcut. Test nonzero crop origins and all right-angle rotations. |
| [RenderTask API](https://mozilla.github.io/pdf.js/api/draft/module-pdfjsLib-RenderTask.html) | Rendering exposes cancellation and completion promises. | Closing a Surface must cancel/release jobs and prevent late publication. A timeout must also end retained work. |
| [React-PDF official README](https://github.com/wojtekmaj/react-pdf) | React-PDF wraps PDF.js Document/Page rendering. It requires worker setup and separate text/annotation layer styles. | A wrapper can help React lifecycle, but does not own evidence, source authorization or exact reference identity. Add it only if the spike proves net simplification. |
| [Electron process sandbox](https://www.electronjs.org/docs/latest/tutorial/sandbox) | Sandboxed renderers lack Node; enabling Node integration disables that sandbox. Preload remains a privileged boundary. | Keep PDF parsing outside privileged Main; a dedicated decoder host gets a minimal typed bridge, not the normal Workspace bridge. |
| [PDF.js historical advisory](https://github.com/mozilla/pdf.js/security/advisories/GHSA-wgrm-67xf-hhpq) | A past vulnerable release could execute attacker-controlled JavaScript under an affected evaluation configuration. | Pin and review the exact dependency at implementation time; do not present this historical issue as a vulnerability in every current version. |

These are moving upstream documentation pages, not proof about an installed PDF.js version. No PDF package was installed or selected in this proposal.

## Candidate comparison

| Candidate | Benefit | Gap / cost | Recommendation |
| --- | --- | --- | --- |
| Chromium's embedded PDF viewer | Existing conventional reading UI | No exact App selection/observation bridge has been demonstrated; embedded object/frame paths conflict with the current shell policy. | Not selected for the shared communication contract. Can be reconsidered if an exact supported bridge is proven. |
| PDF.js display API in an App-owned adapter | Direct control over page renders, text model, coordinates and lifetime | Worker/assets/CSP, canonical text mapping and independent capture must be qualified. | First candidate for R3c1. |
| React-PDF | Reusable React document/page lifecycle | Same PDF.js identity/capture problem; potentially duplicates the dedicated decoder owner. | Compare only if it reduces the actual adapter complexity. |
| New native PDF stack or external conversion service | Different rendering/accessibility options | New packaging/process/dependency surface; no demonstrated need in the first slice. | Deferred. |

Inference: a controlled PDF.js adapter is the strongest starting candidate for shared references, but the code has not established its suitability yet. The first checkpoint is explicitly allowed to reject it.

## Questions the spike must settle

1. Can a small sandboxed decode host return the same bounded page raster and canonical text model used by both viewing and evidence, without broadening the normal shell's capabilities?
2. Can selected text map reversibly to stable item IDs and UTF-16 boundaries under the pinned extraction profile, including ligatures, surrogate pairs, RTL and multiple columns? If not, ship page/region capability first.
3. Can original page coordinates survive rotation, nonzero crop boxes, user-unit scale, zoom and HiDPI with a measured maximum error?
4. What byte/page/pixel/time/cache limits are safe on representative reports? The existing 8 MiB bound is a starting admission limit, not a report-size recommendation.
5. Can cancellation, malformed input, missing fonts, encrypted files and decoder crashes fail locally without freezing Chat or losing a draft?

No source has been used to claim semantic extraction of plots, PDF tables or biological entities. Those capabilities remain out of scope.

## R3c1 follow-through — 2026-09-08

The above record describes the proposal before installing a candidate. [R3c1 native qualification](../../stages/r3c1-pdf-qualification.md) now selects PDF.js 6.3.289 with a bounded page/region capability, using its actual installed declarations and source plus native fixtures. [Upstream release](https://github.com/mozilla/pdf.js/releases/tag/v6.3.289), [display API source](https://github.com/mozilla/pdf.js/blob/v6.3.289/src/display/api.js), [worker image handling](https://github.com/mozilla/pdf.js/blob/v6.3.289/src/core/evaluator.js), [warning implementation](https://github.com/mozilla/pdf.js/blob/v6.3.289/src/shared/util.js).

Runtime evidence takes precedence over API names: strict-mode rendering and complete-list promises did not reliably reject the tested late image-stream failure. The candidate uses warning mode and a pinned display/worker warning-to-failure adapter. Oversized and corrupt-image regressions now pass. This is a measured implementation choice, not a claim that upstream promises guarantee complete visual fidelity. Exact text mapping and broad font/report qualification remain open; see the final verification record for concrete limits.
