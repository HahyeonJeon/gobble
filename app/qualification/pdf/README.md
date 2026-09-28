# Native PDF qualification — R3c1

An isolated source-checkout experiment for Gobble App's proposed PDF adapter. It uses real PDF bytes, PDF.js 6.3.289, a sandboxed Electron decoder and a React inspection screen. It does not register a Project viewer, persist evidence or contact an Agent.

From `app/`, with the existing App dependencies installed:

```sh
node qualification/pdf/qualify.cjs
node qualification/pdf/view.cjs
```

The first command constructs/typechecks the candidate and runs native scenarios. The second opens an interactive sample; close its window to remove its temporary build/profile. Both install the exact candidate into a temporary directory if needed, with lifecycle scripts and optional native canvas disabled. To reuse a previously installed candidate, set `R3C1_CANDIDATE` to the directory containing its `package-lock.json` and `node_modules/pdfjs-dist`. No normal App dependency or account profile is modified. Evidence is written to `docs/desktop-workspace/stages/r3c1-review/`; previous scenario attempts are retained.

## Owners and API

| Files                                   | Responsibility                                                                                                                                           |
| --------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `protocol.ts`                           | Closed result schemas, source/page/rendition identities, diagnostic text model and qualification limits. No production wire version.                     |
| `geometry.ts`, `raster.ts`              | Pure page/pixel conversion, containment, outward raster quantization and Main-side PNG header/dimension checks.                                          |
| `host.ts`                               | One decoder's lifetime, bounded job dispatch, sender/source/model binding, asset/network policy, cancellation and timeout. Imports no fixture generator. |
| `decoder.ts`, `worker.ts`               | One document and one retained page raster; PDF.js parsing, warning-to-failure policy and same-raster capture. Worker assets are bundled and unmodified.  |
| `preload.ts`, `ui-preload.ts`           | Separate job/result and preview-command bridges; no generic filesystem or Workspace capability.                                                          |
| `main.ts`                               | Synthetic source registry, current-host replacement and preview composition. A native-only test observer is deliberately absent from both preloads.      |
| `renderer.tsx`, `style.css`             | Developer inspection controls, painted-rendition readiness, source-coordinate drag and visible evidence. This is not the product Chat layout.            |
| `fixtures.ts`, `fixtures/encrypted.pdf` | Deterministic synthetic inputs; no real user files.                                                                                                      |
| `build.cjs`, `qualify.cjs`, `view.cjs`  | Temporary candidate construction, native evidence, and reviewable demo lifecycle.                                                                        |

`PdfHost.start/open/page/capture/stop` form a deliberately small adapter. Only one job is pending and one page raster is retained. `page` invalidates the prior rendition; `capture` requires its current rendition ID and model hash. `stop` is idempotent and destroys the WebContents/worker. Main supplies immutable source bytes, not paths or arbitrary URLs. Geometry is expressed in PDF page user coordinates, including CropBox origin, UserUnit and intrinsic rotation through the recorded viewport transform.

## Qualified capability and limits

Page and rectangular-region capture are candidates for R3c2. Text is diagnostic-only: a PDF glyph can map to several Unicode code units, and this fixture suite does not prove exact per-character geometry. No text selection API should be inferred from extracted item strings.

Limits: 8 MiB file, 200 pages, 4 million output pixels, 20 million pixels per embedded source image, 5 seconds per Main job, 6 MiB page-PNG base64 and 1 MiB capture-PNG base64. Base64 limits are encoded-string sizes, not decoded byte sizes. One document/page ownership and teardown are verified; these limits are not a hard process-RSS ceiling or a universal adversarial-PDF resource guarantee.

The fixed profile disables XFA, annotations/form appearance, system fonts, dynamic font faces and WASM. CMaps and standard font programs are bundled. Password-protected documents fail locally. No JavaScript actions, URI activation, attachments, OCR, editing, embedded browser or Notebook execution is implemented. R3c2 must disclose this profile's unsupported appearance/content before letting users treat it as a full PDF reader.

PDF.js 6.3.289 can resolve rendering (and the complete operator-list promise) before a late strict-mode stream error is observed. The candidate therefore uses its recoverable-warning mode and rejects every PDF.js warning from both display and worker contexts. The worker is initialized before its port is handed to PDF.js. The real oversized/corrupt-image fixtures verify this failure path. This is a pinned diagnostic adapter, not a universal public PDF.js guarantee; any PDF.js update must rerun and review qualification.

The encrypted fixture was generated locally from `makePdf()` using pypdf 6.10.0 and AES-128 (test passwords `qualification` / `fixture-owner`). Its exact bytes/hash are frozen in the fixture inventory. The Unicode fixture intentionally maps Helvetica glyph codes to ligature, combining, surrogate-pair and RTL text; it is an ambiguity diagnostic, not a representative font-fidelity benchmark. Wider real-report/font/color/accessibility qualification belongs to later integration.

See the [stage decision](../../../docs/desktop-workspace/stages/r3c1-pdf-qualification.md) for measured evidence, limitations and the next checkpoint.
