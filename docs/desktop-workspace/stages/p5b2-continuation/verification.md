# P5B-2 verification and handoff

Date: 2026-09-27, Asia/Seoul. Subject: paths in `changed-paths.json`, relative to `before.json`, with final hashes in `after.json`. All P5B-1 production hashes matched at entry. Inherited modifications and untracked files were retained. No staging, commit, publication or engine-image replacement occurred.

## Environment and commands

Host: Go 1.27.1 darwin/arm64; module directive Go 1.26. Engine verification uses the existing Linux amd64 toolchain image `gobble-p2a1-toolchain:local`, image ID `sha256:bccd458a724b795ac507e3dd0be43e8431269db2819fa544784d73dc751ba22b`, Go 1.26.8. Docker Desktop was started for local verification. Test containers have no network or Docker socket. Source and module caches are mounted read-only; build output is in the disposable `/tmp/gobble-p5b2-go-cache`. No dependencies were downloaded.

Final affected-package run, exit 0, recorded in `verified-linux-race.log`:

```sh
docker run --rm --platform linux/amd64 --network=none \
  --mount type=bind,src=/Users/hahyeon/.codex/.chatgpt-projects/g-p-6a9bae3a3ef481919aec253b9710060a/gobble,dst=/source,readonly \
  --mount type=bind,src=/Users/hahyeon/go/pkg/mod,dst=/gomod,readonly \
  --mount type=bind,src=/tmp/gobble-p5b2-go-cache,dst=/tmp/go-build \
  --workdir /source --env GOTOOLCHAIN=local --env GOPROXY=off \
  --env GOMODCACHE=/gomod --env GOCACHE=/tmp/go-build \
  --entrypoint /usr/local/go/bin/go gobble-p2a1-toolchain:local \
  test -race -count=1 ./internal/engine ./internal/engine/exec \
  ./internal/preparation . ./internal/appservice \
  ./cmd/gobble-service ./cmd/gobble-container
```

| Package/check | Final result |
| --- | --- |
| Engine, including existing Run/Resume/Stop and new continuation tests | Pass, 19.271 s |
| Executor, including read-only submission absence | Pass, 1.180 s |
| Public Gobble API | Pass, 19.940 s |
| App service | Pass, 3.920 s |
| Container launcher | Pass, 1.091 s |
| Preparation and service command | Compile; no test files |
| CLI cross-build | Pass; `cli-build.log` |
| Scoped vet with existing composite-literal exception | Pass; `vet.log` |
| Scoped Go formatting and `git diff --check` | Pass |

Host commands for the final CLI build and vet:

```sh
GOTOOLCHAIN=local GOPROXY=off GOOS=linux GOARCH=amd64 go build -o /tmp/gobble-p5b2-cli ./cmd/gobble
GOTOOLCHAIN=local GOPROXY=off GOOS=linux GOARCH=amd64 go vet -composites=false ./internal/engine ./internal/engine/exec ./internal/preparation . ./cmd/gobble
```

The vet exception remains specific to the two inherited unkeyed literals documented in P5B-1. This is not a clean whole-repository vet or packaging claim. Full module tests were not repeated: the unchanged source-policy and emulated memfd failures are retained in [P5B-1 verification](../p5b1-history/verification.md). The native macOS engine build still depends on Linux-only container ownership code; no fake macOS implementation was introduced.

## Direct evidence

- Real scheduler → durable Stop → read-only review → continuation of the same Run, with deterministic tool execution. Successful Trim stays attempt 1, stopped FastQC becomes attempt 2. A Trim stop restarts Trim at attempt 2 while never-started FastQC stays attempt 1.
- Read-only review success and refusal preserve all workspace file contents. The executor's separate daemon double rejects a remaining container, wrong daemon and unbound runtime identity without invoking removal, cancellation or log copying.
- Changed input, same-size/same-mtime changed output, missing output, partial output, missing tool, uncertain backend, failed/noncanceled task, duplicate attempt, changed plan and history limit all refuse. Old review admission schedules no task and produces no receipt.
- Canceled intent, active lock, old admission schema, altered payload and different workspace refuse without modifying Run state. A changed checkpoint invalidates the review.
- Exact intent replay during and after execution schedules nothing. A competing request cannot acquire the occupied Run. Request/body mismatch fails. Original Start replay does not execute again.
- Two consecutive Stop/Continue cycles preserve Run identity and attempts 1, 2 and 3. An old Stop reports `owner-changed`; the new Stop settles the current lease. Earlier receipt lookup still returns its exact admission.
- Staged-input drift after review is detected before Submit. A destination appearing after admission fails publication without overwriting it. A deterministic install collision verifies rollback removes its own earlier output and preserves the competing writer's file.
- Existing checkpoint fault injection before pointer publication yields no receipt; failure after publication retains the receipt. Replaying that committed intent after simulated controller loss never schedules. This is checkpoint-boundary recovery evidence, not a live process-kill qualification of the complete public admission call.
- Public review, execution and receipt lookup all reject an actual runtime binding different from the accepted engine or daemon.

## Development failures and corrections

Early fixture runs did not simulate executor cancellation settlement, so Stop stayed requested until timeout. The fake executor now records cancellation and reports the backend as stopped. Another test used an existing helper that accepts historical runtime IDs, so it raced the new lease; it now waits for the new admission and submission. The old-Stop assertion initially expected an error instead of the existing `owner-changed` result; the assertion now tests that documented contract. These were fixture/expectation defects, not relaxed production guards. Intermediate failures remain in `focused-final.log` and `race-focused.log`; the mixed initial `focused.log` is not authoritative evidence.

The first race invocation could not mount an absent cache directory. A stage-specific temporary cache was created and the command rerun. Earlier passing aggregate logs precede the final output-collision repair. The final aggregate log covers all production/test changes including that repair, and is the result reported above.

## Execution self-review

Applied the current Coding Execution skill and its Coding Review checklist categories. This is an implementation self-check, not an independent review verdict.

- **Project fit / affected surfaces / structure:** bounded unchanged-design continuation; reviewed reach now includes exclusive publication rollback. Preparation values, engine review/admission, executor absence check and public boundary have separate owners. No App or CLI feature was enabled prematurely.
- **Architecture / patterns / abstraction / modularization:** read-only evidence and execution authority remain separate; existing ownership lock, atomic checkpoint and scheduler are reused. No recovery framework, plugin system, extra journal or source editor was added. The publication test seam is per-call and follows the existing checkpoint fault-boundary approach.
- **Data / API / parameters / naming:** origin, review digest, preceding head, request and current lease have distinct meanings. Receipts acknowledge admission, not success. Replays never imply retry authorization. Caller-facing functions document ordering and runtime trust. Continuation does not accept caller-specified reuse decisions.
- **Correctness / concurrency / compatibility:** recheck under lock precedes admission; staged bytes precede submission; exclusive publication preserves a conflicting destination. Existing prepared Start, old-format inspection and generic Resume guards remain covered by regression tests. The old schema cannot be converted implicitly.
- **Testing / verification / delivery:** final race suite, CLI build, scoped vet, file hashes and dirty-tree preservation are recorded. Early failures and environment limitations remain explicit. There is no native-tool, UI, packaging or release claim.
- **Complexity / readability / reuse / overengineering:** bounded qualified graph, ordinary typed Go values and existing mechanisms; no new dependency. Exact content hashing is required correctness work; no performance improvement is claimed. Large-input review latency remains a later real-tool measurement.
- **Usability / operations:** proposed Flow/Chat interaction remains unchanged. Engine refusal and receipt semantics can support truthful UI states, but typed native error mapping, capability absence, pending-intent restoration, accessibility and shared immutable UI references remain integration obligations.

## Remaining evidence and acceptance boundary

Real Docker/tool execution, actual log preservation, Electron visual review, Agent references, application restart and capability-bearing image qualification remain untested in this increment. No private research data, credentials or provider calls were used. Output publication is not a defense against arbitrary adversarial concurrent filesystem replacement; operations assume the existing trusted-workspace model.

Stop at the engine stage review. Next: native service and Electron Main contracts, capability negotiation and durable intent/receipt restoration, followed by the approved shared visual review and actual Linux end-to-end qualification. App Resume remains disabled until those dependencies are complete.
