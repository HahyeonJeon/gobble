# R3b1 renderer qualification

Isolated synthetic Electron harness. It is not imported by the product and has no Project, Agent,
file-writing or execution capability. Both candidates consume the same dependency projection.

Run from `app`: `node qualification/run-dependencies/qualify.cjs`.
The script installs exact candidate dependencies into its own temporary directory, records their
lockfile, builds under the production asset handler/CSP, runs the native scenarios, and removes
its temporary installation. No graph dependencies enter the product package or lockfile.

Candidates: minimal HTML buttons + SVG edges with a bounded topological layout;
React Flow 12.11.6 + Dagre 3.1.1 (both MIT). React/ReactDOM come from the App's installed 19.2.8.
This is a source-checkout qualification on the current macOS/Electron tuple, not an installed
application, screen-reader study or release claim. Measured limits and the decision are recorded
under `docs/desktop-workspace/stages/r3b1-review`.
