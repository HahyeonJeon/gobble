# R3c1 qualification evidence

Final native run: **2026-09-08T11:14:12.128Z**, 17 scenarios passed, 64 rendered transformation combinations. Source-checkout macOS evidence only. [Machine-readable report](qualification.json), [protected/source hashes](source-verification.json), [fixture manifest](fixtures.json), [asset inventory](assets.json), [candidate lock](candidate-package-lock.json).

## Construction and baseline

Strict TypeScript checking and bundled Main, preloads, React, decoder and real worker succeeded. Target: Electron 44.2.0 / Chromium 152.0.7977.76, macOS 26.5.2 arm64, PDF.js 6.3.289, React 19.2.8, TypeScript 5.9.3. The exact Electron Node/runtime versions are in `qualification.json`. Formatting passed for `app/qualification/pdf/`.

All **434 protected preexisting files are byte-identical**. There are 17 new qualification files; production package manifests/lock, shell CSP, contracts and persisted schema versions were not changed. The normal App profile and signed-in Agent were not used. R3b3's previously reported 263 unit/contract and 43 native scenarios are prior-stage evidence and were not rerun here.

## Observed results

| Claim | Observed evidence |
| --- | --- |
| Same visible content and crop | 64 page rasters across source crop origin, intrinsic/user rotation, UserUnit, scale and DPR. NativeImage comparison found exact pixel equality between each capture and the corresponding page crop; known center color also matched. |
| Coordinate/model stability | Maximum round-trip error 5.684341886080802e-14 PDF user units. Model hashes stable across presentation changes. Eight combinations over 4 million pixels rejected. |
| Native preview | Actual pointer drag captured 80×50 pixels; keyboard action captured the known 120×60 region. Changing the fixture chooser preserved the displayed source name. At 900×650 all action buttons remained in bounds with no horizontal document overflow. |
| Input/output bounds | 8 MiB−1 and 8 MiB admitted; 8 MiB+1 rejected. Page 200 accessible; 201-page document rejected. A noisy page produced 3,263,377 PNG bytes; full-page evidence exceeded its cap while a smaller region succeeded. |
| Invalid/degraded source | Malformed PDF, password requirement, 21.16-megapixel embedded image and corrupt JPEG failed explicitly. A valid source reopened afterward. No ready page was published for either degraded-image fixture. |
| Source/rendition binding | Changed source and changed rendition could not capture the old target. Closed schema rejected an injected path. A foreign WebContents could not complete a pending job. |
| Process isolation | Decoder and preview used different native process IDs. Decoder had sandbox/context isolation, no Node/general Workspace/preview bridge, fixed local assets, denied remote fetch and popup, and 404 for unlisted assets. Actual vendor worker and standard font requests were observed. |
| Cancellation and timeout | Cancellation completed in 12 ms; the 5-second Main deadline ended an intentionally blocked decoder in 5,002 ms. No late page published. Forced decoder crash left the preview alive and recovery succeeded. |
| Cleanup | All 31 retired hosts closed with no pending job. After stopping the active host, one preview window remained. Runner closed the test application and removed its exact temporary build/profile. |

Three short synthetic-report samples: source open **150.2–158.3 ms**, first page **44.2–45.8 ms**, warm page **6.9–7.3 ms**, each at 425,600 pixels. These include native test-bridge overhead and do not characterize large real research reports.

At the final live snapshot, Electron reported working sets of 146,896 KiB for the decoder, 122,512 KiB for the preview, and 299,136 KiB for the Main/test process. Main also generated fixtures and exchanged base64 test images; these figures are not isolated decoder allocations. Process peaks and GPU/utility samples remain in the report. No hard RSS cap or exhaustive memory-leak claim follows from a single bounded run; the measured disposal invariant is closed hosts/no pending work.

## Visual review

Inspected [1280-wide native screenshot](native-page-and-capture.png) and [900×650 native screenshot](native-compact.png). Controls and capture evidence are visible without hover, reader scrolling stays inside the central page area, and the facts sidebar scrolls separately at compact size. The screen is an adapter inspection tool; product navigation, selection overlays, keyboard region editing and Chat integration remain R3c2. This expert check is not representative-user usability evidence.

## Defects found and corrected

Previous attempt JSON files and `construction-failure.log` are retained as historical failure evidence; `qualification.json` is the final result.

1. PDF.js 6 API construction errors: retained the document loading task for destruction and used the typed `PDFWorker.create({port})` factory. No vendor/type casts were added to bypass the APIs.
2. Repeated disposal attempted protocol removal twice. Host disposal is now idempotent.
3. The timeout test awaited JavaScript in a destroyed WebContents, so the observer itself did not settle. This was a test defect. It now observes the Main job deadline/disposal, and each scenario has its own 30-second observer deadline.
4. A supplied worker port received work before vendor initialization. An explicit ready handshake now precedes PDF.js construction.
5. Strict PDF.js mode could complete a partial/blank raster after an image stream failure; awaiting the public complete operator-list promise did not cure it. The final pinned adapter selects recoverable warnings and rejects all PDF.js display/worker warnings. Oversized and corrupt-image regressions pass. The worker uses the tuple's actual `console.warn` path, not a guessed log path. Earlier warning/preflight attempts in the history did not pass and are not the chosen design.
6. Preview review found a source-label mismatch when the chooser changed before opening, and a readiness risk while a new blob image was loading. The displayed source and painted rendition now track their own identities; request epochs invalidate late work after cancellation.

## Qualification boundary

Direct PDF.js was selected because its model/raster/lifecycle calls are required regardless of a React wrapper; no second PDF abstraction was installed. Exact text geometry is unqualified: [diagnostic extraction](text-diagnostics.json) preserves strings and item positions without inventing character boxes. The initial candidate exposes page/region only.

The profile omits annotation/form appearance, XFA, WASM, system fonts and dynamic font faces. It does not provide OCR, password entry, editing, embedded assets/actions, native PDF accessibility, production Project integration or real Agent delivery. Included CMaps/font assets and licenses do not prove all font families or document encodings render correctly. The worker warning adapter and realistic resource envelope require requalification on any dependency/profile change.

Testing disposition: bounded R3c1 claims passed. Owner review of the sample and architecture is the next checkpoint; R3c2 implementation is not started by this report.
