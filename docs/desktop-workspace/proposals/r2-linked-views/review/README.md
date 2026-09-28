# R2 design-sketch verification

Date: 2026-09-08. This record covers the synthetic interactive sketch only. No application implementation, model provider, statistical analysis, actual file loading or migration is exercised.

- [Behavior and layout results](sketch-checks.json): seven scenario groups, six width/theme combinations, no page-script errors. Installed Google Chrome is launched headlessly in an isolated temporary profile through the repository's existing Playwright dependency.
- [Production source boundary](source-boundary.json): all 138 files under `app/contracts/src` and `app/desktop/src` match the [starting hash manifest](baseline.json).
- [Verification script](verify-sketch.cjs): expects the visualize skill's standalone wrapper for `r2-linked-views.html` served at `http://127.0.0.1:8768`. Run with Node from any working directory. The script inspects the wrapper's sandboxed frame and closes all browser contexts it creates.
- Screenshots: [light desktop](light-1024.png), [dark medium](dark-736.png), [light narrow](light-360.png), [dark narrow chat](dark-360-chat.png), [source changed](source-changed.png).

The sketch is stored in the current task's visualization directory. It loads the pinned D3 7.9.0 display library; the source data is synthetic and embedded. Its `Send demo` action generates a labelled local simulated reply. It makes no Agent or data-service calls.

Review corrections included preserving keyboard focus during table updates, independent draft attachments, binding each message button to its own reference, preserving an already open reference view when a new message arrives, and avoiding axis/viewport changes while temporarily following a reference. Pointer controls and native table controls expose the same selected members. Plot labels are measured and optional overlapping labels are removed.

Automated sketch checks are evidence of the represented interaction, not representative-user usability or proof of the future Plotly implementation. The user's design approval and the R2a library/contract investigation remain the next checkpoints.
