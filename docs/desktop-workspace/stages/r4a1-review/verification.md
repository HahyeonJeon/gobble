# R4a1 qualification result

**Passed, 2026-09-09 KST; owner acceptance pending.** This result qualifies the
isolated saved Notebook candidate in the source checkout. It does not claim
production integration, persistence, actual Agent delivery, kernel execution or
packaged/cross-platform support. Review was performed by the implementation author.

## Subject and reproduction

- [Source/configuration/fixture/script hashes](source-subject.json) identify 24
  files; [emitted output hashes](tested-build.json) identify the build exercised.
- [Native/worker report](qualification.json), [pure test output](unit.log),
  [fixture byte hashes](fixtures.json), and [architecture review](architecture-review.md)
  preserve the verification evidence and reviewed corrections.
- [Preservation check](preservation.json): all **554** preexisting protected
  app/internal/cmd/.gobbi files match their starting hashes. New code is confined
  to `app/qualification/notebook/`; production manifests, contracts and compiler
  settings were not changed. This hash check excludes generated/dependency folders
  and is not a separate account-state audit.
- Environment: macOS arm64, Electron **44.2.0**, embedded Node **24.20.0**,
  Chromium **152.0.7977.76**, Vitest **4.1.11**. Runtime versions and process
  observations are in the report. No new dependency or Jupyter installation.
- From `app/`, run `node qualification/notebook/qualify.cjs` to reproduce. It
  builds/typechecks the exact isolated entries, runs the pure suite and opens
  temporary-profile native windows; cleanup closes its windows/workers and removes
  temporary build/profile state. `node qualification/notebook/view.cjs` opens the
  same reader for manual inspection with synthetic fixtures.

## Verification coverage

Strict TypeScript, bundled entry construction and qualification formatting passed.
**27 pure tests and 15 qualification scenarios passed.** The first two scenarios
exercise workers and measured parsing from Node; the remaining **13 exercise the
actual emitted Electron app**. These counts do not include the earlier production
suite, which was not rerun for this isolated change.

| Risk | Observed coverage |
| --- | --- |
| Wrong document, cell or output | nbformat 4.0–4.5, Python/R, legacy ordinals, duplicate/invalid/missing IDs, reordered/changed source, original output index and MIME/digest matching. Old targets fail instead of following new content. |
| Wrong quoted text | String/array source, LF/CRLF, Unicode, escaped terminal controls, bounds, split-surrogate rejection and actual native drag/keyboard capture. Original quote mapping remains separate from display offsets. |
| Wrong image region | PNG/JPEG native decoding, 150% image scale, pointer/keyboard coordinates and exact natural-pixel crop. Malformed images, oversized dimensions and capture byte/edge limits return explicit failures. |
| False observation | Folded/offscreen/partially clipped text, scroll expiry, current receipt generation and returned-range containment. Bounded scans only authorize ranges actually reported. |
| Broken local workflow | Select/attach, draft preservation, captured Show/Return, source revision replacement and prior evidence inspection. This exercises temporary local state, not production persistence. |
| Resource and process ownership | Cancellation, deadline, two-read admission, two independent windows, third-window refusal, close cleanup, sender/preload boundaries, passive content/network/popup/CSP behavior. |
| Browser cost and layout | 1,000 cells, embedded image source, 1 MiB long-line continuation, 900×650 content area and native 150% zoom with the composer reachable. |

## Measurements and decisions

Single-run local measurements from the final report; they are not percentiles,
service guarantees or worst-case adversarial bounds. The parse comparison includes
worker roundtrip **and a direct-parser revision cross-check** in its elapsed time.

| Fixture / operation | Final measurement |
| --- | --- |
| 2,491-byte saved Python fixture | 17.88 ms parse comparison |
| 3,385,667-byte fixture with a real high-entropy embedded PNG | 106.60 ms parse comparison |
| 1,000 cells, 118,760 bytes | 29.78 ms parse comparison |
| 1 MiB aggregate source text | 58.57 ms parse comparison |
| Native long-line open + next + previous; 2,048-unit text windows | **224.66 ms**, below the scenario's 2,000 ms threshold |
| Native 2,000×2,000 image materialization | 30.19 ms; oversized capture rejected separately |
| Native high-entropy 600×600 image | 51.42 ms; capture rejected above 768 KiB |

The rejected 16,384-unit window took about 5,983 ms for the long-line flow and the
first Tab reached approximately 704 MiB peak working set. Reducing the text window
to 2,048 UTF-16 units and geometry scan to 4,096 code points addressed that measured
problem. The final first Tab peak is **348,048 KiB (about 340 MiB)**. Other reported
process peaks are Browser 249,872 KiB, GPU 166,240 KiB, Utility 49,568 KiB and second
Tab 133,904 KiB. These are per-process peaks from Electron, not a simultaneous
aggregate RSS peak. The worker heap budget is not a whole-process memory ceiling.

The embedded-image fixture establishes a concrete need for a named ≤8 MiB Notebook
byte transport in R4a2. Generic text preview should keep its current limit. The
[module guide](../../../../app/qualification/notebook/README.md) records all
document, output, image, capture, text-window and observation limits.

## Visual review and scope limits

Native pointer and keyboard scenarios used actual controls; final screenshots use
the exact reader's `webContents.capturePage()`. Author inspection of the captures
found the composer and main actions reachable at compact size and native zoom.
The controls are deliberately exposed for qualification; their density, raw
offset fields and source hashes are not a final product UI commitment. R4a2 must
reuse the existing compact Pane and composer patterns.

- [Text capture with draft](source-capture.png)
- [Scaled image region and captured evidence](image-region.png)
- [Captured evidence after source changes](earlier-version.png)
- [900×650 reader](compact.png)
- [Native 150% zoom](zoom-150.png)

Earlier reports remain as `previous-*.json`. The rejected zoom screenshot is kept
as `zoom-150-playwright-crop.png`; it is not accepted visual evidence. The
[architecture review](architecture-review.md) classifies the generation defect,
scroll-test defect, screenshot issue and rejected performance candidate.

This is a required-field reader profile, not complete nbformat schema validation.
Markdown is basic and exact ranges use Source. Raw HTML, SVG, JavaScript, widgets,
remote resources, animated PNG and JPEG EXIF/orientation are unsupported. Native
image decode is synchronous and not covered by an interruptible worker deadline.
Large observation scopes can be partial. Semantic chart points, Notebook editing,
execution, persistent attachments, actual Agent delivery, renderer crash recovery,
signed packaging and other platforms have not been qualified by this stage.

The result supports proposing R4a2 integration; it does not authorize that next
stage. See the [next ownership sketch and review gate](../r4a1-notebook-qualification.md).
