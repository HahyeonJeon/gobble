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
| Packed Hello run/release/resume with Go absent from PATH | Passed from clean commit `e60d328`; both runs produced `2` |
| Documentation local links, workflow YAML parsing, diff whitespace | Passed |
| Docker image builds, real Docker smoke, Windows/WSL | Not executed: Docker and Windows/WSL are unavailable here |

The Docker infrastructure from batch 5 has been prepared early so it can be
reviewed now; its environment coverage is still pending execution. Batches 2–4
and the remaining release gates require the design choices in D1–D3. In
particular, the interrupted-checkpoint defect and missing live Docker logs are
not fixed by this first batch. No release, merge, or push is implied by these
local results.

## Checkpoint publication delivery record (batch 2a)

The recommended D2 and D3 designs are approved. The user subsequently accepted
[Docker installation](container-installation.md) as the default beginner route,
with direct Linux installation for advanced users.

Batch 2 is split at its two independent failure boundaries. This section records
2a, checkpoint publication. The later 2b implementation record follows below;
the real Docker release gate remains separate from hermetic tests.

Implemented in 2a:

- Immutable plan/tasks/run generations, file and directory sync, and one atomic
  current pointer. Readers pin the selected generation while retention removes
  obsolete generations. Current and previous generations are retained.
- A compatible legacy-layout reader and conversion on the next allowed write.
  Original flat controls are preserved as evidence. Identity compatibility
  remains enforced; old files do not authorize a downgrade or stale fallback.
- Scheduler writes and Release use the same checkpoint publication mechanism.
  Corrupt committed records are refused even if valid old flat files remain.
- Separate writer-process death tests at five publication boundaries, followed
  by Inspect, Release, and Resume. Completed work remains on attempt 1 and is
  reused. Returned write failures, concurrent reading/retention, legacy
  conversion, and escaping/corrupt committed state are also exercised.

Verification on local Linux/amd64 with Go 1.26.0:

| Check | Result |
|---|---|
| Full `go test -count=1 ./...`, local toolchain, proxy disabled | Passed: 79 packages with tests |
| Race checks for engine, executor, monitor, and TUI | Passed; checkpoint race cases rerun after final reader changes |
| `go vet ./...` | Passed |
| Live engine and `tests/install-e2e` suites compiled | Passed; no real Docker execution claimed |
| Process death during checkpoint publication, then Inspect/Release/Resume | Passed at all five boundaries |
| Generic Hello from clean commit `ea60c6c`, workspace with spaces and Unicode | Passed Run/Release/Resume/Inspect/Release; output `2`, missing-output rerun on attempt 2, coherent state and two retained generations |
| Packed Hello from `ea60c6c`, Go absent from PATH, workspace with spaces and Unicode | Passed the same lifecycle and state assertions |
| Docker controller distribution / Windows / physical power loss | Not executed |

See [checkpoint storage](../checkpoints.md) for the on-disk contract. Neither
checkpoint changes nor the Docker submission protocol alone delivers the image,
native launcher, or a validated Windows installation.

## Docker submission delivery record (batch 2b)

The user has confirmed the installation contract: Docker plus a small launcher
is the default for agent-driven pipeline design and execution; direct Linux
CLI/library installation remains the advanced route. These routes share the
engine rather than maintaining separate schedulers or pipeline formats.

Implemented:

- Durable attempt tokens and Docker endpoint/daemon identity. The scheduler
  acknowledges a created container ID in a complete checkpoint before the
  adapter may issue start. Failed admission checkpoints prevent all submission.
- Separate Docker create/start operations, deterministic names and ownership
  labels, bound endpoint selection, and refusal on daemon/ownership mismatch.
- Reconciliation fences a possibly outstanding start by removing the exact
  owned container. An observation error is not treated as container absence.
- Recovery can retry removal of a container retained after a successful task,
  preserving its successful outcome. This has a regression test.
- New handles use controller-local log paths. Docker polling uses 500 ms rather
  than the process adapter's 20 ms interval. Continuous logs remain batch 3.
- A shared Docker lifecycle fixture for all five assay scenario suites. Assay
  command matching, input facts, and expected outputs remain in their owners;
  fixture execution occurs at start, not create.
- Separate-process death/recovery cases at five submission boundaries, plus
  acknowledgment failure, daemon replacement, lost observation, and removal
  failure tests. Real-Docker controller death tests are added to the CI smoke
  job and compile locally; execution requires an actual Docker host.

See [Docker execution](../docker-execution.md) for the protocol and limitations.
Verification on local Linux/amd64 with Go 1.26.0:

| Check | Result |
|---|---|
| `go test -count=1 -timeout 5m ./...` | Passed: 79 packages with tests |
| Final `go test -race -count=1 ./internal/engine/...` | Passed after separating actual running state from the removal barrier |
| `go vet ./...` | Passed |
| Live engine and installed-runner suites compiled with `-tags=live -run '^$'` | Passed; compilation only |
| Local documentation links and `git diff --check` | Passed |
| Real Docker, image builds, Docker Desktop, Windows | Not executed: no Docker daemon or Windows host in this environment |

A daemon-side late create can leave a recorded, unstarted container; continuous
cleanup of those leftovers is still pending. Stable controller identity,
Docker Desktop mount translation, Stop/Resume UX, live logs, doctor/init, and
the runtime image/launcher remain subsequent implementation work. Real Docker
and Windows results must be obtained before promoting installation support in
the README.

## Lifecycle and installation preview delivery record (batches 3–4)

Implemented after batch 2b, using the accepted shared-engine design:

- Public `Stop(ctx, workspace, ...)`, generic and packed `stop` commands, and
  durable stop requests addressed to one owner lease. A settled explicit Stop
  closes that owner's lock. A canceled wait reports requested, and repeated
  requests are safe. A stale request cannot target a resumed owner.
- Resume reconciles and acquires its new lease under one continuous mutation
  lock. A live scheduler is refused; an exited scheduler no longer requires a
  separate Release. Existing identity/backend proof gates remain enforced.
- Run-level stopping/stopped/interrupted outcomes and a separate inspected
  backend certainty field. Task-level outcomes retain their existing vocabulary
  in this batch. Process death during a running/stopping outcome projects as
  interrupted with recovery required.
- One live Docker log collector per running attempt, separate stdout/stderr,
  and collector shutdown before final collection/removal. The monitor stays
  read-only. The real-Docker CI journey tests live output before Stop.
- A Go runtime image build and a standalone Linux/Windows launcher. It binds a
  selected local daemon, pins the exact runtime image ID per project, uses a
  stable daemon-derived controller hostname, translates host path arguments,
  and maps task mounts from Docker's actual controller mount descriptions.
- Doctor checks tools and, in the runtime, sibling-container read/write access.
  Init creates a teaching pipeline, input data, agent guidance, and a local Git
  commit, refusing existing directories. A generated Linux project was run and
  resumed without Release; output remained `2`.
- README now leads with Docker as the default route and links the concrete
  preview build/setup instructions. Direct Linux installation remains the
  advanced route. No unavailable release download is presented as installable.

Verification:

| Check | Result |
|---|---|
| Full `go test -count=1 -timeout=5m ./...` | Passed: 80 packages with tests |
| Engine/executor/monitor/TUI/launcher race checks | Passed; engine/executor rerun after final stale-Stop and refused-Resume lock fixes |
| Generic CLI + setup + packed-template and launcher tests under race | Passed |
| `go vet ./...` | Passed |
| Native Windows x64 launcher cross-build | Passed; not execution on Windows |
| Separate-process Stop of a live process task, repeated Stop, Resume attempt 2 | Passed |
| Docker controller death followed directly by Resume | Passed at five boundaries using the persisted daemon model |
| Generated project init/run/inspect/resume/stop | Passed on local Linux; result `2` |
| Runtime Docker image builds and launcher integration script | Prepared in CI, not executed locally |
| Docker Desktop, PowerShell interaction, Windows permissions/restart | Not executed |

Remaining release gates: run the real Docker image/launcher journey and Windows
Desktop journey; pin base/runtime image digests for publication; produce release
checksums/installers; measure cached-command startup and realistic pipeline load.
Remaining design work includes detached supervision, large external datasets
without extra staging copies, cleanup of daemon-side late creates, and shared
CLI/packed handler extraction. These records do not mark all v0.2.0 work complete
or imply a release, merge, or push.

### Installed lifecycle verification from `7b3c104`

Built the generic CLI and packed runners from this clean commit. These checks
use real Linux processes, temporary projects/workspaces with spaces and Korean
characters, and actual checkpoint/output files:

| Installed command | Demonstrated journey |
|---|---|
| Generic CLI, generated teaching project | Init; observe process stdout while running; Stop; repeated Stop; Resume without Release; successful attempt 2 and result `2` |
| Packed Hello, no Go on PATH | Run; automatic Resume reusing completed work; remove one result; Resume on attempt 2 with result `2`; Stop and repeated Stop |
| Packed long-running process, no Go on PATH | Run; Stop while active; Resume on attempt 2; Stop while active; both attempts recorded incomplete and both Stop calls settled |

The example pipeline intentionally runs inside a local process. These installed
results verify the CLI/packed protocol and lifecycle, not a Docker runtime or
Windows host. The runtime integration gate remains unexecuted locally.

### CI environment regression fixes after the develop merge

The first develop CI runs exposed three test-environment assumptions:

- ATAC/Methyl lifecycle doubles inherited analysis budgets exceeding the CI
  host's CPU capacity. Stop scenarios then waited for a start signal even when
  Run had already returned a preflight error, masking it with a ten-minute
  timeout. All five assay runtimes now use explicit small fixture resources;
  production defaults stay covered by plan assertions. Stop tests share a
  bounded wait that also reports early Run errors with their defects.
- The hermetic Docker image ran as root, so chmod-based no-read assertions
  failed because root could still read the files. The image now uses an
  unprivileged user with access to its verified dependency cache.
- The packed-runner preservation test intentionally creates a consumer without
  `go.sum`. It now explicitly disables fresh checksum-database lookups in that
  fixture, after cache preparation has verified the dependencies. Normal build
  and product checksum verification remain enabled.

The Docker userspace matrix now pins containers to one available CPU, with
network access disabled, to retain coverage of small hosts.

Local verification: the complete suite passed all 80 test-bearing packages;
all lifecycle scenario packages passed with affinity restricted to one CPU;
the packed-consumer preservation test and `go vet ./...` passed. Improved
Methyl failure diagnostics were also rerun on one CPU.

The original [Docker CI run](https://github.com/HahyeonJeon/gobble/actions/runs/33964259175)
already passed both actual-container jobs (`docker-smoke` and `runtime-install`),
including the no-host-Go launcher journey, live logs, Stop, repeated Stop,
controller death, and automatic Resume. Its userspace matrix failed for the
assumptions above. Windows Desktop and release-publication gates remain open.

The follow-up Docker matrix passed those scenarios and exposed two public
Resume tests sharing a resource-only-change fixture that still requested two
CPUs. The fixture now changes CPU/memory within a one-CPU budget and is named
for its purpose, rather than calling the request "heavy". The new CI restriction
is retained. Both failures were reproduced locally with one-CPU affinity.
After correction, the entire public API test package passed on one CPU.

Additional Stop scenario race verification passed in 439.648 seconds. An
initial three-minute test timeout was too short for the large ATAC graph's
Resume checkpoint serialization under race instrumentation; the same suite
completed with a twelve-minute bound.
