# P5A implementation record

Owner approved the P5A proposal on 2026-09-12. Implementation mode. Baseline is the inherited dirty P4 tree at HEAD 7fe3d5c; do not reset it. Target: local macOS arm64 Electron 44, existing local native service and qualified P4 engine. No engine execution behavior changes.

## Contract refinement

The immutable RunFollowUp record is embedded in the proposal's atomic retained record, rather than stored in a second independently linked journal. Its identity is the containing proposal ID. It records the sent submission and bounded evidence manifests (metadata and content hashes only), validated origin launch/Run/preparation and origin input digest. Main derives it from the actual authorized submission, never Agent arguments or unsent drafts. Native validates Project/Pipeline/Run association. The current adopted proposal supplies this same provenance to the initial launch record before data-copy work. The existing launch checksum protects it. No second commit is necessary at adoption/admission: both relations follow the existing authoritative pointers. This refines storage mechanics without changing the approved User flow.

CRUD / process chain: create metadata in Main's proposal tool handler only after a sent submission and scoped source read; native checks origin and writes proposal; read through existing named list routes; native reads Current's proposal and copies origin into new launch review; no deletion or extra state writer. Main evidence service retains and verifies captured bytes. Renderer consumes metadata through existing preload methods and opens original captured previews; no new generic IPC or Agent tools. Current navigation is a User action, not authoring permission. No Workspace schema changes are needed; contract bundle changes for additive service metadata. Existing frozen schemas remain untouched.

Affected paths: new contracts run-followup.ts; current pipeline-proposal/run-launch contracts and bundle exporter; native run_followup.go and proposal/launch integration; Main pipeline-review follow-up builder/host and service validators; renderer run-feedback components with existing Run/Chat/review/launch callers; scoped tests and this stage's docs. No OS permissions, packaging, engine Resume or runtime image changes.

## Verification request: p5a-feedback-20260912

Development construction: type/schema/build/format. Testing (electron-testing): native association/replay/current-pointer tests; TypeScript invalid origin and Main authority tests; existing proposal/evidence/launch regressions; Electron synthetic fixture workflow and visual review. Pass means exact origin survives proposal and fresh launch, no automatic execution, old evidence remains addressable, unavailable evidence fails visibly. Mock Agent and actual engine evidence must remain separately labeled. Runtime qualification uses synthetic owned files only. Representative-user usability and P5B remain later.

## Ownership and lifecycle

```mermaid
sequenceDiagram
    actor User
    participant UI as App shared Flow and Chat
    participant Main as App Main evidence and proposal host
    participant Native as Native proposal and launch service
    participant Engine as Gobble engine
    User->>UI: Select exact Run task/log and discuss
    UI->>Main: Send captured evidence + explicitly chosen Current
    Main->>Main: Verify sent manifests and stored capture bytes
    Main->>Native: Supported proposal + derived origin metadata
    Native->>Native: Validate same Project/Pipeline/Run; persist proposal atomically
    Native-->>UI: Checked Change spotlight + original evidence
    User->>UI: Adopt proposal
    UI->>Native: Existing exact adoption operation
    User->>UI: Prepare and check new analysis
    Native->>Native: Copy adopted proposal origin into initial launch record
    Native-->>UI: Checked input identity, new destination, all steps run
    User->>UI: Start new analysis
    Native->>Engine: Existing durable admission
    Engine-->>UI: New Run status; earlier Run remains intact
```

RunFollowUp is provenance, not an execution dependency, cache key or diagnosis. Its containing proposal identifies the suggested change; Current's existing adoption pointer identifies the accepted change; the new launch record freezes that relation before admission. An accepted proposal without sent Run evidence remains an ordinary proposal. Historical evidence can describe an older design; editing still targets only the explicitly selected checked Current.

The renderer's run-feedback directory owns presentation and User navigation. Main's pipeline-review builder owns deriving provenance from the current sent submission. Native's run_followup.go owns association checks and retrieving the adopted relation. Existing evidence modules own capture coordinates and content; no new selection or evidence abstraction is introduced. Existing proposal, preparation and launch modules retain their state machines.
