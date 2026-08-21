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

- Reuse valid cached outputs and run only changed work, or run one named task by itself. Until `cache-inputs` is accepted, reuse only when reservedIdentity, command or script, params, env, authored image string, recorded input cheap keys, and published dest cheap keys all match; otherwise treat that task and its downstream dependents as affected. Changed work is tasks whose reuse check failed. Resources and image digest are not identity.
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

- Statement: Library run operation. CLI for the same operation is required at first-horizon exit. Shipped `gobble run --workspace DIR [--cap N] [--sample PATH] [PKG]`; SIGINT/SIGTERM cancel ctx; occupancy stays; contract in [architecture/system.md](../architecture/system.md) Interfaces Current. Scheduler-to-executor seam is `Executor` Submit/Poll/Cancel/Reconcile.
- Current: `Run(ctx, *Graph, workspace, cap)` occupies an existing caller workspace after checks, writes an occupancy owner record under `.gobble/run.json`, isolates tasks, stages plan-relative paths, runs process and Docker tasks, and persists state. Empty Image is process; non-empty Image is docker. When `ctx` is done, in-flight work is canceled, identities persist incomplete, occupancy stays active, and the error is `canceled`. Readiness uses plan `wait` paths only. File wait paths are regular files. Tree wait paths are a directory plus dest `.gobble-tree.json`. Group members stage and publish by name. Tree publishes the declared directory of regular files plus dest `.gobble-tree.json`. Isolate restage records `IO.Source` when dest differs from the authored From path; empty Source keeps Path as both. Stage is hardlink, then process-only symlink, then copy. Docker stage skips symlink. Publish is hardlink then copy; never symlink. Cheap keys (size, mtime, dev, inode) of each consumed workspace input are recorded at task success (R1). Dest cheap keys and content digests are recorded after publish. Inspect remaining and classifyReuse compare those cheap keys and do not hash bytes. Dest cheap mismatch uses `output-missing`. Image digest is recorded on the docker attempt and is not identity. `runDockerCLI` env is `PATH=/usr/bin:/bin`. Task Env applies as docker `-e` only. Non-zero CPU and Memory emit `--cpus` and `--memory` and consume remaining admission; zero is unspecified. Occupancy is active while the owner record is open. A second `Run` while occupy is active is `occupied-workspace`, not resume. After `Release`, occupancy is closed, `run.json` stays, and a later claim uses a lock file plus owner record; leftover dests are still `output-exists`. Cap `0` means `1`. Cap above `64` is refused. Pre-existing declared outputs are `output-exists`. Isolate-keep. Host env is forbidden. Docker `--network=none`. Scheduler maps and `tasks.json` key `reservedIdentity`. Isolate is `.gobble/tasks/<id>/<instanceSeg>/<shard>/<attempt>/work` with empty instance `_`, shard `0`, attempt starting at `1`. Shipped `gobble run --workspace DIR [--cap N] [--sample PATH] [PKG]`; SIGINT/SIGTERM cancel ctx; occupancy stays; contract in [architecture/system.md](../architecture/system.md) Interfaces Current. The synthetic fixture is `testdata/run-local/` (`alpine:3.21` plus process). Live WGS, run-local recover, RNA, and Methyl live Runs are `tests/local-e2e`. The workflow-case golden `testdata/workflow-case/plan.json` records `wait` on task-to-task edges.
- Source: `interfaces`

## Constraints and qualities

- Statement: Local machines and containers only. Docker is required for first-horizon success. Agent-operability and recoverability win. Resource awareness is CPU, memory, and concurrency only.
- Source: `constraints`, `quality-priority`

## Open questions

| Id | Question | Blocking | What would resolve it |
|---|---|---|---|
| cache-inputs | Which long-term inputs participate in reuse? First-horizon workspace rule is recorded under Alternate. | no | An accepted long-term cache fingerprint rule |
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
