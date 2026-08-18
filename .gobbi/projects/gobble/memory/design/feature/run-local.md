# run-local — Feature

Derive every heading from accepted interview topic ids. Never invent a lifecycle
answer. Fill each heading, write `Not applicable — {reason}`, or write
`Open — {what would resolve it}`. A refused use is not a feature file.

Open Actors, Scope, or Behavior on a feature that can sit in the first horizon
is blocking. Ask before leaving those Open.

## Purpose

- Statement: Schedule and execute a valid plan on the local machine. Each task runs itself, normally in its declared Docker image. Ship a local-process executor for tasks that do not require a container. Persist run state and artifacts.
- Source: `core-tasks`

## Actors

- Actor: Agent operator
- Role: Starts a local run. Does not decide task readiness.
- Actor: Human operator
- Role: Same as the agent operator.
- Actor: Gobble scheduler
- Role: Decides readiness from the DAG, per-task CPU and memory, and run-level concurrency.
- Actor: Docker
- Role: Executes a task that declares a container image.
- Source: `task-actors`

## Scope

- In scope: Local schedule and execute using per-task CPU and memory and a run-level concurrency cap; run each task itself, normally in its declared Docker image; local-process executor for tasks that do not require a container; persist run state; manage task work directories and artifacts. First-horizon success requires at least one real Docker task.
- Out of scope: Slurm, cloud batch, Kubernetes, fairness, quotas, job arrays, and treating a process-only stand-in as first-horizon success.
- Source: `task-scope`, `refused-use`

## Behavior

### Normal

- Execute a valid plan locally. Each task runs itself. A bioinformatics task such as `bwa` runs in its own declared image. Persist state until completion or a contained failure.
- Source: `task-behavior`

### Alternate

- Reuse valid cached outputs and run only changed work, or run one named task by itself. Until `cache-inputs` is accepted, reuse only when task identity, declared command or image, declared parameters, and recorded input path plus content fingerprints all match and published outputs still exist; otherwise treat that task and its downstream dependents as affected. Changed work is tasks whose reuse check failed.
- Source: `task-behavior`

### Invalid

- Submit to a non-local backend, start when required local inputs are missing, or treat a process-only stand-in as the first-horizon success pipeline. Missing inputs or a refused backend are structured pre-execution errors. No backend job starts for the refused unit.
- Source: `task-behavior`

## Failure / Recovery

### Failure

- One task or executor failure does not take down the scheduler, other tasks, or persisted run state. The consumer sees structured task state naming the failed unit.
- Source: `failure-recovery`, `failure-containment`

### Recovery

- Use recover-run to retry or resume remaining work. If Docker is down, container tasks fail contained; they are not reported successful.
- Source: `failure-recovery`

## Structure

### Parts

- Statement: Scheduler and executors (local process and Docker), plus run state and artifact files.
- Source: `shape`

### Data

- Statement: Creates and updates run state, task work directories, intermediates, and published outputs in the local run workspace. That workspace is authoritative for the run.
- Source: `data`

### Interfaces

- Statement: Library run operation. CLI for the same operation is required at first-horizon exit. CLI command names stay Open (`invocation-contract`). Scheduler-to-executor seam exists; exact operation names are not settled.
- Current: `Run(*Graph, workspace, cap)` occupies an existing caller workspace after checks, writes an occupancy owner record under `.gobble/run.json`, isolates tasks, stages plan-relative paths, runs process and Docker tasks, and persists state. Readiness uses plan `wait` paths only. Group members stage and publish by name. Non-zero CPU and Memory emit `--cpus` and `--memory` and consume remaining admission; zero is unspecified. Occupancy is active while the owner record is open. A second `Run` while occupy is active is `occupied-workspace`, not resume. After `Release`, occupancy is closed, `run.json` stays, and a later claim uses a lock file plus owner record; leftover dests are still `output-exists`. Cap `0` means `1`. Cap above `64` is refused. Pre-existing declared outputs are `output-exists`. Isolate and occupy stay. There is no CLI. Public `Inspect`, `Resume`, and `Release` are shipped. Reserved isolate is `.gobble/tasks/<id>/_/0/1/work` for empty instance, shard `0`, attempt `1`. The scheduler is still keyed by task.ID. The synthetic fixture is `testdata/run-local/` (`alpine:3.21` plus process). Consumer-test proof is `tests/wgs-e2e/` (public API only). WGS is a consumer test, not a product feature. The workflow-case golden `testdata/workflow-case/plan.json` records `wait` on task-to-task edges.
- Source: `interfaces`

## Constraints and qualities

- Statement: Local machines and containers only. Docker is required for first-horizon success. Agent-operability and recoverability win. Resource awareness is CPU, memory, and concurrency only.
- Source: `constraints`, `quality-priority`

## Open questions

| Id | Question | Blocking | What would resolve it |
|---|---|---|---|
| cache-inputs | Which long-term inputs participate in reuse? Temporary first-horizon rule is recorded under Alternate. | no | An accepted long-term cache fingerprint rule |
| dependency-unavailable | What structured status is shown while Docker is down? | no | A recorded Docker-down status shape |

- Source: open ids used above

## Interview sources

- `core-tasks`
- `task-actors`
- `task-scope`
- `task-behavior`
- `refused-use`
- `failure-recovery`
- `failure-containment`
- `shape`
- `data`
- `interfaces`
- `constraints`
- `quality-priority`
- `cache-inputs`
- `dependency-unavailable`
- Source: the topic ids actually cited
