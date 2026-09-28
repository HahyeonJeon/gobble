# R3c proposal — verification and visual review

Date: 2026-09-08. Subject: generated concept and HTML sketch, not production PDF support.

## Evidence

- [13 browser scenario checks](sketch-checks.json): all passed.
- JavaScript syntax check passed; HTML/CSS/JS formatting passed.
- Browser error/warning log was empty at completion.
- [373-file baseline](production-baseline.json): all recorded App/contract/service/engine files remained byte-identical. No production source, installed dependency, Workspace schema or account state was changed by this proposal.
- [Final proposal file digests](proposal-files.json) identify the reviewed artifacts.

The earlier R3b3 full App result remains 263 unit/contract tests and 43 native Electron scenarios. It was not rerun and is not evidence for PDF decoding. No native PDF, provider, performance, security qualification or representative-user test was performed in this proposal.

## Browser checks

The sketch was controlled through the in-app browser. The checks cover:
1. Text attachment does not auto-send; Show/Return restores page, selection, attachment and an unsent note.
2. Concept B exposes the persistent page rail at desktop width.
3. Historical capture shows only the retained region.
4. Missing capture explains unavailability instead of substituting current source.
5. Scanned mode disables text and supports keyboard region movement/resizing.
6. The 900-pixel shell has no document-level horizontal overflow.
7. Compact Discuss switches to Chat with the same attachment.
8. Native HTML text drag exposes Discuss.
9. Pointer drag draws a region.
10. Escape cancels region mode.
11. Next-page bounds and whole-page selection identify the correct page.
12. Concept B collapses its page rail at compact width.
13. Compact Pages opens temporary navigation.

Controls and states were inspected at 1280×840, 900×650 and 736×650. These are browser viewport checks; they do not establish Electron window behavior or accessibility support for real PDFs. The browser viewport override was reset afterward.

## Visual evidence and corrections

| Evidence | Review result |
| --- | --- |
| [Inline reader, 1280×840](inline-1280.png) | Reading space and right Chat dominate; selection attachment uses the existing composer. Main reading actions remain visible without hover. |
| [Persistent page rail, 1280×840](page-rail-1280.png) | Alternative changes navigation hierarchy and uses additional page width. A remains the recommendation. |
| [Agent reference, 1280×840](reference-1280.png) | Distinct author mark and explicit Return; the User selection/attachment remains separate. |
| [Captured region](captured-region-1280.png) | Corrected the initial misleading full-page historical presentation. Saved evidence now displays only the sample region and explicitly identifies missing surrounding context. |
| [Region selection, 900×650](region-900.png) | Toolbar, visible selection and Discuss stay reachable. Chat scrolls independently above its composer. |
| [Workspace, 736×650](workspace-736.png) and [Chat, 736×650](chat-736.png) | Workspace/Chat switching preserves the attachment; the two central Panes remain available in Workspace. |

A further review correction makes Concept B's rail temporary at compact width. The owner can reproduce it by choosing B and shrinking the window, then opening Pages.

The generated image is a concept illustration. Its synthetic research content and account name are not product data or requirements. The HTML sketch is the precise interaction example and labels simulated behavior.

## Limits and owner decision

The sketch uses ordinary HTML and a normalized demo rectangle. That geometry is deliberately not the proposed PDF user-space implementation. Password entry, actual PDF text-layer fidelity, crop/rotation accuracy, parsing limits and decoder isolation await R3c1.

Owner walkthrough remains pending. This proposal recommends A and requests approval for **R3c1 qualification only** before User workflow integration. No subsequent View family, release, commit or push is implied.
