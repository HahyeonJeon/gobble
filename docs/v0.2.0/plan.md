# v0.2.0 implementation and validation plan

Execute in order. A batch is complete only when its user scenario is demonstrated
and its documentation states what was actually tested.

| Batch | Work | Completion evidence | Decision dependency |
|---|---|---|---|
| 0 | Review baseline, reproduce critical gaps, define vocabulary and design options | Review report, baseline suite, platform probes, interrupted-checkpoint reproduction | None |
| 1 | Separate engine responsibilities; surface cancellation persistence failures; repair stale live assertions; rewrite README and add a small example | Unchanged declaration bodies for moves; regression test fails before fix and passes after; full hermetic suite and relevant race checks | None |
| 2 | Commit state atomically; persist submission intent and reconcile Docker ownership after crashes | Fault injection at every commit/submit boundary; death/restart recovery without duplicate work | D2 |
| 3 | Stop/Resume UX, run outcome vocabulary, shared CLI/packed handlers, live Docker logs | Installed generic and packed runs complete the same lifecycle scenarios | D3 and batch 2 |
| 4 | Docker discovery and capacity, doctor/init, versioned install/bootstrap, Windows route | Clean-machine install to first run; missing requirements produce actionable errors | D1 |
| 5 | Docker and OS matrix, release packaging, final README install instructions and screenshots | Actual matrix results and supported tuples in release notes | D1–D3 |

## Docker testing strategy

Docker can test Linux userspaces and dependency combinations. It cannot turn a
Linux runner into a Windows kernel or prove Docker Desktop/WSL integration.

| Layer | Environments | What it proves |
|---|---|---|
| Hermetic code tests | Linux/amd64; two Debian userspaces via a Go test image | Graph behavior, errors, state/reuse decisions, fake-backend scenarios |
| Real Docker smoke | Linux host with a reachable Docker daemon | Container startup, mounts, UID/GID, resources, live logs, publication, stop, failure, resume |
| Installed command | Clean consumer directory; generic CLI and packed runner without Go | Installation identity, command parity, neutral working directory, recovery |
| Windows integration | Windows x64 + WSL2 + Docker Desktop | Bootstrap, path conversion, Linux filesystem guidance, terminal close, reboot recovery |
| Additional platforms | Native macOS/Windows or ARM only if selected | Build, runtime, dependency images, and lifecycle behavior on the actual tuple |
| Assay execution | Pinned small public fixtures and pinned tool images | Tool invocation/output integration for each assay; separate from scientific validity |

For a containerized test controller using the host Docker socket, bind the test
workspace at the same absolute path on both sides and place `TMPDIR` inside that
shared mount. Otherwise Gobble's generated task paths are absent from the daemon
host. Keep host identity stable across recovery invocations. A hermetic test image
does not need the socket; do not add one by default.

Initially add a reusable hermetic test image and a separate real-Docker CI job.
The release gate must fail if a selected live suite cannot reach Docker, rather
than silently reporting a skipped test as environment support.

## Acceptance scenarios

| Scenario | Required observable result |
|---|---|
| Fresh install, no Go | Authoring setup explains/provisions the compatible toolchain; packed execution works without Go |
| Missing Docker / daemon stopped / denied access | Preflight names the missing requirement before occupying a run |
| Non-default Docker context | Selected local daemon is used consistently; recovery refuses a different owning daemon |
| Tiny teaching example | User can create, inspect, run, and find output without real sequencing data |
| Real assay input missing | Error names the sample/input path and a next step; no undocumented implicit download |
| Spaces / non-ASCII paths | Workspace and samplesheet work through generic and packed commands |
| Task failure in a branch | Successful independent work remains reusable; downstream work is blocked explicitly |
| Stop while task is running | Acknowledged stop, backend disposition, available logs, stopped outcome |
| Stop while pulling an image or submitting | No untracked container or false claim that nothing started |
| Repeated Stop / Stop after success | Idempotent outcome; existing successful work is unchanged |
| Resume after Stop | Checksummed successes reused; incomplete tasks get new attempts |
| Resume while owner lives | No second scheduler or duplicate container |
| Terminal closes / controller SIGKILL | Surviving owned jobs reconciled before restart |
| Restart during checkpoint publication | One complete committed state remains readable and recoverable |
| Disk full during cancellation | Returned error reports failure to persist; no false durable-stop claim |
| Lost Docker observation | Recovery-required state remains visible; restoration permits reconciliation |
| Modified input/config/output | Reuse explanation identifies affected work and downstream reruns |
| Upgrade Gobble then resume | Clear compatible/mismatch result; the previous required executable remains available |
| Open/close monitor repeatedly | Observer never becomes execution owner; live task output remains visible |

Use channels and fault boundaries for deterministic scheduling tests instead of
adding sleeps. Keep small contract tests near implementation; keep real install
and cross-package journeys under `tests/`. Share fake-executor mechanics while
leaving scientific fixture facts in assay-owned packages.

## First-batch delivery record

Batch 0 is complete. The first cleanup batch implements:

- A full README rewrite, an actual terminal-renderer illustration, an
  installation detail page, and an executable tiny FASTA example.
- Separation of resource budget, JSON state types, checkpoint writing, atomic
  file replacement, expansion, and conditions into focused engine files.
  `run.go` decreases from 2,537 to 1,499 lines. An AST comparison confirms all
  123 non-import declarations are preserved by the file split, after isolating
  the intentional persistence-error fix.
- Cancellation now reports checkpoint-write failures alongside `canceled`.
  The new regression failed on the baseline with only a canceled defect, and
  passes with the fix.
- The live Docker publication test now expects a cleared runtime ID after
  successful cleanup and verifies regular stdout/stderr files.
- A reusable Docker toolchain image, two Linux-userspace CI entries, and a
  separate actual-Docker smoke CI job. Build context excludes source/data;
  source and Git metadata are mounted at test runtime.
- Monitoring documentation now states that running Docker logs are not yet
  available, while process log files can grow during execution.

Verification in this environment:

| Check | Result |
|---|---|
| `go test -count=1 ./...` with local toolchain and proxy disabled | Passed: 79 packages with tests |
| `go test -race -count=1 ./internal/engine/... ./monitor/...` | Passed |
| `go vet ./...` | Passed |
| Live engine/install suites compiled with `-tags=live -run '^$'` | Passed; this is compilation, not live execution |
| Hello CLI validate/plan/run/inspect/release | Passed; output `2` |
| Hello missing-output Resume | Passed; output `2`, attempt 2, reason `output-missing` |
| Documentation local links, workflow YAML parsing, diff whitespace | Passed |
| Docker image builds, real Docker smoke, Windows/WSL | Not executed: Docker and Windows/WSL are unavailable here |

The Docker infrastructure from batch 5 has been prepared early so it can be
reviewed now; its environment coverage is still pending execution. Batches 2–4
and the remaining release gates require the design choices in D1–D3. In
particular, the interrupted-checkpoint defect and missing live Docker logs are
not fixed by this first batch. No release, merge, or push is implied by these
local results.
