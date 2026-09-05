# v0.2.0 codebase review

Reviewed on 2026-09-05 against `develop` commit
`ef06b9cb105962cf29087d9a874fb84093813c80`.

Goal: a person unfamiliar with bioinformatics, including a Windows user, can
install Gobble locally, work with a coding agent to design a pipeline, execute
it, understand progress, stop it, and recover work without editing engine state.
Public API and CLI changes are permitted. The design proposals below are still
subject to discussion; permission to change public behavior is not a decision
to adopt a particular architecture.

## Assessment

The graph, artifact, identity, selective reuse, and cancellation machinery is a
useful foundation. The monitor already separates data projection, aggregation,
and terminal presentation. Keep these boundaries. The largest v0.2.0 gaps are
recoverable persistence, the user-facing lifecycle, environment discovery, and
onboarding. Moving files alone will not address them.

The baseline has 440 Go files, including 201 test files. Many module tests live
under `tests/modules`, so a package reporting `[no test files]` is not evidence
that the corresponding module is untested.

## Findings

P0 blocks the intended recovery guarantee; P1 blocks a core user journey;
P2 concerns maintainability or usability. Locations refer to the baseline.

| ID | Priority | Finding and evidence | Required improvement |
|---|---|---|---|
| R01 | P0 | `internal/engine/run.go:persistControl` replaces `plan.json`, `tasks.json`, and `run.json` separately. An interrupted write leaves different snapshot IDs. `readCoherentControl` rejects them, and both Inspect and Release use that reader. | Transactional checkpoint publication and explicit interrupted-write recovery. See decision D2. |
| R02 | P1 | `exec/docker.go:Poll` returns immediately for a running container. `writeDockerLogs` runs only from `finishStopped`. The monitor reads local log files, so ordinary Docker task logs are unavailable while the task runs. | Controller-owned log collection with bounded shutdown and a live Docker regression test. |
| R03 | P1 | `dockerClientEnv` supplies only `PATH=/usr/bin:/bin`. Docker context/host/config environment overrides are dropped. Executable lookup nevertheless uses the parent PATH through `exec.CommandContext`, so the documented lookup restriction is not actually enforced either. | One explicit Docker client configuration shared by discovery, execution, monitoring, and recovery; record daemon identity before starting work. |
| R04 | P1 | No public Stop command. CLI Run/Resume forward SIGINT/SIGTERM; a separate process cannot stop the owner through Gobble. Release rejects a live owner. | A controller stop request, distinct from lock release; do not implement Stop as blindly signaling a recorded PID. |
| R05 | P1 | Cancellation updates task records, but the canceled branch of `sched.loop` does not finalize run status/end time. It also omits a recorded persistence error from its returned defects. | Report persistence failures immediately; decide run-level stopped/interrupted vocabulary with D3. |
| R06 | P1 | Native Windows compilation fails on Unix process groups and no-follow/stat APIs. macOS compilation fails on `syscall.Sysinfo`. `ValidateInstallIdentity` accepts only Linux/amd64, and `packedBuildEnv` forces that target. | Choose an explicit platform promise before portability work. A successful cross-build alone is insufficient. |
| R07 | P1 | Generic graph commands resolve a consumer module, synthesize a Go driver, and compile it for each invocation. Current onboarding requires manual `go mod edit`, a replacement path, and a matching CLI. No `init`, environment doctor, or small beginner example exists. | A first-run tutorial and installer/project bootstrap that handles the authoring toolchain and identity consistently. |
| R08 | P1 | Only Ubuntu hermetic tests and selected race checks run in CI. Live install/recovery tests are behind `live`. Product scenario runtimes substitute Docker calls and generate outputs. | Separate hermetic, actual Docker, installed-binary, and Windows/WSL evidence. Never equate simulated product execution with scientific or container validation. |
| R09 | P2 | `internal/engine/run.go` has 2,537 lines, mixing scheduling, resource accounting, state documents, checkpoint writes, expansion, and conditions. | First split responsibilities within the same package; then introduce stronger boundaries only with behavioral evidence. |
| R10 | P2 | Generic CLI parsing/dispatch and generated packed-runner parsing/dispatch are separately maintained (`parse.go`, `pack_inner.go`, `help.go`). | A shared internal command model and handler layer with generic/packed parity tests. Keep pipeline loading as an adapter. |
| R11 | P2 | `defaultHostCapacity` uses host CPU count and `syscall.Sysinfo` memory, not the Docker daemon's capacity or container memory limit. Every active task also polls Docker through a CLI subprocess after a fixed 20 ms delay. | Measure Docker polling load; discover execution capacity rather than assuming client-host memory equals available execution memory. |
| R12 | P2 | Terms such as product, assay, module, reserved identity, occupancy, release, and recovery actor appear before users understand pipelines and runs. The README starts with exclusions and release archaeology. | A short landing page, one runnable example, a plain-language glossary, and advanced contracts in dedicated guides. Keep wire-format renames explicit. |
| R13 | P2 | `TestRunDockerPublishes` requires a nonempty runtime ID after success even though the Docker adapter clears it after successful removal. The live suite is not a required CI gate. | Correct the assertion to the current cleanup contract and run the real backend suite in CI. |
| R14 | P1 | Image acquisition happens after occupancy, during task submission, and has no separate preparation progress surface. Inputs and reference files must be staged manually. | A preflight/preparation experience with actionable missing-input/image/resource information. Keep requested analysis choices explicit; do not infer reference genomes silently. |

## Reproduced evidence

- Baseline `GOTOOLCHAIN=local GOPROXY=off go test -count=1 ./...`: passed,
  using Go 1.26.0 on Linux/amd64 after dependencies were already present.
- Baseline `go vet ./...`: passed.
- A temporary review probe ran a small pipeline, replaced only the plan's
  snapshot ID (the state immediately after the first write of a new checkpoint),
  and invoked Inspect and Release. Both returned `invalid-path: control snapshot
  is not coherent`. This simulates a precise write boundary, not a hardware
  power-loss experiment. The probe is not retained as a test blessing this bug.
- Windows/amd64 and macOS/arm64 compile probes fail at the platform APIs listed
  in R06. VCS stamping was disabled for these compile-only probes.
- Docker CLI and daemon are unavailable in this environment. Actual container,
  Docker Desktop, WSL, and installed live recovery execution remain unverified.

## Scope boundaries

Implementation update: R01's mixed-snapshot publication defect is addressed by
[checkpoint generations](../checkpoints.md) and process-death recovery tests.
The separate Docker submit-to-runtime-ID gap still needs submission intent and
reconciliation. See the [batch 2a delivery record](plan.md#checkpoint-publication-delivery-record-batch-2a)
for current verification; the findings table above records the reviewed baseline.

The public root Go package is an appropriate Go layout; moving it wholesale
under `src/` would break imports without solving a user problem. Keep assay
configuration and scientific facts owned by their pipeline packages. Deduplicate
test execution mechanics, not each assay's expected stages and outputs.

Keep artifact checksums, selective reruns, output validation, identity checks,
and refusal to reuse unproved backend work. Simplify how users reach these
checks. Do not obtain simpler commands by deleting the checks.

## References

- [Docker contexts](https://docs.docker.com/engine/manage-resources/contexts/)
  describe endpoint selection and environment overrides relevant to R03.
- [Docker bind mounts](https://docs.docker.com/engine/storage/bind-mounts/)
  explains why test-controller and daemon paths must agree.
- [Docker Desktop WSL backend](https://docs.docker.com/desktop/features/wsl/)
  and [WSL best practices](https://docs.docker.com/desktop/features/wsl/best-practices/)
  inform the proposed Windows route; they do not establish Gobble support.
- [uv README](https://github.com/astral-sh/uv) and
  [Nextflow](https://github.com/nextflow-io/nextflow) were reviewed as examples
  of a concise project introduction, installation entrypoint, examples, and
  links to detailed documentation.

Continue with [design decisions](design.md) and the [implementation plan](plan.md).
