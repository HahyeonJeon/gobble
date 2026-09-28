# R3b evidence and alternatives

Reviewed 2026-09-08. This is design research, not implementation or usability proof.

## Local facts and their consequences

| Evidence                                                                      | Verified fact                                                                                                                                                 | Consequence                                                                                                                                          |
| ----------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| [Monitor projection](../../../../internal/engine/monitor.go)                  | Each runtime row has `identity`, authored `task_id`, attempt, template/expanded flags and reported state. Monitor edges are deduplicated authored Task pairs. | Group rows by `task_id`; never draw an edge between arbitrary sample instances or infer sample correspondence.                                       |
| [Run presenter](../../../../app/desktop/src/main/service/run-presentation.ts) | Only the first 1,000 instances are exposed; dependencies address authored IDs. Missing edges currently become an empty array.                                 | Preserve missing-versus-reported topology before showing an empty dependency graph. Group counts and completeness need explicit scope.               |
| [App service](../../../../internal/appservice/runs.go)                        | Registered Runs are read through their pinned runtime. The service returns the monitor observation and revision, not a registered Plan resource.              | R3b is an observed Run dependency overview. A saved Plan viewer needs its own identity, import/read and availability contract.                       |
| [DAG inspector](../../../../internal/engine/inspect.go)                       | `inspectDAGView` exposes the saved Plan DAG separately.                                                                                                       | Do not combine a separately read DAG with live statuses without a proved coherent revision. No engine scheduler change is justified by this UI work. |
| [R3a3 result](../../stages/r3a3-agent-observation.md)                         | Exact task/log capture, explicit Show/Return and real Agent pointers are implemented. Native tests use controlled fixtures.                                   | Extend those owners and evidence paths; retain the compact Run/log/Chat shell. This is not representative research-user testing.                     |

## Scope options

1. **Run dependency exploration — recommended next.** Completes the existing failure-discussion
   path with upstream/downstream context. Main risk: confusing authored dependencies with instance
   execution or inferring a cause from an arrow. Explicit targets and observational counts address it.
2. **Document/report reading next.** Markdown/PDF would broaden file coverage immediately, but
   it would leave the existing Run dependency facts inaccessible. PDF text/page geometry and HTML
   report isolation remain separate, substantial contracts. This can move ahead of graphs if the user
   prioritizes reading papers and reports.

Notebook execution stays R4. It still needs document/session/kernel ownership and a live integration;
adding an unstructured notebook JSON preview would not satisfy that requirement.

## UI concepts

A. **One Run Surface with Tasks / Dependencies modes, logs below — recommended.** One selected
semantic target per Surface, one source observation and mode-specific camera/search state. Selecting
a group exposes its observed instances inside the Run view; choosing an instance enables Open logs.
A mode change does not secretly choose a different target. Short windows can maximize the Run or
use the existing pane switcher.

B. **Separate graph and instance-list Surfaces.** More space for each representation and useful for
persistent graph/table comparison. It consumes both central panes before opening a log, and requires
another explicit shared-state relationship. Prefer it later if real tasks show simultaneous graph/list
comparison is more valuable than immediate log context.

These differ in navigation and state ownership, not color. Both are represented in the sketch.

## Rendering candidates — official sources checked 2026-09-08

- [React Flow accessibility](https://reactflow.dev/learn/advanced-use/accessibility) documents
  focusable nodes/edges, keyboard selection and focus-driven panning. Its defaults include editing
  language/actions, so a read-only viewer must explicitly disable creation, dragging and deletion and
  supply appropriate English instructions. Library features do not prove the App is accessible.
- [React Flow layout guidance](https://reactflow.dev/learn/layouting/layouting) separates rendering
  from layout and compares Dagre and ELK. Dagre has a documented sub-flow limitation. R3b's flat
  authored-group graph avoids nested sample sub-flows; no instance-expansion tree is proposed.
- [React Flow API](https://reactflow.dev/api-reference/react-flow) exposes controlled state and
  interaction configuration. Candidate boundary: one renderer adapter emits semantic IDs only.
- [Dagre repository](https://github.com/dagrejs/dagre) is a candidate for directed layout. Layout
  coordinates cannot become execution facts or reference identities.

**Provisional choice:** evaluate a read-only React Flow + Dagre adapter against a minimal HTML/SVG
renderer in R3b1. Both consume the same pure graph projection. No dependency has been installed or
version selected. Pin and qualify the exact versions under Electron's CSP, keyboard behavior and
small-window limits before promotion. Keep the dependency behind one adapter. If it cannot honor
read-only behavior or represent flat dependencies clearly, use the simpler renderer or return to design.

## Evidence limits

Code and official documentation support the identity/ownership claims. The sketch can test the proposed
flow and legibility, not graph scalability, library behavior, screen-reader usability, source integrity,
provider delivery or general research-user performance. The project owner is invited to try one scenario;
that feedback applies only to this workflow and participant. No recruitment, messages, telemetry or
background monitoring is authorized here.
