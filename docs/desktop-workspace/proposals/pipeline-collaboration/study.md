# Pipeline collaboration — Study

> Status: findings and recommendation, 2026-09-09. Consumer: project owner and the
> subsequent design/planning work. This study does not authorize implementation.

## Study contract

| Field             | Bound value                                                                                                                                                                                |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Question          | How should Gobble and Gobble App support Agent-authored pipeline design, revision, execution and monitoring through shared views?                                                          |
| Purpose           | Return the product to its pipeline collaboration purpose before adding more viewer/editor capabilities.                                                                                    |
| Decision criteria | Clear authority; exact source/Plan/Run identity; useful shared discussion; recoverable effects; proportionate abstractions; existing behavior preserved.                                   |
| Scope             | Current Go DSL/CLI/engine, Go App service, Electron host/provider, contracts and shared workspace; focused upstream protocol/platform research.                                            |
| Evidence boundary | [Locked source set](subject.json), explicitly added dependency evidence, current owner direction and the primary sources below. Sampled architecture review, not a complete product audit. |
| As of             | 2026-09-09; local Codex protocol is pinned to 0.153.4. Current online documentation is not proof of support in that binary.                                                                |
| Output boundary   | This file reports findings. [Design](design.md) chooses a proposed solution; [Plan](plan/plan-index.md) decomposes it. Studied source remains unchanged.                                   |

## Conclusion

The existing product has a strong shared-observation foundation. Its implemented
App vocabulary stops at files, attached Runs, runtime snapshots and references.
The missing connecting workflow is **a pipeline definition, an Agent-authored change,
an exact source revision, a Gobble-validated Plan, an authorized execution and the
resulting Run**. Another editor or viewer family does not establish those links.

The original Go Pipeline API remains the authoring representation. Gobble App
should expose review and discussion of Agent-authored Go/config changes and
Gobble-produced plans. It should not create a second pipeline language, scheduler
or reuse classifier.

## Recommendation

1. Qualify the supported Linux runtime baseline and define the pipeline
   collaboration concepts before implementing effects. Record the native-root Go
   compile limitation separately; the host App service already builds and tests.
2. Introduce one bounded Agent change tool through the existing tool transport.
   The Agent supplies changes; a service-owned candidate records their exact bytes
   and base revision. Keep the App read-only for source authoring.
3. Add Changes and Plan as shared views in the existing central Panes. Reuse the
   right Chat, composer, references, evidence and protected User view state.
4. Add Gobble-owned execution admission and recovery before exposing Start/Stop/
   Resume. A local service receipt alone cannot make a launch exactly once.
5. Qualify one complete pipeline journey before returning to packaging or new
   scientific viewers.

First action: review the proposed concept/ownership diagram and the five bounded
implementation checkpoints. Reconsider the bounded change tool if real pipeline
changes repeatedly require an interactive coding shell; qualify a separate provider
authoring profile at that point rather than broadening the discussion profile.

## Evidence

| Kind                              | Finding and source                                                                                                                                                                                                                          | Consequence                                                                                                                                                                                       |
| --------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Verified local fact               | [Pipeline](../../../../pipeline.go), [Graph](../../../../graph.go), [BuildPlan](../../../../plan.go), [Run](../../../../run.go) are separate Go concepts. BuildPlan validates; Run constructs its own execution document.                   | Preserve these distinctions; an App Plan identifier needs exact execution admission, not a renamed Run snapshot.                                                                                  |
| Verified local fact               | [CLI driver](../../../../cmd/gobble/driver.go) compiles the imported package and calls `userpipe.Pipeline()` before choosing plan/validate/run.                                                                                             | Plan generation executes project Go code. Treat it as an isolated job, with distinct authorization from running analysis tasks.                                                                   |
| Verified local fact               | [Service capabilities](../../../../internal/appservice/service.go) advertise reads plus Project registration/Run attachment. [HTTP tests](../../../../internal/appservice/http_test.go) intentionally reject start/stop routes.             | New workflow is a product extension; current read-only behavior is not a regression.                                                                                                              |
| Verified local fact               | [Codex policy](../../../../app/desktop/src/main/codex/policy.ts) and [conversation binding](../../../../app/desktop/src/main/codex/conversations.ts) pin read-only, isolated discussion with shell disabled.                                | Agent instructions alone cannot deliver editing. Add a scoped effect capability with host enforcement.                                                                                            |
| Verified local fact               | [Reuse classifier](../../../../internal/engine/reuse.go) owns task reuse; [Stop](../../../../internal/engine/stop.go) is lease-addressed internally. The public Stop API cannot receive an earlier observed lease.                          | The App must not infer reuse from graph differences. Conditional Stop must be enforced inside Gobble.                                                                                             |
| Verified local fact               | [Existing architecture contract](../../../../.gobbi/projects/gobble/memory/design/architecture/project-workspace-contract.md) already anticipates base hashes, change sets, durable operations, launch correlation and expected-lease Stop. | Refine and implement existing direction rather than replace it with another architecture.                                                                                                         |
| Verified local fact               | Native Go test compilation fails at `internal/containerenv/runtime.go:113`: only a Linux-suffixed file defines `useProjectOwner`. Caller bytes match HEAD.                                                                                  | Record the host-test limitation separately. Linux/amd64 cross-compilation succeeds; README specifies container execution. Native root-library support is not a new prerequisite for App delivery. |
| Verified upstream fact            | Current [Codex App Server documentation](https://learn.chatgpt.com/docs/app-server) describes file-change events and server-requested approvals.                                                                                            | Native authoring is a future option; first qualify its complete protocol against pinned 0.153.4.                                                                                                  |
| Verified upstream fact            | [Git worktrees](https://git-scm.com/docs/git-worktree) share repository administration, including refs.                                                                                                                                     | A worktree is useful source isolation, not a security sandbox or an independent repository.                                                                                                       |
| Verified upstream fact            | [Electron security guidance](https://www.electronjs.org/docs/latest/tutorial/security) requires validating IPC senders and limiting exposed IPC.                                                                                            | Preserve the narrowed preload bridge. UI and Agent adapters should call the same service commands.                                                                                                |
| Verified upstream fact            | [Go build constraints](https://pkg.go.dev/cmd/go#hdr-Build_constraints) select files by target constraints and OS-specific names.                                                                                                           | The observed native unresolved symbol is target-specific; supported-runtime qualification must use Linux. It is not an observed Docker execution failure.                                         |
| Inference / design recommendation | A small source-change port plus immutable candidate identity uses the present transport without opening a general coding environment.                                                                                                       | Prefer this initial path; measure its usefulness in an actual multi-file pipeline change.                                                                                                         |

## Alternatives

| Alternative                                              | Benefit                                                                                  | Cost / decision                                                                                                                                                                         |
| -------------------------------------------------------- | ---------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Bounded Agent change proposals, service-owned candidates | Reuses current tools, exact base/result comparison, explicit effect scope, no editor UI. | Selected proposal. Must handle multi-file state, conflict recovery and useful errors; first authoring scope is registered text source/config.                                           |
| Native Codex workspace-write session                     | Rich coding tools and native diff/events.                                                | Defer until pinned protocol, read scope, command isolation, approvals and profile lifetime are qualified. Changing `sandboxMode` alone is insufficient.                                 |
| App text/Notebook editor                                 | Direct manual authoring.                                                                 | Excluded by owner direction. Existing readers remain useful communication surfaces.                                                                                                     |
| Visual graph editor or another Pipeline DSL              | Direct graph manipulation.                                                               | Excluded: duplicates Go composition and introduces another source of pipeline truth.                                                                                                    |
| Source hash plus existing CLI run                        | Small interface change.                                                                  | Insufficient: invoking the same Go source can yield a different plan; prepare in isolation and admit the exact engine-owned execution payload, with input/option checks before effects. |
| Service-only operation journal                           | Recoverable client acceptance.                                                           | Necessary but insufficient without Gobble launch correlation and conditional ownership admission.                                                                                       |

## Limits and uncertainty

- No production authoring, Plan admission or execution controls exist yet.
- Current upstream provider documentation is broader than the inspected generated
  local surface. Neither unsupported nor supported native authoring is inferred.
- An immutable source snapshot and prepared execution payload are not promises that datasets or arbitrary Go
  dependencies are reproducible. Capture declared bindings and disclose mutable
  data; reject unsupported ambient dependencies in the first validation profile.
- Multi-file promotion cannot be made atomically visible to arbitrary external
  writers with ordinary file replacement. The proposed single managed writer,
  journal and immutable execution candidate have a narrower guarantee.
- This is author self-review. Deep security, accessibility, scale, packaging and
  actual signed-in authoring need later evidence.
- Native Go compilation failed; focused App service and Codex/collaboration tests
  passed. See [verification](verification.md), including commands and limitations.

## Verification

Traceable source identity, preserved unaided review and focused executable evidence
are recorded in [architecture review](architecture-review.md) and
[verification](verification.md). No implementation approval or full-suite pass is
claimed. All interpretation and solution choices above are marked separately from
observed behavior.
