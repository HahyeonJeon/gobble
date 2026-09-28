# P5B-3 verification

Date: 2026-09-27 KST. Subject: the 29 paths in `changed-paths.json`, compared with `before.json`; final identities in `after.json`. All P5B-2 production hashes matched before work. Every baseline file remains. No staging, commit, branch reset, publication, image replacement, provider call or scientific task execution occurred.

## Environment

Host: macOS arm64, Go 1.27.1, Node 25.7.0, npm 11.10.1. Installed workspace dependencies: TypeScript 5.9.3, Vitest 4.1.11, Electron 44.2.0 and Playwright 1.63.0. Native service tests run on the host. CLI tests cross-compile with the host compiler for Linux amd64 and run in the existing `gobble-p2a1-toolchain:local` container, image ID `sha256:bccd458a724b795ac507e3dd0be43e8431269db2819fa544784d73dc751ba22b`. Containers have no network or Docker socket and a read-only test-binary mount. Go verification uses `GOTOOLCHAIN=local GOPROXY=off`. Existing installed dependencies are used; no install/update is part of this increment.

Actual Electron checks use a disposable test profile and Project. The folder chooser response is simulated; Electron Main, preload, IPC and the native service are real. Codex is pointed at an unavailable executable; no account or Agent inference is needed. Native continuation runtime tests use deterministic Docker command responses, not actual analysis containers.

## Results

| Check | Evidence and result |
| --- | --- |
| Baseline Go launch/preparation tests | `baseline-go.log`: pass, 2.585 s |
| Baseline Main launch tests | `baseline-ts.log`: 4 tests pass |
| Focused continuation native race checks | `native-terminal-final.log`: pass, 4.610 s; includes accepted/unknown/rejected lifecycle, exact controller and restart |
| Final affected native race suite | `native-race-final.log`: service 20.208 s, container launcher 1.454 s; service command compiles with no test files |
| Additional protocol/epoch regression | `native-protocol.log`: pass, 2.259 s; added only tests after the full native run, production sources unchanged |
| Retained receipt after observation loss | `native-retained-receipt.log`: pass; additional assertion on the final test source, production unchanged |
| All App unit/integration tests | `app-tests.log`: 40 files, 376 tests pass, 12.53 s |
| Actual Electron bridge/restart smoke | `electron-verified.log`: 1 test passes, 2.0 s |
| Linux CLI focused and parser/help regression | `cli-focused.log`, `cli-regression.log`: pass |
| App build and final native service rebuild | `app-build.log`, `service-build-final.log`: pass |
| TypeScript and generated schema | `typecheck-final.log`, `schema-check-final.log`: pass; TypeBox generates v27 without changing old bundles |
| Native vet and scoped CLI vet | `native-vet-final.log`, `cli-vet.log`: pass |
| Scoped Prettier, Go formatting, diff whitespace | `app-format-check.log`, recorded scoped gofmt check, `git diff --check`: pass |

Commands run from the repository root unless noted:

```sh
GOTOOLCHAIN=local GOPROXY=off go test -race -count=1 ./internal/appservice ./cmd/gobble-service ./cmd/gobble-container
GOTOOLCHAIN=local GOPROXY=off go test -race ./internal/appservice -run '^TestContinuationRejectsProtocolAndEpochMismatch$' -count=1
GOTOOLCHAIN=local GOPROXY=off go vet ./internal/appservice ./cmd/gobble-service ./cmd/gobble-container
GOTOOLCHAIN=local GOPROXY=off GOOS=linux GOARCH=amd64 go vet -composites=false ./cmd/gobble
GOTOOLCHAIN=local GOPROXY=off GOOS=linux GOARCH=amd64 go test -c -o /tmp/gobble-p5b3-cli.test ./cmd/gobble
docker run --rm --platform linux/amd64 --network=none \
  --mount type=bind,src=/tmp/gobble-p5b3-cli.test,dst=/cli.test,readonly \
  --entrypoint /cli.test gobble-p2a1-toolchain:local \
  -test.run 'TestPreparedContinuationCommands|TestCreationCommands|TestHelp|TestVersion|TestInvocationFailures|TestParse' -test.v
```

From `app/`: `npm run schema:generate`, `npm run schema:check`, `npm run typecheck`, `npm run build`, `npm run build:service`, `./node_modules/.bin/vitest run`, and `./node_modules/.bin/playwright test desktop/tests/electron/continuation-transport.spec.ts`. Prettier checks only this increment's changed App files. The CLI vet exception remains the previously documented inherited composite-literal warnings; it is not a clean whole-module vet claim. Whole-repository source-policy and emulated memfd packaging failures from P5B-1 were not rerun or repaired here.

## Behavior covered

- Exact saved evidence is retained without creating or starting an execution controller. Query mounts make the Run directory read-only.
- The controller can only be created after its exact intent is durably recorded. A wrong bundle mount prevents Start. A normal fixed controller yields one receipt and a current epoch observation.
- Lost create acknowledgement remains unknown. Reopening the profile, listing, repeated refresh and exact confirmation replay create no replacement controller. A later receipt resolves the same request.
- A validated exited controller followed by an absent receipt permits a new review. Unavailable evidence cannot masquerade as rejection. A receipt is preserved on later observation failures.
- Old Start records remain byte-equivalent after continuation Stop. Stop for the origin lease is rejected by the continuation API; the observed continuation lease is retained and settled independently.
- Earlier engine formats, absent capability, wrong capability version, missing/downgraded epoch, cross-Run review evidence, corrupt review digests and corrupt profile records refuse the operation.
- Main rejects unknown execution fields, conflicting action/attempt meaning, paths escaping the saved view, cross-Project/Run/review association, mismatched confirmations and mismatched Stop acknowledgement. Lack of native service capability prevents dispatch.
- HTTP requests require the existing private native token and closed input shapes. The new IPC routes use the existing trusted-window/main-frame check. No Agent tool was added.
- Real Electron exposes the frozen continuation bridge, rejects invalid confirmation at IPC, restores the Project across restart, reads the continuation catalog and leaves the Run catalog empty. This smoke test does not substitute for pending scientific execution restoration through the final UI; the pending-intent cases are native integration tests.

## Repairs and evidence limits

The first native race run (`native-focused.log`) exposed a real introduced aliasing defect: HTTP responses and workers shared an operation pointer. Workers now receive an independent serialized snapshot before starting; the subsequent focused and full race runs pass. Confirmation also refuses a still-active check worker, avoiding a confirmation being saved while its dispatch slot is occupied.

The first Electron assertion expected Runs to be a bare array; the established API returns a Project-scoped catalog containing `runs`, `candidates` and `truncated`. The test was corrected to assert that existing contract. Intermediate failure logs are retained; `electron-verified.log` is authoritative. One edit attempt used a root-relative path while already in `app/`; it changed no file and was corrected before the verified run.

No real continuation-capable engine image has been built or qualified. The original P4/P5A image and workspaces remain intact. Large-input hashing latency, live Docker failure handling, scientific outputs/logs, full pending-run Electron restart, shared-reference replay and accessibility remain the next visual/qualification stages' evidence obligations. A dispatch with unavailable acknowledgement intentionally remains unresolved rather than being retried automatically; broader recovery is outside this slice.

## Execution self-check and handoff

This is a Coding Execution self-check, not an independent review or approval verdict. Applied the current Coding Principles and the existing concrete service class convention from Coding OOP; no new inheritance or interface pattern was needed.

- **Project fit / affected surfaces:** implements approved native/Main integration only. New CLI verbs, capability negotiation, schema-2 Start selection, current-epoch observation, native storage/routes, generated contracts and Main/preload are consistent. Renderer and Agent toolset remain unchanged.
- **Structure / architecture / modularization / abstraction:** engine remains the reuse and admission authority; profile records own transport recovery. Runtime command construction is hidden behind fixed descriptors. Read-only controller inspection is separate from starting it. Main owns response validation, not execution decisions. There is no second scheduler or generic retry framework.
- **Data / API / parameters / naming:** review identity, confirmation identity, origin and current lease have separate types/fields. Stored evidence cannot become permission by being displayed. Saved observation is explicitly a snapshot. Caller inputs contain no paths or executable choices. Service restoration is list/read plus exact receipt refresh.
- **Correctness / concurrency / compatibility:** persistence precedes effects, independent worker snapshots remove aliasing, exact replay never redispatches, terminal rejection requires positive controller evidence and a subsequent receipt query. Older schemas stay inspectable and cannot be converted. Existing launch and service regression suites pass.
- **Testing / verification / delivery:** early failures, repairs and exact final commands are recorded. Source hashes distinguish owned edits from the inherited dirty tree. Generated v27 has a canonical TypeBox owner and generation check. No packaging, full-module, UI-comprehension or real-tool success claim is made.
- **Complexity / reuse / performance / overengineering:** existing ownership, worker lifecycle, checksum storage, runtime runner and IPC registration patterns are retained. Bounds are explicit. No new dependency or speculative optimization. Representative performance and real-tool behavior are unmeasured and deferred, not inferred from mocks.
- **Usability / operations:** unknown versus proved-rejected acknowledgement is visible. Old-engine unavailability has an explicit reason. Actual Flow/Chat controls, busy/freshness presentation, exact references and accessible interaction remain the next accepted scope.

Retain this verified uncommitted tree. Next owner checkpoint: review this transport increment, then implement the approved shared Flow/Chat continuation review before real engine qualification. The implementation guide and README record unit definitions and ownership; no material design decision was reopened.
