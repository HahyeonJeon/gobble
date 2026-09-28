# P5B-3 — Native continuation transport

Owner authorized this increment on 2026-09-27 after reviewing the remaining P5B plan. Gobble's continuation protocol is now connected through the native service, Electron Main, trusted IPC and preload. Visual Flow/Chat integration remains the next stage. No engine image was replaced or live scientific analysis started.

## Ownership

| Owner | Definition and responsibility |
| --- | --- |
| Gobble | Authority for saved-design eligibility, exact reuse, admission receipts, task attempts and execution |
| Native service | Durable controller intent for one exact Project/Run/review; fixed runtime invocation, acknowledgement recovery and epoch-specific Stop |
| Electron Main | Typed User transport with service capability checks and strict response association |
| Preload | Closed, frozen bridge to trusted Main channels; no paths, commands, arbitrary graphs or tool selections |
| Renderer, next stage | Shared Run Flow, selected review detail, explicit confirmation and immutable discussion references |
| Agent | Discussion and design proposals through the existing shared-view system; no continuation execution tool |

A **review ID** identifies a retained read-only check. The engine's **review digest** binds its exact evidence. A **confirmation request ID** identifies the User's execution intent. A **receipt** proves that Gobble admitted that intent. An **execution lease** is the Stop target for that ownership interval. Initial Start and continuation records remain distinct.

```mermaid
sequenceDiagram
  actor User
  participant Main as Electron Main
  participant Native as Native service
  participant Engine as Gobble
  Main->>Native: List retained requests / check support
  User->>Main: Check saved Run
  Main->>Native: Project + launch review + Run reference
  Native->>Engine: Read-only continuation review
  Engine-->>Native: Exact evidence + digest
  Native-->>Main: Retained review
  User->>Main: Confirm exact review
  Main->>Native: Review ID + digest + request ID
  Native->>Native: Save intent before dispatch
  Native->>Engine: Start one validated continuation controller
  Engine-->>Native: Queryable admission receipt
  Note over Main,Engine: Restart restores the same record; refresh never creates or starts a controller
```

## Implemented behavior

- New fixed CLI verbs: `prepared-capabilities`, `prepared-continuation-review`, `prepared-continue`, and `prepared-continuation-receipt`. Execution inputs come from a host-mounted bundle; these verbs never load Project source. Review and receipt queries do not dispatch work.
- The native service checks both the exact installed image and a versioned capability response. The future qualified image must advertise `io.gobble.continuation.scope=single-end-trim-fastqc-v1` and respond with launch schema 2 / continuation version 1. Only then does a **new initial Start** use schema 2. Existing images retain schema-1 Start and inspection. Existing Runs are never converted.
- A read-only check is asynchronous and retains immutable bounded evidence in the native profile. Interrupted checks become `interrupted`; no execution follows. The profile retains at most 64 reviews per Project; the engine's separate 32-continuation-per-Run limit remains unchanged.
- Confirmation accepts only a stored review's exact digest and one request ID. Native persistence precedes controller creation. Repeated confirmation returns the existing operation; conflicting requests cannot replace it.
- Controller inspection verifies image, command, environment, label, mounts and isolation before Start. Continuation uses the original Run directory and sealed payload, with its own intent bundle and controller identity. The initial Start receipt/Stop record is not overwritten.
- List/read restoration performs no runtime action. Refresh only queries the same receipt and current epoch. Missing or interrupted acknowledgements never generate a replacement controller or Start.
- Rejection requires evidence: after a verified controller has exited, a second exact receipt query must prove that it admitted nothing. Only then is a fresh check allowed. Missing controller/status/receipt observations remain `unknown`; this slice deliberately does not redispatch an unconfirmed intent, even if it might have stopped before initial dispatch.
- A confirmed receipt remains authoritative across later observation failures. Execution observations are retained snapshots, not promises of live status. Stop binds to that continuation's receipt lease and the observed current epoch; Gobble independently refuses a delayed Stop addressed to an earlier owner.
- Main rejects malformed, cross-Project, cross-Run, cross-review and wrong-confirmation responses. The service capability check prevents calls against older native services. HTTP authentication and the existing trusted-window/main-frame IPC checks guard the new routes.

## Source organization

- `cmd/gobble/prepared_continuation.go`: fixed prepared wire entry points.
- `internal/appservice/continuation_records.go`: checksummed profile records and exact association to the original launch.
- `continuation_operations.go`: check, confirm, observe and Stop lifecycle; workers own independent snapshots.
- `continuation_runtime.go`: engine capability, bounded wire validation and fixed read-only queries.
- `continuation_routes.go`: authenticated HTTP request shapes.
- Existing `launch_runtime.go`: one fixed controller descriptor shared by Start and continuation, with read-only inspection separated from Start. Schema-2 observations expose current epoch separately from origin.
- `app/contracts/src/run-continuation.ts`: canonical TypeBox schemas and semantic validation. `schema/v27.json` is generated by `contracts/scripts/export-schema.ts`; older bundles remain intact.
- `app/desktop/src/main/service/run-continuation.ts`: concrete typed transport following the existing service layout; registered through Main IPC and preload. No new inheritance, framework, journal or generic retry layer.

## Verification and next stage

[Verification](verification.md) records focused native/CLI/Main checks, native race tests, all 376 App unit/integration tests, type/schema/build checks and the actual Electron bridge/restart smoke test. Native runtime tests use deterministic Docker command responses. They do not qualify the real tool image or pending-execution restoration through the final visual interface.

Next owner checkpoint: connect the approved Run Flow and Chat to these operations. Show reuse/restart/blocked decisions, selected detail and saved references; make User confirmation explicit. Then qualify a new exact engine image with real Trim/FastQC, Stop/Continue, application restart and retained logs. UI layout, accessibility, reference retention and real-tool end-to-end evidence remain those stages' responsibilities.
