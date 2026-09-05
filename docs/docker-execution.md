# Docker execution and recovery

Both planned installation routes use the same engine: a containerized runtime
for beginners and direct Linux installation for advanced users. This document
describes the execution protocol implemented in the development branch. It does
not claim the runtime image or native launcher is already distributed.

## Before a task starts

| Step | Durable evidence before proceeding |
|---|---|
| Admit an attempt | A unique submission token in a complete task checkpoint |
| Select the local Docker engine | The Unix-socket endpoint and daemon ID in a checkpoint |
| Create a stopped container | A deterministic name and `io.gobble.submission` label derived from the token |
| Acknowledge creation | The container ID and owning engine in a complete checkpoint |
| Start the container | The scheduler has acknowledged successful publication of that checkpoint |

The adapter uses `docker create` followed by `docker start`. Docker documents
that creation does not start the container. [Docker create reference](https://docs.docker.com/reference/cli/docker/container/create/)

The adapter requests checkpoints through a per-job callback; the scheduler
remains the single writer. It does not acknowledge a failed write. Initial
admission failure now prevents submission entirely, including process tasks.

A task's internal `submission.created` flag means its created ID was recorded
and a start may be issued. It does not prove the command started or succeeded.
Task outcomes and output checks remain separate.

## What happens after interruption

| Boundary | Recovery behavior |
|---|---|
| Before an engine/creation intent is acknowledged | No Docker create was authorized |
| During creation, before the ID is recorded | Search the owning daemon by the submission label; that request cannot have started a task |
| After recording the ID, before/during start | Resolve the exact container on the owning daemon and settle it before permitting a new attempt |
| After start returns but before task completion is recorded | The container ID was already committed; recovery can locate and stop it |
| After removal but before the task checkpoint is updated | A successful listing on the owning daemon proves absence; an error does not |
| Docker is unreachable, its ID changed, or ownership cannot be verified | Keep the run unresolved and refuse to authorize reruns |

Recovery removes the exact owned container, including a container still in
`created` state. Removal is the barrier against a delayed start request. It
does not start or adopt the old workload. The ordinary Resume path creates a
new attempt only after Release settles the previous owner.

A create request still being processed when the controller dies can leave an
**unstarted** container after an earlier recovery listing found no match. It
cannot start through this protocol: its creator is gone and no durable-ID
acknowledgment occurred. The attempt token remains recorded so the container is
identifiable. Reconciliation removes it when observed; continuous cleanup of
late unstarted containers is not implemented yet.

Old attempts without submission metadata retain the legacy conservative
recovery behavior. This change does not bypass execution-identity checks or the
existing foreign-host ownership gate. The container launcher still needs a
stable controller identity and tested shared mounts for replacement containers.

## Docker selection and logs

The client uses the selected local Docker context, or `DOCKER_HOST` when no
context override applies, and persists the resulting Unix-socket endpoint.
Subsequent submission and recovery calls bind to that endpoint. Daemon identity
is checked before creation/start and during recovery. Remote TCP/SSH backends
are outside the current supported execution contract.

The client environment includes host `PATH`, `HOME`, `XDG_RUNTIME_DIR`, and
selected `DOCKER_*` configuration variables. Declared task environment values
do not configure the client. Endpoint selection is consistent with Docker's
[context precedence](https://docs.docker.com/engine/manage-resources/contexts/).

New handles carry the controller's attempt directory. Log collection uses that
local path instead of interpreting a daemon-host bind source as a controller
path. A following Docker log client writes stdout/stderr while the task runs;
settlement joins that collector before gathering final logs and removing the
container. The monitor only reads these files. Docker polling has a 500 ms interval to avoid launching client
processes at 50 Hz per task; process-task polling remains independent.

## Validation boundary

- Unit tests verify intent/ID acknowledgment order, write failures, cancellation
  before start, daemon replacement, endpoint changes, ownership mismatch,
  observation loss, and failure to remove a container.
- Engine tests terminate a separate controller process before/after create,
  before/after start, and after removal. They use a persisted daemon model and
  exercise actual checkpoint publication, Inspect, Release, Resume, input
  staging, output publication, and attempt accounting.
- The five assay scenario suites share a create/start lifecycle fixture while
  retaining assay-specific command, input, and output checks.
- `TestDockerLiveControllerDeathRecovery` uses a real Docker daemon and is wired
  into the Docker CI smoke job. It is compiled locally but **not executed in
  this environment**, which has no Docker CLI/daemon.

Windows, Docker Desktop mounts, controller-container replacement, physical
power loss, daemon-side processing races, and runtime-image installation still
require their real-environment gates. Mocked daemon tests do not establish
those platform guarantees.
