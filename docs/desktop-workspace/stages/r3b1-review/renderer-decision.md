# R3b1 renderer decision

Decision: use the minimal HTML/SVG adapter for the initial R3b2 User navigation. Keep the
renderer behind semantic group/directed-pair events; add no graph library to the product.
This decision concerns the bounded Run dependency overview. It does not introduce CSV plotting,
scientific graph editing, a viewer plugin registry or a reusable graph editor.

## Evidence and judgment

Both candidates use `projectDependencies`, identical synthetic topology and the production
asset handler/CSP in sandboxed Electron. React Flow's [read-only input options](https://reactflow.dev/api-reference/react-flow)
and [keyboard/accessibility APIs](https://reactflow.dev/learn/advanced-use/accessibility) justified
qualification; [Dagre](https://github.com/dagrejs/dagre) supplies directed graph layout. The pinned
harness uses React Flow 12.11.6 and Dagre 3.1.1, both MIT, in a temporary installation. Their
[resolved package lock](renderer-package-lock.json) is evidence, not a product dependency change.

The [machine-readable result](renderer-qualification.json) records the exact Electron/Chromium/OS
architecture, secure window preferences, scenario outcomes and source hashes. The host is macOS
26.5.2 arm64, Electron 44.2.0 and React 19.2.8. These are one-machine technical measurements, not
performance guarantees or a usability study. The reported bundle size combines both candidates;
it must not be interpreted as either candidate's incremental product cost.

HTML/SVG already supplies the bounded read-only layout, native focus/scroll and exact target
events needed for this checkpoint. Its smaller integration responsibility fits the owner's
request to simplify. React Flow also passed the technical checks; avoiding its dependency is a
scope/maintenance decision, not a claim that it cannot meet the requirements.

## Native scenarios

For both candidates: 8 and 80 groups; exact keyboard group and directed-pair selections; Delete
leaves topology intact; an 81-group input triggers a limit fallback; at 900×650 and native 150%
zoom the selected-target summary remains reachable; 15 unmount/remount cycles after garbage
collection stay under the 15 MiB retained-heap growth guard. Type checking passes, and the
harness reports no page errors or CSP violations. Native captures wait for two paint frames.
No Project files, account, Agent turn or execution control is exposed by the harness.

## Visual review and remaining work

- [HTML/SVG, 8 groups](renderer-simple-8.png): labels and the focused dependency button are readable;
  the selected directed connection is emphasized. The unused space reflects a qualification fixture.
- [HTML/SVG, 80 groups](renderer-simple-80.png): many crossing lines require navigation and the
  equivalent dependency list. Do not treat the canvas as a complete overview of a large Run.
- [React Flow, 80 groups](renderer-flow-80.png): automatic positioning distributes the graph but
  does not remove the need for navigation/list access.
- [HTML/SVG, 150%](renderer-simple-150.png) and [React Flow, 150%](renderer-flow-150.png): the target
  summary stays reachable. The Flow candidate's default canvas controls overlap a node; controls
  would need dedicated space if this adapter were adopted.

R3b2 must design and test camera/Fit/zoom, selected-target reveal, source-relative list navigation,
compact group details and the User's exact group → instance/attempt → logs path. The simple harness
currently demonstrates native scroll/focus, not a finished per-view camera. R3b3 owns temporary
Agent Show/Return and real Agent observation. Screen-reader behavior and representative owner
interaction have not been verified by this harness. Neither candidate is imported into the live UI.

## Failure classification

Early harness runs exposed missing TypeScript props, a duplicate synthetic edge and a reload racing
initial navigation. These are harness/test defects; their failed logs/JSON remain in this directory.
The final harness fixes those inputs and verifies exact target fields, not just a selector's kind.
A later exact-target assertion initially reused the 8-group destination in the 80-group fixture;
the expected fixture-specific directed pair was corrected and the failed report retained.
Screenshots were stabilized after a capture initially lagged the selected-target DOM by one paint.
No product CSP relaxation, test disabling or production graph package was needed.
