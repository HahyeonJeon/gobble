# R3c3 ownership and design review

Request R3c3-2026-09-08. Reviewed the accepted PDF implementation across contracts, Electron
processes, React presenters, source storage and the pinned provider boundary.

| Concept / owner                                  | Responsibility and API boundary                                                                                                                                                                                                                              |
| ------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Project / Go service                             | Registered files and execution-resource identity. A PDF is a versioned file Resource; opening it does not create an engine Run. No engine or pipeline ownership changed.                                                                                     |
| Pane / WorkspaceController                       | A layout slot containing Surface instances. Existing User-protected opening rules apply to Agent-created PDFs. One Project writer owns durable changes.                                                                                                      |
| PDF Surface / PdfViews                           | A reader instance has independent page, scale, rotation and fit settings. It retains a bounded source and decoder only while visible. At most two interactive hosts plus one short-lived exact-reference host.                                               |
| Painted page / RenderSession                     | A receipt identifies one renderer session and current Surface generation. A viewport report binds a clipped PDF rectangle to it. Scroll generations are ephemeral. Receipt checks inspect identity without cloning raster bytes.                             |
| Observation / pdfObservation + ObservedReads     | Pure page/region validation produces a target from actual returned content. The turn-local observedReadId authorizes only a contained, still-current pointer. It is neither persistent evidence nor execution authority.                                     |
| Evidence / PDF capture adapter + EvidenceStorage | A capture freezes PNG bytes, exact PDF source/page/model, transform and crop. Existing limits and manifests apply. Question evidenceId identifies a turn-local captured candidate; it cannot substitute for observedReadId.                                  |
| Authored reference / SharedReference             | A persistent User or Agent pointer with target, author and note. It contains no original source bytes, creates no automatic message and changes no User selection. Main assigns Agent authorship.                                                            |
| Temporary Show / ObservedReferenceViews          | User explicitly asks to inspect a reference. An exact-source page read leaves the interactive decoder and base React reader intact. Return restores base navigation and local reader state. A source mismatch never remaps the target.                       |
| React presenters                                 | PdfView owns reader gestures; usePdfViewport reports geometry; PdfMarks draws separate authored overlays; PdfReferenceContent displays temporary read-only content; PdfSurface preserves the base reader. Chat remains the single discussion/composer owner. |
| Codex adapter / codexToolSchema                  | The published Draft 7 target retains a fixed numeric tuple. The pinned provider receives an equivalent homogeneous array with the same min/max length. Heterogeneous/variadic tuples fail closed. Main's original validation remains authoritative.          |

The new behavior extends existing source/observation/evidence owners. It adds no viewer marketplace,
PDF editor, execution API, separate discussion window or parallel persistence store. Current APIs
remain transport-independent enough for a later MCP adapter, but this stage adds no MCP server.

Resolved review findings:

- PDF questions initially attempted the generic source-preview reader, which intentionally rejects
  raw PDFs. A source-verification API now checks the exact original revision without another decoder.
- Pinned Codex rejected tuple-form schema registration. A provider-specific adapter now preserves
  numeric coordinate length and semantics. Unit tests compare accepted/rejected values and reject
  mixed tuples; the actual runtime accepted registration and completed image-based discussion.
- Repeated viewport checks would clone the retained raster through snapshot. Identity validation
  now uses a private ready-lease lookup; callers requesting data still receive a clone.
- The full pointer note covered selected PDF text. The final overlay shows only its author outside
  the selected rectangle; the note stays in Chat. Native screenshots verify the resulting layout.

Limits remain explicit: no semantic text targets, OCR, password handling, form/annotation appearance,
off-page background Agent browsing, CSV chart creation or automatic source revision substitution.
