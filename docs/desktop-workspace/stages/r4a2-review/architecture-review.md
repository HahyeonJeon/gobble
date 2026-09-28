# R4a2 — ownership and API review

Reviewed by the implementation author, sequentially after construction; this is
not an independent review. Subject: the R4a2 production source listed in
`source-subject.json`, using the preserved pre-stage `baseline.json`.

| Owner                                               | Owns                                                                             | Must delegate                                                     |
| --------------------------------------------------- | -------------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| Go ProjectService                                   | Registered file containment, exact bytes/hash, bounded response                  | Notebook interpretation to Main; Run/Pipeline execution to Gobble |
| `main/notebook`                                     | Private parser worker, source lifetime, text projection, native decoding/capture | Public resource identity and ready authorization to Workspace     |
| `workspace/notebook-views`                          | At most two visible source hosts; refresh, replacement, disposal                 | Parsing to NotebookHost; attachment durability to evidence        |
| WorkspaceController / RenderSession                 | Active Project, Surface identity, ready generation, selection/navigation commits | Bytes and gestures to their respective owners                     |
| Public Notebook contracts                           | Exact versioned target, public projection, coordinate units and evidence shape   | Parsing raw nbformat to Main; authorization to Controller         |
| Notebook renderer components                        | Cell presentation, bounded text windows, local image gesture/Blob lifetime       | Selection persistence and capture to the existing command path    |
| EvidenceCapture / EvidenceStorage / EvidenceService | Immutable publication, draft/recipient authorization, preview, explicit Send     | Current source decoding to the Notebook adapter                   |

## Decisions and corrections

- A Notebook is a file Resource with cell and output addresses. It does not create
  cell Panes, new Projects, another draft, a second attachment store or an execution
  controller. Basic Markdown is passive text rendering; unsupported outputs retain
  their original slots. Saved execution counts do not assert current execution.
- `NotebookTarget` uses explicit schema version 6. The contract includes Project,
  resource, byte revision, profile, cell identity, output MIME/digest and UTF-16 or
  natural-image coordinates. Main resolves exact raw/display boundaries before
  accepting a selection; no source search or repaired identity occurs.
- The named `notebookImage` bridge checks sender, active Project, origin and ready
  acknowledgment before reading the retained source. Review found and closed the
  generic `files.read` path for Notebook bytes, matching PDF's ownership boundary.
- The worker and host are promoted into production-owned files. Temporary harness
  receipt types and address helpers were removed. No production import reaches the
  qualification directory. Parser/target tests reuse frozen synthetic fixtures only.
- TypeBox is pinned as a direct desktop dependency for the private worker-reply and
  target validators. The dependency test permits it only in those two Main files;
  public schema and process-boundary checks remain in place.
- The Notebook View disables attachment actions while a replacement selection is
  being saved. A native scenario exposed that otherwise a rapid Select → Add could
  attach the previous source range. Feedback resets when the target changes.
- Refresh revokes local readiness immediately, while Main retires replaced sources.
  Native assertions wait for the new revision/selection outcome, not a transient
  pre-refresh ready attribute. Existing captured bytes never change during Refresh.
- Local draft preview is addressed by active Project + attachment ID. It validates
  the stored capture and rechecks renderer/draft membership after asynchronous I/O;
  no account, source reread, arbitrary file path or direct blob hash is accepted.
- Renderer loss invalidates leases before an explicit native Reload workspace choice.
  The test simulates that choice and verifies the replacement renderer directly;
  Playwright's old crashed Page handle is not treated as the restored renderer.
- Workspace v12 is frozen before v13 is introduced. Historical schema bundles through
  v14 remain byte-identical; the generated bundle is v15. Shared tool input stays v8,
  with exact comparison against the saved tool schema. Notebook Agent pointing is
  explicitly unsupported here. Existing evidence hashes survive migration.

## Practical limits

This stage validates saved reading and User evidence. Main native image decoding is
bounded by the qualified 4 MP input, not a separate image process. Worker parsing
has a five-second deadline and two-worker admission with a 128 MiB old-generation
limit; these are work/admission constraints, not a whole-App memory guarantee.
Text evidence includes source metadata and raw quote inside the existing 64 KiB cap.
Oversized evidence is refused without resampling or changing the draft. Cell pages
are persisted; fine scroll offsets and local folded/image state are not new storage.

All provider tests use a test-local Codex executable. They prove addressed message
content and existing Send routing, not a real Agent's interpretation. Agent visible
scope, marks, questions and Show/Return remain R4a3. Jupyter, editing/execution,
packaging and other operating systems remain outside this checkpoint.
